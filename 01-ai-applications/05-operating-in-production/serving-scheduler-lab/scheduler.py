"""连续批处理调度模拟器 —— 本人手写 (C5)。下面只有骨架，实现留空。

只用整数步和 token 计数。不实现注意力、vLLM、PagedAttention、GPU 或墙钟延迟。
规则见 EXERCISE.md；对照数字见 fixtures.py。

每个函数的 Examples 是可执行的 doctest，实现后运行：

    python3 -m doctest scheduler.py -v

骨架状态下 doctest 报 NotImplementedError 是预期结果。
"""

from typing import Literal, TypedDict


class Request(TypedDict):
    """一条已到达的生成请求。"""

    id: str
    prompt: int
    output: int
    arrival: int


class Admission(TypedDict):
    """一次准入决定。outcome 只有 admit / overload，reason 说明是哪一条闸门。"""

    outcome: Literal["admit", "overload"]
    reason: Literal["ok", "kv", "queue"]


class Step(TypedDict):
    """一步里实际安排的 token。tokens 必须等于 prefill + decode。"""

    step: int
    prefill: int
    decode: int
    tokens: int


class Schedule(TypedDict):
    """一次模拟的完成步、首 token 步，以及每步的 token 账。"""

    completions: dict[str, int]
    ttft: dict[str, int]
    steps: list[Step]


def prefill_work(prompt_len: int, shared_prefix: int) -> int:
    """前缀复用之后还要做的 prefill token 数。

    Args:
        prompt_len: 这条请求的 prompt 长度。
        shared_prefix: 已经能复用的前缀长度。只减少 prefill，不减少 decode。

    Returns:
        prompt_len - shared_prefix。两端相等时为 0。

    Raises:
        ValueError: 参数不是 int（bool 也不行），或 shared_prefix 不在 0 到 prompt_len 之间。

    Examples:
        >>> prefill_work(16, 4)
        12
        >>> prefill_work(4, 0)
        4
        >>> prefill_work(4, 4)
        0
        >>> prefill_work(4, 5)  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: shared_prefix
        >>> prefill_work(True, 0)  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: prompt_len
    """
    raise NotImplementedError


def decode_work(output_len: int) -> int:
    """还要生成的 decode token 数。前缀复用不会减少这个数。

    Args:
        output_len: 计划生成的 token 数。

    Returns:
        output_len 本身。

    Raises:
        ValueError: 参数不是 int（bool 也不行），或小于 0。

    Examples:
        >>> decode_work(2)
        2
        >>> decode_work(0)
        0
        >>> decode_work(-1)  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: output_len
    """
    raise NotImplementedError


def classify_admission(
    live_kv: int,
    queue_len: int,
    kv_need: int,
    *,
    capacity: int,
    max_queue: int | None,
) -> Admission:
    """决定一个新请求是准入还是 overload。

    先看 KV：live_kv + kv_need > capacity 则拒绝，reason 是 kv。
    相等时可以准入。再看队列：max_queue 不是 None 且 queue_len >= max_queue
    则拒绝，reason 是 queue。max_queue 为 None 表示无限队列，不检查队列；
    KV 闸门仍然有效。两条都触发时 reason 取 kv。

    Args:
        live_kv: 已经在跑的序列当前占用的 KV token。
        queue_len: 已经在等待队列里的请求数。
        kv_need: 这个新请求要预留的 KV token。
        capacity: KV 上限。
        max_queue: 队列上限；None 表示不设上限。

    Returns:
        outcome 为 admit 或 overload，reason 为 ok、kv 或 queue。

    Raises:
        ValueError: 整数参数不是 int（bool 也不行）或小于 0；max_queue 既不是 None 也不是合法整数。

    Examples:
        >>> classify_admission(16, 0, 8, capacity=24, max_queue=2) == {"outcome": "admit", "reason": "ok"}
        True
        >>> classify_admission(20, 0, 8, capacity=24, max_queue=2) == {"outcome": "overload", "reason": "kv"}
        True
        >>> classify_admission(0, 2, 4, capacity=24, max_queue=2) == {"outcome": "overload", "reason": "queue"}
        True
        >>> classify_admission(0, 2, 4, capacity=24, max_queue=None) == {"outcome": "admit", "reason": "ok"}
        True
        >>> classify_admission(20, 0, 8, capacity=24, max_queue=None) == {"outcome": "overload", "reason": "kv"}
        True
        >>> classify_admission(-1, 0, 1, capacity=24, max_queue=2)  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: live_kv
    """
    raise NotImplementedError


def screen_arrivals(
    live_kv: int,
    kv_needs: list[int],
    *,
    capacity: int,
    max_queue: int | None,
) -> list[Admission]:
    """按到达顺序给一批新请求做准入。

    已预留 KV 从 live_kv 起步，队列长度从 0 起步。每个请求都用
    classify_admission 看当前预留和当前队列。admit 才把 kv_need 加进预留、
    并把队列长度加 1。overload 两个数都不变，后面的请求还能再试。

    Args:
        live_kv: 屏幕开始时已经在跑的 KV。
        kv_needs: 按到达顺序排列的预留需求。
        capacity: KV 上限。
        max_queue: 队列上限；None 表示无限队列。

    Returns:
        与 kv_needs 等长的决定列表。

    Raises:
        ValueError: kv_needs 不是 list，或其中的数、live_kv、capacity、max_queue 不合法。

    Examples:
        >>> [item["outcome"] for item in screen_arrivals(0, [4, 4, 4], capacity=24, max_queue=2)]
        ['admit', 'admit', 'overload']
        >>> screen_arrivals(0, [4, 4, 4], capacity=24, max_queue=2)[2]["reason"]
        'queue'
        >>> [item["outcome"] for item in screen_arrivals(0, [4, 4, 4], capacity=24, max_queue=None)]
        ['admit', 'admit', 'admit']
        >>> screen_arrivals(0, [], capacity=24, max_queue=2)
        []
    """
    raise NotImplementedError


def simulate(requests: list[Request], policy: str) -> Schedule:
    """按一种策略把请求跑完，返回完成步、TTFT 和每步的 token 账。

    步号从 0 起。一步里的 tokens = prefill + decode，不超过
    fixtures.TOKEN_BUDGET_PER_STEP。一条请求一步最多 decode 1 个 token。
    本步刚做完 prefill 的请求，下一步才能 decode。完成步是 decode 减到 0
    的那一步；TTFT 是第一次 decode 的那一步。不修改传入的请求。

    policy 三选一，细则在 EXERCISE.md：

    - chunked：先给已经 prefill 完的请求 decode，再按列表顺序给每条还在
      prefill 的请求最多一个 fixtures.CHUNK_SIZE 的 chunk。
    - monolithic：decode 阶段与 chunked 相同。prefill 只给剩余 prefill
      最长的一条，它可以拿走本步剩下的全部 budget。
    - static：当前批跑完才接下一批。批内还有 prefill 时，只给列表里第一条
      没做完的 prefill，同一步不 decode。全部 prefill 完了才按列表顺序 decode。

    Args:
        requests: 请求列表。prompt >= 0，output >= 1，arrival >= 0，id 非空且不重复。
        policy: static、chunked 或 monolithic。

    Returns:
        completions、ttft 和 steps。空列表给出三个空结果。

    Raises:
        ValueError: policy 不认识，或请求字段不合法。

    Examples:
        >>> pair = [
        ...     {"id": "R_short", "prompt": 4, "output": 2, "arrival": 0},
        ...     {"id": "R_long_prefill", "prompt": 16, "output": 2, "arrival": 0},
        ... ]
        >>> simulate(pair, "chunked")["completions"]["R_short"]
        2
        >>> simulate(pair, "static")["completions"]["R_short"]
        4
        >>> simulate(pair, "monolithic")["completions"]["R_long_prefill"]
        3
        >>> simulate([{"id": "R_zero_prompt", "prompt": 0, "output": 1, "arrival": 0}], "chunked")["completions"]
        {'R_zero_prompt': 0}
        >>> simulate([], "chunked") == {"completions": {}, "ttft": {}, "steps": []}
        True
        >>> simulate(pair, "fair")  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: policy
    """
    raise NotImplementedError

