import test from "node:test";
import {runContract} from "../../../../support/js/contract-support.mjs";
import {evaluate} from "./flyweight.mjs";
test("gof.structural.flyweight shared contract",async()=>{await runContract(import.meta.url,evaluate);});
