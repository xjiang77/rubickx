# L4 Distributed Semantics

这是一个 deterministic semantics lab：不用网络、真实时钟、数据库或外部依赖，以小型 history、集合与 access-set 模型验证五类判断。

1. `CheckReadYourWrites` / `CheckMonotonicReads` 找出 session history 的反例。
2. `CheckReplicaFreshness` 检查 replica read 是否满足 caller 的 minimum version。
3. `Majority` / `QuorumsIntersect` / `CanCommit` 演示 3/2、5/3 network partition 中的 majority reachability。
4. `FindWriteSkewPairs` 找出两个 disjoint writers 之间的 mutual rw-antidependency。
5. `ApplyNaive` / `ApplyIdempotent` 对比 at-least-once duplicate 对业务 effect 的影响，并让同一 ID 的冲突 payload fail explicitly。

## Proof Boundary

本目录是**语义模型，不是 database、replication 或 consensus implementation**：

- 没有 term、vote persistence、leader election、replicated log、commit index、snapshot 或 state machine。
- `CanCommit` 只检查可达 voter 是否形成 majority，不证明 Raft/Paxos safety 或 liveness。
- session checker 只验证给定 history 的 RYW/monotonic-read property，不证明系统 linearizable、serializable 或 causally consistent。
- `CheckReplicaFreshness` 假设 version 已由外部系统定义；它不证明 version order、replication durability、failover 或 convergence。
- `FindWriteSkewPairs` 只识别两 transaction 的一种潜在 write-skew shape，不解析 SQL predicate，也不是通用 serializability checker。
- idempotent apply 只展示 stable identity 的去重效果，不覆盖 durable inbox、concurrent consumer、external side effect 或 reconciliation。

## Run

在 repository 的 `go/` 目录运行：

```bash
go test -race ./foundations/l4-distributed-semantics
go vet ./foundations/l4-distributed-semantics
```

或验证全部 foundations：

```bash
go test -race ./foundations/...
go vet ./foundations/...
```

## Knowledge Links

- [Distributed Systems canonical notes](https://github.com/xjiang7712/dragon-vault/tree/main/02_Knowledge/02_Engineering/09%20-%20Systems%20%26%20Mechanisms/01%20-%20Distributed%20Systems) — 00-07 的 failure、replication、partitioning、isolation、consensus 与 recovery 主线。
- `02 - Eng - Replication：Leader、Lag 与 Conflict` — minimum-version read 与 replica lag 的 contract。
- `03 - Eng - Partitioning：Routing、Skew 与 Rebalancing` — 区分 data sharding 与本 lab 的 network partition majority model。
- `04 - Eng - Consistency Models：History 与可见性` — session history 和 consistency properties 的边界。
- `05 - Eng - Transaction Isolation：Anomaly、MVCC 与 Serializability` — write-skew shape 与真实数据库并发测试的分界。
- `06 - Eng - Consensus：Quorum、Leader 与 State Machine` — quorum intersection 的作用与 proof boundary。
- `07 - Eng - Distributed Transactions：Commit、Saga 与 Reconciliation` — duplicate delivery、idempotent effect 与 reconciliation。
