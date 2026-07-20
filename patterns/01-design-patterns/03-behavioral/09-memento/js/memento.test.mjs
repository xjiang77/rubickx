import test from "node:test";
import {runContract} from "../../../../support/js/contract-support.mjs";
import {evaluate} from "./memento.mjs";
test("gof.behavioral.memento shared contract",async()=>{await runContract(import.meta.url,evaluate);});
