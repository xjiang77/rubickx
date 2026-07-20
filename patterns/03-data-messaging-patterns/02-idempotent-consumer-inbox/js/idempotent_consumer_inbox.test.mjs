import test from "node:test";
import {runContract} from "../../../support/js/contract-support.mjs";
import {evaluate} from "./idempotent_consumer_inbox.mjs";

test("data-messaging.idempotent-consumer-inbox shared contract", async () => {
    await runContract(import.meta.url, evaluate);
});
