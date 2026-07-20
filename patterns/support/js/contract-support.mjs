import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";

export async function runContract(testModuleUrl, evaluate) {
  const contract = JSON.parse(
    await readFile(new URL("../fixtures/contract.json", testModuleUrl), "utf8"),
  );

  for (const scenario of contract.cases) {
    try {
      const result = await evaluate(structuredClone(scenario.input));
      assert.equal(
        scenario.expected_error,
        undefined,
        `${scenario.id}: expected an error, got ${JSON.stringify(result)}`,
      );
      assert.deepEqual(result, scenario.expected, scenario.id);
    } catch (error) {
      if (scenario.expected_error === undefined) {
        throw error;
      }
      assert.equal(error.code, scenario.expected_error.code, scenario.id);
    }
  }
}
