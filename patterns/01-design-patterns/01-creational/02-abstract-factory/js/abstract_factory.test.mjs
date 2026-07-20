import test from "node:test";
import { runContract } from "../../../../support/js/contract-support.mjs";
import { evaluate } from "./abstract_factory.mjs";

test("gof.creational.abstract-factory shared contract", async () => {
  await runContract(import.meta.url, evaluate);
});
