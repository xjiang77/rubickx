import test from "node:test";
import {runContract} from "../../../support/js/contract-support.mjs";
import {evaluate} from "./pipeline.mjs";

test("concurrency.pipeline shared contract", async () => {
    await runContract(import.meta.url, evaluate);
});
