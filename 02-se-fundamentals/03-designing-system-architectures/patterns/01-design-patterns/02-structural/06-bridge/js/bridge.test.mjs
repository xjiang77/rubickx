import test from "node:test";
import {runContract} from "../../../../support/js/contract-support.mjs";
import {evaluate} from "./bridge.mjs";
test("gof.structural.bridge shared contract",async()=>{await runContract(import.meta.url,evaluate);});
