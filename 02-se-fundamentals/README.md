# 02 — Software engineering fundamentals

对应 Skills Map 第二项技能：理解软件如何工作，才能在成本、扩展性、可靠性、速度与安全之间做出取舍——选技术栈、设计架构与数据存储、写测试。

本目录的 track 都遵循同一方法：同一个问题在多种语言里重写，在差异里看清每种语言的设计取向；完成度以各 track 自己的 `PROGRESS.md` 为准。

| Track | 内容 | 验证（在仓库根目录执行） |
| --- | --- | --- |
| `algo/` | NeetCode 题目，Python / Go / JavaScript / Java / Scala 五语言并排，每题 `NOTES.md` 讲写法差异 | `make -C 02-se-fundamentals/algo test` |
| `patterns/` | 42 项工程模式（GoF 23、Reliability 6、Data & Messaging 7、Concurrency 6），四语言实现 + 共享 fixture + contract tests | `make -C 02-se-fundamentals/patterns verify` |
| `system-design/` | 系统设计核心组件四语言实现（Rate Limiter 五种算法）+ Go HTTP / Redis / UI lab | `make -C 02-se-fundamentals/system-design test` |
| `network-security/` | 10 个 loopback-only labs：网络路径、浏览器与 API 信任边界、SSO / federation、持续访问与证据闭环 | `make -C 02-se-fundamentals/network-security verify` |
| `systems-foundations/` | 执行模型（goroutine、race）与分布式语义（session、quorum、isolation、duplicate effect）的 deterministic 实验 | `cd 02-se-fundamentals/systems-foundations && go test -race ./...` |
