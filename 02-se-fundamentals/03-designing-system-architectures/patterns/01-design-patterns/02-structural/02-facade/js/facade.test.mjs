import test from "node:test";
import {runContract} from "../../../../support/js/contract-support.mjs";
import {evaluate} from "./facade.mjs";
test("gof.structural.facade shared contract",async()=>{await runContract(import.meta.url,evaluate);});
