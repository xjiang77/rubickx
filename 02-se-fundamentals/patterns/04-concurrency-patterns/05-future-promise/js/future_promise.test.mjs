import test from "node:test";
import {runContract} from "../../../support/js/contract-support.mjs";
import {evaluate} from "./future_promise.mjs";

test("concurrency.future-promise shared contract", async () => {
    await runContract(import.meta.url, evaluate);
});
