from __future__ import annotations

import asyncio
import copy
import inspect
import json
from pathlib import Path


def _resolve(value):
    if inspect.isawaitable(value):
        return asyncio.run(value)
    return value


def run_contract(test_file, evaluate):
    contract_path = Path(test_file).resolve().parents[1] / "fixtures" / "contract.json"
    contract = json.loads(contract_path.read_text(encoding="utf-8"))

    for case in contract["cases"]:
        try:
            result = _resolve(evaluate(copy.deepcopy(case["input"])))
        except Exception as error:
            if "expected_error" not in case:
                raise AssertionError(f"{case['id']}: unexpected error: {error}") from error
            actual_code = getattr(error, "code", None)
            assert actual_code == case["expected_error"]["code"], (
                f"{case['id']}: error code {actual_code!r} != "
                f"{case['expected_error']['code']!r}"
            )
        else:
            assert "expected_error" not in case, (
                f"{case['id']}: expected error {case['expected_error']['code']!r}, "
                f"got result {result!r}"
            )
            assert result == case["expected"], (
                f"{case['id']}: result {result!r} != {case['expected']!r}"
            )

