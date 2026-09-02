import test from "node:test";
import {runContract} from "../../../support/js/contract-support.mjs";
import {evaluate} from "./event_sourcing.mjs";

test("data-messaging.event-sourcing shared contract", async () => {
    await runContract(import.meta.url, evaluate);
});
