import test from "node:test";
import {runContract} from "../../../support/js/contract-support.mjs";
import {evaluate} from "./retry.mjs";
test("reliability.retry shared contract",async()=>{await runContract(import.meta.url,evaluate);});
