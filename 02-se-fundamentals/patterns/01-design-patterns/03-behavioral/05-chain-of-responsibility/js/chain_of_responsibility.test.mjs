import test from "node:test";
import {runContract} from "../../../../support/js/contract-support.mjs";
import {evaluate} from "./chain_of_responsibility.mjs";
test("gof.behavioral.chain-of-responsibility shared contract",async()=>{await runContract(import.meta.url,evaluate);});
