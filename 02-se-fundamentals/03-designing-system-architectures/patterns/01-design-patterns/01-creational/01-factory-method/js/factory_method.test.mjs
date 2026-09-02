import test from "node:test";
import { runContract } from "../../../../support/js/contract-support.mjs";
import { evaluate } from "./factory_method.mjs";

test("gof.creational.factory-method shared contract", async () => {
  await runContract(import.meta.url, evaluate);
});
