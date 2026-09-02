import test from "node:test";
import { runContract } from "../../../../support/js/contract-support.mjs";
import { evaluate } from "./builder.mjs";

test("gof.creational.builder shared contract", async () => {
  await runContract(import.meta.url, evaluate);
});
