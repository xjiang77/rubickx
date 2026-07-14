import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AlgorithmVisualizer } from "./AlgorithmVisualizer";
import type { AlgorithmVisualState } from "./algorithmVisualState";

describe("AlgorithmVisualizer", () => {
  it("shows token availability separately from request admission", () => {
    const state: AlgorithmVisualState = {
      kind: "token-bucket",
      actor: "client-a",
      timestampMs: 1_000,
      stepId: "token.refill",
      admission: "evaluating",
      capacity: 2,
      used: 0.5,
      available: 1.5,
      loadState: "active",
      tokens: 1.5,
      ratePerSecond: 1,
      lastRefillMs: 1_000,
      delta: 1,
    };

    render(<AlgorithmVisualizer state={state} />);

    expect(screen.getByLabelText("Capacity status")).toHaveTextContent("ACTIVE");
    expect(screen.getByLabelText("Admission")).toHaveTextContent("EVALUATING");
    expect(screen.getByLabelText("Admission")).not.toHaveAttribute("aria-live");
    expect(screen.queryByRole("status", { name: "Admission" })).not.toBeInTheDocument();
    expect(screen.getByText("client-a")).toBeInTheDocument();
    expect(screen.getAllByText("1,000 ms")).toHaveLength(2);

    const progress = screen.getByRole("progressbar", { name: "Token availability" });
    expect(progress).toHaveAttribute("aria-valuemin", "0");
    expect(progress).toHaveAttribute("aria-valuemax", "2");
    expect(progress).toHaveAttribute("aria-valuenow", "1.5");
    expect(progress).toHaveAttribute("aria-valuetext", "1.5 of 2 tokens available");
    expect(screen.getByText("+1 refilled")).toBeInTheDocument();
  });

  it.each([
    {
      name: "fixed window",
      state: {
        kind: "fixed-window",
        actor: "client-a",
        timestampMs: 1_000,
        stepId: "fixed.locate-window",
        admission: "evaluating",
        capacity: 3,
        used: 0,
        available: 3,
        loadState: "idle",
        windowStartMs: 1_000,
        windowEndMs: 2_000,
        windowProgress: 0,
        rollover: true,
      } satisfies AlgorithmVisualState,
      progressName: "Fixed window usage",
      valueNow: "0",
      valueText: "0 of 3 window capacity used",
      visibleText: "Window rolled over",
    },
    {
      name: "sliding log",
      state: {
        kind: "sliding-window-log",
        actor: "client-a",
        timestampMs: 1_500,
        stepId: "sliding-log.evict",
        admission: "evaluating",
        capacity: 3,
        used: 1,
        available: 2,
        loadState: "active",
        rangeStartMs: 500,
        rangeEndMs: 1_500,
        entries: [{ key: "1000:1:0", atMs: 1_000, cost: 1, position: 0.5 }],
        evictedCount: 1,
      } satisfies AlgorithmVisualState,
      progressName: "Sliding window log usage",
      valueNow: "1",
      valueText: "1 of 3 rolling window capacity used",
      visibleText: "1 live entry",
    },
    {
      name: "sliding counter",
      state: {
        kind: "sliding-window-counter",
        actor: "client-a",
        timestampMs: 1_500,
        stepId: "sliding-counter.estimate",
        admission: "evaluating",
        capacity: 2,
        used: 1.5,
        available: 0.5,
        loadState: "active",
        currentWindowStartMs: 1_000,
        currentWindowEndMs: 2_000,
        currentCount: 0.5,
        previousCount: 2,
        previousWeight: 0.5,
        weightedPreviousCount: 1,
        estimatedCount: 1.5,
      } satisfies AlgorithmVisualState,
      progressName: "Sliding window estimated usage",
      valueNow: "1.5",
      valueText: "1.5 of 2 estimated window capacity used",
      visibleText: "0.5 current + 2 × 0.5 = 1.5",
    },
    {
      name: "leaky bucket",
      state: {
        kind: "leaky-bucket",
        actor: "client-a",
        timestampMs: 1_000,
        stepId: "leaky.drain",
        admission: "evaluating",
        capacity: 2,
        used: 1,
        available: 1,
        loadState: "active",
        water: 1,
        ratePerSecond: 1,
        lastLeakMs: 1_000,
        delta: -1,
      } satisfies AlgorithmVisualState,
      progressName: "Leaky bucket queue usage",
      valueNow: "1",
      valueText: "1 of 2 queue capacity used",
      visibleText: "-1 drained",
    },
  ])("renders the $name visual and accessibility contract", ({ state, progressName, valueNow, valueText, visibleText }) => {
    render(<AlgorithmVisualizer state={state} />);

    const progress = screen.getByRole("progressbar", { name: progressName });
    expect(progress).toHaveAttribute("aria-valuenow", valueNow);
    expect(progress).toHaveAttribute("aria-valuetext", valueText);
    expect(screen.getByText(visibleText)).toBeInTheDocument();
  });

  it("announces only a terminal allow or deny decision", () => {
    const state: AlgorithmVisualState = {
      kind: "token-bucket",
      actor: "client-a",
      timestampMs: 1_000,
      stepId: "token.decision",
      admission: "deny",
      decision: {
        allowed: false,
        remaining: 0,
        retryAfterMs: 1_000,
        resetAtMs: 3_000,
        reason: "insufficient_tokens",
      },
      capacity: 2,
      used: 2,
      available: 0,
      loadState: "exhausted",
      tokens: 0,
      ratePerSecond: 1,
      lastRefillMs: 1_000,
      delta: 0,
    };

    render(<AlgorithmVisualizer state={state} />);

    expect(screen.getByLabelText("Admission")).toHaveTextContent("DENY");
    expect(screen.getByLabelText("Admission")).toHaveAttribute("aria-live", "polite");
    expect(screen.getByRole("status", { name: "Admission" })).toBeInTheDocument();
  });

  it("renders a counter rotate as an indeterminate estimate", () => {
    const state: AlgorithmVisualState = {
      kind: "sliding-window-counter",
      actor: "client-a",
      timestampMs: 1_000,
      stepId: "sliding-counter.rotate",
      admission: "evaluating",
      capacity: 2,
      loadState: "estimating",
      currentWindowStartMs: 1_000,
      currentWindowEndMs: 2_000,
      currentCount: 0,
      previousCount: 2,
    };

    render(<AlgorithmVisualizer state={state} />);

    const progress = screen.getByRole("progressbar", { name: "Sliding window estimated usage" });
    expect(progress).not.toHaveAttribute("aria-valuenow");
    expect(progress).toHaveAttribute("aria-valuetext", "Estimated window usage pending");
    expect(screen.getByLabelText("Capacity status")).toHaveTextContent("ESTIMATING");
  });

  it("shows a safe fallback without a progressbar", () => {
    const state: AlgorithmVisualState = {
      kind: "unavailable",
      actor: "client-a",
      timestampMs: 0,
      stepId: "token.refill",
      admission: "evaluating",
      reason: "Token-bucket state is incomplete",
    };

    render(<AlgorithmVisualizer state={state} />);

    expect(screen.getByText("State unavailable")).toBeInTheDocument();
    expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
  });
});
