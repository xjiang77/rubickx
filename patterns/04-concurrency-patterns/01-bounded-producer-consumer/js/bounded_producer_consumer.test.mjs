import test from "node:test";
import {runContract} from "../../../support/js/contract-support.mjs";
import {evaluate} from "./bounded_producer_consumer.mjs";

test("concurrency.bounded-producer-consumer shared contract", async () => {
    await runContract(import.meta.url, evaluate);
});
