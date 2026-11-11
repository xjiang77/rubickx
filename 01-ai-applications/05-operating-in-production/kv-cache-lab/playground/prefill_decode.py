#!/usr/bin/env python3
"""从 prefill 走到 decode 的对照表。只打印，不实现 kv_cache.py。

运行：python3 prefill_decode.py
不访问网络。数字与 EXERCISE.md、fixtures.py 相同，方便在写函数之前先看见形状和次数。
"""


def demo_lines() -> list[str]:
    """固定的对照行。

    Examples:
        >>> demo_lines()[0]
        'prefill 5: Q=K=V=(2, 2, 5, 4), cache length 5'
        >>> demo_lines()[2]
        'token-forward n=4 m=3: no cache 4+5+6=15, with cache 4+3=7'
        >>> any("327680" in line for line in demo_lines())
        True
    """
    return [
        "prefill 5: Q=K=V=(2, 2, 5, 4), cache length 5",
        "decode 6: Q=(2, 2, 1, 4), K=V=(2, 2, 6, 4), cache length 6",
        "token-forward n=4 m=3: no cache 4+5+6=15, with cache 4+3=7",
        "token-forward n=128 m=32: no cache 4592, with cache 160",
        "llama3-70b fp16 per token: 327680 bytes (320 KiB)",
        "llama3-70b fp16 1K/8K/32K/128K: 320 MiB, 2.5 GiB, 10 GiB, 40 GiB",
    ]


def main() -> None:
    """打印对照表。"""
    print("KV cache 概念对照（不是 kv_cache.py 的实现）")
    for line in demo_lines():
        print(line)


if __name__ == "__main__":
    main()
