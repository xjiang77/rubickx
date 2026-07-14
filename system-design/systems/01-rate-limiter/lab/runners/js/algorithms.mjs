/**
 * Deterministic rate-limiter algorithms with observer-driven semantic traces.
 *
 * A synchronous allow path finishes in one event-loop turn and does not interleave with another
 * synchronous call. Read-modify-write split across await points can still race. Cross-process and
 * distributed quotas therefore require Redis atomic commands/Lua; the event loop is not a lock.
 */

export class InvalidRequest extends Error {}

function number(value, name, { positive = false } = {}) {
  if (typeof value !== "number" || !Number.isFinite(value) || (positive && value <= 0)) {
    throw new InvalidRequest(`${name} must be ${positive ? "a positive finite number" : "finite"}`);
  }
  return value;
}

function quantity(value, name) {
  const result = number(value, name, { positive: true });
  if (result > Number.MAX_SAFE_INTEGER) {
    throw new InvalidRequest(
      `${name} must be a positive finite number no greater than ${Number.MAX_SAFE_INTEGER}`,
    );
  }
  return result;
}

function rounded(value) {
  // Quantize the actual IEEE-754 value; exact binary ties go away from zero.
  const magnitude = Math.floor(Math.abs(value) * 1_000_000 + 0.5) / 1_000_000;
  const result = Math.sign(value) * magnitude;
  if (Object.is(result, -0) || result === 0) return 0;
  return Number.isInteger(result) ? Math.trunc(result) : result;
}

function normalise(value) {
  if (typeof value === "number") return rounded(value);
  if (Array.isArray(value)) return value.map(normalise);
  if (value && typeof value === "object") {
    return Object.fromEntries(Object.entries(value).map(([key, item]) => [key, normalise(item)]));
  }
  return value;
}

function snapshot(value) {
  return normalise(structuredClone(value));
}

class Observer {
  constructor() {
    this.events = [];
  }

  emit(stepId, item, before, after, decision, reason) {
    this.events.push({
      seq: this.events.length + 1,
      stepId,
      actor: item.key,
      timestampMs: rounded(item.atMs),
      before: snapshot(before),
      after: snapshot(after),
      decision: decision === null ? null : snapshot(decision),
      reason,
    });
  }
}

function decision(allowed, remaining, retryAfterMs, resetAtMs, reason) {
  return {
    allowed,
    remaining: rounded(Math.max(0, remaining)),
    retryAfterMs: rounded(Math.max(0, retryAfterMs)),
    resetAtMs: rounded(Math.max(0, resetAtMs)),
    reason,
  };
}

function validateTimeline(value) {
  if (!Array.isArray(value)) throw new InvalidRequest("requestTimeline must be an array");
  if (value.length > 100) throw new InvalidRequest("requestTimeline must contain at most 100 items");
  let previous = -1;
  return value.map((raw, index) => {
    if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
      throw new InvalidRequest(`requestTimeline[${index}] must be an object`);
    }
    const atMs = number(raw.atMs, `requestTimeline[${index}].atMs`);
    const cost = quantity(raw.cost, `requestTimeline[${index}].cost`);
    const key = raw.key;
    if (!Number.isSafeInteger(atMs) || atMs < 0 || atMs < previous) {
      throw new InvalidRequest(
        "requestTimeline atMs must be non-negative, non-decreasing safe integer milliseconds",
      );
    }
    if (
      typeof key !== "string"
      || key.length === 0
      || new TextEncoder().encode(key).byteLength > 128
    ) {
      throw new InvalidRequest(
        `requestTimeline[${index}].key must be a non-empty UTF-8 string of at most 128 bytes`,
      );
    }
    previous = atMs;
    return { atMs, cost, key };
  });
}

function windowConfig(config) {
  if (!config || typeof config !== "object" || Array.isArray(config)) {
    throw new InvalidRequest("config must be an object");
  }
  const limit = quantity(config.limit, "config.limit");
  const windowMs = number(config.windowMs, "config.windowMs", { positive: true });
  if (!Number.isSafeInteger(windowMs)) {
    throw new InvalidRequest("config.windowMs must be a positive safe integer in milliseconds");
  }
  return { limit, windowMs };
}

function bucketConfig(config) {
  if (!config || typeof config !== "object" || Array.isArray(config)) {
    throw new InvalidRequest("config must be an object");
  }
  const capacity = quantity(config.capacity, "config.capacity");
  const rate = quantity(config.ratePerSecond, "config.ratePerSecond");
  const maximumRecoveryMs = capacity / (rate / 1000);
  if (!Number.isFinite(maximumRecoveryMs) || maximumRecoveryMs > Number.MAX_SAFE_INTEGER) {
    throw new InvalidRequest(
      "config capacity and ratePerSecond must yield a finite maximum recovery "
      + `time no greater than ${Number.MAX_SAFE_INTEGER} milliseconds`,
    );
  }
  return { capacity, rate };
}

function runFixedWindow(config, timeline, observer) {
  const { limit, windowMs } = windowConfig(config);
  const states = new Map();
  const decisions = [];
  for (const item of timeline) {
    const windowStartMs = Math.floor(item.atMs / windowMs) * windowMs;
    const state = states.get(item.key) ?? { windowStartMs, count: 0 };
    states.set(item.key, state);
    const before = structuredClone(state);
    // @step:fixed.locate-window
    if (state.windowStartMs !== windowStartMs) {
      state.windowStartMs = windowStartMs;
      state.count = 0;
    }
    observer.emit("fixed.locate-window", item, before, state, null, "window_selected");

    const decisionBefore = structuredClone(state);
    const allowed = state.count + item.cost <= limit;
    if (allowed) state.count += item.cost;
    const reason = allowed ? "within_limit" : (item.cost > limit ? "cost_exceeds_limit" : "limit_exceeded");
    const resetAtMs = windowStartMs + windowMs;
    // @step:fixed.decision
    const result = decision(
      allowed,
      limit - state.count,
      allowed || item.cost > limit ? 0 : resetAtMs - item.atMs,
      resetAtMs,
      reason,
    );
    observer.emit("fixed.decision", item, decisionBefore, state, result, reason);
    decisions.push(result);
  }
  return decisions;
}

function runSlidingLog(config, timeline, observer) {
  const { limit, windowMs } = windowConfig(config);
  const states = new Map();
  const decisions = [];
  for (const item of timeline) {
    const entries = states.get(item.key) ?? [];
    states.set(item.key, entries);
    const before = { entries: structuredClone(entries), used: entries.reduce((sum, entry) => sum + entry.cost, 0) };
    const cutoff = item.atMs - windowMs;
    // @step:sliding-log.evict
    while (entries.length > 0 && entries[0].atMs <= cutoff) entries.shift();
    let used = entries.reduce((sum, entry) => sum + entry.cost, 0);
    const afterEvict = { entries: structuredClone(entries), used };
    observer.emit("sliding-log.evict", item, before, afterEvict, null, "expired_entries_removed");

    const allowed = used + item.cost <= limit;
    if (allowed) {
      entries.push({ atMs: item.atMs, cost: item.cost });
      used += item.cost;
    }
    const reason = allowed ? "within_limit" : (item.cost > limit ? "cost_exceeds_limit" : "limit_exceeded");
    let retryAfterMs = 0;
    if (!allowed && item.cost <= limit) {
      const requiredRelease = used + item.cost - limit;
      let released = 0;
      for (const entry of entries) {
        released += entry.cost;
        if (released + 1e-9 >= requiredRelease) {
          retryAfterMs = Math.max(0, entry.atMs + windowMs - item.atMs);
          break;
        }
      }
    }
    const resetAtMs = allowed
      ? (entries[0]?.atMs ?? item.atMs - windowMs) + windowMs
      : item.atMs + retryAfterMs;
    const after = { entries: structuredClone(entries), used };
    // @step:sliding-log.decision
    const result = decision(allowed, limit - used, retryAfterMs, resetAtMs, reason);
    observer.emit("sliding-log.decision", item, afterEvict, after, result, reason);
    decisions.push(result);
  }
  return decisions;
}

function runSlidingCounter(config, timeline, observer) {
  const { limit, windowMs } = windowConfig(config);
  const states = new Map();
  const decisions = [];
  for (const item of timeline) {
    const currentWindowStartMs = Math.floor(item.atMs / windowMs) * windowMs;
    const state = states.get(item.key) ?? {
      currentWindowStartMs,
      currentCount: 0,
      previousCount: 0,
    };
    states.set(item.key, state);
    const before = structuredClone(state);
    const windowsElapsed = Math.floor((currentWindowStartMs - state.currentWindowStartMs) / windowMs);
    // @step:sliding-counter.rotate
    if (windowsElapsed === 1) {
      state.previousCount = state.currentCount;
      state.currentCount = 0;
      state.currentWindowStartMs = currentWindowStartMs;
    } else if (windowsElapsed > 1) {
      state.previousCount = 0;
      state.currentCount = 0;
      state.currentWindowStartMs = currentWindowStartMs;
    }
    observer.emit("sliding-counter.rotate", item, before, state, null, "windows_rotated");

    const elapsed = item.atMs - currentWindowStartMs;
    const previousWeight = Math.max(0, 1 - elapsed / windowMs);
    // @step:sliding-counter.estimate
    let estimatedCount = state.currentCount + state.previousCount * previousWeight;
    const estimateState = { ...state, previousWeight, estimatedCount };
    observer.emit("sliding-counter.estimate", item, state, estimateState, null, "weighted_count_estimated");

    const allowed = estimatedCount + item.cost <= limit;
    if (allowed) {
      state.currentCount += item.cost;
      estimatedCount += item.cost;
    }
    const reason = allowed ? "within_limit" : (item.cost > limit ? "cost_exceeds_limit" : "limit_exceeded");
    let resetAtMs = currentWindowStartMs + windowMs;
    let retryAfterMs = 0;
    if (!allowed && item.cost <= limit) {
      const excess = Math.max(0, estimatedCount + item.cost - limit);
      const untilBoundary = Math.max(0, currentWindowStartMs + windowMs - item.atMs);
      if (state.currentCount + item.cost <= limit && state.previousCount > 0) {
        retryAfterMs = Math.min(untilBoundary, (excess * windowMs) / state.previousCount);
      } else {
        retryAfterMs = untilBoundary;
        if (state.currentCount > 0) {
          retryAfterMs += Math.max(
            0,
            ((state.currentCount + item.cost - limit) * windowMs) / state.currentCount,
          );
        }
      }
      resetAtMs = item.atMs + retryAfterMs;
    }
    const after = { ...state, previousWeight, estimatedCount };
    // @step:sliding-counter.decision
    const result = decision(
      allowed,
      limit - estimatedCount,
      retryAfterMs,
      resetAtMs,
      reason,
    );
    observer.emit("sliding-counter.decision", item, estimateState, after, result, reason);
    decisions.push(result);
  }
  return decisions;
}

function runTokenBucket(config, timeline, observer) {
  const { capacity, rate } = bucketConfig(config);
  const states = new Map();
  const decisions = [];
  for (const item of timeline) {
    const state = states.get(item.key) ?? { tokens: capacity, lastRefillMs: item.atMs };
    states.set(item.key, state);
    const before = structuredClone(state);
    const elapsed = Math.max(0, item.atMs - state.lastRefillMs);
    // @step:token.refill
    state.tokens = Math.min(capacity, state.tokens + (elapsed * rate) / 1000);
    state.lastRefillMs = item.atMs;
    observer.emit("token.refill", item, before, state, null, "tokens_refilled");

    const decisionBefore = structuredClone(state);
    const allowed = state.tokens + 1e-9 >= item.cost;
    if (allowed) state.tokens -= item.cost;
    const reason = allowed
      ? "token_available"
      : (item.cost > capacity ? "cost_exceeds_capacity" : "insufficient_tokens");
    const retryAfterMs = allowed || item.cost > capacity
      ? 0
      : ((item.cost - state.tokens) / rate) * 1000;
    const resetAtMs = item.atMs + ((capacity - state.tokens) / rate) * 1000;
    // @step:token.decision
    const result = decision(allowed, state.tokens, retryAfterMs, resetAtMs, reason);
    observer.emit("token.decision", item, decisionBefore, state, result, reason);
    decisions.push(result);
  }
  return decisions;
}

function runLeakyBucket(config, timeline, observer) {
  const { capacity, rate } = bucketConfig(config);
  const states = new Map();
  const decisions = [];
  for (const item of timeline) {
    const state = states.get(item.key) ?? { water: 0, lastLeakMs: item.atMs };
    states.set(item.key, state);
    const before = structuredClone(state);
    const elapsed = Math.max(0, item.atMs - state.lastLeakMs);
    // @step:leaky.drain
    state.water = Math.max(0, state.water - (elapsed * rate) / 1000);
    state.lastLeakMs = item.atMs;
    observer.emit("leaky.drain", item, before, state, null, "queued_work_drained");

    const decisionBefore = structuredClone(state);
    const allowed = state.water + item.cost <= capacity + 1e-9;
    if (allowed) state.water += item.cost;
    const reason = allowed
      ? "queue_has_capacity"
      : (item.cost > capacity ? "cost_exceeds_capacity" : "queue_full");
    const retryAfterMs = allowed || item.cost > capacity
      ? 0
      : ((state.water + item.cost - capacity) / rate) * 1000;
    const resetAtMs = item.atMs + (state.water / rate) * 1000;
    // @step:leaky.decision
    const result = decision(allowed, capacity - state.water, retryAfterMs, resetAtMs, reason);
    observer.emit("leaky.decision", item, decisionBefore, state, result, reason);
    decisions.push(result);
  }
  return decisions;
}

const runners = new Map([
  ["fixed-window", runFixedWindow],
  ["sliding-window-log", runSlidingLog],
  ["sliding-window-counter", runSlidingCounter],
  ["token-bucket", runTokenBucket],
  ["leaky-bucket", runLeakyBucket],
]);

export function runRequest(request) {
  if (!request || typeof request !== "object" || Array.isArray(request)) {
    throw new InvalidRequest("request must be an object");
  }
  const runner = runners.get(request.algorithm);
  if (!runner) throw new InvalidRequest(`unsupported algorithm: ${String(request.algorithm)}`);
  const timeline = validateTimeline(request.requestTimeline);
  const observer = new Observer();
  const decisions = runner(request.config, timeline, observer);
  return { events: observer.events, decisions };
}
