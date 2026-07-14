import type { Decision, TraceEvent } from "./types";

export type AdmissionState = "evaluating" | "allow" | "deny";
export type LoadState = "idle" | "active" | "full" | "exhausted" | "estimating";

interface VisualStateBase {
  actor: string;
  timestampMs: number;
  stepId: string;
  admission: AdmissionState;
  decision?: Decision;
}

export interface FixedWindowVisualState extends VisualStateBase {
  kind: "fixed-window";
  capacity: number;
  used: number;
  available: number;
  loadState: Exclude<LoadState, "exhausted" | "estimating">;
  windowStartMs: number;
  windowEndMs: number;
  windowProgress: number;
  rollover: boolean;
}

export interface UnavailableVisualState extends VisualStateBase {
  kind: "unavailable";
  reason: string;
}

export interface SlidingWindowLogEntry {
  key: string;
  atMs: number;
  cost: number;
  position: number;
}

export interface SlidingWindowLogVisualState extends VisualStateBase {
  kind: "sliding-window-log";
  capacity: number;
  used: number;
  available: number;
  loadState: Exclude<LoadState, "exhausted" | "estimating">;
  rangeStartMs: number;
  rangeEndMs: number;
  entries: SlidingWindowLogEntry[];
  evictedCount: number;
}

export interface SlidingWindowCounterVisualState extends VisualStateBase {
  kind: "sliding-window-counter";
  capacity: number;
  currentWindowStartMs: number;
  currentWindowEndMs: number;
  currentCount: number;
  previousCount: number;
  loadState: "idle" | "active" | "full" | "estimating";
  used?: number;
  available?: number;
  previousWeight?: number;
  weightedPreviousCount?: number;
  estimatedCount?: number;
}

export interface TokenBucketVisualState extends VisualStateBase {
  kind: "token-bucket";
  capacity: number;
  used: number;
  available: number;
  loadState: "idle" | "active" | "exhausted";
  tokens: number;
  ratePerSecond: number;
  lastRefillMs: number;
  delta: number;
}

export interface LeakyBucketVisualState extends VisualStateBase {
  kind: "leaky-bucket";
  capacity: number;
  used: number;
  available: number;
  loadState: "idle" | "active" | "full";
  water: number;
  ratePerSecond: number;
  lastLeakMs: number;
  delta: number;
}

export type AlgorithmVisualState =
  | FixedWindowVisualState
  | SlidingWindowLogVisualState
  | SlidingWindowCounterVisualState
  | TokenBucketVisualState
  | LeakyBucketVisualState
  | UnavailableVisualState;

export function deriveAlgorithmVisualState({
  algorithm,
  config,
  event,
}: {
  algorithm: string;
  config: Record<string, number>;
  event: TraceEvent;
}): AlgorithmVisualState {
  const common = {
    actor: event.actor,
    timestampMs: event.timestampMs,
    stepId: event.stepId,
    admission: admissionFor(event),
    decision: event.decision,
  } as const;
  const before = stateSnapshot(event.before);
  const after = stateSnapshot(event.after);
  if (!before || !after) {
    return { ...common, kind: "unavailable", reason: "Trace snapshots are incomplete" };
  }

  if (algorithm === "fixed-window") {
    const capacity = positiveTraceQuantity(config.limit);
    const windowMs = positiveFinite(config.windowMs);
    const count = capacity === undefined ? undefined : boundedQuantity(after.count, capacity);
    const beforeCount = capacity === undefined ? undefined : boundedQuantity(before.count, capacity);
    const windowStartMs = nonNegativeFinite(after.windowStartMs);
    const beforeStart = nonNegativeFinite(before.windowStartMs);
    if (
      capacity === undefined
      || windowMs === undefined
      || count === undefined
      || beforeCount === undefined
      || windowStartMs === undefined
      || beforeStart === undefined
    ) {
      return { ...common, kind: "unavailable", reason: "Fixed-window state is incomplete" };
    }

    return {
      ...common,
      kind: "fixed-window",
      capacity,
      used: count,
      available: traceRound6(Math.max(0, capacity - count)),
      loadState: usedLoadState(count, capacity),
      windowStartMs,
      windowEndMs: windowStartMs + windowMs,
      windowProgress: clamp01((event.timestampMs - windowStartMs) / windowMs),
      rollover: beforeStart !== undefined && beforeStart !== windowStartMs,
    };
  }

  if (algorithm === "sliding-window-log") {
    const capacity = positiveTraceQuantity(config.limit);
    const windowMs = positiveFinite(config.windowMs);
    const used = capacity === undefined ? undefined : boundedQuantity(after.used, capacity);
    const beforeUsed = capacity === undefined ? undefined : boundedQuantity(before.used, capacity);
    const afterEntries = logEntries(after.entries);
    const beforeEntries = logEntries(before.entries);
    if (
      capacity === undefined
      || windowMs === undefined
      || used === undefined
      || beforeUsed === undefined
      || !afterEntries
      || !beforeEntries
    ) {
      return { ...common, kind: "unavailable", reason: "Sliding-window-log state is incomplete" };
    }

    const rangeStartMs = event.timestampMs - windowMs;
    const duplicateOrdinals = new Map<string, number>();
    return {
      ...common,
      kind: "sliding-window-log",
      capacity,
      used,
      available: traceRound6(Math.max(0, capacity - used)),
      loadState: usedLoadState(used, capacity),
      rangeStartMs,
      rangeEndMs: event.timestampMs,
      entries: afterEntries.map((entry) => {
        const signature = `${entry.atMs}:${entry.cost}`;
        const ordinal = duplicateOrdinals.get(signature) ?? 0;
        duplicateOrdinals.set(signature, ordinal + 1);
        return {
          ...entry,
          key: `${signature}:${ordinal}`,
          position: clamp01((entry.atMs - rangeStartMs) / windowMs),
        };
      }),
      evictedCount: Math.max(0, beforeEntries.length - afterEntries.length),
    };
  }

  if (algorithm === "sliding-window-counter") {
    const capacity = positiveTraceQuantity(config.limit);
    const windowMs = positiveFinite(config.windowMs);
    const currentWindowStartMs = nonNegativeFinite(after.currentWindowStartMs);
    const currentCount = capacity === undefined ? undefined : boundedQuantity(after.currentCount, capacity);
    const previousCount = capacity === undefined ? undefined : boundedQuantity(after.previousCount, capacity);
    if (
      capacity === undefined
      || windowMs === undefined
      || currentWindowStartMs === undefined
      || currentCount === undefined
      || previousCount === undefined
    ) {
      return { ...common, kind: "unavailable", reason: "Sliding-window-counter state is incomplete" };
    }

    const base = {
      ...common,
      kind: "sliding-window-counter" as const,
      capacity,
      currentWindowStartMs,
      currentWindowEndMs: currentWindowStartMs + windowMs,
      currentCount,
      previousCount,
    };
    const previousWeight = unitInterval(after.previousWeight);
    const estimatedCount = capacity === undefined ? undefined : boundedQuantity(after.estimatedCount, capacity);
    const hasPreviousWeight = Object.prototype.hasOwnProperty.call(after, "previousWeight");
    const hasEstimatedCount = Object.prototype.hasOwnProperty.call(after, "estimatedCount");
    if (!hasPreviousWeight && !hasEstimatedCount && event.stepId === "sliding-counter.rotate") {
      return { ...base, loadState: "estimating" };
    }
    if (previousWeight === undefined || estimatedCount === undefined) {
      return { ...common, kind: "unavailable", reason: "Sliding-window-counter estimate is incomplete" };
    }

    return {
      ...base,
      loadState: usedLoadState(estimatedCount, capacity),
      used: estimatedCount,
      available: traceRound6(Math.max(0, capacity - estimatedCount)),
      previousWeight,
      weightedPreviousCount: traceRound6(previousCount * previousWeight),
      estimatedCount,
    };
  }

  if (algorithm === "token-bucket") {
    const capacity = positiveTraceQuantity(config.capacity);
    const ratePerSecond = positiveFinite(config.ratePerSecond);
    if (capacity === undefined || ratePerSecond === undefined) {
      return { ...common, kind: "unavailable", reason: "Token-bucket config is incomplete" };
    }
    const tokens = boundedQuantity(after.tokens, capacity);
    const beforeTokens = boundedQuantity(before.tokens, capacity);
    const lastRefillMs = nonNegativeFinite(after.lastRefillMs);
    if (tokens === undefined || beforeTokens === undefined || lastRefillMs === undefined) {
      return { ...common, kind: "unavailable", reason: "Token-bucket state is incomplete" };
    }
    return {
      ...common,
      kind: "token-bucket",
      capacity,
      used: traceRound6(Math.max(0, capacity - tokens)),
      available: tokens,
      loadState: tokenLoadState(tokens, capacity),
      tokens,
      ratePerSecond,
      lastRefillMs,
      delta: traceRound6(tokens - beforeTokens),
    };
  }

  if (algorithm === "leaky-bucket") {
    const capacity = positiveTraceQuantity(config.capacity);
    const ratePerSecond = positiveFinite(config.ratePerSecond);
    if (capacity === undefined || ratePerSecond === undefined) {
      return { ...common, kind: "unavailable", reason: "Leaky-bucket config is incomplete" };
    }
    const water = boundedQuantity(after.water, capacity);
    const beforeWater = boundedQuantity(before.water, capacity);
    const lastLeakMs = nonNegativeFinite(after.lastLeakMs);
    if (water === undefined || beforeWater === undefined || lastLeakMs === undefined) {
      return { ...common, kind: "unavailable", reason: "Leaky-bucket state is incomplete" };
    }
    return {
      ...common,
      kind: "leaky-bucket",
      capacity,
      used: water,
      available: traceRound6(Math.max(0, capacity - water)),
      loadState: usedLoadState(water, capacity),
      water,
      ratePerSecond,
      lastLeakMs,
      delta: traceRound6(water - beforeWater),
    };
  }

  return { ...common, kind: "unavailable", reason: `Unsupported visual state for ${algorithm}` };
}

function admissionFor(event: TraceEvent): AdmissionState {
  if (!event.decision) return "evaluating";
  return event.decision.allowed ? "allow" : "deny";
}

function stateSnapshot(value: unknown): Record<string, unknown> | undefined {
  return value !== null && typeof value === "object" && !Array.isArray(value)
    ? value as Record<string, unknown>
    : undefined;
}

function positiveFinite(value: unknown) {
  return typeof value === "number" && Number.isFinite(value) && value > 0 ? value : undefined;
}

function positiveTraceQuantity(value: unknown) {
  const quantity = positiveFinite(value);
  if (quantity === undefined) return undefined;
  const quantized = traceRound6(quantity);
  return quantized > 0 ? quantized : undefined;
}

function traceRound6(value: number) {
  const magnitude = Math.floor(Math.abs(value) * 1_000_000 + 0.5) / 1_000_000;
  const result = Math.sign(value) * magnitude;
  return Object.is(result, -0) ? 0 : result;
}

function nonNegativeFinite(value: unknown) {
  return typeof value === "number" && Number.isFinite(value) && value >= 0 ? value : undefined;
}

function boundedQuantity(value: unknown, capacity: number) {
  const quantity = nonNegativeFinite(value);
  if (quantity === undefined || quantity > capacity + 1e-9) return undefined;
  return Math.min(quantity, capacity);
}

function unitInterval(value: unknown) {
  return typeof value === "number" && Number.isFinite(value) && value >= 0 && value <= 1 ? value : undefined;
}

function logEntries(value: unknown) {
  if (!Array.isArray(value)) return undefined;
  const entries: Array<{ atMs: number; cost: number }> = [];
  for (const item of value) {
    if (!item || typeof item !== "object") return undefined;
    const atMs = nonNegativeFinite((item as Record<string, unknown>).atMs);
    const cost = positiveFinite((item as Record<string, unknown>).cost);
    if (atMs === undefined || cost === undefined) return undefined;
    entries.push({ atMs, cost });
  }
  return entries;
}

function usedLoadState(used: number, capacity: number): "idle" | "active" | "full" {
  if (used <= 1e-9) return "idle";
  if (used + 1e-9 >= capacity) return "full";
  return "active";
}

function tokenLoadState(tokens: number, capacity: number): "idle" | "active" | "exhausted" {
  if (tokens <= 1e-9) return "exhausted";
  if (tokens + 1e-9 >= capacity) return "idle";
  return "active";
}

function clamp01(value: number) {
  return Math.max(0, Math.min(1, value));
}
