import test from "node:test";
import {runContract} from "../../../../support/js/contract-support.mjs";
import {evaluate} from "./decorator.mjs";
test("gof.structural.decorator shared contract",async()=>{await runContract(import.meta.url,evaluate);});
