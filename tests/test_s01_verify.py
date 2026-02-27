#!/usr/bin/env python3
"""
test_s01_verify.py - Verification test for s01 agent loop

Tests:
1. API connectivity (simple message)
2. Tool use round-trip (agent calls bash, gets result, responds)

Usage:
    Ensure .env is configured with ANTHROPIC_API_KEY and MODEL_ID, then:
    python tests/test_s01_verify.py
"""

import os
import subprocess
import sys

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from dotenv import load_dotenv

load_dotenv(override=True)


def test_api_connectivity():
    """Test 1: Basic API call works."""
    from anthropic import Anthropic

    client = Anthropic(base_url=os.getenv("ANTHROPIC_BASE_URL"))
    resp = client.messages.create(
        model=os.environ["MODEL_ID"],
        messages=[{"role": "user", "content": "Say OK"}],
        max_tokens=10,
    )
    assert resp.stop_reason == "end_turn", f"Unexpected stop_reason: {resp.stop_reason}"
    text = resp.content[0].text.strip()
    assert len(text) > 0, "Empty response"
    print(f"  API connectivity: OK (response: {text!r})")


def test_tool_use_roundtrip():
    """Test 2: Model calls a tool, receives result, and responds."""
    from anthropic import Anthropic

    client = Anthropic(base_url=os.getenv("ANTHROPIC_BASE_URL"))
    tools = [{
        "name": "bash",
        "description": "Run a shell command.",
        "input_schema": {
            "type": "object",
            "properties": {"command": {"type": "string"}},
            "required": ["command"],
        },
    }]

    messages = [{"role": "user", "content": "Run: echo AGENT_TEST_OK"}]

    # First call - expect tool_use
    resp = client.messages.create(
        model=os.environ["MODEL_ID"],
        system="You are a coding agent. Use bash to solve tasks. Act, don't explain.",
        messages=messages, tools=tools, max_tokens=2000,
    )
    assert resp.stop_reason == "tool_use", f"Expected tool_use, got: {resp.stop_reason}"

    # Find the tool call
    tool_block = next(b for b in resp.content if b.type == "tool_use")
    cmd = tool_block.input["command"]
    print(f"  Model called: $ {cmd}")

    # Execute and feed result back
    r = subprocess.run(cmd, shell=True, capture_output=True, text=True, timeout=10)
    output = (r.stdout + r.stderr).strip()
    print(f"  Tool output: {output!r}")

    messages.append({"role": "assistant", "content": resp.content})
    messages.append({"role": "user", "content": [
        {"type": "tool_result", "tool_use_id": tool_block.id, "content": output}
    ]})

    # Second call - expect end_turn with text
    resp2 = client.messages.create(
        model=os.environ["MODEL_ID"],
        system="You are a coding agent. Use bash to solve tasks. Act, don't explain.",
        messages=messages, tools=tools, max_tokens=2000,
    )
    assert resp2.stop_reason == "end_turn", f"Expected end_turn, got: {resp2.stop_reason}"
    print(f"  Tool use round-trip: OK")


if __name__ == "__main__":
    print("=== s01 Agent Loop Verification ===\n")

    tests = [
        ("API Connectivity", test_api_connectivity),
        ("Tool Use Round-trip", test_tool_use_roundtrip),
    ]

    passed = 0
    for name, fn in tests:
        print(f"[Test] {name}")
        try:
            fn()
            passed += 1
            print(f"  PASSED\n")
        except Exception as e:
            print(f"  FAILED: {e}\n")

    print(f"=== Results: {passed}/{len(tests)} passed ===")
    sys.exit(0 if passed == len(tests) else 1)
