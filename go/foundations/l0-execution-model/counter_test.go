package main

import "testing"

// Mutex / Atomic 版在 go test -race 下也应全绿（无竞态 + 结果正确）。
// 复用 main.go 里的 hammer()。

func TestMutexCounter_Correct(t *testing.T) {
	c := &MutexCounter{}
	hammer(c.Inc, 100, 1000)
	if got, want := c.Value(), 100*1000; got != want {
		t.Fatalf("MutexCounter = %d, want %d", got, want)
	}
}

func TestAtomicCounter_Correct(t *testing.T) {
	c := &AtomicCounter{}
	hammer(c.Inc, 100, 1000)
	if got, want := c.Value(), int64(100*1000); got != want {
		t.Fatalf("AtomicCounter = %d, want %d", got, want)
	}
}

// 故意不为 UnsafeCounter 写"结果相等"的断言：它在 -race 下会被判负，
// 普通 test 下结果也不稳定。要亲眼看竞态，用：
//   go run -race ./foundations/l0-execution-model

func BenchmarkMutexCounter(b *testing.B) {
	c := &MutexCounter{}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc()
		}
	})
}

func BenchmarkAtomicCounter(b *testing.B) {
	c := &AtomicCounter{}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc()
		}
	})
}
