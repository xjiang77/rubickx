from __future__ import annotations

from dataclasses import dataclass
from typing import Protocol, Sequence


@dataclass(frozen=True)
class Message:
    role: str
    content: str


@dataclass(frozen=True)
class ChatRequest:
    model: str
    messages: Sequence[Message]
    requires_tools: bool = False


@dataclass(frozen=True)
class ChatResponse:
    content: str
    finish_reason: str


@dataclass(frozen=True)
class LegacyRequest:
    deployment: str
    prompt: str


@dataclass(frozen=True)
class LegacyResponse:
    output: str
    stop_code: str


class ProviderError(Exception):
    def __init__(self, code: str, message: str) -> None:
        super().__init__(message)
        self.code = code


class NormalizedError(Exception):
    def __init__(self, code: str, retryable: bool, message: str) -> None:
        super().__init__(message)
        self.code = code
        self.retryable = retryable


class LegacyClient(Protocol):
    def generate(self, request: LegacyRequest) -> LegacyResponse: ...


class LegacyProviderAdapter:
    _MODEL_MAP = {"chat-pro": "legacy-chat-pro"}
    _STOP_MAP = {"stop": "stop", "length": "length"}

    def __init__(self, client: LegacyClient) -> None:
        self._client = client

    def complete(self, request: ChatRequest) -> ChatResponse:
        if request.requires_tools:
            raise NormalizedError(
                "unsupported_capability",
                False,
                "legacy provider does not support tools",
            )

        deployment = self._MODEL_MAP.get(request.model)
        if deployment is None:
            raise NormalizedError(
                "invalid_request",
                False,
                f"unknown model: {request.model}",
            )

        legacy_request = LegacyRequest(
            deployment=deployment,
            prompt="\n".join(
                f"{message.role}: {message.content}" for message in request.messages
            ),
        )

        try:
            legacy_response = self._client.generate(legacy_request)
        except ProviderError as error:
            raise self._normalize_error(error) from error

        return ChatResponse(
            content=legacy_response.output,
            finish_reason=self._STOP_MAP.get(legacy_response.stop_code, "unknown"),
        )

    @staticmethod
    def _normalize_error(error: ProviderError) -> NormalizedError:
        if error.code == "OVER_QUOTA":
            return NormalizedError("rate_limited", True, str(error))
        if error.code == "BAD_REQUEST":
            return NormalizedError("invalid_request", False, str(error))
        return NormalizedError("upstream_failure", True, str(error))
