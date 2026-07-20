import test from "node:test";
import {runContract} from "../../../../support/js/contract-support.mjs";
import {evaluate} from "./interpreter.mjs";
test("gof.behavioral.interpreter shared contract",async()=>{await runContract(import.meta.url,evaluate);});
