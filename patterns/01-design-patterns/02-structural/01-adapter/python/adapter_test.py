import json
from pathlib import Path

import pytest

from adapter import (
    ChatRequest,
    LegacyProviderAdapter,
    LegacyResponse,
    Message,
    NormalizedError,
    ProviderError,
)


CONTRACT = json.loads(
    (Path(__file__).parents[1] / "fixtures" / "contract.json").read_text()
)


def contract_case(case_id):
    return next(case for case in CONTRACT["cases"] if case["id"] == case_id)


class FakeLegacyClient:
    def __init__(self, response=None, error=None):
        self.response = response
        self.error = error
        self.last_request = None
        self.calls = 0

    def generate(self, request):
        self.calls += 1
        self.last_request = request
        if self.error is not None:
            raise self.error
        return self.response


def request_from(case):
    request = case["input"]["request"]
    return ChatRequest(
        model=request["model"],
        messages=tuple(Message(**message) for message in request["messages"]),
        requires_tools=request["requires_tools"],
    )


def test_maps_shared_contract_fixture():
    case = contract_case("maps_request_and_response")
    provider = case["input"]["provider_response"]
    client = FakeLegacyClient(
        response=LegacyResponse(output=provider["output"], stop_code=provider["stop_code"])
    )

    response = LegacyProviderAdapter(client).complete(request_from(case))

    assert vars(client.last_request) == case["expected"]["mapped_request"]
    assert response.content == case["expected"]["response"]["content"]
    assert response.finish_reason == case["expected"]["response"]["finish_reason"]


def test_fails_explicitly_for_unknown_model():
    case = contract_case("rejects_unknown_model")

    with pytest.raises(NormalizedError) as captured:
        LegacyProviderAdapter(FakeLegacyClient()).complete(request_from(case))

    assert captured.value.code == case["expected_error"]["code"]
    assert captured.value.retryable is case["expected_error"]["retryable"]


def test_normalizes_provider_error():
    case = contract_case("normalizes_provider_error")
    provider_error = case["input"]["provider_error"]
    client = FakeLegacyClient(
        error=ProviderError(provider_error["code"], provider_error["message"])
    )

    with pytest.raises(NormalizedError) as captured:
        LegacyProviderAdapter(client).complete(request_from(case))

    assert captured.value.code == case["expected_error"]["code"]
    assert captured.value.retryable is case["expected_error"]["retryable"]


def test_rejects_unsupported_before_provider_call():
    case = contract_case("rejects_unsupported_before_provider_call")
    client = FakeLegacyClient()

    with pytest.raises(NormalizedError) as captured:
        LegacyProviderAdapter(client).complete(request_from(case))

    assert captured.value.code == case["expected_error"]["code"]
    assert captured.value.retryable is case["expected_error"]["retryable"]
    assert client.calls == case["expected"]["provider_calls"]

