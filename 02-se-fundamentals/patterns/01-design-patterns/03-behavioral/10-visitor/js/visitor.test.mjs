import test from "node:test";
import {runContract} from "../../../../support/js/contract-support.mjs";
import {evaluate} from "./visitor.mjs";
test("gof.behavioral.visitor shared contract",async()=>{await runContract(import.meta.url,evaluate);});
