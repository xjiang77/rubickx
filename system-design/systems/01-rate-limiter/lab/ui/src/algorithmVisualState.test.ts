import { describe, expect, it } from "vitest";
import { deriveAlgorithmVisualState } from "./algorithmVisualState";
import type { StateSnapshot, TraceEvent } from "./types";

function traceEvent({
  stepId,
  timestampMs,
  before,
  after,
  decision,
}: {
  stepId: string;
  timestampMs: number;
  before: StateSnapshot;
  after: StateSnapshot;
  decision?: TraceEvent["decision"];
}): TraceEvent {
  return {
    seq: 1,
    stepId,
    actor: "client-a",
    timestampMs,
    before,
    after,
    decision,
    reason: "test_step",
  };
}

describe("deriveAlgorithmVisualState", () => {
  it("derives a fixed-window rollover without predicting its decision", () => {
    const visual = deriveAlgorithmVisualState({
      algorithm: "fixed-window",
      config: { limit: 3, windowMs: 1_000 },
      event: traceEvent({
        stepId: "fixed.locate-window",
        timestampMs: 1_000,
        before: { windowStartMs: 0, count: 3 },
        after: { windowStartMs: 1_000, count: 0 },
      }),
    });

    expect(visual).toMatchObject({
      kind: "fixed-window",
      actor: "client-a",
      timestampMs: 1_000,
      capacity: 3,
      used: 0,
      available: 3,
      loadState: "idle",
      admission: "evaluating",
      windowStartMs: 1_000,
      windowEndMs: 2_000,
      windowProgress: 0,
      rollover: true,
    });
  });

  it("derives the live entries and eviction from a sliding-window log", () => {
    const visual = deriveAlgorithmVisualState({
      algorithm: "sliding-window-log",
      config: { limit: 3, windowMs: 1_000 },
      event: traceEvent({
        stepId: "sliding-log.evict",
        timestampMs: 1_500,
        before: {
          entries: [{ atMs: 500, cost: 1 }, { atMs: 1_000, cost: 1 }],
          used: 2,
        },
        after: { entries: [{ atMs: 1_000, cost: 1 }], used: 1 },
      }),
    });

    expect(visual).toMatchObject({
      kind: "sliding-window-log",
      capacity: 3,
      used: 1,
      available: 2,
      loadState: "active",
      admission: "evaluating",
      rangeStartMs: 500,
      rangeEndMs: 1_500,
      evictedCount: 1,
      entries: [{ key: "1000:1:0", atMs: 1_000, cost: 1, position: 0.5 }],
    });
  });

  it("keeps surviving log entry identities stable after prefix eviction", () => {
    const beforeEviction = deriveAlgorithmVisualState({
      algorithm: "sliding-window-log",
      config: { limit: 3, windowMs: 1_000 },
      event: traceEvent({
        stepId: "sliding-log.decision",
        timestampMs: 1_000,
        before: { entries: [{ atMs: 500, cost: 1 }], used: 1 },
        after: {
          entries: [{ atMs: 500, cost: 1 }, { atMs: 1_000, cost: 1 }],
          used: 2,
        },
      }),
    });
    const afterEviction = deriveAlgorithmVisualState({
      algorithm: "sliding-window-log",
      config: { limit: 3, windowMs: 1_000 },
      event: traceEvent({
        stepId: "sliding-log.evict",
        timestampMs: 1_500,
        before: {
          entries: [{ atMs: 500, cost: 1 }, { atMs: 1_000, cost: 1 }],
          used: 2,
        },
        after: { entries: [{ atMs: 1_000, cost: 1 }], used: 1 },
      }),
    });

    expect(beforeEviction.kind).toBe("sliding-window-log");
    expect(afterEviction.kind).toBe("sliding-window-log");
    if (beforeEviction.kind === "sliding-window-log" && afterEviction.kind === "sliding-window-log") {
      expect(beforeEviction.entries[1].key).toBe(afterEviction.entries[0].key);
    }
  });

  it("accepts independently quantized sliding-log totals without recomputing trace state", () => {
    const entries = Array.from({ length: 4 }, (_, index) => ({ atMs: index, cost: 0.25 }));
    const visual = deriveAlgorithmVisualState({
      algorithm: "sliding-window-log",
      config: { limit: 2, windowMs: 1_000 },
      event: traceEvent({
        stepId: "sliding-log.decision",
        timestampMs: 4,
        before: { entries: entries.slice(0, 3), used: 0.750001 },
        after: { entries, used: 1.000002 },
      }),
    });

    expect(visual).toMatchObject({
      kind: "sliding-window-log",
      used: 1.000002,
      available: 0.999998,
    });
  });

  it("quantizes config capacity with the same six-decimal policy as trace snapshots", () => {
    const visual = deriveAlgorithmVisualState({
      algorithm: "fixed-window",
      config: { limit: 1.0000006, windowMs: 1_000 },
      event: traceEvent({
        stepId: "fixed.decision",
        timestampMs: 0,
        before: { windowStartMs: 0, count: 0 },
        after: { windowStartMs: 0, count: 1.000001 },
      }),
    });

    expect(visual).toMatchObject({
      kind: "fixed-window",
      capacity: 1.000001,
      used: 1.000001,
      available: 0,
      loadState: "full",
    });
  });

  it("uses the runner-provided weighted estimate for a sliding-window counter", () => {
    const visual = deriveAlgorithmVisualState({
      algorithm: "sliding-window-counter",
      config: { limit: 2, windowMs: 1_000 },
      event: traceEvent({
        stepId: "sliding-counter.estimate",
        timestampMs: 1_500,
        before: { currentWindowStartMs: 1_000, currentCount: 0.5, previousCount: 2 },
        after: {
          currentWindowStartMs: 1_000,
          currentCount: 0.5,
          previousCount: 2,
          previousWeight: 0.5,
          estimatedCount: 1.5,
        },
      }),
    });

    expect(visual).toMatchObject({
      kind: "sliding-window-counter",
      capacity: 2,
      used: 1.5,
      available: 0.5,
      loadState: "active",
      admission: "evaluating",
      currentWindowStartMs: 1_000,
      currentCount: 0.5,
      previousCount: 2,
      previousWeight: 0.5,
      weightedPreviousCount: 1,
      estimatedCount: 1.5,
    });
  });

  it("keeps a counter rotate step valid while its estimate is pending", () => {
    const visual = deriveAlgorithmVisualState({
      algorithm: "sliding-window-counter",
      config: { limit: 2, windowMs: 1_000 },
      event: traceEvent({
        stepId: "sliding-counter.rotate",
        timestampMs: 1_000,
        before: { currentWindowStartMs: 0, currentCount: 2, previousCount: 0 },
        after: { currentWindowStartMs: 1_000, currentCount: 0, previousCount: 2 },
      }),
    });

    expect(visual).toMatchObject({
      kind: "sliding-window-counter",
      loadState: "estimating",
      admission: "evaluating",
      currentCount: 0,
      previousCount: 2,
    });
    expect(visual).not.toHaveProperty("estimatedCount");
  });

  it("rejects a non-rotate counter step whose estimate is missing", () => {
    const visual = deriveAlgorithmVisualState({
      algorithm: "sliding-window-counter",
      config: { limit: 2, windowMs: 1_000 },
      event: traceEvent({
        stepId: "sliding-counter.estimate",
        timestampMs: 1_000,
        before: { currentWindowStartMs: 1_000, currentCount: 0, previousCount: 2 },
        after: { currentWindowStartMs: 1_000, currentCount: 0, previousCount: 2 },
      }),
    });

    expect(visual.kind).toBe("unavailable");
  });

  it.each(["before", "after"] as const)("returns unavailable instead of throwing for a null %s snapshot", (field) => {
    const event = traceEvent({
      stepId: "token.refill",
      timestampMs: 0,
      before: { tokens: 2, lastRefillMs: 0 },
      after: { tokens: 2, lastRefillMs: 0 },
    });
    event[field] = null as unknown as StateSnapshot;

    expect(() => deriveAlgorithmVisualState({
      algorithm: "token-bucket",
      config: { capacity: 2, ratePerSecond: 1 },
      event,
    })).not.toThrow();
    expect(deriveAlgorithmVisualState({
      algorithm: "token-bucket",
      config: { capacity: 2, ratePerSecond: 1 },
      event,
    }).kind).toBe("unavailable");
  });

  it("treats token fill as availability and exposes refill or consume deltas", () => {
    const refilled = deriveAlgorithmVisualState({
      algorithm: "token-bucket",
      config: { capacity: 2, ratePerSecond: 1 },
      event: traceEvent({
        stepId: "token.refill",
        timestampMs: 1_000,
        before: { tokens: 0.5, lastRefillMs: 0 },
        after: { tokens: 1.5, lastRefillMs: 1_000 },
      }),
    });
    expect(refilled).toMatchObject({
      kind: "token-bucket",
      capacity: 2,
      available: 1.5,
      used: 0.5,
      loadState: "active",
      admission: "evaluating",
      tokens: 1.5,
      ratePerSecond: 1,
      lastRefillMs: 1_000,
      delta: 1,
    });

    const exhausted = deriveAlgorithmVisualState({
      algorithm: "token-bucket",
      config: { capacity: 2, ratePerSecond: 1 },
      event: traceEvent({
        stepId: "token.decision",
        timestampMs: 1_000,
        before: { tokens: 0, lastRefillMs: 1_000 },
        after: { tokens: 0, lastRefillMs: 1_000 },
        decision: {
          allowed: false,
          remaining: 0,
          retryAfterMs: 1_000,
          resetAtMs: 3_000,
          reason: "insufficient_tokens",
        },
      }),
    });
    expect(exhausted).toMatchObject({
      kind: "token-bucket",
      loadState: "exhausted",
      admission: "deny",
      available: 0,
      delta: 0,
    });
  });

  it("treats leaky-bucket fill as queued work and exposes drain or enqueue deltas", () => {
    const drained = deriveAlgorithmVisualState({
      algorithm: "leaky-bucket",
      config: { capacity: 2, ratePerSecond: 1 },
      event: traceEvent({
        stepId: "leaky.drain",
        timestampMs: 1_000,
        before: { water: 2, lastLeakMs: 0 },
        after: { water: 1, lastLeakMs: 1_000 },
      }),
    });
    expect(drained).toMatchObject({
      kind: "leaky-bucket",
      capacity: 2,
      used: 1,
      available: 1,
      loadState: "active",
      admission: "evaluating",
      water: 1,
      ratePerSecond: 1,
      lastLeakMs: 1_000,
      delta: -1,
    });

    const filled = deriveAlgorithmVisualState({
      algorithm: "leaky-bucket",
      config: { capacity: 2, ratePerSecond: 1 },
      event: traceEvent({
        stepId: "leaky.decision",
        timestampMs: 1_000,
        before: { water: 1, lastLeakMs: 1_000 },
        after: { water: 2, lastLeakMs: 1_000 },
        decision: {
          allowed: true,
          remaining: 0,
          retryAfterMs: 0,
          resetAtMs: 3_000,
          reason: "queued",
        },
      }),
    });
    expect(filled).toMatchObject({
      kind: "leaky-bucket",
      loadState: "full",
      admission: "allow",
      used: 2,
      delta: 1,
    });
  });

  it("classifies a full token bucket as idle rather than full", () => {
    const visual = deriveAlgorithmVisualState({
      algorithm: "token-bucket",
      config: { capacity: 2, ratePerSecond: 1 },
      event: traceEvent({
        stepId: "token.refill",
        timestampMs: 0,
        before: { tokens: 2, lastRefillMs: 0 },
        after: { tokens: 2, lastRefillMs: 0 },
      }),
    });

    expect(visual).toMatchObject({
      kind: "token-bucket",
      loadState: "idle",
      available: 2,
      used: 0,
    });
  });

  it.each([
    {
      name: "an unknown algorithm",
      algorithm: "unknown",
      config: { limit: 2, windowMs: 1_000 },
      before: {},
      after: {},
    },
    {
      name: "a non-positive capacity",
      algorithm: "token-bucket",
      config: { capacity: 0, ratePerSecond: 1 },
      before: { tokens: 0, lastRefillMs: 0 },
      after: { tokens: 0, lastRefillMs: 0 },
    },
    {
      name: "a non-finite state value",
      algorithm: "token-bucket",
      config: { capacity: 2, ratePerSecond: 1 },
      before: { tokens: 2, lastRefillMs: 0 },
      after: { tokens: Number.NaN, lastRefillMs: 0 },
    },
    {
      name: "an over-capacity state value",
      algorithm: "leaky-bucket",
      config: { capacity: 2, ratePerSecond: 1 },
      before: { water: 2, lastLeakMs: 0 },
      after: { water: 3, lastLeakMs: 0 },
    },
    {
      name: "an over-limit fixed-window count",
      algorithm: "fixed-window",
      config: { limit: 2, windowMs: 1_000 },
      before: { windowStartMs: 0, count: 2 },
      after: { windowStartMs: 0, count: 3 },
    },
    {
      name: "an over-limit sliding-log usage",
      algorithm: "sliding-window-log",
      config: { limit: 2, windowMs: 1_000 },
      before: { entries: [], used: 0 },
      after: { entries: [{ atMs: 0, cost: 3 }], used: 3 },
    },
    {
      name: "an over-limit sliding-counter estimate",
      algorithm: "sliding-window-counter",
      config: { limit: 2, windowMs: 1_000 },
      before: { currentWindowStartMs: 0, currentCount: 2, previousCount: 0 },
      after: {
        currentWindowStartMs: 0,
        currentCount: 2,
        previousCount: 0,
        previousWeight: 1,
        estimatedCount: 3,
      },
    },
  ] as Array<{
    name: string;
    algorithm: string;
    config: Record<string, number>;
    before: StateSnapshot;
    after: StateSnapshot;
  }>)("returns unavailable for $name", ({ algorithm, config, before, after }) => {
    const visual = deriveAlgorithmVisualState({
      algorithm,
      config,
      event: traceEvent({ stepId: "test", timestampMs: 0, before, after }),
    });

    expect(visual.kind).toBe("unavailable");
    expect(visual).toHaveProperty("reason");
  });
});
