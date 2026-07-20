import test from "node:test";
import {runContract} from "../../../support/js/contract-support.mjs";
import {evaluate} from "./fan_out_fan_in.mjs";

test("concurrency.fan-out-fan-in shared contract", async () => {
    await runContract(import.meta.url, evaluate);
});
