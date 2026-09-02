import { act, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { App } from "./App";
import { renderWithI18n as render } from "./test/renderWithI18n";

const catalog = {
  algorithms: [
    { id: "token-bucket", label: "Token Bucket", description: "Burst-friendly quota." },
    { id: "fixed-window", label: "Fixed Window", description: "One counter per window." },
  ],
  languages: [
    { id: "python", label: "Python", debuggable: false },
    { id: "go", label: "Go", debuggable: true },
    { id: "java", label: "Java", debuggable: false },
    { id: "javascript", label: "JavaScript", debuggable: false },
  ],
  scenarios: [
    {
      id: "burst-capacity",
      label: "Burst capacity",
      tier: "core",
      algorithm: "token-bucket",
      lesson: "A token bucket admits a short burst, then exposes refill timing.",
      defaultConfig: { capacity: 2, ratePerSecond: 1 },
      requestTimeline: [{ atMs: 0, cost: 1, key: "client-a" }],
      brief: {
        traffic: "client-a @ 0 ms",
        expected: {
          summary: "The initial token is available immediately.",
          admissions: ["allow"],
        },
      },
    },
    {
      id: "http-contract",
      label: "HTTP 429 contract",
      tier: "system",
      algorithm: "fixed-window",
      lesson: "Clients need quota headers, not only status 429.",
      defaultConfig: { limit: 3, windowMs: 1_000 },
      brief: {
        traffic: "Repeat the same client key within one window.",
        expected: {
          summary: "The first three requests pass; later requests return 429 with quota headers.",
          cases: [
            { when: "Requests #1–3", result: "HTTP 200", kind: "allow" },
            { when: "Request #4+", result: "HTTP 429 + Retry-After", kind: "deny" },
          ],
        },
      },
    },
    {
      id: "local-vs-shared",
      label: "Local vs shared",
      tier: "system",
      algorithm: "fixed-window",
      lesson: "Memory is replica-local; Redis shares quota.",
      defaultConfig: { limit: 3, windowMs: 1_000 },
      brief: {
        replicaScoped: true,
        traffic: "Reuse one client key across Replica A and Replica B.",
        expected: {
          summary: "The selected store decides whether replicas share quota.",
          cases: [
            { when: "Memory", result: "Each replica owns limit 3", kind: "observe" },
            { when: "Redis", result: "Both replicas share limit 3", kind: "observe" },
          ],
        },
      },
    },
  ],
  modes: ["semantic", "debug"],
};

const run = {
  runId: "run-1",
  language: "python",
  algorithm: "token-bucket",
  events: [
    {
      seq: 1,
      stepId: "token.refill",
      actor: "client-a",
      timestampMs: 0,
      before: { tokens: 2, lastRefillMs: 0 },
      after: { tokens: 2, lastRefillMs: 0 },
      reason: "refill from elapsed time",
      source: { path: "python/token_bucket.py", line: 8 },
    },
    {
      seq: 2,
      stepId: "token.decision",
      actor: "client-a",
      timestampMs: 0,
      before: { tokens: 2, lastRefillMs: 0 },
      after: { tokens: 1, lastRefillMs: 0 },
      decision: { allowed: true, remaining: 1, retryAfterMs: 0, resetAtMs: 0, reason: "token available" },
      reason: "consume one token",
      source: { path: "python/token_bucket.py", line: 9 },
    },
  ],
  decisions: [
    { allowed: true, remaining: 1, retryAfterMs: 0, resetAtMs: 0, reason: "token available" },
  ],
  source: {
    language: "python",
    path: "python/token_bucket.py",
    content: "def allow():\n    refill()\n    tokens -= 1\n",
    anchors: { "token.refill": 2, "token.decision": 3 },
  },
};

const multiActorRun = {
  ...run,
  events: [
    ...run.events,
    {
      seq: 3,
      stepId: "token.refill",
      actor: "client-b",
      timestampMs: 100,
      before: { tokens: 2, lastRefillMs: 100 },
      after: { tokens: 2, lastRefillMs: 100 },
      reason: "refill client-b",
      source: { path: "python/token_bucket.py", line: 2 },
    },
    {
      seq: 4,
      stepId: "token.decision",
      actor: "client-b",
      timestampMs: 100,
      before: { tokens: 2, lastRefillMs: 100 },
      after: { tokens: 1, lastRefillMs: 100 },
      decision: { allowed: true, remaining: 1, retryAfterMs: 0, resetAtMs: 1_100, reason: "token available" },
      reason: "consume client-b token",
      source: { path: "python/token_bucket.py", line: 3 },
    },
  ],
};

function jsonResponse(body: unknown, status = 200) {
  return Promise.resolve(new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  }));
}

function persistedPageEvent(type: "pagehide" | "pageshow") {
  const event = new Event(type) as PageTransitionEvent;
  Object.defineProperty(event, "persisted", { value: true });
  return event;
}

describe("Rate Limiter Lab", () => {
  it("shows policy, traffic, expected behavior and lesson before the first run", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockImplementation(() => jsonResponse(catalog));
    render(<App />);

    const brief = await screen.findByRole("region", { name: /scenario brief/i });
    expect(within(brief).getByText("capacity 2 · refill 1/s")).toBeInTheDocument();
    expect(within(brief).getByText("client-a @ 0 ms")).toBeInTheDocument();
    const sequence = within(brief).getByRole("list", { name: /expected admission sequence/i });
    expect(within(sequence).getByText("#1")).toBeInTheDocument();
    expect(within(sequence).getByText("ALLOW")).toBeInTheDocument();
    expect(within(brief).getByText("The initial token is available immediately.")).toBeInTheDocument();
    expect(within(brief).getByText("A token bucket admits a short burst, then exposes refill timing.")).toBeInTheDocument();
    expect(fetchMock.mock.calls.filter(([url]) => url === "/api/runs")).toHaveLength(0);
  });

  it("switches visible chrome to Chinese without fetching or translating catalog data", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockImplementation(() => jsonResponse(catalog));
    const user = userEvent.setup();
    render(<App />);

    await user.click(await screen.findByRole("button", { name: "中文" }));

    expect(screen.getByRole("heading", { name: "场景简报" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "运行场景" })).toBeInTheDocument();
    expect(screen.getByText("client-a @ 0 ms")).toBeInTheDocument();
    expect(screen.getByRole("combobox", { name: "Scenario" })).toBeInTheDocument();
    expect(screen.getByRole("region", { name: "Scenario brief" })).toBeInTheDocument();
    expect(fetchMock.mock.calls.filter(([url]) => url === "/api/catalog")).toHaveLength(1);
    expect(fetchMock.mock.calls.filter(([url]) => url === "/api/runs")).toHaveLength(0);
  });

  it("preserves the trace cursor when the interface language changes", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch")
      .mockImplementationOnce(() => jsonResponse(catalog))
      .mockImplementationOnce(() => jsonResponse(run));
    const user = userEvent.setup();
    render(<App />);

    await user.click(await screen.findByRole("button", { name: "Run scenario" }));
    await screen.findByText("Step 1 / 2");
    await user.click(screen.getByRole("button", { name: "Step forward" }));
    expect(screen.getByText("Step 2 / 2")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "中文" }));

    expect(screen.getByText("步骤 2 / 2")).toBeInTheDocument();
    expect(screen.getByText("consume one token")).toBeInTheDocument();
    expect(fetchMock.mock.calls.filter(([url]) => url === "/api/catalog")).toHaveLength(1);
    expect(fetchMock.mock.calls.filter(([url]) => url === "/api/runs")).toHaveLength(1);
  });

  it("preserves system demo options when the interface language changes", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockImplementation(() => jsonResponse(catalog));
    const user = userEvent.setup();
    render(<App />);

    await user.selectOptions(await screen.findByRole("combobox", { name: "Scenario" }), "local-vs-shared");
    await user.selectOptions(screen.getByRole("combobox", { name: "Store" }), "redis");
    await user.selectOptions(screen.getByRole("combobox", { name: "Failure policy" }), "fail-closed");
    await user.selectOptions(screen.getByRole("combobox", { name: "Replica" }), "b");
    await user.click(screen.getByRole("button", { name: "中文" }));

    expect(screen.getByRole("combobox", { name: "Store" })).toHaveValue("redis");
    expect(screen.getByRole("combobox", { name: "Failure policy" })).toHaveValue("fail-closed");
    expect(screen.getByRole("combobox", { name: "Replica" })).toHaveValue("b");
    expect(screen.getByText("limit 3 · window 1,000 ms · store Redis · failure fail-closed · replica B")).toBeInTheDocument();
    expect(fetchMock.mock.calls.filter(([url]) => url === "/api/catalog")).toHaveLength(1);
    expect(fetchMock.mock.calls.filter(([url]) => url === "/api/runs")).toHaveLength(0);
  });

  it("uses a neutral expectation for a custom algorithm without fetching a run", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockImplementation(() => jsonResponse(catalog));
    const user = userEvent.setup();
    render(<App />);

    await user.selectOptions(await screen.findByRole("combobox", { name: /algorithm/i }), "fixed-window");

    const brief = screen.getByRole("region", { name: /scenario brief/i });
    expect(within(brief).getByText("limit 3 · window 1,000 ms")).toBeInTheDocument();
    expect(within(brief).getByText("Custom comparison · Run to observe this algorithm")).toBeInTheDocument();
    expect(within(brief).queryByText("#1 ALLOW")).not.toBeInTheDocument();
    expect(fetchMock.mock.calls.filter(([url]) => url === "/api/runs")).toHaveLength(0);
  });

  it("updates the system policy preview from store, failure, and replica controls", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockImplementation(() => jsonResponse(catalog));
    const user = userEvent.setup();
    render(<App />);

    await user.selectOptions(await screen.findByRole("combobox", { name: /scenario/i }), "local-vs-shared");
    await user.selectOptions(screen.getByRole("combobox", { name: /store/i }), "redis");
    await user.selectOptions(screen.getByRole("combobox", { name: /failure policy/i }), "fail-closed");
    await user.selectOptions(screen.getByRole("combobox", { name: /replica/i }), "b");

    const brief = screen.getByRole("region", { name: /scenario brief/i });
    expect(within(brief).getByText("limit 3 · window 1,000 ms · store Redis · failure fail-closed · replica B")).toBeInTheDocument();
    expect(within(brief).getByText("Memory")).toBeInTheDocument();
    expect(within(brief).getByText("Each replica owns limit 3")).toBeInTheDocument();
    expect(within(brief).getByText("Redis")).toBeInTheDocument();
    expect(within(brief).getByText("Both replicas share limit 3")).toBeInTheDocument();
    expect(screen.getAllByText("Memory is replica-local; Redis shares quota.")).toHaveLength(1);
    expect(fetchMock.mock.calls.filter(([url]) => url === "/api/runs")).toHaveLength(0);
  });

  it("runs a scenario and replays its real trace without rerunning it", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch")
      .mockImplementationOnce(() => jsonResponse(catalog))
      .mockImplementationOnce(() => jsonResponse(run));
    const user = userEvent.setup();

    render(<App />);

    expect(await screen.findByRole("heading", { name: /rate limiter lab/i })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /run scenario/i }));

    expect(await screen.findByText("refill from elapsed time")).toBeInTheDocument();
    expect(screen.getByRole("progressbar", { name: "Token availability" })).toHaveAttribute("aria-valuenow", "2");
    expect(screen.getByLabelText("Capacity status")).toHaveTextContent("IDLE");
    expect(screen.getByLabelText("Admission")).toHaveTextContent("EVALUATING");
    expect(screen.getByLabelText("Admission")).not.toHaveAttribute("aria-live");
    expect(screen.getByText("Step 1 / 2")).toBeInTheDocument();
    expect(screen.getByText("Line 2", { selector: "span" })).toHaveAttribute("aria-current", "true");

    await user.click(screen.getByRole("button", { name: /step forward/i }));
    expect(screen.getByText("Step 2 / 2")).toBeInTheDocument();
    expect(screen.getByText("consume one token")).toBeInTheDocument();
    expect(screen.getByRole("progressbar", { name: "Token availability" })).toHaveAttribute("aria-valuenow", "1");
    expect(screen.getByLabelText("Capacity status")).toHaveTextContent("ACTIVE");
    expect(screen.getByLabelText("Admission")).toHaveTextContent("ALLOW");
    expect(screen.getByLabelText("Admission")).toHaveAttribute("aria-live", "polite");
    expect(screen.getByText("Line 3", { selector: "span" })).toHaveAttribute("aria-current", "true");

    const rawState = screen.getByText("Raw trace state").closest("details");
    expect(rawState).not.toHaveAttribute("open");
    await user.click(screen.getByText("Raw trace state"));
    expect(rawState).toHaveAttribute("open");

    await user.click(screen.getByRole("button", { name: /rewind/i }));
    expect(screen.getByText("Step 1 / 2")).toBeInTheDocument();
    expect(screen.getByLabelText("Admission")).toHaveTextContent("EVALUATING");
    expect(fetchMock.mock.calls.filter(([url]) => url === "/api/runs")).toHaveLength(1);

    await user.click(screen.getByRole("button", { name: /restart trace/i }));
    expect(screen.getByText("Step 1 / 2")).toBeInTheDocument();
    expect(fetchMock.mock.calls.filter(([url]) => url === "/api/runs")).toHaveLength(1);
  });

  it("switches actor state atomically and rewinds without carrying over admission", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch")
      .mockImplementationOnce(() => jsonResponse(catalog))
      .mockImplementationOnce(() => jsonResponse(multiActorRun));
    const user = userEvent.setup();
    render(<App />);

    await user.click(await screen.findByRole("button", { name: /run scenario/i }));
    await screen.findByText("refill from elapsed time");
    await user.click(screen.getByRole("button", { name: /step forward/i }));

    let visual = screen.getByLabelText("Limiter state");
    expect(within(visual).getByText("client-a")).toBeInTheDocument();
    expect(within(visual).getByLabelText("Admission")).toHaveTextContent("ALLOW");
    expect(within(visual).getByRole("progressbar", { name: "Token availability" })).toHaveAttribute("aria-valuenow", "1");

    await user.click(screen.getByRole("button", { name: /step forward/i }));
    visual = screen.getByLabelText("Limiter state");
    expect(within(visual).getByText("client-b")).toBeInTheDocument();
    expect(within(visual).getByLabelText("Admission")).toHaveTextContent("EVALUATING");
    expect(within(visual).getByRole("progressbar", { name: "Token availability" })).toHaveAttribute("aria-valuenow", "2");

    await user.click(screen.getByRole("button", { name: /rewind/i }));
    visual = screen.getByLabelText("Limiter state");
    expect(within(visual).getByText("client-a")).toBeInTheDocument();
    expect(within(visual).getByLabelText("Admission")).toHaveTextContent("ALLOW");
    expect(fetchMock.mock.calls.filter(([url]) => url === "/api/runs")).toHaveLength(1);
  });

  it("plays and pauses the same trace without rerunning it", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch")
      .mockImplementationOnce(() => jsonResponse(catalog))
      .mockImplementationOnce(() => jsonResponse(multiActorRun));
    const user = userEvent.setup();
    render(<App />);

    await user.click(await screen.findByRole("button", { name: /run scenario/i }));
    await screen.findByText("Step 1 / 4");

    vi.useFakeTimers();
    try {
      act(() => screen.getByRole("button", { name: /play trace/i }).click());
      act(() => vi.advanceTimersByTime(650));
      expect(screen.getByText("Step 2 / 4")).toBeInTheDocument();

      act(() => screen.getByRole("button", { name: /pause trace/i }).click());
      act(() => vi.advanceTimersByTime(1_300));
      expect(screen.getByText("Step 2 / 4")).toBeInTheDocument();
      expect(fetchMock.mock.calls.filter(([url]) => url === "/api/runs")).toHaveLength(1);
    } finally {
      vi.useRealTimers();
    }
  });

  it("uses algorithm defaults after the user leaves the scenario algorithm", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch")
      .mockImplementationOnce(() => jsonResponse(catalog))
      .mockImplementationOnce(() => jsonResponse({ ...run, algorithm: "fixed-window" }));
    const user = userEvent.setup();
    render(<App />);

    await user.selectOptions(await screen.findByRole("combobox", { name: /algorithm/i }), "fixed-window");
    await user.click(screen.getByRole("button", { name: /run scenario/i }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
    const request = JSON.parse((fetchMock.mock.calls[1][1] as RequestInit).body as string);
    expect(request.config).toEqual({ limit: 3, windowMs: 1_000 });
  });

  it("clears a completed trace when the language changes", async () => {
    vi.spyOn(globalThis, "fetch")
      .mockImplementationOnce(() => jsonResponse(catalog))
      .mockImplementationOnce(() => jsonResponse(run));
    const user = userEvent.setup();
    render(<App />);

    await user.click(await screen.findByRole("button", { name: /run scenario/i }));
    expect(await screen.findByText("refill from elapsed time")).toBeInTheDocument();

    await user.selectOptions(screen.getByRole("combobox", { name: /language/i }), "java");
    expect(screen.getByText("No trace yet.")).toBeInTheDocument();
    expect(screen.queryByText("refill from elapsed time")).not.toBeInTheDocument();
    expect(screen.queryByText(/Step 1 \/ 2/)).not.toBeInTheDocument();
    expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Admission")).not.toBeInTheDocument();
  });

  it("ignores a stale semantic response after the selection changes", async () => {
    let resolveRun!: (response: Response) => void;
    const pendingRun = new Promise<Response>((resolve) => { resolveRun = resolve; });
    const fetchMock = vi.spyOn(globalThis, "fetch")
      .mockImplementationOnce(() => jsonResponse(catalog))
      .mockImplementationOnce(() => pendingRun);
    const user = userEvent.setup();
    render(<App />);

    await user.click(await screen.findByRole("button", { name: /run scenario/i }));
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
    await user.selectOptions(screen.getByRole("combobox", { name: /language/i }), "java");
    resolveRun(new Response(JSON.stringify(run), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }));

    await waitFor(() => expect(screen.getByText("No trace yet.")).toBeInTheDocument());
    expect(screen.queryByText("refill from elapsed time")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /run scenario/i })).toBeEnabled();
  });

  it("locks system scenarios and debug mode to Go", async () => {
    vi.spyOn(globalThis, "fetch").mockImplementation(() => jsonResponse(catalog));
    const user = userEvent.setup();
    render(<App />);

    const language = await screen.findByRole("combobox", { name: /language/i });
    expect(language).toHaveValue("python");

    await user.selectOptions(screen.getByRole("combobox", { name: /scenario/i }), "http-contract");
    expect(language).toHaveValue("go");
    expect(screen.getByText(/system scenarios run through the go end-to-end path/i)).toBeInTheDocument();

    await user.selectOptions(screen.getByRole("combobox", { name: /scenario/i }), "burst-capacity");
    await user.selectOptions(screen.getByRole("combobox", { name: /language/i }), "java");
    await user.selectOptions(screen.getByRole("combobox", { name: /mode/i }), "debug");
    expect(language).toHaveValue("go");
    expect(screen.getByText(/delve dap is available for go only/i)).toBeInTheDocument();
  });

  it("sends system scenarios through the real demo endpoint and renders rate-limit headers", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch")
      .mockImplementationOnce(() => jsonResponse(catalog))
      .mockImplementationOnce(() => Promise.resolve(new Response(
        JSON.stringify({ error: "rate limit exceeded", key: "alice" }),
        {
          status: 429,
          headers: {
            "Content-Type": "application/json",
            "RateLimit-Limit": "3",
            "RateLimit-Remaining": "0",
            "RateLimit-Reset": "1",
            "Retry-After": "1",
          },
        },
      )));
    const user = userEvent.setup();
    render(<App />);

    await user.selectOptions(await screen.findByRole("combobox", { name: /scenario/i }), "http-contract");
    await user.click(screen.getByRole("button", { name: /^send request$/i }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
    const [url, init] = fetchMock.mock.calls[1];
    expect(url).toBe("/demo/http-contract?store=memory&failure=fail-open&replica=a&limit=3&window_ms=1000");
    expect(init).toMatchObject({ headers: { "X-RateLimit-Key": "alice" } });
    expect(fetchMock.mock.calls.some(([target]) => target === "/api/runs")).toBe(false);

    const exchange = await screen.findByLabelText("HTTP exchange");
    expect(within(exchange).getByText("429")).toBeInTheDocument();
    expect(within(exchange).getByText("RateLimit-Remaining")).toBeInTheDocument();
    expect(within(exchange).getByText("0", { selector: "dd" })).toBeInTheDocument();
    expect(within(exchange).getByText(/rate limit exceeded/)).toBeInTheDocument();
  });

  it("exposes replica A/B for the local-vs-shared scenario", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch")
      .mockImplementationOnce(() => jsonResponse(catalog))
      .mockImplementationOnce(() => jsonResponse({ scope: "replica-local", replica: "b" }));
    const user = userEvent.setup();
    render(<App />);

    await user.selectOptions(await screen.findByRole("combobox", { name: /scenario/i }), "local-vs-shared");
    const replica = screen.getByRole("combobox", { name: /replica/i });
    await user.selectOptions(replica, "b");
    await user.click(screen.getByRole("button", { name: /^send request$/i }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
    expect(fetchMock.mock.calls[1][0]).toBe(
      "/demo/local-vs-shared?store=memory&failure=fail-open&replica=b&limit=3&window_ms=1000",
    );
  });

  it("shows API errors as a recoverable status", async () => {
    vi.spyOn(globalThis, "fetch")
      .mockImplementationOnce(() => jsonResponse(catalog))
      .mockImplementationOnce(() => jsonResponse({ error: { code: "runner_failed", message: "Python runner exited" } }, 500));
    const user = userEvent.setup();
    render(<App />);

    await screen.findByRole("button", { name: /run scenario/i });
    await user.click(screen.getByRole("button", { name: /run scenario/i }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Python runner exited");
    await waitFor(() => expect(screen.getByRole("button", { name: /run scenario/i })).toBeEnabled());
  });

  it("drives a real Go debug session through create, next, and stop", async () => {
    const paused = {
      sessionId: "debug-1",
      status: "paused",
      source: {
        path: "go/tokenbucket.go",
        content: "func allow() {\n  tokens := refill()\n  return tokens > 0\n}",
      },
      line: 2,
      stackFrames: [{ id: 1, name: "TokenBucket.Allow", file: "go/tokenbucket.go", line: 2 }],
      locals: [{ name: "tokens", value: "2.0", type: "float64" }],
    };
    const next = {
      ...paused,
      line: 3,
      stackFrames: [{ id: 1, name: "TokenBucket.Allow", file: "go/tokenbucket.go", line: 3 }],
      locals: [{ name: "tokens", value: "1.0", type: "float64" }],
    };
    const fetchMock = vi.spyOn(globalThis, "fetch")
      .mockImplementationOnce(() => jsonResponse(catalog))
      .mockImplementationOnce(() => jsonResponse(paused))
      .mockImplementationOnce(() => jsonResponse(next))
      .mockImplementationOnce(() => Promise.resolve(new Response(null, { status: 204 })));
    const user = userEvent.setup();
    render(<App />);

    await user.selectOptions(await screen.findByRole("combobox", { name: /mode/i }), "debug");
    await user.click(screen.getByRole("button", { name: /start debug/i }));

    const createRequest = fetchMock.mock.calls[1];
    expect(createRequest[0]).toBe("/api/debug/sessions");
    expect(JSON.parse((createRequest[1] as RequestInit).body as string)).toEqual({
      algorithm: "token-bucket",
      config: { capacity: 2, ratePerSecond: 1 },
      requestTimeline: [{ atMs: 0, cost: 1, key: "client-a" }],
      breakpointStepId: "token.refill",
    });

    const debugSource = await screen.findByLabelText("Debug source");
    expect(within(debugSource).getByText("Line 2", { selector: "span" })).toHaveAttribute("aria-current", "true");
    expect(screen.getByText("TokenBucket.Allow")).toBeInTheDocument();
    expect(screen.getByText("2.0")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Next" }));
    expect(within(debugSource).getByText("Line 3", { selector: "span" })).toHaveAttribute("aria-current", "true");
    expect(screen.getByText("1.0")).toBeInTheDocument();

    const nextRequest = fetchMock.mock.calls[2];
    expect(nextRequest[0]).toBe("/api/debug/sessions/debug-1/commands");
    expect(JSON.parse((nextRequest[1] as RequestInit).body as string)).toEqual({ command: "next" });

    await user.click(screen.getByRole("button", { name: "Stop" }));
    expect(await screen.findByText("Delve is ready")).toBeInTheDocument();
    expect(fetchMock).toHaveBeenLastCalledWith("/api/debug/sessions/debug-1", { method: "DELETE" });
  });

  it("keeps an active debug session and runtime data intact when chrome switches to Chinese", async () => {
    const paused = {
      sessionId: "debug-i18n",
      status: "paused",
      source: {
        path: "go/tokenbucket.go",
        content: "func allow() {\n  tokens := refill()\n  return tokens > 0\n}",
      },
      line: 2,
      stackFrames: [{ id: 1, name: "TokenBucket.Allow", file: "go/tokenbucket.go", line: 2 }],
      locals: [{ name: "tokens", value: "2.0", type: "float64" }],
    };
    const fetchMock = vi.spyOn(globalThis, "fetch").mockImplementation((input, init) => {
      const url = String(input);
      if (url === "/api/catalog") return jsonResponse(catalog);
      if (url === "/api/debug/sessions" && init?.method === "POST") return jsonResponse(paused);
      if (url === "/api/debug/sessions/debug-i18n" && init?.method === "DELETE") {
        return Promise.resolve(new Response(null, { status: 204 }));
      }
      throw new Error(`Unexpected request: ${url}`);
    });
    const user = userEvent.setup();
    render(<App />);

    await user.selectOptions(await screen.findByRole("combobox", { name: "Mode" }), "debug");
    await user.click(screen.getByRole("button", { name: "Start debug" }));
    expect(await screen.findByText("paused")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "中文" }));

    expect(screen.getByRole("heading", { name: "运行时状态" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "下一步" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "局部变量" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "调用栈" })).toBeInTheDocument();
    expect(screen.getByText("paused")).toBeInTheDocument();
    expect(screen.getByText("TokenBucket.Allow")).toBeInTheDocument();
    expect(screen.getByText("tokens")).toBeInTheDocument();
    expect(screen.getByText("float64")).toBeInTheDocument();
    expect(screen.getByText("2.0")).toBeInTheDocument();
    const debugSource = screen.getByLabelText("Debug source");
    expect(within(debugSource).getByText("go/tokenbucket.go")).toBeInTheDocument();
    expect(within(debugSource).getByText("行 2", { selector: "span" })).toHaveAttribute("aria-current", "true");
    expect(within(debugSource).getByText("tokens := refill()")).toBeInTheDocument();
    expect(fetchMock.mock.calls.filter(([, init]) => (init as RequestInit | undefined)?.method === "DELETE")).toHaveLength(0);
  });

  it("best-effort deletes an active debug session when leaving debug mode", async () => {
    const paused = {
      sessionId: "debug-to-clean",
      status: "paused",
      source: "go/tokenbucket.go",
      line: 19,
      stackFrames: [],
      locals: [],
    };
    const fetchMock = vi.spyOn(globalThis, "fetch")
      .mockImplementationOnce(() => jsonResponse(catalog))
      .mockImplementationOnce(() => jsonResponse(paused))
      .mockImplementationOnce(() => Promise.resolve(new Response(null, { status: 204 })));
    const user = userEvent.setup();
    render(<App />);

    const mode = await screen.findByRole("combobox", { name: /mode/i });
    await user.selectOptions(mode, "debug");
    await user.click(screen.getByRole("button", { name: /start debug/i }));
    expect(await screen.findByText("paused")).toBeInTheDocument();

    await user.selectOptions(mode, "semantic");
    await waitFor(() => expect(fetchMock).toHaveBeenLastCalledWith(
      "/api/debug/sessions/debug-to-clean",
      { method: "DELETE" },
    ));
  });

  it("uses keepalive on pagehide and reconciles busy debug UI after bfcache restore", async () => {
    const paused = {
      sessionId: "debug-pagehide",
      status: "paused",
      source: "go/tokenbucket.go",
      line: 19,
      stackFrames: [],
      locals: [],
    };
    const pendingCommand = new Promise<Response>(() => undefined);
    const fetchMock = vi.spyOn(globalThis, "fetch")
      .mockImplementationOnce(() => jsonResponse(catalog))
      .mockImplementationOnce(() => jsonResponse(paused))
      .mockImplementationOnce(() => pendingCommand)
      .mockImplementationOnce(() => Promise.resolve(new Response(null, { status: 204 })));
    const user = userEvent.setup();
    render(<App />);

    await user.selectOptions(await screen.findByRole("combobox", { name: /mode/i }), "debug");
    await user.click(screen.getByRole("button", { name: /start debug/i }));
    expect(await screen.findByText("paused")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Next" }));
    expect(screen.getByRole("button", { name: /working/i })).toBeDisabled();

    act(() => window.dispatchEvent(persistedPageEvent("pagehide")));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith(
      "/api/debug/sessions/debug-pagehide",
      { method: "DELETE", keepalive: true },
    ));

    act(() => window.dispatchEvent(persistedPageEvent("pageshow")));

    expect(await screen.findByText("Delve is ready")).toBeInTheDocument();
    expect(screen.queryByText("paused")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /start debug/i })).toBeEnabled();
  });

  it("clears a stale debug error after bfcache restore", async () => {
    const paused = {
      sessionId: "debug-pagehide-error",
      status: "paused",
      source: "go/tokenbucket.go",
      line: 19,
      stackFrames: [],
      locals: [],
    };
    const fetchMock = vi.spyOn(globalThis, "fetch")
      .mockImplementationOnce(() => jsonResponse(catalog))
      .mockImplementationOnce(() => jsonResponse(paused))
      .mockImplementationOnce(() => jsonResponse({
        error: { code: "debug_failed", message: "stale command error" },
      }, 500))
      .mockImplementationOnce(() => Promise.resolve(new Response(null, { status: 204 })));
    const user = userEvent.setup();
    render(<App />);

    await user.selectOptions(await screen.findByRole("combobox", { name: /mode/i }), "debug");
    await user.click(screen.getByRole("button", { name: /start debug/i }));
    await user.click(await screen.findByRole("button", { name: "Next" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("stale command error");

    act(() => window.dispatchEvent(persistedPageEvent("pagehide")));
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith(
      "/api/debug/sessions/debug-pagehide-error",
      { method: "DELETE", keepalive: true },
    ));
    act(() => window.dispatchEvent(persistedPageEvent("pageshow")));

    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(await screen.findByText("Delve is ready")).toBeInTheDocument();
  });

  it("ignores a stale debug command rejection after switching mode and starting newer work", async () => {
    const paused = {
      sessionId: "debug-stale-command",
      status: "paused",
      source: "go/tokenbucket.go",
      line: 19,
      stackFrames: [],
      locals: [],
    };
    let rejectCommand!: (reason?: unknown) => void;
    const pendingCommand = new Promise<Response>((_resolve, reject) => { rejectCommand = reject; });
    let resolveRun!: (response: Response) => void;
    const pendingRun = new Promise<Response>((resolve) => { resolveRun = resolve; });
    const fetchMock = vi.spyOn(globalThis, "fetch").mockImplementation((input, init) => {
      const url = String(input);
      if (url === "/api/catalog") return jsonResponse(catalog);
      if (url === "/api/debug/sessions" && init?.method === "POST") return jsonResponse(paused);
      if (url.endsWith("/commands")) return pendingCommand;
      if (url === "/api/debug/sessions/debug-stale-command" && init?.method === "DELETE") {
        return Promise.resolve(new Response(null, { status: 204 }));
      }
      if (url === "/api/runs") return pendingRun;
      throw new Error(`Unexpected request: ${url}`);
    });
    const user = userEvent.setup();
    render(<App />);

    const mode = await screen.findByRole("combobox", { name: /mode/i });
    await user.selectOptions(mode, "debug");
    await user.click(screen.getByRole("button", { name: /start debug/i }));
    await user.click(await screen.findByRole("button", { name: "Next" }));
    await waitFor(() => expect(fetchMock.mock.calls.some(([url]) => String(url).endsWith("/commands"))).toBe(true));

    await user.selectOptions(mode, "semantic");
    await user.click(screen.getByRole("button", { name: /run scenario/i }));
    expect(screen.getByRole("button", { name: /working/i })).toBeDisabled();

    await act(async () => rejectCommand(new Error("late debug command failed")));

    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /working/i })).toBeDisabled();

    await act(async () => resolveRun(new Response(JSON.stringify(run), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    })));
    expect(await screen.findByText("refill from elapsed time")).toBeInTheDocument();
  });
});
