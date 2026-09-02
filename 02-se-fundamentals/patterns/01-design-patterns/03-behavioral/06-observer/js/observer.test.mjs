import test from "node:test";
import {runContract} from "../../../../support/js/contract-support.mjs";
import {evaluate} from "./observer.mjs";
test("gof.behavioral.observer shared contract",async()=>{await runContract(import.meta.url,evaluate);});
