import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { App } from "./App";
import { SOFT_LINES } from "./config";
import { I18nProvider } from "./i18n";

const rootElement = document.getElementById("root")!;
if (SOFT_LINES) {
  rootElement.dataset.softLines = "true";
} else {
  delete rootElement.dataset.softLines;
}

createRoot(rootElement).render(
  <StrictMode>
    <I18nProvider>
      <App />
    </I18nProvider>
  </StrictMode>,
);
