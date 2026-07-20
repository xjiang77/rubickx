import test from "node:test";
import {runContract} from "../../../../support/js/contract-support.mjs";
import {evaluate} from "./iterator.mjs";
test("gof.behavioral.iterator shared contract",async()=>{await runContract(import.meta.url,evaluate);});
