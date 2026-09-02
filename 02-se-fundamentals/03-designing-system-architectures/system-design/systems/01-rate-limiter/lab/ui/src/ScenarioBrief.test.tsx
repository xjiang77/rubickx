import { screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ScenarioBrief } from "./ScenarioBrief";
import type { ScenarioBriefViewModel } from "./scenarioBriefModel";
import { renderWithI18n as render } from "./test/renderWithI18n";

describe("ScenarioBrief", () => {
  it("makes a core scenario policy, traffic, expected admissions, and lesson visible before a run", () => {
    const model: ScenarioBriefViewModel = {
      policy: "capacity 3 · refill 2/s",
      traffic: "alice · 5 requests @ 0/0/0/0/500 ms",
      expected: {
        summary: "The initial burst exhausts the bucket before refill restores one token.",
        groups: [
          { kind: "allow", label: "#1–3" },
          { kind: "deny", label: "#4" },
          { kind: "allow", label: "#5 after 500 ms" },
        ],
        cases: [],
      },
      lesson: {
        label: "Lesson",
        text: "Burst capacity absorbs short spikes without raising the steady refill rate.",
      },
      goPath: "Go runs in-process through the semantic trace adapter.",
    };

    render(<ScenarioBrief model={model} />);

    const brief = screen.getByRole("region", { name: "Scenario brief" });
    expect(within(brief).getByRole("heading", { name: "Policy" })).toBeInTheDocument();
    expect(within(brief).getByText("capacity 3 · refill 2/s")).toBeInTheDocument();
    expect(within(brief).getByRole("heading", { name: "Traffic" })).toBeInTheDocument();
    expect(within(brief).getByText("alice · 5 requests @ 0/0/0/0/500 ms")).toBeInTheDocument();

    const sequence = within(brief).getByRole("list", { name: "Expected admission sequence" });
    expect(within(sequence).getAllByText("ALLOW")).toHaveLength(2);
    expect(within(sequence).getByText("DENY")).toBeInTheDocument();
    expect(within(sequence).getByText("#1–3")).toBeInTheDocument();
    expect(within(sequence).getByText("#4")).toBeInTheDocument();
    expect(within(sequence).getByText("#5 after 500 ms")).toBeInTheDocument();
    expect(within(brief).getByText(model.expected.summary)).toBeInTheDocument();

    expect(within(brief).getByText("Lesson")).toBeInTheDocument();
    expect(within(brief).getByText(model.lesson!.text)).toBeInTheDocument();
    expect(within(brief).getByText("Go path")).toBeInTheDocument();
    expect(within(brief).getByText(model.goPath!)).toBeInTheDocument();
  });

  it("renders conditional system outcomes with explicit decision text", () => {
    const model: ScenarioBriefViewModel = {
      policy: "limit 3 · window 1,000 ms · store Redis · failure fail-open",
      traffic: "alice · Burst ×10",
      expected: {
        summary: "The result depends on Redis health and the selected failure policy.",
        groups: [],
        cases: [
          { when: "Redis healthy", result: "3×200 + 7×429", kind: "observe" },
          { when: "Redis unavailable · fail-open", result: "200 + degraded + bypass", kind: "allow" },
          { when: "Redis unavailable · fail-closed", result: "503 + degraded + enforced", kind: "deny" },
        ],
      },
      lesson: {
        label: "Concept",
        text: "Availability policy changes behavior during a shared-store outage.",
      },
    };

    render(<ScenarioBrief model={model} />);

    const cases = screen.getByRole("list", { name: "Expected behavior cases" });
    expect(within(cases).getByText("OBSERVE")).toBeInTheDocument();
    expect(within(cases).getByText("ALLOW")).toBeInTheDocument();
    expect(within(cases).getByText("DENY")).toBeInTheDocument();
    expect(within(cases).getByText("Redis healthy")).toBeInTheDocument();
    expect(within(cases).getByText("3×200 + 7×429")).toBeInTheDocument();
    expect(screen.queryByText("Go path")).not.toBeInTheDocument();
  });

  it("keeps a custom comparison concise when no canonical decision sequence applies", () => {
    const model: ScenarioBriefViewModel = {
      policy: "limit 3 · window 1,000 ms",
      traffic: "alice · 6 requests @ 800/900/999/1,000/1,001/1,100 ms",
      expected: {
        summary: "Custom comparison · Run to observe this algorithm",
        groups: [],
        cases: [],
      },
    };

    render(<ScenarioBrief model={model} />);

    expect(screen.getByText("Custom comparison · Run to observe this algorithm")).toBeInTheDocument();
    expect(screen.queryByRole("list", { name: "Expected admission sequence" })).not.toBeInTheDocument();
    expect(screen.queryByRole("list", { name: "Expected behavior cases" })).not.toBeInTheDocument();
  });

  it("translates chrome while preserving scenario content and decision tokens", () => {
    localStorage.setItem("rl-lab-uiLang", "zh");
    const model: ScenarioBriefViewModel = {
      policy: "limit 3 · window 1,000 ms",
      traffic: "alice @ 0/100/200 ms",
      expected: {
        summary: "The first three requests are allowed.",
        groups: [{ kind: "allow", label: "#1–3" }],
        cases: [],
      },
      lesson: { label: "Concept", text: "Catalog lesson stays English." },
      goPath: "System scenarios run through the Go end-to-end path.",
    };

    render(<ScenarioBrief model={model} />);

    const brief = screen.getByRole("region", { name: "Scenario brief" });
    expect(within(brief).getByRole("heading", { name: "场景简报" })).toBeInTheDocument();
    expect(within(brief).getByRole("heading", { name: "策略" })).toBeInTheDocument();
    expect(within(brief).getByRole("heading", { name: "流量" })).toBeInTheDocument();
    expect(within(brief).getByRole("heading", { name: "预期" })).toBeInTheDocument();
    expect(within(brief).getByText("概念")).toBeInTheDocument();
    expect(within(brief).getByText("Go 链路")).toBeInTheDocument();
    expect(within(brief).getByText("System 场景通过 Go 端到端链路运行。")).toBeInTheDocument();
    expect(within(brief).getByText(model.policy)).toBeInTheDocument();
    expect(within(brief).getByText(model.traffic)).toBeInTheDocument();
    expect(within(brief).getByText(model.expected.summary)).toBeInTheDocument();
    expect(within(brief).getByText(model.lesson!.text)).toBeInTheDocument();
    expect(within(brief).getByText("ALLOW")).toBeInTheDocument();
  });
});
