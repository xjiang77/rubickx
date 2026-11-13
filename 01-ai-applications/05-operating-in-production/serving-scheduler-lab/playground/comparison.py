#!/usr/bin/env python3
"""静态批、连续批和背压的对照表。只打印，不调用 scheduler.py。

运行：

    python3 01-ai-applications/05-operating-in-production/serving-scheduler-lab/playground/comparison.py

不访问网络。数字与 fixtures.py、EXERCISE.md 相同，方便在写函数之前先看见步数。
"""


def demo_lines() -> list[str]:
    """固定的对照行。

    Examples:
        >>> demo_lines()[0]
        'budget 8 chunk 4 kv 24 queue 2'
        >>> any("static 5, monolithic 4, chunked 2" in line for line in demo_lines())
        True
        >>> any("overload kv" in line for line in demo_lines())
        True
        >>> any("prefill 12, decode 2" in line for line in demo_lines())
        True
    """
    return [
        "budget 8 chunk 4 kv 24 queue 2",
        "mixed R_short completion: static 5, monolithic 4, chunked 2",
        "mixed R_short ttft: static 4, chunked 1",
        "mixed R_long_prefill completion: monolithic 3, chunked 5",
        "mixed R_long_decode completion: static 11, chunked 10",
        "next batch R_b completion: static 5, chunked 3",
        "backpressure live 20 + need 8: overload kv",
        "backpressure queue 2: overload queue",
        "unbounded queue of 3 needs: admit admit admit",
        "prefix 16 shared 4: prefill 12, decode 2",
    ]


def main() -> None:
    """打印对照表。"""
    print("Serving 调度对照（不是 scheduler.py 的实现）")
    for line in demo_lines():
        print(line)


if __name__ == "__main__":
    main()
