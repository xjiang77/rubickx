#!/usr/bin/env python3
"""KV cache 概念练习验证器。

运行（仓库根目录）：make test-kvcache
实现由本人手写（C5）。不得放宽比较、删除用例，或改用 pytest.skip / xfail。
骨架状态下本脚本预期失败，退出码非 0。
数值来自 fixtures.py，对应正文第 1、3、7 节。
"""
import doctest
import os
import sys
import traceback
from collections.abc import Iterator
from typing import Any

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


def _clone(tree: Any) -> Any:
    """嵌套列表的副本，只给验证器用。"""
    if isinstance(tree, list):
        return [_clone(item) for item in tree]
    return tree


def _fill(layers: int, heads: int, positions: int, dim: int, start: float) -> list:
    """造一块 [layer][head][pos][dim]，数值从 start 递增。"""
    value = start
    block = []
    for _ in range(layers):
        layer = []
        for _ in range(heads):
            rows = []
            for _ in range(positions):
                rows.append([value + offset for offset in range(dim)])
                value += dim
            layer.append(rows)
        block.append(layer)
    return block


def cases() -> Iterator[Case]:
    """逐条产出第 1、3、7 节和前缀规则的用例。"""
    from kv_cache import KVCache, kv_cache_bytes, prefix_reuse_hit, qkv_shapes, token_forwards

    prefill = qkv_shapes(fx.TOY_LAYERS, fx.TOY_HEADS, fx.TOY_HEAD_DIM, fx.PREFILL_LEN, phase="prefill")
    yield "第3节 prefill Q", prefill["q"] == fx.PREFILL_SHAPE
    yield "第3节 prefill K", prefill["k"] == fx.PREFILL_SHAPE
    yield "第3节 prefill V", prefill["v"] == fx.PREFILL_SHAPE
    decode = qkv_shapes(fx.TOY_LAYERS, fx.TOY_HEADS, fx.TOY_HEAD_DIM, fx.DECODE_LEN, phase="decode")
    yield "第3节 decode Q", decode["q"] == fx.DECODE_Q_SHAPE
    yield "第3节 decode K", decode["k"] == fx.DECODE_KV_SHAPE
    yield "第3节 decode V", decode["v"] == fx.DECODE_KV_SHAPE
    yield "phase 非法", raises(ValueError, qkv_shapes, 2, 2, 4, 5, phase="both")
    yield "seq_len 拒绝 bool", raises(ValueError, qkv_shapes, 2, 2, 4, True, phase="prefill")

    yield "第1节短例无 cache 4+5+6", token_forwards(fx.SHORT_PROMPT, fx.SHORT_OUTPUT, use_cache=False) == fx.SHORT_NO_CACHE
    yield "第1节短例有 cache 4+3", token_forwards(fx.SHORT_PROMPT, fx.SHORT_OUTPUT, use_cache=True) == fx.SHORT_CACHED
    yield "第1节对照 n=128 m=32 无 cache", token_forwards(fx.LONG_PROMPT, fx.LONG_OUTPUT, use_cache=False) == fx.LONG_NO_CACHE
    yield "第1节对照 n=128 m=32 有 cache", token_forwards(fx.LONG_PROMPT, fx.LONG_OUTPUT, use_cache=True) == fx.LONG_CACHED
    yield "第3节 trace 再生成 1 token 无 cache", token_forwards(fx.TRACE_PROMPT, fx.TRACE_OUTPUT, use_cache=False) == fx.TRACE_NO_CACHE
    yield "第3节 trace 再生成 1 token 有 cache", token_forwards(fx.TRACE_PROMPT, fx.TRACE_OUTPUT, use_cache=True) == fx.TRACE_CACHED
    yield "token_forwards 拒绝 bool", raises(ValueError, token_forwards, True, 1, use_cache=True)

    yield "第7节 Llama3-70B FP16 单 token", kv_cache_bytes(
        fx.LLAMA_LAYERS, fx.LLAMA_KV_HEADS, fx.LLAMA_HEAD_DIM, 1, fx.FP16_BYTES,
    ) == fx.LLAMA_BYTES_PER_TOKEN
    for seq, nbytes in fx.LLAMA_CONTEXT_BYTES.items():
        yield f"第7节表 seq={seq}", kv_cache_bytes(
            fx.LLAMA_LAYERS, fx.LLAMA_KV_HEADS, fx.LLAMA_HEAD_DIM, seq, fx.FP16_BYTES,
        ) == nbytes
    yield "第3节玩具 prefill 字节", kv_cache_bytes(2, 2, 4, 5, 2) == fx.TOY_BYTES_PREFILL
    yield "第3节玩具 decode 字节", kv_cache_bytes(2, 2, 4, 6, 2) == fx.TOY_BYTES_DECODE
    yield "字节数拒绝 0", raises(ValueError, kv_cache_bytes, 80, 8, 128, 1, 0)

    cache = KVCache(fx.TOY_LAYERS, fx.TOY_HEADS, fx.TOY_HEAD_DIM)
    yield "空 cache 长度 0", cache.seq_len == 0
    empty_k, empty_v = cache.get()
    yield "空 cache 仍有 2 层", len(empty_k) == fx.TOY_LAYERS and len(empty_v) == fx.TOY_LAYERS
    yield "空 cache 每层 2 头", len(empty_k[0]) == fx.TOY_HEADS and empty_k[1][1] == []

    prefill_k = _fill(2, 2, 5, 4, 1.0)
    prefill_v = _fill(2, 2, 5, 4, 100.0)
    cache.append(_clone(prefill_k), _clone(prefill_v))
    yield "prefill 追加后长度 5", cache.seq_len == 5
    step_k = _fill(2, 2, 1, 4, 50.0)
    step_v = _fill(2, 2, 1, 4, 70.0)
    cache.append(step_k, step_v)
    yield "decode 追加后长度 6", cache.seq_len == 6
    got_k, got_v = cache.get()
    yield "追加后 K 的第 6 个 token 是新写入的", got_k[0][0][5] == step_k[0][0][0]
    yield "追加后 V 的前 5 个 token 还在", got_v[1][1][:5] == prefill_v[1][1]
    got_k[0][0][5][0] = -1.0
    again_k, _again_v = cache.get()
    yield "get 返回副本", again_k[0][0][5][0] == step_k[0][0][0][0]
    yield "错误形状不改变长度", raises(ValueError, cache.append, [[[[1.0]]]], [[[[1.0]]]]) and cache.seq_len == 6

    same = dict(cached_model=fx.MODEL, request_model=fx.MODEL, cached_adapter=fx.ADAPTER, request_adapter=fx.ADAPTER)
    yield "前缀完全一致且可带后缀", prefix_reuse_hit(fx.CACHED_PREFIX, fx.REQUEST_CONTINUATION, **same) is True
    yield "前缀换时间戳则 miss", prefix_reuse_hit(fx.CACHED_PREFIX, fx.REQUEST_WITH_TIMESTAMP, **same) is False
    yield "model 不同则 miss", prefix_reuse_hit(
        fx.CACHED_PREFIX, fx.CACHED_PREFIX, fx.MODEL, fx.OTHER_MODEL, fx.ADAPTER, fx.ADAPTER,
    ) is False
    yield "adapter 不同则 miss", prefix_reuse_hit(
        fx.CACHED_PREFIX, fx.CACHED_PREFIX, fx.MODEL, fx.MODEL, fx.ADAPTER, fx.OTHER_ADAPTER,
    ) is False
    yield "空前缀不能复用", prefix_reuse_hit([], fx.CACHED_PREFIX, **same) is False
    yield "请求比前缀短则 miss", prefix_reuse_hit(fx.CACHED_PREFIX, fx.CACHED_PREFIX[:2], **same) is False
    yield "adapter 不是 str", raises(
        ValueError, prefix_reuse_hit, fx.CACHED_PREFIX, fx.CACHED_PREFIX, fx.MODEL, fx.MODEL, fx.ADAPTER, None,
    )


def run_doctests() -> bool:
    """运行 kv_cache.py 与 playground 的 doctest。"""
    import kv_cache
    import prefill_decode

    ok = True
    for module in (kv_cache, prefill_decode):
        result = doctest.testmod(module, verbose=False, optionflags=doctest.REPORT_ONLY_FIRST_FAILURE)
        status = "PASS" if result.failed == 0 else "FAIL"
        print(f"{status} doctest {module.__name__}: {result.attempted - result.failed}/{result.attempted}")
        ok = ok and result.failed == 0
    return ok


def main() -> None:
    ok = True
    try:
        for name, passed in cases():
            print(("PASS " if passed else "FAIL ") + name)
            ok = ok and passed
    except NotImplementedError:
        print("FAIL kv_cache.py 尚未实现（NotImplementedError）")
        ok = False
    except Exception:
        traceback.print_exc()
        ok = False
    ok = run_doctests() and ok
    print("RESULT:", "PASS" if ok else "FAIL")
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
