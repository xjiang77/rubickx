# Progress - Patterns 四语言实现

本文件是动态完成度 SSOT。`[x]` 只表示该 pattern 的 canonical note、shared contract、四语言实现和四套 tests 已存在，并且 `make -C patterns test-pattern PATTERN=<id>` 已通过。身份、family、顺序和路径由 `catalog.json` 管理。

## Design / Creational

- [x] `gof.creational.factory-method` - Factory Method
- [x] `gof.creational.abstract-factory` - Abstract Factory
- [x] `gof.creational.builder` - Builder
- [x] `gof.creational.prototype` - Prototype
- [x] `gof.creational.singleton` - Singleton

## Design / Structural

- [x] `gof.structural.adapter` - Adapter
- [x] `gof.structural.facade` - Facade
- [x] `gof.structural.decorator` - Decorator
- [x] `gof.structural.proxy` - Proxy
- [x] `gof.structural.composite` - Composite
- [x] `gof.structural.bridge` - Bridge
- [x] `gof.structural.flyweight` - Flyweight

## Design / Behavioral

- [x] `gof.behavioral.strategy` - Strategy
- [x] `gof.behavioral.state` - State
- [x] `gof.behavioral.template-method` - Template Method
- [x] `gof.behavioral.command` - Command
- [x] `gof.behavioral.chain-of-responsibility` - Chain of Responsibility
- [x] `gof.behavioral.observer` - Observer
- [x] `gof.behavioral.mediator` - Mediator
- [x] `gof.behavioral.iterator` - Iterator
- [x] `gof.behavioral.memento` - Memento
- [x] `gof.behavioral.visitor` - Visitor
- [x] `gof.behavioral.interpreter` - Interpreter

## Reliability

- [x] `reliability.timeout` - Timeout
- [x] `reliability.retry` - Retry
- [x] `reliability.circuit-breaker` - Circuit Breaker
- [x] `reliability.bulkhead` - Bulkhead
- [x] `reliability.load-shedding` - Load Shedding
- [x] `reliability.hedged-requests` - Hedged Requests

## Data & Messaging

- [x] `data-messaging.transactional-outbox` - Transactional Outbox
- [x] `data-messaging.idempotent-consumer-inbox` - Idempotent Consumer and Inbox
- [x] `data-messaging.saga` - Saga
- [x] `data-messaging.publisher-subscriber` - Publisher-Subscriber
- [x] `data-messaging.dead-letter-channel` - Dead Letter Channel
- [x] `data-messaging.cqrs` - CQRS
- [x] `data-messaging.event-sourcing` - Event Sourcing

## Concurrency

- [x] `concurrency.bounded-producer-consumer` - Bounded Producer-Consumer
- [x] `concurrency.worker-pool` - Worker Pool
- [x] `concurrency.pipeline` - Pipeline
- [x] `concurrency.fan-out-fan-in` - Fan-out and Fan-in
- [x] `concurrency.future-promise` - Future and Promise
- [x] `concurrency.structured-concurrency-cancellation` - Structured Concurrency and Cancellation

## Final gate

全部 42 项勾选后运行：

```bash
make -C patterns verify
make -C patterns verify-vault VAULT_ROOT=/Users/kevinxjiang/Obsidian/dragon-vault
```
