# 02-mini-redis —— Go Sprint Capstone 1(W2–W4)

配套 vault:`[[05 - Go Sprint 8-Week Plan]]` 线 C。目标不是复刻 Redis,而是用一个可压测的真实服务打穿:协议解析 → 存储引擎 → 过期与淘汰 → 并发模型 → 性能诊断(G3 证据)。

## Milestones

| M | 周 | 交付 | 验证 |
|---|----|------|------|
| M0 | W2 | RESP2 协议解析器(inline + bulk)| 协议 fuzz-ish 单测:残包/粘包/非法输入 |
| M1 | W2 | TCP server + GET/SET/DEL/PING | redis-cli 直连可用 |
| M2 | W3 | dict 引擎 + EXPIRE/TTL(惰性+定期)+ LRU/LFU 淘汰(maxmemory)| 过期语义单测 + 淘汰命中率对比 |
| M3 | W4 | pipeline + 并发模型定型(每连接 goroutine + 单写者 or 分片锁,写 ADR 说明取舍)| `go test -race` 全绿 |
| M4 | W4 | 压测报告:redis-benchmark 对比官方 Redis,pprof 定位 top3 瓶颈并优化一轮 | `docs/benchmark.md`(qps/延迟/火焰图 + 不证明什么)|

## 约束

- 只依赖标准库。
- 每个 milestone 先在 `docs/` 写半页设计(问题/取舍/失败语义),再动手。
- 实验结果必须写明 workload 与 fixture,不冒充 production-ready。

## 结构(按需生长,不预建空目录)

```
02-mini-redis/
  cmd/miniredis/     入口
  internal/resp/     协议
  internal/store/    引擎(dict/ttl/evict)
  internal/server/   连接与并发模型
  docs/              设计半页 + 压测报告
```

W5–W7 的 Capstone 2(Raft, MIT 6.5840)另建 `03-raft/`,由 coach session 在 W5 生成骨架。
