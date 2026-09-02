import test from "node:test";
import {runContract} from "../../../support/js/contract-support.mjs";
import {evaluate} from "./dead_letter_channel.mjs";

test("data-messaging.dead-letter-channel shared contract", async () => {
    await runContract(import.meta.url, evaluate);
});
