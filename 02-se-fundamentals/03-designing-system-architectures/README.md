# Designing system architectures

设计系统架构。专注系统架构设计与权衡：在复杂性、性能、成本与可靠性之间做出合理取舍。

## 来源

[Andrew Ng 原文](https://x.com/AndrewYNg/status/2093388974194872781)

## 当前状态

**已有局部实践**。已有组件、模式与机制实验；局部测试不代表完整生产架构验证。

## 主归属实践

- [system-design](system-design) — 系统组件与 HTTP / Redis / UI lab。
- [patterns](patterns) — 42 项四语言工程模式与共享 contract tests。
- [systems-foundations](systems-foundations) — Go 执行模型与分布式语义实验。

## 跨能力链接

- [Managing data](../02-managing-data/README.md)
- [Making systems secure and reliable](../04-making-systems-secure-and-reliable/README.md)

## 验证入口

从仓库根目录执行（Nanochat 命令在具体系统目录执行）：

- `make -C 02-se-fundamentals/03-designing-system-architectures/system-design test`
- `make -C 02-se-fundamentals/03-designing-system-architectures/patterns verify`
- `cd 02-se-fundamentals/03-designing-system-architectures/systems-foundations && go test -race ./... && go vet ./...`

## 下一步

结合真实负载记录架构取舍、故障边界与演进方案。
