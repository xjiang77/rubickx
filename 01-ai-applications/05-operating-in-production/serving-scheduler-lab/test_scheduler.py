#!/usr/bin/env python3
"""Serving 调度概念练习验证器。

运行（仓库根目录）：make test-serving
实现由本人手写（C5）。不得放宽比较、删除用例，或改用 pytest.skip / xfail。
骨架状态下本脚本预期失败，退出码非 0。
数值来自 fixtures.py，全是写死的整数步，不测墙钟。
"""
import doctest
import os
import sys
import traceback
from collections.abc import Iterator

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
sys.path.insert(0, os.path.join(HERE, "playground"))

import fixtures as fx

Case = tuple[str, bool]


def raises(exc: type[BaseException], fn, *args, **kwargs) -> bool:
    """调用 fn 是否抛出 exc。"""
    try:
        fn(*args, **kwargs)
    except exc:
        return True
    return False


def _rows(steps: list[dict]) -> list[tuple[int, int, int, int]]:
    """把一步的账收成 (step, prefill, decode, tokens)。"""
    return [(s["step"], s["prefill"], s["decode"], s["tokens"]) for s in steps]


def _consistent(steps: list[tuple[int, int, int, int]], total: int) -> bool:
    """字面量自身：步号连续，prefill + decode = tokens，且不超过 budget，总和等于 total。"""
    if [row[0] for row in steps] != list(range(len(steps))):
        return False
    if any(prefill < 0 or decode < 0 or prefill + decode != tokens or tokens > fx.TOKEN_BUDGET_PER_STEP
           for _, prefill, decode, tokens in steps):
        return False
    return sum(row[3] for row in steps) == total


def check_fixtures() -> bool:
    """先核对写死的对照数彼此一致。这一步不调用 scheduler.py。"""
    ok = True
    bundles = [
        ("static", fx.STATIC_STEPS, fx.MIXED_TOTAL_TOKENS),
        ("chunked", fx.CHUNKED_STEPS, fx.MIXED_TOTAL_TOKENS),
        ("monolithic", fx.MONOLITHIC_STEPS, fx.MIXED_TOTAL_TOKENS),
        ("late static", fx.LATE_STATIC_STEPS, fx.LATE_TOTAL_TOKENS),
        ("late chunked", fx.LATE_CHUNKED_STEPS, fx.LATE_TOTAL_TOKENS),
    ]
    for name, steps, total in bundles:
        passed = _consistent(steps, total)
        print(("PASS " if passed else "FAIL ") + f"fixtures {name} 步账自洽")
        ok = ok and passed
    passed = (
        fx.CHUNKED_COMPLETION["R_short"] < fx.MONOLITHIC_COMPLETION["R_short"]
        < fx.STATIC_COMPLETION["R_short"]
        and fx.MONOLITHIC_COMPLETION["R_long_prefill"] < fx.CHUNKED_COMPLETION["R_long_prefill"]
        and fx.LATE_CHUNKED_COMPLETION["R_b"] < fx.LATE_STATIC_COMPLETION["R_b"]
        and fx.SECOND_PREFILL_WORK == 12
        and fx.SECOND_PROMPT - fx.SHARED_PREFIX == 12
        and fx.SECOND_DECODE_WORK == fx.SECOND_OUTPUT
        and fx.FIT_LIVE_KV + fx.FIT_KV_NEED == fx.KV_CAPACITY
        and fx.OVER_LIVE_KV + fx.OVER_KV_NEED == 28
        and 28 > fx.KV_CAPACITY
    )
    print(("PASS " if passed else "FAIL ") + "fixtures 完成步与闸门字面量")
    return ok and passed


def check_playground() -> bool:
    """对照表不调用待写函数，但必须印出 fixtures 里的同一组数字。"""
    import comparison

    text = "\n".join(comparison.demo_lines())
    needles = [
        f"budget {fx.TOKEN_BUDGET_PER_STEP} chunk {fx.CHUNK_SIZE} kv {fx.KV_CAPACITY} queue {fx.MAX_QUEUE}",
        f"static {fx.STATIC_COMPLETION['R_short']}, monolithic {fx.MONOLITHIC_COMPLETION['R_short']}, chunked {fx.CHUNKED_COMPLETION['R_short']}",
        f"static {fx.STATIC_TTFT['R_short']}, chunked {fx.CHUNKED_TTFT['R_short']}",
        f"monolithic {fx.MONOLITHIC_COMPLETION['R_long_prefill']}, chunked {fx.CHUNKED_COMPLETION['R_long_prefill']}",
        f"static {fx.LATE_STATIC_COMPLETION['R_b']}, chunked {fx.LATE_CHUNKED_COMPLETION['R_b']}",
        "overload kv",
        "overload queue",
        "admit admit admit",
        f"prefill {fx.SECOND_PREFILL_WORK}, decode {fx.SECOND_DECODE_WORK}",
    ]
    ok = True
    for needle in needles:
        passed = needle in text
        print(("PASS " if passed else "FAIL ") + f"对照表 {needle}")
        ok = ok and passed
    return ok


def cases() -> Iterator[Case]:
    """逐条产出调度、背压和前缀复用的用例。"""
    from scheduler import classify_admission, decode_work, prefill_work, screen_arrivals, simulate

    yield "前缀 4，第二个 prefill 12", prefill_work(fx.SECOND_PROMPT, fx.SHARED_PREFIX) == fx.SECOND_PREFILL_WORK
    yield "前缀不减少 decode", decode_work(fx.SECOND_OUTPUT) == fx.SECOND_DECODE_WORK
    yield "没有前缀时 prefill 等于 prompt", prefill_work(fx.SECOND_PROMPT, 0) == fx.SECOND_PROMPT
    yield "整段都是前缀时 prefill 为 0", prefill_work(fx.SHARED_PREFIX, fx.SHARED_PREFIX) == 0
    yield "前缀长于 prompt", raises(ValueError, prefill_work, fx.SHARED_PREFIX, fx.SECOND_PROMPT)
    yield "prefill 拒绝 bool", raises(ValueError, prefill_work, True, 0)
    yield "decode 可以为 0", decode_work(0) == 0
    yield "decode 拒绝负数", raises(ValueError, decode_work, -1)

    yield "KV 正好满额可以准入", classify_admission(
        fx.FIT_LIVE_KV, 0, fx.FIT_KV_NEED, capacity=fx.KV_CAPACITY, max_queue=fx.MAX_QUEUE,
    ) == fx.ADMIT_OK
    yield "活 KV 放不下则 overload kv", classify_admission(
        fx.OVER_LIVE_KV, 0, fx.OVER_KV_NEED, capacity=fx.KV_CAPACITY, max_queue=fx.MAX_QUEUE,
    ) == fx.OVERLOAD_KV
    yield "队列已满则 overload queue", classify_admission(
        0, fx.FULL_QUEUE, fx.SMALL_KV_NEED, capacity=fx.KV_CAPACITY, max_queue=fx.MAX_QUEUE,
    ) == fx.OVERLOAD_QUEUE
    yield "队列还有一个空位", classify_admission(
        0, fx.ROOM_QUEUE, fx.SMALL_KV_NEED, capacity=fx.KV_CAPACITY, max_queue=fx.MAX_QUEUE,
    ) == fx.ADMIT_OK
    yield "无限队列不因队列长度拒绝", classify_admission(
        0, fx.FULL_QUEUE, fx.SMALL_KV_NEED, capacity=fx.KV_CAPACITY, max_queue=None,
    ) == fx.ADMIT_OK
    yield "无限队列仍然拒绝超 KV", classify_admission(
        fx.OVER_LIVE_KV, 0, fx.OVER_KV_NEED, capacity=fx.KV_CAPACITY, max_queue=None,
    ) == fx.OVERLOAD_KV
    yield "两条闸门同时触发时 reason 是 kv", classify_admission(
        fx.OVER_LIVE_KV, fx.FULL_QUEUE, fx.OVER_KV_NEED, capacity=fx.KV_CAPACITY, max_queue=fx.MAX_QUEUE,
    ) == fx.OVERLOAD_KV
    yield "准入拒绝负数", raises(
        ValueError, classify_admission, -1, 0, 1, capacity=fx.KV_CAPACITY, max_queue=fx.MAX_QUEUE,
    )

    yield "队列上限 2，第三次 overload", screen_arrivals(
        0, fx.QUEUE_NEEDS, capacity=fx.KV_CAPACITY, max_queue=fx.MAX_QUEUE,
    ) == fx.QUEUE_SCREEN
    unbounded = screen_arrivals(0, fx.QUEUE_NEEDS, capacity=fx.KV_CAPACITY, max_queue=None)
    yield "无限队列三次都准入", unbounded == fx.QUEUE_SCREEN_UNBOUNDED
    yield "无限队列长度 3", sum(item["outcome"] == "admit" for item in unbounded) == fx.UNBOUNDED_QUEUE_LEN
    yield "超 KV 的请求不占队列名额", screen_arrivals(
        fx.OVER_LIVE_KV, fx.KV_NEEDS, capacity=fx.KV_CAPACITY, max_queue=fx.MAX_QUEUE,
    ) == fx.KV_SCREEN
    yield "空到达列表", screen_arrivals(0, [], capacity=fx.KV_CAPACITY, max_queue=fx.MAX_QUEUE) == []

    req = {"id": "R_short", "prompt": 4, "output": 2, "arrival": 0}
    snapshot = {"id": "R_short", "prompt": 4, "output": 2, "arrival": 0}
    alone = simulate([req], "chunked")
    yield "不修改调用方的请求", req == snapshot
    yield "只有短请求时 step 2 完成", alone["completions"] == {"R_short": 2}
    yield "prompt 为 0 时当步 decode", simulate(
        [{"id": "R_zero_prompt", "prompt": 0, "output": 1, "arrival": 0}], "static",
    )["completions"] == {"R_zero_prompt": 0}

    for policy, completion, ttft, steps in (
        ("static", fx.STATIC_COMPLETION, fx.STATIC_TTFT, fx.STATIC_STEPS),
        ("chunked", fx.CHUNKED_COMPLETION, fx.CHUNKED_TTFT, fx.CHUNKED_STEPS),
        ("monolithic", fx.MONOLITHIC_COMPLETION, fx.MONOLITHIC_TTFT, fx.MONOLITHIC_STEPS),
    ):
        got = simulate(fx.MIXED, policy)
        yield f"{policy} 完成步", got["completions"] == completion
        yield f"{policy} TTFT", got["ttft"] == ttft
        yield f"{policy} 每步 token", _rows(got["steps"]) == steps
        yield f"{policy} 步内不超过 budget", all(
            s["tokens"] <= fx.TOKEN_BUDGET_PER_STEP and s["tokens"] == s["prefill"] + s["decode"]
            for s in got["steps"]
        )

    late_static = simulate(fx.LATE, "static")
    late_chunked = simulate(fx.LATE, "chunked")
    yield "静态批等当前批跑完", late_static["completions"] == fx.LATE_STATIC_COMPLETION
    yield "静态批下一批的 TTFT", late_static["ttft"] == fx.LATE_STATIC_TTFT
    yield "静态批下一批的步账", _rows(late_static["steps"]) == fx.LATE_STATIC_STEPS
    yield "连续批可以插进后到的请求", late_chunked["completions"] == fx.LATE_CHUNKED_COMPLETION
    yield "连续批后到请求的 TTFT", late_chunked["ttft"] == fx.LATE_CHUNKED_TTFT
    yield "连续批后到请求的步账", _rows(late_chunked["steps"]) == fx.LATE_CHUNKED_STEPS
    yield "空请求列表", simulate([], "monolithic") == {"completions": {}, "ttft": {}, "steps": []}
    yield "未知策略", raises(ValueError, simulate, fx.MIXED, "fair")
    yield "重复 id", raises(ValueError, simulate, [req, dict(snapshot)], "chunked")
    yield "output 至少为 1", raises(
        ValueError, simulate, [{"id": "R_short", "prompt": 4, "output": 0, "arrival": 0}], "chunked",
    )


def run_doctests() -> bool:
    """运行 scheduler.py 与 playground 的 doctest。"""
    import comparison
    import scheduler

    ok = True
    for module in (scheduler, comparison):
        result = doctest.testmod(module, verbose=False, optionflags=doctest.REPORT_ONLY_FIRST_FAILURE)
        status = "PASS" if result.failed == 0 else "FAIL"
        print(f"{status} doctest {module.__name__}: {result.attempted - result.failed}/{result.attempted}")
        ok = ok and result.failed == 0
    return ok


def main() -> None:
    ok = check_fixtures() and check_playground()
    try:
        for name, passed in cases():
            print(("PASS " if passed else "FAIL ") + name)
            ok = ok and passed
    except NotImplementedError:
        print("FAIL scheduler.py 尚未实现（NotImplementedError）")
        ok = False
    except Exception:
        traceback.print_exc()
        ok = False
    ok = run_doctests() and ok
    print("RESULT:", "PASS" if ok else "FAIL")
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
