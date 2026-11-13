"""1.5.2 概念练习的固定数值。只有字面量，没有实现。

来源是 vault 笔记
`02_Knowledge/01-ai-applications/05-operating-in-production/01-inference-and-serving/03 - AI - LLM Serving Runtime：Scheduler、KV Cache 与 Backpressure.md`
（知识状态仍是待核验）。本练习只核对能手算的调度规则，不把正文标成已核验。

步数是模拟步，不是墙钟。每一步的 (step, prefill, decode, tokens) 都是写死的整数。
"""

NOTE = "03 - AI - LLM Serving Runtime：Scheduler、KV Cache 与 Backpressure.md"

TOKEN_BUDGET_PER_STEP = 8
KV_CAPACITY = 24
MAX_QUEUE = 2
CHUNK_SIZE = 4

# 混合负载。默认都在 step 0 到达。顺序就是调度时的列表顺序。
R_SHORT_PROMPT = 4
R_SHORT_OUTPUT = 2
R_LONG_PREFILL_PROMPT = 16
R_LONG_PREFILL_OUTPUT = 2
R_LONG_DECODE_PROMPT = 4
R_LONG_DECODE_OUTPUT = 8

MIXED = [
    {"id": "R_short", "prompt": 4, "output": 2, "arrival": 0},
    {"id": "R_long_prefill", "prompt": 16, "output": 2, "arrival": 0},
    {"id": "R_long_decode", "prompt": 4, "output": 8, "arrival": 0},
]

# 三条请求的全部工作：prefill 4+16+4，decode 2+2+8。三种策略都要做完这些 token。
MIXED_TOTAL_TOKENS = 36

# 静态批：当前批的 prefill 全部做完才 decode；同一步只给一条请求做 prefill。
# 短请求完成步 5，TTFT 4。
STATIC_COMPLETION = {"R_short": 5, "R_long_prefill": 5, "R_long_decode": 11}
STATIC_TTFT = {"R_short": 4, "R_long_prefill": 4, "R_long_decode": 4}
STATIC_STEPS = [
    (0, 4, 0, 4),
    (1, 8, 0, 8),
    (2, 8, 0, 8),
    (3, 4, 0, 4),
    (4, 0, 3, 3),
    (5, 0, 3, 3),
    (6, 0, 1, 1),
    (7, 0, 1, 1),
    (8, 0, 1, 1),
    (9, 0, 1, 1),
    (10, 0, 1, 1),
    (11, 0, 1, 1),
]

# chunked：decode 优先，然后每条 prefill 每步最多一个 4-token chunk。
# 短请求完成步 2，早于静态批的 5，也早于 monolithic 的 4。
CHUNKED_COMPLETION = {"R_short": 2, "R_long_prefill": 5, "R_long_decode": 10}
CHUNKED_TTFT = {"R_short": 1, "R_long_prefill": 4, "R_long_decode": 3}
CHUNKED_STEPS = [
    (0, 8, 0, 8),
    (1, 7, 1, 8),
    (2, 5, 1, 6),
    (3, 4, 1, 5),
    (4, 0, 2, 2),
    (5, 0, 2, 2),
    (6, 0, 1, 1),
    (7, 0, 1, 1),
    (8, 0, 1, 1),
    (9, 0, 1, 1),
    (10, 0, 1, 1),
]

# monolithic：最长的 prefill 可以一次占满剩余 budget。长 prefill 完成步 3，短请求被挤到 4。
MONOLITHIC_COMPLETION = {"R_short": 4, "R_long_prefill": 3, "R_long_decode": 11}
MONOLITHIC_TTFT = {"R_short": 3, "R_long_prefill": 2, "R_long_decode": 4}
MONOLITHIC_STEPS = [
    (0, 8, 0, 8),
    (1, 8, 0, 8),
    (2, 4, 1, 5),
    (3, 4, 2, 6),
    (4, 0, 2, 2),
    (5, 0, 1, 1),
    (6, 0, 1, 1),
    (7, 0, 1, 1),
    (8, 0, 1, 1),
    (9, 0, 1, 1),
    (10, 0, 1, 1),
    (11, 0, 1, 1),
]

# 下一批：R_b 在 step 1 到达，此时 R_a 的静态批还没跑完。
LATE = [
    {"id": "R_a", "prompt": 4, "output": 2, "arrival": 0},
    {"id": "R_b", "prompt": 4, "output": 2, "arrival": 1},
]
LATE_TOTAL_TOKENS = 12  # (4+2) + (4+2)
LATE_STATIC_COMPLETION = {"R_a": 2, "R_b": 5}
LATE_STATIC_TTFT = {"R_a": 1, "R_b": 4}
LATE_STATIC_STEPS = [
    (0, 4, 0, 4),
    (1, 0, 1, 1),
    (2, 0, 1, 1),
    (3, 4, 0, 4),
    (4, 0, 1, 1),
    (5, 0, 1, 1),
]
LATE_CHUNKED_COMPLETION = {"R_a": 2, "R_b": 3}
LATE_CHUNKED_TTFT = {"R_a": 1, "R_b": 2}
LATE_CHUNKED_STEPS = [
    (0, 4, 0, 4),
    (1, 4, 1, 5),
    (2, 0, 2, 2),
    (3, 0, 1, 1),
]

# 前缀复用：第二个请求的 prefill = prompt - 4，decode 仍是 output。
SHARED_PREFIX = 4
SECOND_PROMPT = 16
SECOND_OUTPUT = 2
SECOND_PREFILL_WORK = 12  # 16 - 4
SECOND_DECODE_WORK = 2

# 准入。live + need > capacity 才拒绝；相等可以准入。队列长度 >= MAX_QUEUE 才拒绝。
FIT_LIVE_KV = 16
FIT_KV_NEED = 8  # 16 + 8 = 24
OVER_LIVE_KV = 20
OVER_KV_NEED = 8  # 20 + 8 = 28
SMALL_KV_NEED = 4
FULL_QUEUE = 2
ROOM_QUEUE = 1

ADMIT_OK = {"outcome": "admit", "reason": "ok"}
OVERLOAD_KV = {"outcome": "overload", "reason": "kv"}
OVERLOAD_QUEUE = {"outcome": "overload", "reason": "queue"}

# 队列闸门：live KV 为 0，三次各要 4。有上限时第三次拒绝；无限队列三次都接收。
QUEUE_NEEDS = [4, 4, 4]
QUEUE_SCREEN = [
    {"outcome": "admit", "reason": "ok"},
    {"outcome": "admit", "reason": "ok"},
    {"outcome": "overload", "reason": "queue"},
]
QUEUE_SCREEN_UNBOUNDED = [
    {"outcome": "admit", "reason": "ok"},
    {"outcome": "admit", "reason": "ok"},
    {"outcome": "admit", "reason": "ok"},
]
UNBOUNDED_QUEUE_LEN = 3

# KV 闸门：已有 20，先来一个 8（拒绝，不占名额），再来 4（24，接收），后面的 4 都放不下。
KV_NEEDS = [8, 4, 4, 4]
KV_SCREEN = [
    {"outcome": "overload", "reason": "kv"},
    {"outcome": "admit", "reason": "ok"},
    {"outcome": "overload", "reason": "kv"},
    {"outcome": "overload", "reason": "kv"},
]
