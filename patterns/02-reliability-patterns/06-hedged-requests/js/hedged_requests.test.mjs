import test from "node:test";
import {runContract} from "../../../support/js/contract-support.mjs";
import {evaluate} from "./hedged_requests.mjs";
test("reliability.hedged-requests shared contract",async()=>{await runContract(import.meta.url,evaluate);});
