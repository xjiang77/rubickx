package main

import (
	"sync"
	"sync/atomic"
)

// 三种计数器，演示并发下"读-改-写"的正确性。
// 对照 [[Eng - Redis：架构、实现与高阶实战]] §7：Redis 用"单线程服务端"消除竞态，
// Go 单机进程内则靠 Mutex 或 atomic 消除竞态。

// UnsafeCounter：无任何同步的计数器。
// Inc() 的 c.n++ 实际是 load → +1 → store 三步，多个 goroutine 交叉执行会丢更新。
// 等价于 Redis 笔记里的 GET-then-SET bug；也等价于 Java 多线程对普通 int 做 count++。
type UnsafeCounter struct{ n int }

func (c *UnsafeCounter) Inc()       { c.n++ }
func (c *UnsafeCounter) Value() int { return c.n }

// MutexCounter：用互斥锁保护临界区。
// Java 锚点：≈ synchronized 块 / ReentrantLock。
type MutexCounter struct {
	mu sync.Mutex
	n  int
}

func (c *MutexCounter) Inc() {
	c.mu.Lock()
	c.n++
	c.mu.Unlock()
}

func (c *MutexCounter) Value() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

// AtomicCounter：用原子操作，无锁。
// Java 锚点：≈ AtomicInteger / AtomicLong（底层都是 CPU 的 CAS/FAA 指令）。
type AtomicCounter struct{ n int64 }

func (c *AtomicCounter) Inc()         { atomic.AddInt64(&c.n, 1) }
func (c *AtomicCounter) Value() int64 { return atomic.LoadInt64(&c.n) }
