import { describe, expect, it } from "vitest";
import { deriveScenarioBriefModel } from "./scenarioBriefModel";
import type { CatalogScenario } from "./types";

describe("deriveScenarioBriefModel", () => {
  it("formats a canonical token-bucket scenario and groups consecutive admissions", () => {
    const scenario = {
      id: "burst-capacity",
      label: "Burst capacity",
      tier: "core",
      algorithm: "token-bucket",
      lesson: "A short burst consumes the available tokens.",
      brief: {
        traffic: "alice @ 0/0/0/0/500 ms",
        expected: {
          summary: "The first burst fills the quota, then refill admits the retry.",
          admissions: ["allow", "allow", "allow", "deny", "allow"],
        },
      },
    } as unknown as CatalogScenario;

    expect(deriveScenarioBriefModel({
      scenario,
      algorithm: "token-bucket",
      config: { capacity: 3, ratePerSecond: 2 },
    })).toEqual({
      policy: "capacity 3 · refill 2/s",
      traffic: "alice @ 0/0/0/0/500 ms",
      expected: {
        summary: "The first burst fills the quota, then refill admits the retry.",
        groups: [
          { kind: "allow", label: "#1–3" },
          { kind: "deny", label: "#4" },
          { kind: "allow", label: "#5" },
        ],
        cases: [],
      },
      lesson: {
        label: "Lesson",
        text: "A short burst consumes the available tokens.",
      },
    });
  });

  it.each([
    ["fixed-window", { limit: 3, windowMs: 1_000 }, "limit 3 · window 1,000 ms"],
    ["sliding-window-log", { limit: 3, windowMs: 1_000 }, "limit 3 · window 1,000 ms"],
    ["sliding-window-counter", { limit: 1.5, windowMs: 750 }, "limit 1.5 · window 750 ms"],
    ["leaky-bucket", { capacity: 3, ratePerSecond: 2 }, "capacity 3 · drain 2/s"],
  ])("formats the current %s policy", (algorithm, config, policy) => {
    const scenario = {
      id: "scenario",
      label: "Scenario",
      tier: "core",
      algorithm,
      brief: {
        traffic: "alice @ 0 ms",
        expected: { summary: "One request is admitted.", admissions: ["allow"] },
      },
    } as CatalogScenario;

    expect(deriveScenarioBriefModel({ scenario, algorithm, config }).policy).toBe(policy);
  });

  it("keeps current policy and traffic but removes canonical expectations for a custom comparison", () => {
    const scenario = {
      id: "burst-capacity",
      label: "Burst capacity",
      tier: "core",
      algorithm: "token-bucket",
      lesson: "A token bucket exposes refill timing.",
      brief: {
        policy: "canonical policy",
        traffic: "alice @ 0/0/0/0/500 ms",
        expected: {
          summary: "Canonical outcome",
          admissions: ["allow", "deny"],
          cases: [{ when: "canonical", result: "only", kind: "observe" }],
        },
      },
    } as CatalogScenario;

    expect(deriveScenarioBriefModel({
      scenario,
      algorithm: "fixed-window",
      config: { limit: 3, windowMs: 1_000 },
    })).toMatchObject({
      policy: "limit 3 · window 1,000 ms",
      traffic: "alice @ 0/0/0/0/500 ms",
      expected: {
        summary: "Custom comparison · Run to observe this algorithm",
        groups: [],
        cases: [],
      },
      lesson: {
        label: "Lesson",
        text: "A token bucket exposes refill timing.",
      },
    });
  });

  it("uses a system policy override, appends live demo options, and exposes conditional cases", () => {
    const scenario: CatalogScenario = {
      id: "policy-composition",
      label: "Policy composition",
      tier: "system",
      algorithm: "fixed-window",
      lesson: "Compose policies without hiding which one rejected.",
      brief: {
        policy: "per-client 3 · endpoint-wide 6",
        traffic: "alice · 4 sequential HTTP requests",
        expected: {
          summary: "The per-client policy rejects before the endpoint-wide policy.",
          cases: [
            { when: "alice requests #1–3", result: "HTTP 200", kind: "allow" },
            { when: "alice request #4", result: "HTTP 429 · per-client", kind: "deny" },
          ],
        },
      },
    };

    expect(deriveScenarioBriefModel({
      scenario,
      algorithm: "fixed-window",
      config: { limit: 3, windowMs: 1_000 },
      demoOptions: {
        store: "redis",
        failure: "fail-closed",
        replica: "b",
        clientKey: "alice",
        limit: 3,
        windowMs: 1_000,
      },
    })).toEqual({
      policy: "per-client 3 · endpoint-wide 6 · store Redis · failure fail-closed",
      traffic: "alice · 4 sequential HTTP requests",
      expected: {
        summary: "The per-client policy rejects before the endpoint-wide policy.",
        groups: [],
        cases: [
          { when: "alice requests #1–3", result: "HTTP 200", kind: "allow" },
          { when: "alice request #4", result: "HTTP 429 · per-client", kind: "deny" },
        ],
      },
      lesson: {
        label: "Lesson",
        text: "Compose policies without hiding which one rejected.",
      },
      goPath: "System scenarios run through the Go end-to-end path.",
    });
  });

  it("shows the replica option only for a replica-scoped system scenario", () => {
    const scenario: CatalogScenario = {
      id: "local-vs-shared",
      label: "Local vs shared quota",
      tier: "system",
      algorithm: "fixed-window",
      brief: {
        replicaScoped: true,
        traffic: "alice alternates between Replica A and Replica B",
        expected: {
          summary: "The store controls quota scope.",
          cases: [],
        },
      },
    };

    const model = deriveScenarioBriefModel({
      scenario,
      algorithm: "fixed-window",
      config: { limit: 3, windowMs: 1_000 },
      demoOptions: {
        store: "redis",
        failure: "fail-closed",
        replica: "b",
        clientKey: "alice",
        limit: 3,
        windowMs: 1_000,
      },
    });

    expect(model.policy).toBe("limit 3 · window 1,000 ms · store Redis · failure fail-closed · replica B");
  });

  it("keeps a catalog-defined conceptual policy free of live options", () => {
    const scenario: CatalogScenario = {
      id: "multi-region-quota",
      label: "Multi-region quota",
      tier: "system",
      algorithm: "fixed-window",
      lesson: "Global accuracy trades latency against availability.",
      brief: {
        conceptual: true,
        policy: "No live limiter",
        traffic: "No HTTP requests · decision exercise",
        expected: {
          summary: "Compare regional allocation with a global quota.",
          cases: [{ when: "Regional allocation", result: "Lower latency", kind: "observe" }],
        },
      },
    };

    const model = deriveScenarioBriefModel({
      scenario,
      algorithm: "fixed-window",
      config: { limit: 3, windowMs: 1_000 },
      demoOptions: {
        store: "redis",
        failure: "fail-open",
        replica: "a",
        clientKey: "alice",
        limit: 3,
        windowMs: 1_000,
      },
    });

    expect(model.policy).toBe("No live limiter");
    expect(model.lesson).toEqual({
      label: "Concept",
      text: "Global accuracy trades latency against availability.",
    });
    expect(model.goPath).toBeUndefined();

    const overridden = deriveScenarioBriefModel({
      scenario,
      algorithm: "token-bucket",
      config: { capacity: 3, ratePerSecond: 2 },
      demoOptions: {
        store: "memory",
        failure: "fail-open",
        replica: "a",
        clientKey: "alice",
        limit: 3,
        windowMs: 1_000,
      },
    });
    expect(overridden.policy).toBe("No live limiter");
    expect(overridden.goPath).toBeUndefined();
  });

  it("does not infer conceptual behavior from policy display copy", () => {
    const scenario: CatalogScenario = {
      id: "copy-only",
      label: "Copy only",
      tier: "system",
      algorithm: "fixed-window",
      brief: {
        policy: "No live limiter",
        traffic: "One live request",
        expected: { summary: "The display copy is not metadata.", cases: [] },
      },
    };

    const model = deriveScenarioBriefModel({
      scenario,
      algorithm: "fixed-window",
      config: { limit: 3, windowMs: 1_000 },
      demoOptions: {
        store: "memory",
        failure: "fail-open",
        replica: "a",
        clientKey: "alice",
        limit: 3,
        windowMs: 1_000,
      },
    });

    expect(model.goPath).toBe("System scenarios run through the Go end-to-end path.");
  });

  it("returns explicit unavailable copy instead of leaking invalid catalog or config values", () => {
    const scenario = {
      id: "malformed",
      label: "Malformed",
      tier: "core",
      algorithm: "token-bucket",
      brief: { traffic: "   ", expected: { summary: "   " } },
    } as CatalogScenario;

    expect(deriveScenarioBriefModel({
      scenario,
      algorithm: "token-bucket",
      config: { capacity: Number.NaN, ratePerSecond: 2 },
    })).toMatchObject({
      policy: "Policy unavailable",
      traffic: "Traffic unavailable",
      expected: {
        summary: "Expected behavior unavailable",
        groups: [],
        cases: [],
      },
    });
  });
});
