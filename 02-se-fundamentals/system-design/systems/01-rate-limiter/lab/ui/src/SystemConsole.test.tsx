import { screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { SystemConsole, SystemExchange, SystemRequestLog } from "./SystemConsole";
import { renderWithI18n } from "./test/renderWithI18n";
import type { DemoExchange, DemoOptions } from "./types";

const options: DemoOptions = {
  store: "memory",
  failure: "fail-open",
  replica: "a",
  clientKey: "alice",
  limit: 3,
  windowMs: 1_000,
};

const exchange: DemoExchange = {
  id: 1,
  sentAt: 123,
  url: "/demo/orders?store=redis&failure=fail-open&replica=b",
  key: "alice",
  status: 200,
  statusText: "Server supplied status",
  headers: {
    limit: "3",
    remaining: "2",
    reset: "1000",
    retryAfter: "",
    degraded: "true",
  },
  body: JSON.stringify({ allowed: true, reason: "server data stays English" }),
};

describe("SystemConsole i18n", () => {
  it("translates the conceptual scenario chrome into Chinese", () => {
    renderWithI18n(
      <SystemConsole
        options={options}
        busy={false}
        conceptual
        burstPreferred={false}
        showReplica={false}
        onOptions={vi.fn()}
        onSend={vi.fn()}
      />,
      { lang: "zh" },
    );

    expect(screen.getByText("概念场景")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "多区域配额" })).toBeInTheDocument();
    expect(screen.getByText("比较区域分配与强一致的全局配额。")).toBeInTheDocument();
    expect(screen.getByText("低延迟")).toBeInTheDocument();
    expect(screen.getByText("全局精确")).toBeInTheDocument();
    expect(screen.getByText("可用性")).toBeInTheDocument();
    expect(screen.getByText("不模拟任何集群。本场景是一次决策练习。")).toBeInTheDocument();
  });

  it("translates form chrome while preserving English ARIA names and protocol values", () => {
    renderWithI18n(
      <SystemConsole
        options={options}
        busy
        conceptual={false}
        burstPreferred
        showReplica
        onOptions={vi.fn()}
        onSend={vi.fn()}
      />,
      { lang: "zh" },
    );

    expect(screen.getByText("真实 HTTP 中间件")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "向 Go 链路发送流量" })).toBeInTheDocument();
    expect(screen.getByText("逐个响应地检查公开的限流契约。")).toBeInTheDocument();

    expect(screen.getByRole("combobox", { name: "Store" })).toHaveValue("memory");
    expect(screen.getByRole("combobox", { name: "Failure policy" })).toHaveValue("fail-open");
    expect(screen.getByRole("combobox", { name: "Replica" })).toHaveValue("a");
    expect(screen.getByRole("textbox", { name: "Client key" })).toHaveValue("alice");
    expect(screen.getByText("存储")).toBeInTheDocument();
    expect(screen.getByText("故障策略")).toBeInTheDocument();
    expect(screen.getByText("副本")).toBeInTheDocument();
    expect(screen.getByText("客户端 Key")).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "内存" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "Redis" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "失败放行" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "失败拒绝" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "副本 A" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "副本 B" })).toBeInTheDocument();

    const policy = screen.getByText("限额").closest("dl");
    expect(policy).not.toBeNull();
    expect(within(policy!).getByText("窗口")).toBeInTheDocument();
    expect(within(policy!).getByText("传输")).toBeInTheDocument();
    expect(within(policy!).getByText("3")).toBeInTheDocument();
    expect(within(policy!).getByText("1000 ms")).toBeInTheDocument();
    expect(within(policy!).getByText("GET")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "发送请求" })).toBeDisabled();
    expect(screen.getByRole("button", { name: /连发 ×10/ })).toBeDisabled();
    expect(screen.getByText("默认")).toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("正在发送流量…");
  });

  it("translates request and exchange empty states without changing their ARIA labels", () => {
    renderWithI18n(
      <>
        <SystemRequestLog exchanges={[]} />
        <SystemExchange />
      </>,
      { lang: "zh" },
    );

    expect(screen.getByText("尚无 HTTP 请求。")).toBeInTheDocument();
    expect(screen.getByText("使用系统控制台驱动中间件。")).toBeInTheDocument();
    const exchangeState = screen.getByLabelText("HTTP exchange");
    expect(within(exchangeState).getByText("响应契约将显示在这里")).toBeInTheDocument();
    expect(within(exchangeState).getByText("状态、配额响应头与响应体来自真实端点。")).toBeInTheDocument();
  });

  it.each([
    { status: 200, expected: "允许" },
    { status: 429, expected: "受限" },
  ])("uses the translated $expected status instead of server statusText for HTTP $status", ({ status, expected }) => {
    renderWithI18n(
      <SystemExchange exchange={{ ...exchange, status }} />,
      { lang: "zh" },
    );

    const rendered = screen.getByLabelText("HTTP exchange");
    expect(within(rendered).getByText("HTTP 状态")).toBeInTheDocument();
    expect(within(rendered).getByText(expected)).toBeInTheDocument();
    expect(within(rendered).queryByText(exchange.statusText)).not.toBeInTheDocument();
    expect(within(rendered).getByText("存储不可用 · 降级响应")).toBeInTheDocument();
    expect(within(rendered).getByText(exchange.url)).toBeInTheDocument();
    expect(within(rendered).getByText(`X-RateLimit-Key: ${exchange.key}`)).toBeInTheDocument();
    expect(within(rendered).getByText("RateLimit-Limit")).toBeInTheDocument();
    expect(within(rendered).getByText("RateLimit-Remaining")).toBeInTheDocument();
    expect(within(rendered).getByText("RateLimit-Reset")).toBeInTheDocument();
    expect(within(rendered).getByText("Retry-After")).toBeInTheDocument();
    expect(within(rendered).getByRole("heading", { name: "响应体" })).toBeInTheDocument();
    expect(within(rendered).getByText(/server data stays English/)).toBeInTheDocument();
  });

  it("keeps server statusText and HTTP data unchanged in English", () => {
    renderWithI18n(<SystemExchange exchange={{ ...exchange, status: 429 }} />);

    const rendered = screen.getByLabelText("HTTP exchange");
    expect(within(rendered).getByText("HTTP status")).toBeInTheDocument();
    expect(within(rendered).getByText("Server supplied status")).toBeInTheDocument();
    expect(within(rendered).getByText(exchange.url)).toBeInTheDocument();
    expect(within(rendered).getByText(`X-RateLimit-Key: ${exchange.key}`)).toBeInTheDocument();
    expect(within(rendered).getByRole("heading", { name: "Response body" })).toBeInTheDocument();
    expect(within(rendered).getByText(/server data stays English/)).toBeInTheDocument();
  });
});
