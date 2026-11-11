"""KV cache 概念练习 —— 本人手写 (C5)。下面只有骨架，实现留空。

只用 Python 标准库。不得 import numpy、torch 或其他数值库。
规格见 EXERCISE.md。数值见 fixtures.py，来自知识库正文
`02 - AI - KV-Cache Deep Dive.md` 第 1、3、7 节。

这不是 vLLM / PagedAttention，也不是 nanochat systems/06-engine 的推理引擎。
只核对形状、token-forward 次数、内存字节、cache 的追加，以及前缀能否复用。

每个函数的 Examples 是可执行的 doctest，实现后运行：

    python3 -m doctest kv_cache.py -v

骨架状态下 doctest 报 NotImplementedError 是预期结果。
"""

from typing import TypeAlias

Shape: TypeAlias = tuple[int, int, int, int]
"""(num_layers, num_heads, seq_len, head_dim)。没有 batch 维。"""

KVTensor: TypeAlias = list[list[list[list[float]]]]
"""K 或 V。[layer][head][position] 是长度为 head_dim 的向量。int 也可传入。"""


def qkv_shapes(
    num_layers: int,
    num_heads: int,
    head_dim: int,
    seq_len: int,
    *,
    phase: str,
) -> dict[str, Shape]:
    """当前这次 forward 里，注意力用到的 Q、K、V 形状。

    第 3 节把轴序固定为 (layers, heads, seq, head_dim)。
    prefill 一次算出整段 prompt 的 Q、K、V，三者 seq 都是 seq_len。
    decode 只为新 token 算 Q；K、V 是已经追加过这个新 token 的整段 cache，
    seq 仍是 seq_len，Q 的 seq 是 1。

    Args:
        num_layers: 层数，正整数。
        num_heads: 头数，正整数。第 3 节的玩具里 Q 头与 KV 头相同。
        head_dim: 每个头的维度，正整数。
        seq_len: prefill 时是 prompt 长度；decode 时是追加新 token 之后的 cache 长度。
        phase: "prefill" 或 "decode"。

    Returns:
        字典，键为 "q"、"k"、"v"，值是形状元组。

    Raises:
        ValueError: 尺寸不是正整数，或 phase 不是 prefill / decode。bool 不算整数。

    Examples:
        第 3 节：2 层、2 头、head_dim=4。5-token prefill 之后，以及 decode 到第 6 个 token 时：

        >>> qkv_shapes(2, 2, 4, 5, phase="prefill")
        {'q': (2, 2, 5, 4), 'k': (2, 2, 5, 4), 'v': (2, 2, 5, 4)}
        >>> qkv_shapes(2, 2, 4, 6, phase="decode")
        {'q': (2, 2, 1, 4), 'k': (2, 2, 6, 4), 'v': (2, 2, 6, 4)}
        >>> qkv_shapes(2, 2, 4, 0, phase="prefill")  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: seq_len 必须是正整数
    """
    raise NotImplementedError


def token_forwards(prompt_len: int, output_len: int, *, use_cache: bool) -> int:
    """生成 output_len 个新 token 时，实际送进模型的 token 次数。

    第 1 节。prompt 长度 n，新 token 数 m。

    无 cache：发出第 k 个新 token 的那一步（k 从 0 计到 m-1）重算长度为 n+k 的整段。
    次数是 m*n + m*(m-1)/2。

    有 cache：prefill 送进 n 个 token，之后 m 个新 token 各送 1 次。次数是 n+m。

    Args:
        prompt_len: prompt 的 token 数，正整数。
        output_len: 要新生成的 token 数，正整数。
        use_cache: True 用 cache 的次数，False 用无 cache 的次数。

    Returns:
        token-forward 次数，int。

    Raises:
        ValueError: prompt_len 或 output_len 不是正整数。bool 不算整数。

    Examples:
        短例供手算。n=4、m=3 时，无 cache 是 4+5+6。

        >>> token_forwards(4, 3, use_cache=False)
        15
        >>> token_forwards(4, 3, use_cache=True)
        7
        >>> token_forwards(128, 32, use_cache=False)
        4592
        >>> token_forwards(128, 32, use_cache=True)
        160
        >>> token_forwards(5, 1, use_cache=True)
        6
        >>> token_forwards(True, 3, use_cache=True)  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: prompt_len 必须是正整数
    """
    raise NotImplementedError


def kv_cache_bytes(
    num_layers: int,
    num_kv_heads: int,
    head_dim: int,
    seq_len: int,
    bytes_per_element: int,
) -> int:
    """一份 KV cache 的字节数。

    第 7 节：2 × layers × kv_heads × head_dim × seq_len × bytes_per_element。
    前面的 2 是 K 与 V 各一份。不算 batch，不算页表和分配器的额外开销。

    Args:
        num_layers: 层数，正整数。
        num_kv_heads: KV 头数，正整数。GQA 时它小于 query 头数。
        head_dim: 每个头的维度，正整数。
        seq_len: 序列长度，正整数。
        bytes_per_element: 每个元素的字节数，正整数。FP16 是 2。

    Returns:
        字节数，int。

    Raises:
        ValueError: 任一参数不是正整数。bool 不算整数。

    Examples:
        Llama 3 70B、FP16、单 token：2×80×8×128×2 = 327680，即 320 KiB。

        >>> kv_cache_bytes(80, 8, 128, 1, 2)
        327680
        >>> kv_cache_bytes(80, 8, 128, 1024, 2)
        335544320
        >>> kv_cache_bytes(2, 2, 4, 5, 2)
        320
        >>> kv_cache_bytes(80, 8, 128, 1, 0)  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: bytes_per_element 必须是正整数
    """
    raise NotImplementedError


class KVCache:
    """按层、按头存放 K 和 V，并沿 position 轴追加。

    布局与 qkv_shapes 相同：[layer][head][position][head_dim]。
    不做注意力，不分配 GPU 页。空 cache 的 seq 长度是 0。

    Args:
        num_layers: 层数，正整数。
        num_heads: 每层的 KV 头数，正整数。
        head_dim: 每个头的维度，正整数。

    Raises:
        ValueError: 尺寸不是正整数。
    """

    def __init__(self, num_layers: int, num_heads: int, head_dim: int) -> None:
        raise NotImplementedError

    @property
    def seq_len(self) -> int:
        """当前 cache 里的 token 数。空 cache 为 0。

        Examples:
            >>> KVCache(1, 1, 2).seq_len
            0
        """
        raise NotImplementedError

    def append(self, k: KVTensor, v: KVTensor) -> None:
        """把一段新的 K、V 接到 cache 末尾。

        k、v 的层数、头数、head_dim 必须与 cache 一致，且两者的 position 数相同。
        可以一次追加多个 token（prefill），也可以追加 1 个（decode）。
        写入 cache 的是副本：之后修改传入的列表，cache 不变。
        形状不对时抛出 ValueError，并且 cache 保持调用前的内容。

        Args:
            k: 新的 K，布局 [layer][head][position][head_dim]。
            v: 新的 V，形状与 k 相同。

        Raises:
            ValueError: 层、头、head_dim 或 position 数对不上，或向量长度不对。

        Examples:
            >>> cache = KVCache(1, 1, 2)
            >>> cache.append([[[[1.0, 0.0], [0.0, 1.0]]]], [[[[0.5, 0.5], [1.0, 0.0]]]])
            >>> cache.seq_len
            2
            >>> cache.get()[0]
            [[[[1.0, 0.0], [0.0, 1.0]]]]
            >>> block = [[[[9.0, 8.0]]]]
            >>> cache.append(block, [[[[1.0, 1.0]]]])
            >>> block[0][0][0][0] = 0.0
            >>> cache.get()[0][0][0][2][0] == 9.0
            True
            >>> cache.append([[[[1.0]]]], [[[[1.0]]]])  # doctest: +IGNORE_EXCEPTION_DETAIL
            Traceback (most recent call last):
            ValueError: head_dim 必须是 2
        """
        raise NotImplementedError

    def get(self) -> tuple[KVTensor, KVTensor]:
        """返回 (K, V) 的副本。

        修改返回值不会改 cache。空 cache 仍有全部层和头，只是 position 列表为空。

        Returns:
            (k, v)。每个都是 [layer][head][position][head_dim]。

        Examples:
            >>> k, v = KVCache(1, 2, 1).get()
            >>> k
            [[[], []]]
            >>> v[0][1]
            []
        """
        raise NotImplementedError


def prefix_reuse_hit(
    cached_prefix: list[int],
    request_tokens: list[int],
    cached_model: str,
    request_model: str,
    cached_adapter: str,
    request_adapter: str,
) -> bool:
    """这次请求能否复用已经存下的前缀。纯规则，不是 GPU 页表。

    三前提缺一则 miss：

    1. cached_prefix 非空，且 request_tokens 以它开头，token id 逐个相等。
    2. 前缀里的 token 有任何不同（例如插进或换掉一个时间戳）则不相等。
    3. model 字符串相同，并且 adapter 字符串相同。

    请求比前缀更长是允许的：多出来的后缀不参与比较。
    请求比前缀短，或前缀为空，都不能复用。

    Args:
        cached_prefix: 已缓存的前缀 token id。
        request_tokens: 这次请求的 token id。
        cached_model: 缓存所属的模型名。
        request_model: 这次请求的模型名。
        cached_adapter: 缓存所属的 adapter 名。没有 adapter 时用同一个占位字符串。
        request_adapter: 这次请求的 adapter 名。

    Returns:
        三前提都满足时 True，否则 False。

    Raises:
        ValueError: 前缀或请求不是 list，或 model / adapter 不是 str。

    Examples:
        >>> prefix_reuse_hit([11, 22, 33], [11, 22, 33, 44], "llama3-70b", "llama3-70b", "none", "none")
        True
        >>> prefix_reuse_hit([11, 22, 33], [11, 22, 99, 44], "llama3-70b", "llama3-70b", "none", "none")
        False
        >>> prefix_reuse_hit([11, 22, 33], [11, 22, 33], "llama3-70b", "llama3-8b", "none", "none")
        False
        >>> prefix_reuse_hit([11, 22, 33], [11, 22, 33], "llama3-70b", "llama3-70b", "none", "lora-a")
        False
        >>> prefix_reuse_hit([], [11, 22, 33], "llama3-70b", "llama3-70b", "none", "none")
        False
        >>> prefix_reuse_hit([11], [11], "llama3-70b", "llama3-70b", "none", None)  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: request_adapter 必须是 str
    """
    raise NotImplementedError
