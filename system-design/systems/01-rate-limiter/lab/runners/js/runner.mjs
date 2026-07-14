#!/usr/bin/env node
import { createInterface } from "node:readline";

import { InvalidRequest, runRequest } from "./algorithms.mjs";

function error(code, message) {
  return { error: { code, message } };
}

export function handleLine(line) {
  let request;
  try {
    request = JSON.parse(line);
  } catch {
    return error("invalid_json", "invalid JSON");
  }
  try {
    return runRequest(request);
  } catch (failure) {
    if (failure instanceof InvalidRequest) return error("invalid_request", failure.message);
    return error("internal_error", `runner failed: ${failure?.constructor?.name ?? "Error"}`);
  }
}

const input = createInterface({ input: process.stdin, crlfDelay: Infinity });
for await (const line of input) {
  if (line.trim().length === 0) continue;
  process.stdout.write(`${JSON.stringify(handleLine(line))}\n`);
}
