import test from "node:test";
import {runContract} from "../../../support/js/contract-support.mjs";
import {evaluate} from "./cqrs.mjs";

test("data-messaging.cqrs shared contract", async () => {
    await runContract(import.meta.url, evaluate);
});
