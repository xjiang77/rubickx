package ratelimiter

import (
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeClock 是可手动推进的假时钟。
type fakeClock struct{ t time.Time }

func (c *fakeClock) Now() time.Time          { return c.t }
func (c *fakeClock) Advance(d time.Duration) { c.t = c.t.Add(d) }
func (c *fakeClock) Set(t time.Time)         { c.t = t }

func newFakeClock() *fakeClock { return &fakeClock{t: time.Unix(0, 0)} }

func TestTokenBucketBurstThenEmpty(t *testing.T) {
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

func TestTokenBucketRefillOverTime(t *testing.T) {
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

func TestTokenBucketCap(t *testing.T) {
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

// TestTokenBucketConcurrentSafe：无补充、恰好 1000 令牌；100 goroutine ×50 抢，
// 线程安全则恰好放行 1000。配 go test -race 验证无数据竞态。
func TestTokenBucketConcurrentSafe(t *testing.T) {
	b := New(1000, 0)
	start := make(chan struct{})
	var allowed int64
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 50; j++ {
				if b.AllowN(1) {
					atomic.AddInt64(&allowed, 1)
				}
			}
		}()
	}
	close(start)
	wg.Wait()
	if allowed != 1000 {
		t.Fatalf("concurrent allowed=%d, want exactly 1000", allowed)
	}
}

func TestTokenBucketInvalidConfigPanics(t *testing.T) {
	tests := []struct {
		name       string
		capacity   float64
		refillRate float64
		now        func() time.Time
	}{
		{name: "zero capacity", capacity: 0, refillRate: 1, now: time.Now},
		{name: "negative capacity", capacity: -1, refillRate: 1, now: time.Now},
		{name: "NaN capacity", capacity: math.NaN(), refillRate: 1, now: time.Now},
		{name: "infinite capacity", capacity: math.Inf(1), refillRate: 1, now: time.Now},
		{name: "negative refill rate", capacity: 1, refillRate: -1, now: time.Now},
		{name: "NaN refill rate", capacity: 1, refillRate: math.NaN(), now: time.Now},
		{name: "infinite refill rate", capacity: 1, refillRate: math.Inf(1), now: time.Now},
		{name: "nil clock", capacity: 1, refillRate: 1, now: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected constructor to panic")
				}
			}()
			NewWithClock(tt.capacity, tt.refillRate, tt.now)
		})
	}
}

func TestTokenBucketInvalidCostLeavesStateUnchanged(t *testing.T) {
	clk := newFakeClock()
	b := NewWithClock(2, 2, clk.Now)
	if !b.AllowN(2) {
		t.Fatal("expected initial tokens to be available")
	}
	clk.Advance(time.Second)

	for _, n := range []float64{0, -1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		beforeTokens, beforeLast := b.Tokens(), b.last
		if b.AllowN(n) {
			t.Fatalf("AllowN(%v) = true, want false", n)
		}
		if b.Tokens() != beforeTokens || b.last != beforeLast {
			t.Fatalf("AllowN(%v) changed state", n)
		}
	}

	if !b.AllowN(2) {
		t.Fatal("invalid costs advanced the time baseline")
	}
}

func TestTokenBucketFractionalAndMultiTokenCosts(t *testing.T) {
	b := NewWithClock(3, 0, newFakeClock().Now)
	if !b.AllowN(1.5) || !b.AllowN(1.5) {
		t.Fatal("expected two fractional costs to consume the bucket")
	}
	if b.AllowN(0.1) {
		t.Fatal("expected deny after fractional costs consumed the bucket")
	}
}

func TestTokenBucketClockRollbackDoesNotMoveBaseline(t *testing.T) {
	clk := newFakeClock()
	b := NewWithClock(2, 1, clk.Now)
	if !b.AllowN(2) {
		t.Fatal("expected initial tokens to be available")
	}

	clk.Advance(time.Second)
	if !b.Allow() {
		t.Fatal("expected one refilled token")
	}
	baseline := b.last

	clk.Set(time.Unix(0, 0))
	if b.Allow() {
		t.Fatal("clock rollback must not create tokens")
	}
	if b.last != baseline {
		t.Fatal("clock rollback moved the time baseline")
	}

	clk.Set(time.Unix(0, 0).Add(1500 * time.Millisecond))
	if b.Allow() {
		t.Fatal("expected only half a token since the pre-rollback baseline")
	}
	if got := b.Tokens(); got != 0.5 {
		t.Fatalf("tokens=%v, want 0.5", got)
	}
}
