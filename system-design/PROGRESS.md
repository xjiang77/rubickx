# Progress — system-design 四语言板

| # | 系统 | 组件 | Python | Go | Java | JavaScript | Lab | 验证 |
|---|------|------|--------|----|------|------------|-----|------|
| 01 | Rate Limiter | Fixed Window | ✅ | ✅ | ✅ | ✅ | semantic trace | four-language parity |
| 01 | Rate Limiter | Sliding Window Log | ✅ | ✅ | ✅ | ✅ | semantic trace | four-language parity |
| 01 | Rate Limiter | Sliding Window Counter | ✅ | ✅ | ✅ | ✅ | semantic trace | four-language parity |
| 01 | Rate Limiter | Token Bucket | ✅ | ✅ | ✅ | ✅ | semantic trace | four-language parity |
| 01 | Rate Limiter | Leaky Bucket | ✅ | ✅ | ✅ | ✅ | semantic trace | four-language parity |
| 01 | Rate Limiter | HTTP + Memory/Redis + DAP | — | ✅ | — | — | Web E2E | browser + real Redis + Delve |
| 02 | Payment | 幂等键 / 状态机 | ⬜ | ⬜ | ⬜ | ⬜ | — | — |
| 03 | Message Queue | 进程内 broker / offset | ⬜ | ⬜ | ⬜ | ⬜ | — | — |
| 04 | Twitter Feed | fan-out / 时间线合并 | ⬜ | ⬜ | ⬜ | ⬜ | — | — |

> 顺序对齐 vault Dashboard 的 L6 高频优先。核心算法四语言并排；完整系统链路保持 Go-only。表中 ✅ 只有在对应自动化测试与真实 surface verifier 通过后才成立。
