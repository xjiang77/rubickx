import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { readFileSync } from "node:fs";
import test from "node:test";

const runner = new URL("./runner.mjs", import.meta.url);
const fixtures = JSON.parse(readFileSync(new URL("../../fixtures/core-parity.json", import.meta.url)));

function request(algorithm, config, requestTimeline) {
  return {
    scenarioId: "contract-test",
    algorithm,
    language: "javascript",
    config,
    requestTimeline,
    storeMode: "memory",
  };
}

function runLines(...requests) {
  const input = `${requests
    .map((value) => (typeof value === "string" ? value : JSON.stringify(value)))
    .join("\n")}\n`;
  const result = spawnSync(process.execPath, [runner.pathname], {
    input,
    encoding: "utf8",
  });
  assert.equal(result.status, 0, result.stderr);
  return result.stdout.trim().split("\n").map(JSON.parse);
}

const burstTimeline = [
  { atMs: 0, cost: 1, key: "alice" },
  { atMs: 0, cost: 1, key: "alice" },
  { atMs: 0, cost: 1, key: "alice" },
  { atMs: 1000, cost: 1, key: "alice" },
];

test("JSONL seam runs real token bucket and recovers after malformed JSON", () => {
  const [error, response] = runLines(
    "not json",
    request("token-bucket", { capacity: 2, ratePerSecond: 1 }, burstTimeline),
  );
  assert.equal(error.error.code, "invalid_json");
  assert.deepEqual(response.decisions.map(({ allowed }) => allowed), [true, true, false, true]);
  assert.equal(response.events.at(-1).stepId, "token.decision");
  assert.deepEqual(Object.keys(response.events.at(-1)).sort(), [
    "actor", "after", "before", "decision", "reason", "seq", "stepId", "timestampMs",
  ]);
});

for (const [algorithm, config, expected] of [
  ["fixed-window", { limit: 2, windowMs: 1000 }, [true, true, false, true]],
  ["sliding-window-log", { limit: 2, windowMs: 1000 }, [true, true, false, true]],
  ["sliding-window-counter", { limit: 2, windowMs: 1000 }, [true, true, false, false]],
  ["leaky-bucket", { capacity: 2, ratePerSecond: 1 }, [true, true, false, true]],
]) {
  test(`JSONL seam supports ${algorithm}`, () => {
    const response = runLines(request(algorithm, config, burstTimeline))[0];
    assert.deepEqual(response.decisions.map(({ allowed }) => allowed), expected);
  });
}

test("invalid request does not stop the next JSONL request", () => {
  const [error, response] = runLines(
    request("fixed-window", { limit: 0, windowMs: 1000 }, []),
    request("fixed-window", { limit: 1, windowMs: 1000 }, [{ atMs: 0, cost: 1, key: "alice" }]),
  );
  assert.equal(error.error.code, "invalid_request");
  assert.equal(response.decisions[0].allowed, true);
});

test("token bucket sync allow is serialized but cross-await allow can over-admit", async () => {
  let tokens = 1;
  let release;
  const gate = new Promise((resolve) => { release = resolve; });
  const unsafeAllow = async () => {
    const observed = tokens;
    if (observed < 1) return false;
    await gate;
    tokens = observed - 1;
    return true;
  };
  const first = unsafeAllow();
  const second = unsafeAllow();
  release();
  assert.deepEqual(
    await Promise.all([first, second]),
    [true, true],
    "both callers read one token before awaiting and incorrectly ALLOW",
  );
  assert.equal(tokens, 0);

  let syncTokens = 1;
  const syncAllow = () => {
    if (syncTokens < 1) return false;
    syncTokens -= 1;
    return true;
  };
  assert.deepEqual(
    [syncAllow(), syncAllow()],
    [true, false],
    "synchronous allow calls finish one at a time in the event-loop turn",
  );
});

for (const fixture of fixtures) {
  test(`shared fixture: ${fixture.name}`, () => {
    const response = runLines(
      request(fixture.algorithm, fixture.config, fixture.requestTimeline),
    )[0];
    assert.deepEqual(response.decisions.map(({ allowed }) => allowed), fixture.expectedAllowed);
    assert.deepEqual(response.decisions.map(({ remaining }) => remaining), fixture.expectedRemaining);
    assert.equal(response.decisions.at(-1).reason, fixture.expectedLastReason);
    assert.equal(response.decisions.at(-1).retryAfterMs, fixture.expectedLastRetryAfterMs);
    if (Object.hasOwn(fixture, "expectedLastResetAtMs")) {
      assert.equal(response.decisions.at(-1).resetAtMs, fixture.expectedLastResetAtMs);
    }
  });
}

for (const [algorithm, config] of [
  ["fixed-window", { limit: 2, windowMs: 1000 }],
  ["sliding-window-log", { limit: 2, windowMs: 1000 }],
  ["sliding-window-counter", { limit: 2, windowMs: 1000 }],
  ["token-bucket", { capacity: 2, ratePerSecond: 1 }],
  ["leaky-bucket", { capacity: 2, ratePerSecond: 1 }],
]) {
  test(`empty timeline is a successful no-op for ${algorithm}`, () => {
    assert.deepEqual(runLines(request(algorithm, config, []))[0], { events: [], decisions: [] });
  });
}

for (const [name, timeline] of [
  ["negative time", [{ atMs: -1, cost: 1, key: "alice" }]],
  ["fractional time", [{ atMs: 0.5, cost: 1, key: "alice" }]],
  ["time above MAX_SAFE_INTEGER", [
    { atMs: Number.MAX_SAFE_INTEGER + 1, cost: 1, key: "alice" },
  ]],
  ["non-monotonic time", [
    { atMs: 2, cost: 1, key: "alice" },
    { atMs: 1, cost: 1, key: "alice" },
  ]],
]) {
  test(`${name} is rejected at the public seam`, () => {
    const response = runLines(request("fixed-window", { limit: 2, windowMs: 1000 }, timeline))[0];
    assert.equal(response.error.code, "invalid_request");
    if (name === "time above MAX_SAFE_INTEGER") {
      assert.match(response.error.message, /safe integer milliseconds/);
    }
  });
}

test("more than 100 timeline items are rejected", () => {
  const timeline = Array.from({ length: 101 }, (_, atMs) => ({ atMs, cost: 1, key: "alice" }));
  const response = runLines(request("fixed-window", { limit: 2, windowMs: 1000 }, timeline))[0];
  assert.equal(response.error.code, "invalid_request");
});

for (const [algorithm, config] of [
  ["fixed-window", { limit: 2, windowMs: 1000 }],
  ["token-bucket", { capacity: 2, ratePerSecond: 1 }],
  ["leaky-bucket", { capacity: 2, ratePerSecond: 1 }],
]) {
  test(`${algorithm} trace is state-continuous within each request`, () => {
    const response = runLines(request(algorithm, config, [
      { atMs: 0, cost: 1, key: "alice" },
      { atMs: 1000, cost: 1, key: "alice" },
    ]))[0];
    assert.equal(response.events.length, 4);
    for (let index = 0; index < response.events.length; index += 2) {
      assert.deepEqual(response.events[index].after, response.events[index + 1].before);
    }
  });
}

for (const [algorithm, config] of [
  ["fixed-window", { limit: 0, windowMs: 1000 }],
  ["fixed-window", { limit: 2, windowMs: 0.5 }],
  ["token-bucket", { capacity: 2, ratePerSecond: 0 }],
]) {
  test(`non-positive ${algorithm} config is rejected`, () => {
    const response = runLines(request(algorithm, config, []))[0];
    assert.equal(response.error.code, "invalid_request");
  });
}

test("window above MAX_SAFE_INTEGER is rejected at the public seam", () => {
  const response = runLines(request("fixed-window", {
    limit: 2,
    windowMs: Number.MAX_SAFE_INTEGER + 1,
  }, []))[0];
  assert.equal(response.error.code, "invalid_request");
  assert.match(response.error.message, /positive safe integer/);
});

for (const [name, algorithm, config, timeline] of [
  ["limit above max safe", "fixed-window", { limit: 1e308, windowMs: 1000 }, [
    { atMs: 0, cost: 1, key: "alice" },
  ]],
  ["capacity above max safe", "token-bucket", { capacity: 1e308, ratePerSecond: 1 }, [
    { atMs: 0, cost: 1, key: "alice" },
  ]],
  ["rate above max safe", "token-bucket", { capacity: 1, ratePerSecond: 1e308 }, [
    { atMs: 0, cost: 1, key: "alice" },
  ]],
  ["cost above max safe", "fixed-window", { limit: 2, windowMs: 1000 }, [
    { atMs: 0, cost: 1e308, key: "alice" },
  ]],
  ["recovery time above max safe", "token-bucket", { capacity: 1, ratePerSecond: 5e-324 }, [
    { atMs: 0, cost: 1, key: "alice" },
  ]],
]) {
  test(`${name} quantity is rejected at the public seam`, () => {
    const response = runLines(request(algorithm, config, timeline))[0];
    assert.equal(response.error.code, "invalid_request");
  });
}

test("MAX_SAFE_INTEGER quantities are accepted at the public seam", () => {
  const maximum = Number.MAX_SAFE_INTEGER;
  const responses = runLines(
    request("fixed-window", { limit: maximum, windowMs: 1000 }, [
      { atMs: 0, cost: maximum, key: "alice" },
    ]),
    request("token-bucket", { capacity: maximum, ratePerSecond: maximum }, [
      { atMs: 0, cost: maximum, key: "alice" },
    ]),
  );
  assert.deepEqual(responses.map(({ decisions }) => decisions[0].allowed), [true, true]);
});

for (const [name, key] of [
  ["ASCII 129-byte", "a".repeat(129)],
  ["multibyte 129-byte", "界".repeat(43)],
]) {
  test(`${name} key is rejected at the public seam`, () => {
    const response = runLines(request("fixed-window", { limit: 1, windowMs: 1000 }, [
      { atMs: 0, cost: 1, key },
    ]))[0];
    assert.equal(response.error.code, "invalid_request");
  });
}

test("multibyte key at 128 UTF-8 bytes is accepted", () => {
  const key = `${"界".repeat(42)}ab`;
  const response = runLines(request("fixed-window", { limit: 1, windowMs: 1000 }, [
    { atMs: 0, cost: 1, key },
  ]))[0];
  assert.equal(response.decisions[0].allowed, true);
  assert.equal(response.events[0].actor, key);
});

for (const [name, timeline] of [
  ["cost", [{ atMs: 0, key: "alice" }]],
  ["key", [{ atMs: 0, cost: 1 }]],
]) {
  test(`missing ${name} is rejected at the public seam`, () => {
    const response = runLines(request("fixed-window", { limit: 1, windowMs: 1000 }, timeline))[0];
    assert.equal(response.error.code, "invalid_request");
  });
}
