import test from "node:test";
import {runContract} from "../../../../support/js/contract-support.mjs";
import {evaluate} from "./proxy.mjs";
test("gof.structural.proxy shared contract",async()=>{await runContract(import.meta.url,evaluate);});
