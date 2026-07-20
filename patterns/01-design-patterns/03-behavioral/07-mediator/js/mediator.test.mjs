import test from "node:test";
import {runContract} from "../../../../support/js/contract-support.mjs";
import {evaluate} from "./mediator.mjs";
test("gof.behavioral.mediator shared contract",async()=>{await runContract(import.meta.url,evaluate);});
