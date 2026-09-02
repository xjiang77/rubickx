import test from "node:test";
import { runContract } from "../../../../support/js/contract-support.mjs";
import { evaluate } from "./prototype.mjs";

test("gof.creational.prototype shared contract", async () => {
  await runContract(import.meta.url, evaluate);
});
