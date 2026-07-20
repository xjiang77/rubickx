import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import {
  LegacyProviderAdapter,
  NormalizedError,
  ProviderError,
} from "./adapter.mjs";

const contract = JSON.parse(
  await readFile(new URL("../fixtures/contract.json", import.meta.url), "utf8"),
);

const contractCase = (id) => contract.cases.find((candidate) => candidate.id === id);

const requestFrom = (scenario) => ({
  model: scenario.input.request.model,
  messages: scenario.input.request.messages,
  requiresTools: scenario.input.request.requires_tools,
});

class FakeLegacyClient {
  constructor({ response, error } = {}) {
    this.response = response;
    this.error = error;
    this.lastRequest = undefined;
    this.calls = 0;
  }

  async generate(request) {
    this.calls += 1;
    this.lastRequest = request;
    if (this.error !== undefined) {
      throw this.error;
    }
    return this.response;
  }
}

test("maps the shared contract fixture", async () => {
  const scenario = contractCase("maps_request_and_response");
  const provider = scenario.input.provider_response;
  const client = new FakeLegacyClient({
    response: { output: provider.output, stopCode: provider.stop_code },
  });

  const response = await new LegacyProviderAdapter(client).complete(requestFrom(scenario));

  assert.deepEqual(client.lastRequest, scenario.expected.mapped_request);
  assert.deepEqual(response, {
    content: scenario.expected.response.content,
    finishReason: scenario.expected.response.finish_reason,
  });
});

test("fails explicitly for unknown models", async () => {
  const scenario = contractCase("rejects_unknown_model");

  await assert.rejects(
    new LegacyProviderAdapter(new FakeLegacyClient()).complete(requestFrom(scenario)),
    (error) =>
      error instanceof NormalizedError &&
      error.code === scenario.expected_error.code &&
      error.retryable === scenario.expected_error.retryable,
  );
});

test("normalizes provider errors", async () => {
  const scenario = contractCase("normalizes_provider_error");
  const providerError = scenario.input.provider_error;
  const client = new FakeLegacyClient({
    error: new ProviderError(providerError.code, providerError.message),
  });

  await assert.rejects(
    new LegacyProviderAdapter(client).complete(requestFrom(scenario)),
    (error) =>
      error instanceof NormalizedError &&
      error.code === scenario.expected_error.code &&
      error.retryable === scenario.expected_error.retryable,
  );
});

test("rejects unsupported capabilities before provider invocation", async () => {
  const scenario = contractCase("rejects_unsupported_before_provider_call");
  const client = new FakeLegacyClient();

  await assert.rejects(
    new LegacyProviderAdapter(client).complete(requestFrom(scenario)),
    (error) =>
      error instanceof NormalizedError &&
      error.code === scenario.expected_error.code &&
      error.retryable === scenario.expected_error.retryable,
  );
  assert.equal(client.calls, scenario.expected.provider_calls);
});
