import "@testing-library/jest-dom/vitest";
import { afterEach, beforeEach, vi } from "vitest";

beforeEach(() => {
  try {
    localStorage.removeItem("rl-lab-uiLang");
  } catch {
    // A test may deliberately replace the storage implementation.
  }
  document.documentElement.lang = "en";
});

afterEach(() => {
  vi.unstubAllEnvs();
  vi.restoreAllMocks();
});
