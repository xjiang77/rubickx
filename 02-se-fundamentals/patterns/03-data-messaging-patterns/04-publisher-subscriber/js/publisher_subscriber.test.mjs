import test from "node:test";
import {runContract} from "../../../support/js/contract-support.mjs";
import {evaluate} from "./publisher_subscriber.mjs";

test("data-messaging.publisher-subscriber shared contract", async () => {
    await runContract(import.meta.url, evaluate);
});
