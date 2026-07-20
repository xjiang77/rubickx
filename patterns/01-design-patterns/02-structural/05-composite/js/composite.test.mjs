import test from "node:test";
import {runContract} from "../../../../support/js/contract-support.mjs";
import {evaluate} from "./composite.mjs";
test("gof.structural.composite shared contract",async()=>{await runContract(import.meta.url,evaluate);});
