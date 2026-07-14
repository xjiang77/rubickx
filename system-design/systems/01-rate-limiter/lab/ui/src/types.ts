export type Mode = "semantic" | "debug";

export interface CatalogAlgorithm {
  id: string;
  label: string;
  description: string;
}

export interface CatalogLanguage {
  id: string;
  label: string;
  debuggable?: boolean;
}

export type ScenarioExpectedKind = "allow" | "deny" | "observe";

export interface ScenarioExpectedCase {
  when: string;
  result: string;
  kind: ScenarioExpectedKind;
}

export interface ScenarioExpectation {
  summary: string;
  admissions?: Array<Exclude<ScenarioExpectedKind, "observe">>;
  cases?: ScenarioExpectedCase[];
}

export interface ScenarioBrief {
  policy?: string;
  traffic: string;
  expected: ScenarioExpectation;
  replicaScoped?: boolean;
  conceptual?: boolean;
}

export interface CatalogScenario {
  id: string;
  label: string;
  tier: "core" | "system";
  algorithm?: string;
  description?: string;
  lesson?: string;
  defaultLanguage?: string;
  defaultConfig?: Record<string, number>;
  requestTimeline?: RequestPoint[];
  brief?: ScenarioBrief;
}

export interface Catalog {
  algorithms: CatalogAlgorithm[];
  languages: CatalogLanguage[];
  scenarios: CatalogScenario[];
  modes: Mode[];
}

export interface RequestPoint {
  atMs: number;
  cost: number;
  key: string;
}

export interface Decision {
  allowed: boolean;
  remaining: number;
  retryAfterMs: number;
  resetAtMs: number;
  reason: string;
}

export type StateSnapshot = Record<string, unknown>;

export interface TraceEvent {
  seq: number;
  stepId: string;
  actor: string;
  timestampMs: number;
  before: StateSnapshot;
  after: StateSnapshot;
  decision?: Decision;
  reason: string;
  source?: {
    path: string;
    line: number;
  };
}

export interface SourceDocument {
  language: string;
  path: string;
  content: string;
  anchors?: Record<string, number>;
  stepLines?: Record<string, number>;
}

export interface RunResponse {
  runId: string;
  language: string;
  algorithm: string;
  events: TraceEvent[];
  decisions: Decision[];
  source: SourceDocument;
}

export interface RunRequest {
  scenarioId: string;
  algorithm: string;
  language: string;
  config: Record<string, number>;
  requestTimeline: RequestPoint[];
  storeMode: "memory" | "redis";
}

export interface DebugFrame {
  id?: number;
  name: string;
  file?: string;
  line?: number;
}

export interface DebugLocal {
  name: string;
  value: unknown;
  type?: string;
}

export interface DebugSnapshot {
  sessionId: string;
  status: "starting" | "paused" | "running" | "stopped" | "error" | string;
  source: string | { path?: string; content?: string };
  line: number;
  stackFrames: DebugFrame[];
  locals: DebugLocal[] | Record<string, unknown>;
}

export type DebugCommand = "next" | "continue" | "restart" | "stop";

export interface DebugSessionRequest {
  algorithm: string;
  config: Record<string, number>;
  requestTimeline: RequestPoint[];
  breakpointStepId: string;
}

export interface DemoOptions {
  store: "memory" | "redis";
  failure: "fail-open" | "fail-closed";
  replica: "a" | "b";
  clientKey: string;
  limit: number;
  windowMs: number;
}

export interface DemoResult {
  url: string;
  key: string;
  status: number;
  statusText: string;
  headers: {
    limit: string;
    remaining: string;
    reset: string;
    retryAfter: string;
    degraded: string;
  };
  body: string;
}

export interface DemoExchange extends DemoResult {
  id: number;
  sentAt: number;
}
