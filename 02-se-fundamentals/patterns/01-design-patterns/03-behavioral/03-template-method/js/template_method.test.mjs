import test from "node:test";
import {runContract} from "../../../../support/js/contract-support.mjs";
import {evaluate} from "./template_method.mjs";
test("gof.behavioral.template-method shared contract",async()=>{await runContract(import.meta.url,evaluate);});
