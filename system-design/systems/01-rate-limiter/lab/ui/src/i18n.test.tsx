import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useI18n, I18nProvider, LangToggle } from "./i18n";

const STORAGE_KEY = "rl-lab-uiLang";

function Probe() {
  const { lang, t } = useI18n();
  return (
    <>
      <span>{lang}</span>
      <span>{t.tRunScenario}</span>
      <LangToggle />
    </>
  );
}

describe("I18nProvider", () => {
  it("defaults to English and exposes an accessible language toggle", async () => {
    render(
      <I18nProvider>
        <Probe />
      </I18nProvider>,
    );

    expect(screen.getByText("Run scenario")).toBeInTheDocument();
    expect(screen.getByRole("group", { name: "Interface language" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "EN" })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByRole("button", { name: "中文" })).toHaveAttribute("aria-pressed", "false");
    await waitFor(() => expect(document.documentElement).toHaveAttribute("lang", "en"));
  });

  it("switches to Chinese, persists the choice, and updates the document language", async () => {
    const user = userEvent.setup();
    render(
      <I18nProvider>
        <Probe />
      </I18nProvider>,
    );

    await user.click(screen.getByRole("button", { name: "中文" }));

    expect(screen.getByText("运行场景")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "中文" })).toHaveAttribute("aria-pressed", "true");
    expect(localStorage.getItem(STORAGE_KEY)).toBe("zh");
    await waitFor(() => expect(document.documentElement).toHaveAttribute("lang", "zh-CN"));
  });

  it("restores a language selected before an actual remount", async () => {
    const user = userEvent.setup();
    const mounted = render(
      <I18nProvider>
        <Probe />
      </I18nProvider>,
    );

    await user.click(screen.getByRole("button", { name: "中文" }));
    expect(localStorage.getItem(STORAGE_KEY)).toBe("zh");
    mounted.unmount();

    render(
      <I18nProvider>
        <Probe />
      </I18nProvider>,
    );

    expect(screen.getByText("运行场景")).toBeInTheDocument();
    await waitFor(() => expect(document.documentElement).toHaveAttribute("lang", "zh-CN"));
  });

  it("switches language with keyboard focus and Enter", async () => {
    const user = userEvent.setup();
    render(
      <I18nProvider>
        <Probe />
      </I18nProvider>,
    );

    await user.tab();
    expect(screen.getByRole("button", { name: "EN" })).toHaveFocus();
    await user.tab();
    expect(screen.getByRole("button", { name: "中文" })).toHaveFocus();
    await user.keyboard("{Enter}");

    expect(screen.getByText("运行场景")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "中文" })).toHaveAttribute("aria-pressed", "true");
  });

  it("falls back to English for an invalid persisted language", () => {
    localStorage.setItem(STORAGE_KEY, "fr");

    render(
      <I18nProvider>
        <Probe />
      </I18nProvider>,
    );

    expect(screen.getByText("Run scenario")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "EN" })).toHaveAttribute("aria-pressed", "true");
  });

  it("falls back safely when storage reads fail", () => {
    vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
      throw new DOMException("Storage disabled");
    });

    render(
      <I18nProvider>
        <Probe />
      </I18nProvider>,
    );

    expect(screen.getByText("Run scenario")).toBeInTheDocument();
  });

  it("keeps switching languages when storage writes fail", async () => {
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new DOMException("Storage full");
    });
    const user = userEvent.setup();

    render(
      <I18nProvider>
        <Probe />
      </I18nProvider>,
    );
    await user.click(screen.getByRole("button", { name: "中文" }));

    expect(screen.getByText("运行场景")).toBeInTheDocument();
    await waitFor(() => expect(document.documentElement).toHaveAttribute("lang", "zh-CN"));
  });

  it("fails clearly when useI18n is rendered without its provider", () => {
    expect(() => render(<Probe />)).toThrow("useI18n must be used within I18nProvider");
  });
});

describe("soft lines configuration", () => {
  it("enables soft lines only for the exact true environment value", async () => {
    vi.stubEnv("VITE_SOFT_LINES", "true");
    vi.resetModules();
    expect((await import("./config")).SOFT_LINES).toBe(true);

    vi.stubEnv("VITE_SOFT_LINES", "TRUE");
    vi.resetModules();
    expect((await import("./config")).SOFT_LINES).toBe(false);
  });
});
