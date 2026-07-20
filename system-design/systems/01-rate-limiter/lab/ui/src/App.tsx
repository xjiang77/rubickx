import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "./api";
import { AlgorithmVisualizer } from "./AlgorithmVisualizer";
import { deriveAlgorithmVisualState } from "./algorithmVisualState";
import { LangToggle, useI18n } from "./i18n";
import { ScenarioBrief } from "./ScenarioBrief";
import { deriveScenarioBriefModel } from "./scenarioBriefModel";
import { SystemConsole, SystemExchange, SystemRequestLog } from "./SystemConsole";
import type {
  Catalog,
  CatalogScenario,
  DebugCommand,
  DebugLocal,
  DebugSnapshot,
  DemoExchange,
  DemoOptions,
  Mode,
  RequestPoint,
  RunRequest,
  RunResponse,
  StateSnapshot,
  TraceEvent,
} from "./types";
import "./styles.css";

const fallbackTimeline: RequestPoint[] = [
  { atMs: 0, cost: 1, key: "client-a" },
  { atMs: 0, cost: 1, key: "client-a" },
  { atMs: 120, cost: 1, key: "client-a" },
  { atMs: 240, cost: 1, key: "client-b" },
  { atMs: 1_000, cost: 1, key: "client-a" },
];

function defaultConfig(algorithm: string): Record<string, number> {
  if (algorithm === "token-bucket" || algorithm === "leaky-bucket") {
    return { capacity: 3, ratePerSecond: 2 };
  }
  return { limit: 3, windowMs: 1_000 };
}

const debugStepByAlgorithm: Record<string, string> = {
  "fixed-window": "fixed.locate-window",
  "sliding-window-log": "sliding-log.evict",
  "sliding-window-counter": "sliding-counter.rotate",
  "token-bucket": "token.refill",
  "leaky-bucket": "leaky.drain",
};

function configForSelection(scenario: CatalogScenario | undefined, algorithm: string) {
  if (scenario?.algorithm === algorithm && scenario.defaultConfig) return scenario.defaultConfig;
  return defaultConfig(algorithm);
}

function defaultDemoOptions(scenario?: CatalogScenario): DemoOptions {
  const config = scenario?.defaultConfig ?? defaultConfig(scenario?.algorithm ?? "fixed-window");
  const redisFirst = scenario
    ? ["redis-atomicity", "redis-outage", "hot-key-sharding", "hot-key"].includes(scenario.id)
    : false;
  return {
    store: redisFirst ? "redis" : "memory",
    failure: "fail-open",
    replica: "a",
    clientKey: scenario?.requestTimeline?.[0]?.key ?? "alice",
    limit: config.limit ?? config.capacity ?? 3,
    windowMs: config.windowMs ?? 1_000,
  };
}

function prefersBurst(scenarioId: string) {
  return ["redis-atomicity", "hot-key-sharding", "hot-key"].includes(scenarioId);
}

function isSystemScenario(scenario: CatalogScenario | undefined) {
  return scenario?.tier === "system";
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : "Something went wrong";
}

function Toolbar({
  catalog,
  scenarioId,
  algorithm,
  language,
  mode,
  busy,
  showAction,
  onScenario,
  onAlgorithm,
  onLanguage,
  onMode,
  onRun,
}: {
  catalog: Catalog;
  scenarioId: string;
  algorithm: string;
  language: string;
  mode: Mode;
  busy: boolean;
  showAction: boolean;
  onScenario: (value: string) => void;
  onAlgorithm: (value: string) => void;
  onLanguage: (value: string) => void;
  onMode: (value: Mode) => void;
  onRun: () => void;
}) {
  const { t } = useI18n();
  const selectedScenario = catalog.scenarios.find((item) => item.id === scenarioId);
  const goOnly = mode === "debug" || isSystemScenario(selectedScenario);

  return (
    <section className={`toolbar${showAction ? "" : " compact-toolbar"}`} aria-label="Lab controls">
      <label>
        <span>{t.tScenario}</span>
        <select aria-label="Scenario" value={scenarioId} onChange={(event) => onScenario(event.target.value)}>
          {catalog.scenarios.map((scenario) => (
            <option key={scenario.id} value={scenario.id}>{scenario.label}</option>
          ))}
        </select>
      </label>
      <label>
        <span>{t.tAlgorithm}</span>
        <select
          aria-label="Algorithm"
          value={algorithm}
          disabled={isSystemScenario(selectedScenario)}
          onChange={(event) => onAlgorithm(event.target.value)}
        >
          {catalog.algorithms.map((item) => (
            <option key={item.id} value={item.id}>{item.label}</option>
          ))}
        </select>
      </label>
      <label>
        <span>{t.tLanguage}</span>
        <select aria-label="Language" value={language} onChange={(event) => onLanguage(event.target.value)}>
          {catalog.languages.map((item) => (
            <option key={item.id} value={item.id} disabled={goOnly && item.id !== "go"}>{item.label}</option>
          ))}
        </select>
      </label>
      <label>
        <span>{t.tMode}</span>
        <select aria-label="Mode" value={mode} onChange={(event) => onMode(event.target.value as Mode)}>
          {catalog.modes.map((item) => (
            <option key={item} value={item}>{item === "debug" ? t.tModeDebug : t.tModeSemantic}</option>
          ))}
        </select>
      </label>
      {showAction && (
        <button className="primary-action" type="button" onClick={onRun} disabled={busy}>
          {busy ? t.tWorking : mode === "debug" ? t.tStartDebug : t.tRunScenario}
        </button>
      )}
    </section>
  );
}

function RequestTimeline({ events, cursor }: { events: TraceEvent[]; cursor: number }) {
  const { t } = useI18n();
  if (events.length === 0) {
    return (
      <div className="empty-state compact">
        <p>{t.tNoTrace}</p>
        <span>{t.tNoTraceSub}</span>
      </div>
    );
  }

  return (
    <ol className="timeline-list" aria-label="Request timeline">
      {events.map((event, index) => (
        <li key={`${event.seq}-${event.stepId}`} className={index === cursor ? "active" : index < cursor ? "complete" : ""}>
          <span className="timeline-marker">{event.seq}</span>
          <div>
            <strong>{event.actor || "request"}</strong>
            <span>{event.timestampMs} ms · {event.stepId}</span>
          </div>
          {event.decision && <DecisionBadge allowed={event.decision.allowed} />}
        </li>
      ))}
    </ol>
  );
}

function DecisionBadge({ allowed }: { allowed: boolean }) {
  return <span className={`decision-badge ${allowed ? "allowed" : "denied"}`}>{allowed ? "ALLOW" : "DENY"}</span>;
}

function AlgorithmState({
  algorithm,
  config,
  event,
}: {
  algorithm: string;
  config: Record<string, number>;
  event?: TraceEvent;
}) {
  const { t } = useI18n();
  if (!event) {
    return (
      <div className="empty-state visual-empty">
        <div className="empty-glyph" aria-hidden="true">○</div>
        <p>{t.tStateHere}</p>
        <span>{t.tStateSub}</span>
      </div>
    );
  }

  const visualState = deriveAlgorithmVisualState({ algorithm, config, event });

  return (
    <div className="state-view">
      <div className="state-heading">
        <div>
          <span className="eyebrow">{t.tCurrentStep}</span>
          <h3>{event.stepId}</h3>
        </div>
      </div>

      <AlgorithmVisualizer key={`${algorithm}:${event.actor}`} state={visualState} />

      <p className="reason-text">{event.reason}</p>
      {event.decision && (
        <dl className="decision-facts">
          <div><dt>{t.tRemaining}</dt><dd>{event.decision.remaining}</dd></div>
          <div><dt>{t.tRetryAfter}</dt><dd>{event.decision.retryAfterMs} ms</dd></div>
          <div><dt>{t.tResetAt}</dt><dd>{event.decision.resetAtMs} ms</dd></div>
        </dl>
      )}
      <details className="raw-state-disclosure">
        <summary>{t.tRawTrace}</summary>
        <div className="state-columns">
          <StateBlock label={t.tBefore} state={event.before} />
          <StateBlock label={t.tAfter} state={event.after} />
        </div>
      </details>
    </div>
  );
}

function StateBlock({ label, state }: { label: string; state: StateSnapshot }) {
  return (
    <div className="state-block">
      <span>{label}</span>
      <pre>{JSON.stringify(state, null, 2)}</pre>
    </div>
  );
}

function SourceViewer({ run, event }: { run?: RunResponse; event?: TraceEvent }) {
  const { t } = useI18n();
  const activeLineRef = useRef<HTMLElement | null>(null);
  const activeLine = run && event
    ? (run.source.anchors ?? run.source.stepLines ?? {})[event.stepId] ?? event.source?.line
    : undefined;

  useEffect(() => {
    activeLineRef.current?.scrollIntoView?.({ block: "center", inline: "nearest" });
  }, [activeLine]);

  if (!run) {
    return (
      <div className="empty-state compact">
        <p>{t.tSourceFollows}</p>
        <span>{t.tSourceSub}</span>
      </div>
    );
  }

  const lines = run.source.content.replace(/\n$/, "").split("\n");

  return (
    <div className="source-viewer">
      <div className="source-path"><span>{run.source.language}</span>{run.source.path}</div>
      <pre className="source-code" aria-label="Implementation source">
        {lines.map((line, index) => {
          const lineNumber = index + 1;
          const active = lineNumber === activeLine;
          return (
            <code key={lineNumber} ref={active ? activeLineRef : undefined} className={active ? "active-line" : ""}>
              <span className="line-number" aria-current={active ? "true" : undefined}>{t.tLine} {lineNumber}</span>
              <span className="line-content">{line || " "}</span>
            </code>
          );
        })}
      </pre>
    </div>
  );
}

function DebugSourceViewer({ snapshot }: { snapshot: DebugSnapshot }) {
  const { t } = useI18n();
  const activeLineRef = useRef<HTMLElement | null>(null);

  useEffect(() => {
    activeLineRef.current?.scrollIntoView?.({ block: "center", inline: "nearest" });
  }, [snapshot.line]);

  const source = typeof snapshot.source === "string" ? undefined : snapshot.source;
  if (!source?.content) {
    return (
      <div className="debug-source-summary" aria-label="Debug source">
        <span>{typeof snapshot.source === "string" ? snapshot.source : source?.path}</span>
        <strong>{t.tPausedAtLine} {snapshot.line}</strong>
        <p>{t.tDebugSrcUnavailable}</p>
      </div>
    );
  }

  return (
    <div className="source-viewer" aria-label="Debug source">
      <div className="source-path"><span>go · delve</span>{source.path}</div>
      <pre className="source-code">
        {source.content.replace(/\n$/, "").split("\n").map((line, index) => {
          const lineNumber = index + 1;
          const active = lineNumber === snapshot.line;
          return (
            <code key={lineNumber} ref={active ? activeLineRef : undefined} className={active ? "active-line" : ""}>
              <span className="line-number" aria-current={active ? "true" : undefined}>{t.tLine} {lineNumber}</span>
              <span className="line-content">{line || " "}</span>
            </code>
          );
        })}
      </pre>
    </div>
  );
}

function TraceControls({
  cursor,
  total,
  playing,
  onStep,
  onRewind,
  onRestart,
  onPlay,
}: {
  cursor: number;
  total: number;
  playing: boolean;
  onStep: () => void;
  onRewind: () => void;
  onRestart: () => void;
  onPlay: () => void;
}) {
  const { t } = useI18n();
  if (total === 0) return null;
  return (
    <div className="trace-controls" aria-label="Trace playback">
      <button type="button" onClick={onRestart} aria-label="Restart trace">↺</button>
      <button type="button" onClick={onRewind} disabled={cursor <= 0} aria-label="Rewind">←</button>
      <button className="play-button" type="button" onClick={onPlay} aria-label={playing ? "Pause trace" : "Play trace"}>
        {playing ? "Ⅱ" : "▶"}
      </button>
      <button type="button" onClick={onStep} disabled={cursor >= total - 1} aria-label="Step forward">→</button>
      <span>{t.tStepWord} {cursor + 1} / {total}</span>
    </div>
  );
}

function DebugPanel({
  snapshot,
  busy,
  onCommand,
}: {
  snapshot?: DebugSnapshot;
  busy: boolean;
  onCommand: (command: DebugCommand) => void;
}) {
  const { t } = useI18n();
  if (!snapshot) {
    return (
      <div className="empty-state visual-empty">
        <div className="empty-glyph debug" aria-hidden="true">›_</div>
        <p>{t.tDelveReady}</p>
        <span>{t.tDelveSub}</span>
      </div>
    );
  }

  const locals: DebugLocal[] = Array.isArray(snapshot.locals)
    ? snapshot.locals
    : Object.entries(snapshot.locals).map(([name, value]) => ({ name, value }));

  return (
    <div className="debug-panel">
      <div className="debug-status">
        <span className={`status-dot ${snapshot.status}`} />
        <strong>{snapshot.status}</strong>
        <span>{t.tLine} {snapshot.line}</span>
      </div>
      <div className="debug-actions">
        <button type="button" disabled={busy} onClick={() => onCommand("next")}>{t.tNext}</button>
        <button type="button" disabled={busy} onClick={() => onCommand("continue")}>{t.tContinue}</button>
        <button type="button" disabled={busy} onClick={() => onCommand("restart")}>{t.tRestart}</button>
        <button className="danger-quiet" type="button" disabled={busy} onClick={() => onCommand("stop")}>{t.tStop}</button>
      </div>
      <section>
        <h3>{t.tLocals}</h3>
        {locals.length ? (
          <dl className="locals-list">
            {locals.map((item) => (
              <div key={item.name}>
                <dt>{item.name}{item.type ? <small>{item.type}</small> : null}</dt>
                <dd>{typeof item.value === "string" ? item.value : JSON.stringify(item.value)}</dd>
              </div>
            ))}
          </dl>
        ) : <p className="muted">{t.tNoLocals}</p>}
      </section>
      <section>
        <h3>{t.tCallStack}</h3>
        <ol className="stack-list">
          {snapshot.stackFrames.map((frame, index) => (
            <li key={frame.id ?? `${frame.name}-${index}`}>
              <strong>{frame.name}</strong>
              <span>{frame.file ?? ""}{frame.line ? `:${frame.line}` : ""}</span>
            </li>
          ))}
        </ol>
      </section>
    </div>
  );
}

export function App() {
  const { t } = useI18n();
  const [catalog, setCatalog] = useState<Catalog>();
  const [catalogError, setCatalogError] = useState("");
  const [scenarioId, setScenarioId] = useState("");
  const [algorithm, setAlgorithm] = useState("");
  const [language, setLanguage] = useState("");
  const [mode, setMode] = useState<Mode>("semantic");
  const [run, setRun] = useState<RunResponse>();
  const [cursor, setCursor] = useState(-1);
  const [playing, setPlaying] = useState(false);
  const [debugSnapshot, setDebugSnapshot] = useState<DebugSnapshot>();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [demoOptions, setDemoOptions] = useState<DemoOptions>(() => defaultDemoOptions());
  const [demoExchanges, setDemoExchanges] = useState<DemoExchange[]>([]);
  const debugSessionRef = useRef<string | undefined>(undefined);
  const debugEpochRef = useRef(0);
  const requestEpochRef = useRef(0);

  const invalidateInFlightRequests = useCallback(() => {
    requestEpochRef.current += 1;
    setBusy(false);
  }, []);

  const releaseActiveDebugSession = useCallback(() => {
    debugEpochRef.current += 1;
    const sessionId = debugSessionRef.current;
    debugSessionRef.current = undefined;
    setDebugSnapshot(undefined);
    if (sessionId) void api.stopDebugSession(sessionId).catch(() => undefined);
  }, []);

  const loadCatalog = useCallback(async (signal?: AbortSignal) => {
    setCatalogError("");
    try {
      const response = await api.catalog(signal);
      setCatalog(response);
      const firstScenario = response.scenarios[0];
      setScenarioId(firstScenario?.id ?? "");
      setAlgorithm(firstScenario?.algorithm ?? response.algorithms[0]?.id ?? "");
      setLanguage(firstScenario?.tier === "system" ? "go" : firstScenario?.defaultLanguage ?? response.languages[0]?.id ?? "go");
      setMode(response.modes[0] ?? "semantic");
      setDemoOptions(defaultDemoOptions(firstScenario));
    } catch (loadError) {
      if ((loadError as DOMException)?.name !== "AbortError") setCatalogError(errorMessage(loadError));
    }
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    void loadCatalog(controller.signal);
    return () => controller.abort();
  }, [loadCatalog]);

  useEffect(() => () => {
    requestEpochRef.current += 1;
    debugEpochRef.current += 1;
    const sessionId = debugSessionRef.current;
    debugSessionRef.current = undefined;
    if (sessionId) void api.stopDebugSession(sessionId).catch(() => undefined);
  }, []);

  useEffect(() => {
    const releaseOnPageHide = () => {
      requestEpochRef.current += 1;
      debugEpochRef.current += 1;
      const sessionId = debugSessionRef.current;
      debugSessionRef.current = undefined;
      if (sessionId) {
        void api.stopDebugSession(sessionId, { keepalive: true }).catch(() => undefined);
      }
    };
    const reconcileAfterPageShow = (event: PageTransitionEvent) => {
      if (!event.persisted) return;
      requestEpochRef.current += 1;
      debugEpochRef.current += 1;
      debugSessionRef.current = undefined;
      setDebugSnapshot(undefined);
      setBusy(false);
      setError("");
      setPlaying(false);
    };
    window.addEventListener("pagehide", releaseOnPageHide);
    window.addEventListener("pageshow", reconcileAfterPageShow);
    return () => {
      window.removeEventListener("pagehide", releaseOnPageHide);
      window.removeEventListener("pageshow", reconcileAfterPageShow);
    };
  }, []);

  useEffect(() => {
    if (!playing || !run || cursor >= run.events.length - 1) {
      if (playing && run && cursor >= run.events.length - 1) setPlaying(false);
      return;
    }
    const timer = window.setTimeout(() => setCursor((value) => Math.min(value + 1, run.events.length - 1)), 650);
    return () => window.clearTimeout(timer);
  }, [cursor, playing, run]);

  const selectedScenario = catalog?.scenarios.find((item) => item.id === scenarioId);
  const currentEvent = run && cursor >= 0 ? run.events[cursor] : undefined;
  const systemOnly = isSystemScenario(selectedScenario);
  const systemWorkspace = systemOnly && mode === "semantic";
  const latestExchange = demoExchanges.at(-1);
  const effectiveConfig = configForSelection(selectedScenario, algorithm);
  const scenarioBrief = selectedScenario
    ? deriveScenarioBriefModel({
        scenario: selectedScenario,
        algorithm,
        config: effectiveConfig,
        demoOptions,
      })
    : undefined;
  const conceptualScenario = selectedScenario?.brief?.conceptual === true;

  const onScenario = (value: string) => {
    if (!catalog) return;
    invalidateInFlightRequests();
    releaseActiveDebugSession();
    const next = catalog.scenarios.find((item) => item.id === value);
    setScenarioId(value);
    if (next?.algorithm) setAlgorithm(next.algorithm);
    if (isSystemScenario(next)) setLanguage("go");
    setDemoOptions(defaultDemoOptions(next));
    setDemoExchanges([]);
    setRun(undefined);
    setCursor(-1);
    setPlaying(false);
    setError("");
  };

  const onAlgorithm = (value: string) => {
    invalidateInFlightRequests();
    releaseActiveDebugSession();
    setAlgorithm(value);
    setRun(undefined);
    setCursor(-1);
    setPlaying(false);
  };

  const onMode = (value: Mode) => {
    invalidateInFlightRequests();
    releaseActiveDebugSession();
    setMode(value);
    if (value === "debug") setLanguage("go");
    setRun(undefined);
    setCursor(-1);
    setPlaying(false);
    setError("");
  };

  const onLanguage = (value: string) => {
    invalidateInFlightRequests();
    releaseActiveDebugSession();
    setLanguage(value);
    setRun(undefined);
    setCursor(-1);
    setPlaying(false);
    setError("");
  };

  const execute = async () => {
    if (systemWorkspace) return;
    const requestEpoch = requestEpochRef.current + 1;
    requestEpochRef.current = requestEpoch;
    setBusy(true);
    setError("");
    setPlaying(false);
    try {
      if (mode === "debug") {
        const epoch = debugEpochRef.current + 1;
        debugEpochRef.current = epoch;
        const previousSession = debugSessionRef.current;
        debugSessionRef.current = undefined;
        setDebugSnapshot(undefined);
        if (previousSession) await api.stopDebugSession(previousSession).catch(() => undefined);
        const response = await api.createDebugSession({
          algorithm,
          config: effectiveConfig,
          requestTimeline: selectedScenario?.requestTimeline ?? fallbackTimeline,
          breakpointStepId: debugStepByAlgorithm[algorithm],
        });
        if (debugEpochRef.current !== epoch || requestEpochRef.current !== requestEpoch) {
          void api.stopDebugSession(response.sessionId).catch(() => undefined);
          return;
        }
        debugSessionRef.current = response.sessionId;
        setDebugSnapshot(response);
      } else {
        const payload: RunRequest = {
          scenarioId,
          algorithm,
          language,
          config: effectiveConfig,
          requestTimeline: selectedScenario?.requestTimeline ?? fallbackTimeline,
          storeMode: "memory",
        };
        const response = await api.run(payload);
        if (requestEpochRef.current !== requestEpoch) return;
        setRun(response);
        setCursor(response.events.length ? 0 : -1);
      }
    } catch (runError) {
      if (requestEpochRef.current === requestEpoch) setError(errorMessage(runError));
    } finally {
      if (requestEpochRef.current === requestEpoch) setBusy(false);
    }
  };

  const sendDemo = async (count: number) => {
    if (!selectedScenario || !systemOnly || conceptualScenario) return;
    const requestEpoch = requestEpochRef.current + 1;
    requestEpochRef.current = requestEpoch;
    setBusy(true);
    setError("");
    try {
      const responses = await Promise.all(
        Array.from({ length: count }, () => api.demo(selectedScenario.id, demoOptions)),
      );
      if (requestEpochRef.current !== requestEpoch) return;
      const sentAt = Date.now();
      setDemoExchanges((current) => {
        const nextID = (current.at(-1)?.id ?? 0) + 1;
        return [
          ...current,
          ...responses.map((response, index) => ({ ...response, id: nextID + index, sentAt })),
        ];
      });
    } catch (demoError) {
      if (requestEpochRef.current === requestEpoch) setError(errorMessage(demoError));
    } finally {
      if (requestEpochRef.current === requestEpoch) setBusy(false);
    }
  };

  const debugCommand = async (command: DebugCommand) => {
    if (!debugSnapshot) return;
    const sessionId = debugSnapshot.sessionId;
    const epoch = debugEpochRef.current;
    const requestEpoch = requestEpochRef.current;
    const isCurrentCommand = () => (
      debugEpochRef.current === epoch
      && requestEpochRef.current === requestEpoch
      && debugSessionRef.current === sessionId
    );
    setBusy(true);
    setError("");
    try {
      if (command === "stop") {
        await api.stopDebugSession(sessionId);
        if (!isCurrentCommand()) return;
        debugSessionRef.current = undefined;
        debugEpochRef.current += 1;
        setDebugSnapshot(undefined);
        setBusy(false);
      } else {
        const response = await api.debugCommand(sessionId, command);
        if (isCurrentCommand()) setDebugSnapshot(response);
      }
    } catch (debugError) {
      if (isCurrentCommand()) setError(errorMessage(debugError));
    } finally {
      if (isCurrentCommand()) setBusy(false);
    }
  };

  if (!catalog && !catalogError) {
    return (
      <main className="shell loading-shell">
        <div className="loading-mark" />
        <p>{t.tLoadingLab}</p>
      </main>
    );
  }

  if (!catalog) {
    return (
      <main className="shell loading-shell">
        <div className="empty-glyph error" aria-hidden="true">!</div>
        <h1>{t.tLoadFailed}</h1>
        <p role="alert">{catalogError}</p>
        <button className="primary-action" type="button" onClick={() => void loadCatalog()}>{t.tTryAgain}</button>
      </main>
    );
  }

  return (
    <main className="shell" data-has-error={error ? "true" : undefined}>
      <header className="page-header">
        <div>
          <span className="kicker">SYSTEM DESIGN · 01</span>
          <h1>Rate Limiter Lab</h1>
          <p>{t.tTagline}</p>
        </div>
        <div className="header-meta">
          <LangToggle />
          <div className="server-state"><span /> {t.tLocalLab}</div>
        </div>
      </header>

      <Toolbar
        catalog={catalog}
        scenarioId={scenarioId}
        algorithm={algorithm}
        language={language}
        mode={mode}
        busy={busy}
        showAction={!systemWorkspace}
        onScenario={onScenario}
        onAlgorithm={onAlgorithm}
        onLanguage={onLanguage}
        onMode={onMode}
        onRun={() => void execute()}
      />

      {scenarioBrief && (
        <ScenarioBrief
          model={{
            ...scenarioBrief,
            goPath: mode === "debug"
              ? t.tDelveGoOnly
              : scenarioBrief.goPath,
          }}
        />
      )}
      {error && <div className="error-banner" role="alert"><strong>{t.tRequestFailed}</strong><span>{error}</span></div>}

      <div className="workspace-grid">
        <section className="panel timeline-panel">
          <div className="panel-heading">
            <div><span className="panel-index">01</span><h2>{t.tPanel1Title}</h2></div>
            <span>{systemWorkspace ? `${demoExchanges.length} ${t.tRequestsUnit}` : `${run?.events.length ?? 0} ${t.tEventsUnit}`}</span>
          </div>
          {systemWorkspace
            ? <SystemRequestLog exchanges={demoExchanges} />
            : <RequestTimeline events={run?.events ?? []} cursor={cursor} />}
        </section>

        <section className="panel state-panel">
          <div className="panel-heading">
            <div><span className="panel-index">02</span><h2>{mode === "debug" ? t.tRuntimeState : systemWorkspace ? t.tSystemConsole : t.tAlgorithmState}</h2></div>
            <span>{systemWorkspace ? selectedScenario?.label : catalog.algorithms.find((item) => item.id === algorithm)?.label}</span>
          </div>
          {systemWorkspace
            ? (
              <SystemConsole
                options={demoOptions}
                busy={busy}
                conceptual={conceptualScenario}
                burstPreferred={prefersBurst(scenarioId)}
                showReplica={selectedScenario?.brief?.replicaScoped === true}
                onOptions={setDemoOptions}
                onSend={(count) => void sendDemo(count)}
              />
            )
            : mode === "debug"
            ? <DebugPanel snapshot={debugSnapshot} busy={busy} onCommand={(command) => void debugCommand(command)} />
            : <AlgorithmState algorithm={algorithm} config={effectiveConfig} event={currentEvent} />}
        </section>

        <section className="panel source-panel">
          <div className="panel-heading">
            <div><span className="panel-index">03</span><h2>{systemWorkspace ? t.tHttpExchange : t.tSourceDecision}</h2></div>
            <span>{systemWorkspace ? demoOptions.store : language}</span>
          </div>
          {systemWorkspace ? (
            <SystemExchange exchange={latestExchange} />
          ) : mode === "debug" && debugSnapshot ? (
            <DebugSourceViewer snapshot={debugSnapshot} />
          ) : <SourceViewer run={run} event={currentEvent} />}
        </section>
      </div>

      {mode === "semantic" && !systemOnly && (
        <TraceControls
          cursor={cursor}
          total={run?.events.length ?? 0}
          playing={playing}
          onStep={() => setCursor((value) => Math.min(value + 1, (run?.events.length ?? 1) - 1))}
          onRewind={() => { setPlaying(false); setCursor((value) => Math.max(0, value - 1)); }}
          onRestart={() => { setPlaying(false); setCursor(run?.events.length ? 0 : -1); }}
          onPlay={() => {
            if (run && cursor >= run.events.length - 1) setCursor(0);
            setPlaying((value) => !value);
          }}
        />
      )}

      <footer className="page-footer">
        <span>{t.tFooterLeft}</span>
        <span>Python · Go · Java · JavaScript</span>
      </footer>
    </main>
  );
}
