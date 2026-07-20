import test from "node:test";
import assert from "node:assert/strict";
import {runContract} from "../../../support/js/contract-support.mjs";
import {evaluate} from "./structured_concurrency_cancellation.mjs";

test("concurrency.structured-concurrency-cancellation shared contract", async () => {
    await runContract(import.meta.url, evaluate);
});

test("failure cancels sibling and all promises join", async () => {
    let releaseStart;
    let releaseCancellation;
    const start = new Promise((resolve) => { releaseStart = resolve; });
    const cancelled = new Promise((resolve) => { releaseCancellation = resolve; });
    const events = [];

    const failing = (async () => {
        await start;
        events.push("failed");
        releaseCancellation();
    })();
    const sibling = (async () => {
        await start;
        await cancelled;
        events.push("cancelled");
    })();

    releaseStart();
    await Promise.all([failing, sibling]);
    assert.deepEqual(events, ["failed", "cancelled"]);
});
