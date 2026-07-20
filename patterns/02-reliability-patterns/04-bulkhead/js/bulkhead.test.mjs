import test from "node:test";
import {runContract} from "../../../support/js/contract-support.mjs";
import {evaluate} from "./bulkhead.mjs";
test("reliability.bulkhead shared contract",async()=>{await runContract(import.meta.url,evaluate);});
