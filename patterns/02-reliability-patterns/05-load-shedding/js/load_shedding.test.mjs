import test from "node:test";
import {runContract} from "../../../support/js/contract-support.mjs";
import {evaluate} from "./load_shedding.mjs";
test("reliability.load-shedding shared contract",async()=>{await runContract(import.meta.url,evaluate);});
