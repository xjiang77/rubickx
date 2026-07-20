import test from "node:test";
import {runContract} from "../../../support/js/contract-support.mjs";
import {evaluate} from "./saga.mjs";

test("data-messaging.saga shared contract", async () => {
    await runContract(import.meta.url, evaluate);
});
