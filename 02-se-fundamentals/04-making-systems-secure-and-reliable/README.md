# Making systems secure and reliable

安全与可靠性。通过信任边界、测试、隔离和降级控制系统风险。

## 来源

[Andrew Ng 原文](https://x.com/AndrewYNg/status/2093388974194872781)

## 当前状态

**已有局部实践**。10 个 loopback 安全 labs 与 reliability patterns 只证明给定实验条件。

## 主归属实践

- [network-security](network-security) — 10 个 loopback 网络安全 labs。

## 跨能力链接

- [Designing system architectures](../03-designing-system-architectures/README.md)
- [Reliability patterns](../03-designing-system-architectures/patterns/02-reliability-patterns)

## 验证入口

从仓库根目录执行（Nanochat 命令在具体系统目录执行）：

- `make -C 02-se-fundamentals/04-making-systems-secure-and-reliable/network-security verify`

## 下一步

将故障注入与安全边界映射到具体系统并保留证据。
