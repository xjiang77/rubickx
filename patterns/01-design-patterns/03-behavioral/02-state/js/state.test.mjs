import test from "node:test";
import {runContract} from "../../../../support/js/contract-support.mjs";
import {evaluate} from "./state.mjs";
test("gof.behavioral.state shared contract",async()=>{await runContract(import.meta.url,evaluate);});
