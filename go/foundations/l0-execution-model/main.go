// foundations/l0-execution-model
//
// GP0 · L0 执行模型的可跑实验：goroutine 有多轻 + 并发下的数据竞态。
// 配套笔记：Vault `Session Log - GP0 系统基础前置` / `Redis：契约、状态与运维`
//
// 一句话桥接：Redis 用单 shard execution owner 串行执行单条 command；
// Go 单机用 Mutex/atomic 保护同一 read-modify-write。两者的 atomicity scope 不同。
//
// 跑法：
//
//	go run ./foundations/l0-execution-model            # 看现象（含丢更新）
//	go run -race ./foundations/l0-execution-model      # 竞态检测器会抓 UnsafeCounter
//	go test -race ./foundations/l0-execution-model     # Mutex/Atomic 版应全绿
//	go test -bench . ./foundations/l0-execution-model  # atomic vs mutex 性能对比
package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

func main() {
	demoGoroutineCheap()
	fmt.Println()
	demoRace()
}

// demoGoroutineCheap：起 10 万个 goroutine，体会它有多轻。
// Java 锚点：一个 OS 线程默认栈约 1MB，开 10 万个直接 OOM；
// goroutine 初始栈仅 ~2KB，由 Go runtime（GMP 调度器）在少量 OS 线程上多路复用，
// 几十万个稀松平常。概念上 ≈ Java 21 的虚拟线程(Project Loom)。
func demoGoroutineCheap() {
	const n = 100_000
	var wg sync.WaitGroup
	var m0, m1 runtime.MemStats

	runtime.ReadMemStats(&m0)
	start := time.Now()

	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			time.Sleep(time.Millisecond) // 模拟一点点工作
		}()
	}
	wg.Wait()

	runtime.ReadMemStats(&m1)
	fmt.Printf("[goroutine] 起 %d 个 goroutine：用时 %v，期间约分配 %d KB 堆，当前 OS 线程数约 %d\n",
		n, time.Since(start).Round(time.Millisecond),
		(m1.TotalAlloc-m0.TotalAlloc)/1024, runtime.GOMAXPROCS(0))
}

// demoRace：100 个 goroutine 各自 +1 共 per 次，对比三种计数器的最终值。
func demoRace() {
	const goroutines, per = 100, 1000
	want := goroutines * per

	unsafe := &UnsafeCounter{}
	mu := &MutexCounter{}
	at := &AtomicCounter{}

	hammer(unsafe.Inc, goroutines, per)
	hammer(mu.Inc, goroutines, per)
	hammer(at.Inc, goroutines, per)

	fmt.Printf("[race] 期望值        = %d\n", want)
	fmt.Printf("[race] UnsafeCounter = %d  <- 通常 < 期望（丢更新；go run -race 会报 DATA RACE）\n", unsafe.Value())
	fmt.Printf("[race] MutexCounter  = %d  <- 锁保护，永远正确\n", mu.Value())
	fmt.Printf("[race] AtomicCounter = %d  <- 原子操作，永远正确\n", at.Value())
}

// hammer：开 goroutines 个 goroutine，每个调用 inc() per 次，全部结束后返回。
func hammer(inc func(), goroutines, per int) {
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < per; j++ {
				inc()
			}
		}()
	}
	wg.Wait()
}
