package ratelimiter

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeClock 是可手动推进的假时钟。
type fakeClock struct{ t time.Time }

func (c *fakeClock) Now() time.Time          { return c.t }
func (c *fakeClock) Advance(d time.Duration) { c.t = c.t.Add(d) }

func newFakeClock() *fakeClock { return &fakeClock{t: time.Unix(0, 0)} }

func TestBurstThenEmpty(t *testing.T) {
	clk := newFakeClock()
	b := NewWithClock(5, 2, clk.Now)
	allowed := 0
	for i := 0; i < 5; i++ {
		if b.Allow() {
			allowed++
		}
	}
	if allowed != 5 {
		t.Fatalf("burst allowed=%d, want 5", allowed)
	}
	if b.Allow() {
		t.Fatal("expected deny when bucket empty")
	}
}

func TestRefillOverTime(t *testing.T) {
	clk := newFakeClock()
	b := NewWithClock(5, 2, clk.Now)
	for i := 0; i < 5; i++ {
		b.Allow()
	}
	clk.Advance(time.Second) // 补 2 个令牌
	if !b.Allow() || !b.Allow() {
		t.Fatal("expected 2 allows after 1s refill")
	}
	if b.Allow() {
		t.Fatal("expected deny after consuming refill")
	}
}

func TestCap(t *testing.T) {
	clk := newFakeClock()
	b := NewWithClock(5, 10, clk.Now)
	clk.Advance(100 * time.Second) // 理论补 1000，但封顶 5
	allowed := 0
	for i := 0; i < 10; i++ {
		if b.Allow() {
			allowed++
		}
	}
	if allowed != 5 {
		t.Fatalf("capped allowed=%d, want 5", allowed)
	}
}

// TestConcurrentSafe：无补充、恰好 1000 令牌；100 goroutine ×50 抢，
// 线程安全则恰好放行 1000。配 go test -race 验证无数据竞态。
func TestConcurrentSafe(t *testing.T) {
	b := New(1000, 0)
	var allowed int64
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				if b.AllowN(1) {
					atomic.AddInt64(&allowed, 1)
				}
			}
		}()
	}
	wg.Wait()
	if allowed != 1000 {
		t.Fatalf("concurrent allowed=%d, want exactly 1000", allowed)
	}
}
