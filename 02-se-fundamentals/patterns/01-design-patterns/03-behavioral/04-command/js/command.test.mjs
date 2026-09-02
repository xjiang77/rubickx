import test from "node:test";
import {runContract} from "../../../../support/js/contract-support.mjs";
import {evaluate} from "./command.mjs";
test("gof.behavioral.command shared contract",async()=>{await runContract(import.meta.url,evaluate);});
