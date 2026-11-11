"""1.5.1 概念练习的固定数值。只有字面量，没有实现。

来源是 vault 笔记
`02_Knowledge/01-ai-applications/05-operating-in-production/01-inference-and-serving/02 - AI - KV-Cache Deep Dive.md`
（知识状态仍是待核验）。本环境的凭证读不到该私有仓库，下面的闭式与字节按笔记里能对上的恒等式写死，并在 EXERCISE.md 写明。

- 第 3 节：2 层、2 头、head_dim=4 的 5-token trace。
- 第 1 节：无 cache 与有 cache 的 token-forward 闭式。短例供手算；n=128、m=32 是同一闭式的对照。
- 第 7 节：`2 × layers × kv_heads × head_dim × seq × bytes`。Llama 3 70B FP16 单 token 为 327680 字节（320 KiB，笔记里的「约 320 KB」）。1K/8K/32K/128K 按 1024 进位，才能同时得到 320 MiB、2.5 GiB、10 GiB、40 GiB。
"""

NOTE = "02 - AI - KV-Cache Deep Dive.md"

# 第 3 节简化设定。形状轴序：(layers, heads, seq, head_dim)。没有 batch 维。
TOY_LAYERS = 2
TOY_HEADS = 2
TOY_HEAD_DIM = 4
PREFILL_LEN = 5
DECODE_LEN = 6

PREFILL_SHAPE = (TOY_LAYERS, TOY_HEADS, PREFILL_LEN, TOY_HEAD_DIM)
DECODE_Q_SHAPE = (TOY_LAYERS, TOY_HEADS, 1, TOY_HEAD_DIM)
DECODE_KV_SHAPE = (TOY_LAYERS, TOY_HEADS, DECODE_LEN, TOY_HEAD_DIM)

# 第 1 节。无 cache = m*n + m*(m-1)/2 ；有 cache = n+m。
SHORT_PROMPT = 4
SHORT_OUTPUT = 3
SHORT_NO_CACHE = 15  # 4+5+6
SHORT_CACHED = 7  # 4+3

LONG_PROMPT = 128
LONG_OUTPUT = 32
LONG_NO_CACHE = 4592  # 32*128 + 32*31/2
LONG_CACHED = 160  # 128+32

TRACE_PROMPT = 5
TRACE_OUTPUT = 1
TRACE_NO_CACHE = 5  # 只重算长度为 5 的那一步
TRACE_CACHED = 6  # 5+1

# 第 7 节。Llama 3 70B，GQA，FP16。
LLAMA_LAYERS = 80
LLAMA_KV_HEADS = 8
LLAMA_HEAD_DIM = 128
FP16_BYTES = 2
LLAMA_BYTES_PER_TOKEN = 327_680  # 2*80*8*128*2 = 320 KiB

# 表：1K/8K/32K/128K。键是 token 数，值是字节。
LLAMA_CONTEXT_BYTES = {
    1024: 335_544_320,
    8192: 2_684_354_560,
    32_768: 10_737_418_240,
    131_072: 42_949_672_960,
}

# 同一公式在第 3 节玩具上的字节（FP16）。
TOY_BYTES_PREFILL = 320  # 2*2*2*4*5*2
TOY_BYTES_DECODE = 384  # 2*2*2*4*6*2

# 第 7 节之外、笔记里的跨请求复用三前提：前缀 token 逐个相同，且 model、adapter 都相同。
CACHED_PREFIX = [11, 22, 33]
REQUEST_CONTINUATION = [11, 22, 33, 44]
REQUEST_WITH_TIMESTAMP = [11, 22, 99, 44]
MODEL = "llama3-70b"
OTHER_MODEL = "llama3-8b"
ADAPTER = "none"
OTHER_ADAPTER = "lora-a"
