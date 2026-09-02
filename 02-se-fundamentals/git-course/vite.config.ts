import { resolve } from "node:path";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

export default defineConfig({
  base: "./",
  plugins: [react()],
  build: {
    outDir: "dist",
    emptyOutDir: true,
    rollupOptions: {
      input: {
        catalog: resolve(import.meta.dirname, "index.html"),
        git: resolve(import.meta.dirname, "git/index.html"),
      },
    },
  },
  test: {
    environment: "jsdom",
    setupFiles: ["./src/test/setup.ts"],
    css: true,
    globals: true,
    exclude: ["e2e/**", "node_modules/**", "dist/**"],
  },
});
