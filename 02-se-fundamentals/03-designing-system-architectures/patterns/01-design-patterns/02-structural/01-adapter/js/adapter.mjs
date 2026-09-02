export class ProviderError extends Error {
  constructor(code, message) {
    super(message);
    this.code = code;
  }
}

export class NormalizedError extends Error {
  constructor(code, retryable, message) {
    super(message);
    this.code = code;
    this.retryable = retryable;
  }
}

export class LegacyProviderAdapter {
  constructor(client) {
    this.client = client;
  }

  async complete(request) {
    if (!request || !Array.isArray(request.messages)) {
      throw new NormalizedError(
        "invalid_request",
        false,
        "request.messages must be an array",
      );
    }
    if (request.requiresTools === true) {
      throw new NormalizedError(
        "unsupported_capability",
        false,
        "legacy provider does not support tools",
      );
    }

    const deployment = { "chat-pro": "legacy-chat-pro" }[request.model];
    if (deployment === undefined) {
      throw new NormalizedError(
        "invalid_request",
        false,
        "unknown model: " + request.model,
      );
    }

    const legacyRequest = {
      deployment,
      prompt: request.messages
        .map((message) => message.role + ": " + message.content)
        .join("\n"),
    };

    let legacyResponse;
    try {
      legacyResponse = await this.client.generate(legacyRequest);
    } catch (error) {
      throw normalizeError(error);
    }

    return {
      content: legacyResponse.output,
      finishReason:
        { stop: "stop", length: "length" }[legacyResponse.stopCode] ?? "unknown",
    };
  }
}

function normalizeError(error) {
  if (!(error instanceof ProviderError)) {
    return new NormalizedError("upstream_failure", true, error.message);
  }
  if (error.code === "OVER_QUOTA") {
    return new NormalizedError("rate_limited", true, error.message);
  }
  if (error.code === "BAD_REQUEST") {
    return new NormalizedError("invalid_request", false, error.message);
  }
  return new NormalizedError("upstream_failure", true, error.message);
}
