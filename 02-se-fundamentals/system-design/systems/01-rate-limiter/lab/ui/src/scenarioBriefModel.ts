import type {
  CatalogScenario,
  DemoOptions,
  ScenarioExpectedCase,
} from "./types";

export interface ScenarioAdmissionGroup {
  kind: "allow" | "deny";
  label: string;
}

export interface ScenarioBriefViewModel {
  policy: string;
  traffic: string;
  expected: {
    summary: string;
    groups: ScenarioAdmissionGroup[];
    cases: ScenarioExpectedCase[];
  };
  lesson?: {
    label: "Lesson" | "Concept";
    text: string;
  };
  goPath?: string;
}

export interface DeriveScenarioBriefModelInput {
  scenario: CatalogScenario;
  algorithm: string;
  config: Record<string, number>;
  demoOptions?: DemoOptions;
}

function formatNumber(value: number) {
  return new Intl.NumberFormat("en-US", { maximumFractionDigits: 6 }).format(value);
}

function hasPositiveFiniteValues(...values: number[]) {
  return values.every((value) => Number.isFinite(value) && value > 0);
}

function formatPolicy(algorithm: string, config: Record<string, number>) {
  if (algorithm === "token-bucket") {
    if (!hasPositiveFiniteValues(config.capacity, config.ratePerSecond)) return "Policy unavailable";
    return `capacity ${formatNumber(config.capacity)} · refill ${formatNumber(config.ratePerSecond)}/s`;
  }
  if (algorithm === "leaky-bucket") {
    if (!hasPositiveFiniteValues(config.capacity, config.ratePerSecond)) return "Policy unavailable";
    return `capacity ${formatNumber(config.capacity)} · drain ${formatNumber(config.ratePerSecond)}/s`;
  }
  if (["fixed-window", "sliding-window-log", "sliding-window-counter"].includes(algorithm)) {
    if (!hasPositiveFiniteValues(config.limit, config.windowMs)) return "Policy unavailable";
    return `limit ${formatNumber(config.limit)} · window ${formatNumber(config.windowMs)} ms`;
  }
  return "Policy unavailable";
}

function groupAdmissions(admissions: Array<"allow" | "deny">): ScenarioAdmissionGroup[] {
  const groups: ScenarioAdmissionGroup[] = [];
  let start = 0;
  while (start < admissions.length) {
    const kind = admissions[start];
    let end = start;
    while (end + 1 < admissions.length && admissions[end + 1] === kind) end += 1;
    groups.push({
      kind,
      label: start === end ? `#${start + 1}` : `#${start + 1}–${end + 1}`,
    });
    start = end + 1;
  }
  return groups;
}

function withDemoOptions(
  policy: string,
  demoOptions: DemoOptions | undefined,
  replicaScoped: boolean,
  conceptual: boolean,
) {
  if (!demoOptions || conceptual) return policy;
  const store = demoOptions.store === "redis" ? "Redis" : "Memory";
  const options = [`store ${store}`, `failure ${demoOptions.failure}`];
  if (replicaScoped) options.push(`replica ${demoOptions.replica.toUpperCase()}`);
  return `${policy} · ${options.join(" · ")}`;
}

function cleanText(value: string | undefined, fallback: string) {
  const cleaned = value?.trim();
  return cleaned || fallback;
}

export function deriveScenarioBriefModel({
  scenario,
  algorithm,
  config,
  demoOptions,
}: DeriveScenarioBriefModelInput): ScenarioBriefViewModel {
  const brief = scenario.brief;
  const expected = brief?.expected;
  const conceptual = brief?.conceptual === true;
  const customComparison = !conceptual && Boolean(scenario.algorithm && scenario.algorithm !== algorithm);
  const basePolicy = customComparison
    ? formatPolicy(algorithm, config)
    : brief?.policy?.trim() || formatPolicy(algorithm, config);
  return {
    policy: scenario.tier === "system"
      ? withDemoOptions(basePolicy, demoOptions, brief?.replicaScoped === true, conceptual)
      : basePolicy,
    traffic: cleanText(brief?.traffic, "Traffic unavailable"),
    expected: {
      summary: customComparison
        ? "Custom comparison · Run to observe this algorithm"
        : cleanText(expected?.summary, "Expected behavior unavailable"),
      groups: customComparison ? [] : groupAdmissions(expected?.admissions ?? []),
      cases: customComparison ? [] : expected?.cases ?? [],
    },
    ...(scenario.lesson?.trim()
      ? { lesson: { label: conceptual ? "Concept" as const : "Lesson" as const, text: scenario.lesson.trim() } }
      : {}),
    ...(scenario.tier === "system" && !conceptual
      ? { goPath: "System scenarios run through the Go end-to-end path." }
      : {}),
  };
}
