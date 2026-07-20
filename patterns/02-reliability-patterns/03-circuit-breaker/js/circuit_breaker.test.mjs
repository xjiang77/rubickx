import test from "node:test";
import {runContract} from "../../../support/js/contract-support.mjs";
import {evaluate} from "./circuit_breaker.mjs";
test("reliability.circuit-breaker shared contract",async()=>{await runContract(import.meta.url,evaluate);});
