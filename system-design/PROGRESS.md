# Progress — system-design 三语言板

| # | 系统 | 组件 | Python | Go | Java | NOTES | 验证 |
|---|------|------|--------|----|----|-------|------|
| 01 | Rate Limiter | Token Bucket | ✅ | ✅ | ✅ | ✅ | py ✅ / go·java 待本机 |
| 01 | Rate Limiter | Sliding Window Counter | ⬜ | ⬜ | ⬜ | ⬜ | — |
| 02 | Payment | 幂等键 / 状态机 | ⬜ | ⬜ | ⬜ | ⬜ | — |
| 03 | Message Queue | 进程内 broker / offset | ⬜ | ⬜ | ⬜ | ⬜ | — |
| 04 | Twitter Feed | fan-out / 时间线合并 | ⬜ | ⬜ | ⬜ | ⬜ | — |

> 顺序对齐 vault Dashboard 的 L6 高频优先。每个组件三语言并排 + NOTES 对比。
> 沙箱只有 Python（pytest 可跑）；Go/Java 在本机 `make test-go` / `make test-java`。
