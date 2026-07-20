import { render, type RenderOptions, type RenderResult } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";
import { I18nProvider, type UiLang } from "../i18n";

interface RenderWithI18nOptions extends Omit<RenderOptions, "wrapper"> {
  lang?: UiLang;
}

export function renderWithI18n(
  ui: ReactElement,
  { lang, ...renderOptions }: RenderWithI18nOptions = {},
): RenderResult {
  if (lang) {
    localStorage.setItem("rl-lab-uiLang", lang);
  }

  function Wrapper({ children }: { children: ReactNode }) {
    return <I18nProvider>{children}</I18nProvider>;
  }

  return render(ui, { wrapper: Wrapper, ...renderOptions });
}
