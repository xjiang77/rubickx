import type { AlgorithmVisualState, AdmissionState, LoadState } from "./algorithmVisualState";

export function AlgorithmVisualizer({ state }: { state: AlgorithmVisualState }) {
  if (state.kind === "unavailable") {
    return (
      <div className="visual-unavailable" role="note">
        <strong>State unavailable</strong>
        <span>{state.reason}</span>
      </div>
    );
  }

  return (
    <section className={`algorithm-visualizer ${state.kind}`} aria-label="Limiter state">
      <div className="visual-context">
        <div><span>Quota key</span><strong>{state.actor || "request"}</strong></div>
        <div><span>Trace time</span><strong>{formatMs(state.timestampMs)}</strong></div>
        <div><span>Capacity</span><strong>{formatNumber(state.capacity)}</strong></div>
      </div>
      <div className="visual-statuses">
        <div className={`visual-status load ${state.loadState}`} aria-label="Capacity status">
          <span>Capacity</span>{statusLabel(state.loadState)}
        </div>
        <div
          className={`visual-status admission ${state.admission}`}
          aria-label="Admission"
          role={state.admission === "evaluating" ? undefined : "status"}
          aria-live={state.admission === "evaluating" ? undefined : "polite"}
        >
          <span>Request</span>{admissionLabel(state.admission)}
        </div>
      </div>
      {state.kind === "fixed-window" && <FixedWindowVisual state={state} />}
      {state.kind === "sliding-window-log" && <SlidingLogVisual state={state} />}
      {state.kind === "sliding-window-counter" && <SlidingCounterVisual state={state} />}
      {state.kind === "token-bucket" && <TokenBucketVisual state={state} />}
      {state.kind === "leaky-bucket" && <LeakyBucketVisual state={state} />}
    </section>
  );
}

function FixedWindowVisual({ state }: { state: Extract<AlgorithmVisualState, { kind: "fixed-window" }> }) {
  return (
    <div className="window-visual-card">
      <UsageBar
        label="Fixed window usage"
        caption="Window used"
        value={state.used}
        capacity={state.capacity}
        valueText={`${formatNumber(state.used)} of ${formatNumber(state.capacity)} window capacity used`}
      />
      <div className="window-time-track" aria-hidden="true">
        <span className="window-time-fill" style={{ width: `${state.windowProgress * 100}%` }} />
        <i className="window-now-marker" style={{ left: `${state.windowProgress * 100}%` }} />
      </div>
      <div className="window-time-labels">
        <span>{formatMs(state.windowStartMs)}</span>
        <strong>now · {formatMs(state.timestampMs)}</strong>
        <span>{formatMs(state.windowEndMs)}</span>
      </div>
      <p className="visual-delta">{state.rollover ? "Window rolled over" : "Current fixed window"}</p>
    </div>
  );
}

function SlidingLogVisual({ state }: { state: Extract<AlgorithmVisualState, { kind: "sliding-window-log" }> }) {
  return (
    <div className="window-visual-card">
      <UsageBar
        label="Sliding window log usage"
        caption="Rolling window used"
        value={state.used}
        capacity={state.capacity}
        valueText={`${formatNumber(state.used)} of ${formatNumber(state.capacity)} rolling window capacity used`}
      />
      <div className="sliding-log-track" aria-hidden="true">
        {state.entries.map((entry) => (
          <i
            key={entry.key}
            className="log-entry-dot"
            style={{ left: `${entry.position * 100}%` }}
          />
        ))}
      </div>
      <div className="window-time-labels">
        <span>{formatMs(state.rangeStartMs)}</span>
        <strong>rolling now</strong>
        <span>{formatMs(state.rangeEndMs)}</span>
      </div>
      <p className="visual-delta split-delta">
        <span>{plural(state.entries.length, "live entry", "live entries")}</span>
        <span>{plural(state.evictedCount, "evicted", "evicted")}</span>
      </p>
    </div>
  );
}

function SlidingCounterVisual({ state }: { state: Extract<AlgorithmVisualState, { kind: "sliding-window-counter" }> }) {
  const estimated = state.estimatedCount;
  const previousWidth = percent(state.weightedPreviousCount ?? 0, state.capacity);
  const currentWidth = Math.min(percent(state.currentCount, state.capacity), 100 - previousWidth);
  return (
    <div className="window-visual-card">
      <div className="capacity-labels visual-capacity-labels">
        <span>Weighted window used</span>
        <strong>{estimated === undefined ? "estimating…" : `${formatNumber(estimated)} / ${formatNumber(state.capacity)}`}</strong>
      </div>
      <div
        className={`counter-stack${estimated === undefined ? " estimating" : ""}`}
        role="progressbar"
        aria-label="Sliding window estimated usage"
        aria-valuemin={0}
        aria-valuemax={state.capacity}
        aria-valuenow={estimated}
        aria-valuetext={estimated === undefined
          ? "Estimated window usage pending"
          : `${formatNumber(estimated)} of ${formatNumber(state.capacity)} estimated window capacity used`}
      >
        <span className="counter-previous" style={{ width: `${previousWidth}%` }} />
        <span className="counter-current" style={{ left: `${previousWidth}%`, width: `${currentWidth}%` }} />
      </div>
      <div className="counter-legend">
        <span><i className="previous-key" />weighted previous</span>
        <span><i className="current-key" />current</span>
      </div>
      <p className="counter-equation">
        {estimated === undefined || state.previousWeight === undefined
          ? "Estimate pending after window rotation"
          : `${formatNumber(state.currentCount)} current + ${formatNumber(state.previousCount)} × ${formatNumber(state.previousWeight)} = ${formatNumber(estimated)}`}
      </p>
      <div className="window-time-labels counter-time-labels">
        <span>previous</span>
        <strong>{formatMs(state.currentWindowStartMs)}</strong>
        <span>{formatMs(state.currentWindowEndMs)}</span>
      </div>
    </div>
  );
}

function TokenBucketVisual({ state }: { state: Extract<AlgorithmVisualState, { kind: "token-bucket" }> }) {
  const fill = percent(state.available, state.capacity);
  return (
    <div className="bucket-visual-grid">
      <div
        className="bucket-vessel token-vessel"
        role="progressbar"
        aria-label="Token availability"
        aria-valuemin={0}
        aria-valuemax={state.capacity}
        aria-valuenow={state.available}
        aria-valuetext={`${formatNumber(state.available)} of ${formatNumber(state.capacity)} tokens available`}
      >
        <span className="bucket-fill" style={{ height: `${fill}%` }} />
        <span className="bucket-value">{formatNumber(state.available)}</span>
      </div>
      <div className="bucket-copy">
        <span>Available tokens</span>
        <strong>{formatNumber(state.available)} / {formatNumber(state.capacity)}</strong>
        <p>{tokenDeltaLabel(state.delta, state.stepId)}</p>
        <dl>
          <div><dt>Refill rate</dt><dd>{formatNumber(state.ratePerSecond)} / sec</dd></div>
          <div><dt>Refill checkpoint</dt><dd>{formatMs(state.lastRefillMs)}</dd></div>
        </dl>
      </div>
    </div>
  );
}

function LeakyBucketVisual({ state }: { state: Extract<AlgorithmVisualState, { kind: "leaky-bucket" }> }) {
  const fill = percent(state.used, state.capacity);
  return (
    <div className="bucket-visual-grid">
      <div
        className="bucket-vessel leaky-vessel"
        role="progressbar"
        aria-label="Leaky bucket queue usage"
        aria-valuemin={0}
        aria-valuemax={state.capacity}
        aria-valuenow={state.used}
        aria-valuetext={`${formatNumber(state.used)} of ${formatNumber(state.capacity)} queue capacity used`}
      >
        <span className="bucket-fill" style={{ height: `${fill}%` }} />
        <span className="bucket-value">{formatNumber(state.used)}</span>
      </div>
      <div className="bucket-copy">
        <span>Queued work</span>
        <strong>{formatNumber(state.used)} / {formatNumber(state.capacity)}</strong>
        <p>{leakyDeltaLabel(state.delta, state.stepId)}</p>
        <dl>
          <div><dt>Drain rate</dt><dd>{formatNumber(state.ratePerSecond)} / sec</dd></div>
          <div><dt>Drain checkpoint</dt><dd>{formatMs(state.lastLeakMs)}</dd></div>
        </dl>
      </div>
    </div>
  );
}

function UsageBar({
  label,
  caption,
  value,
  capacity,
  valueText,
}: {
  label: string;
  caption: string;
  value: number;
  capacity: number;
  valueText: string;
}) {
  return (
    <div className="usage-bar-group">
      <div className="capacity-labels visual-capacity-labels">
        <span>{caption}</span>
        <strong>{formatNumber(value)} / {formatNumber(capacity)}</strong>
      </div>
      <div
        className="visual-meter"
        role="progressbar"
        aria-label={label}
        aria-valuemin={0}
        aria-valuemax={capacity}
        aria-valuenow={value}
        aria-valuetext={valueText}
      >
        <span style={{ width: `${percent(value, capacity)}%` }} />
      </div>
    </div>
  );
}

function tokenDeltaLabel(delta: number, stepId: string) {
  if (delta > 0) return `+${formatNumber(delta)} refilled`;
  if (delta < 0) return `${formatNumber(delta)} consumed`;
  return stepId === "token.decision" ? "No tokens consumed" : "No refill needed";
}

function leakyDeltaLabel(delta: number, stepId: string) {
  if (delta < 0) return `${formatNumber(delta)} drained`;
  if (delta > 0) return `+${formatNumber(delta)} enqueued`;
  return stepId === "leaky.decision" ? "Queue unchanged" : "Nothing to drain";
}

function statusLabel(state: LoadState) {
  return state.toUpperCase();
}

function admissionLabel(state: AdmissionState) {
  return state.toUpperCase();
}

function percent(value: number, capacity: number) {
  return Math.max(0, Math.min(100, value / capacity * 100));
}

function formatMs(value: number) {
  return `${formatNumber(value)} ms`;
}

function formatNumber(value: number) {
  return value.toLocaleString("en-US", { maximumFractionDigits: 6 });
}

function plural(value: number, singular: string, pluralValue: string) {
  return `${formatNumber(value)} ${value === 1 ? singular : pluralValue}`;
}
