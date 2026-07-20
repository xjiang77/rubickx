package ratelimiter

import (
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSlidingWindowCurrentWindowLimit(t *testing.T) {
	clk := newFakeClock()
	c := NewSlidingWindowCounterWithClock(3, 10*time.Second, clk.Now)
	for i := 0; i < 3; i++ {
		if !c.Allow() {
			t.Fatalf("request %d denied, want allow", i+1)
		}
	}
	if c.Allow() {
		t.Fatal("expected deny at the current-window limit")
	}
}

func TestSlidingWindowExactBoundaryPreventsDoubleBurst(t *testing.T) {
	clk := newFakeClock()
	c := NewSlidingWindowCounterWithClock(5, 10*time.Second, clk.Now)
	if !c.AllowN(5) {
		t.Fatal("expected initial burst")
	}

	clk.Advance(10 * time.Second)
	if c.Allow() {
		t.Fatal("previous window must have full weight at the exact boundary")
	}
	if c.previousCount != 5 || c.currentCount != 0 {
		t.Fatal("denied boundary request changed counts")
	}
}

func TestSlidingWindowHalfWindowWeightsPreviousCountByHalf(t *testing.T) {
	clk := newFakeClock()
	c := NewSlidingWindowCounterWithClock(4, 10*time.Second, clk.Now)
	if !c.AllowN(4) {
		t.Fatal("expected initial burst")
	}

	clk.Advance(15 * time.Second)
	if !c.AllowN(2) {
		t.Fatal("expected two units beside a half-weight previous count")
	}
	if c.AllowN(0.1) {
		t.Fatal("expected estimate to be at the limit")
	}
}

func TestSlidingWindowTwoWindowsClearHistory(t *testing.T) {
	clk := newFakeClock()
	c := NewSlidingWindowCounterWithClock(3, 10*time.Second, clk.Now)
	if !c.AllowN(3) {
		t.Fatal("expected initial burst")
	}

	clk.Advance(20 * time.Second)
	if !c.AllowN(3) {
		t.Fatal("history should be empty after two windows")
	}
	if c.previousCount != 0 || c.currentCount != 3 {
		t.Fatal("unexpected state after clearing old history")
	}
}

func TestSlidingWindowRejectedRequestIsNotCounted(t *testing.T) {
	c := NewSlidingWindowCounterWithClock(5, time.Minute, newFakeClock().Now)
	if !c.AllowN(4) {
		t.Fatal("expected first request to be allowed")
	}
	if c.AllowN(2) {
		t.Fatal("expected request beyond the limit to be denied")
	}
	if !c.Allow() {
		t.Fatal("denied request must not consume capacity")
	}
}

func TestSlidingWindowConcurrentAllowsExactlyLimit(t *testing.T) {
	now := time.Unix(0, 0)
	c := NewSlidingWindowCounterWithClock(1000, time.Minute, func() time.Time { return now })
	start := make(chan struct{})
	var allowed int64
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 50; j++ {
				if c.Allow() {
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

func TestSlidingWindowInvalidConfigPanics(t *testing.T) {
	tests := []struct {
		name   string
		limit  float64
		window time.Duration
		now    func() time.Time
	}{
		{name: "zero limit", limit: 0, window: time.Second, now: time.Now},
		{name: "negative limit", limit: -1, window: time.Second, now: time.Now},
		{name: "NaN limit", limit: math.NaN(), window: time.Second, now: time.Now},
		{name: "infinite limit", limit: math.Inf(1), window: time.Second, now: time.Now},
		{name: "zero window", limit: 1, window: 0, now: time.Now},
		{name: "negative window", limit: 1, window: -time.Second, now: time.Now},
		{name: "nil clock", limit: 1, window: time.Second, now: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected constructor to panic")
				}
			}()
			NewSlidingWindowCounterWithClock(tt.limit, tt.window, tt.now)
		})
	}
}

func TestSlidingWindowInvalidCostLeavesStateUnchanged(t *testing.T) {
	clk := newFakeClock()
	c := NewSlidingWindowCounterWithClock(5, 10*time.Second, clk.Now)
	if !c.AllowN(3) {
		t.Fatal("expected initial request to be allowed")
	}
	clk.Advance(10 * time.Second)

	for _, n := range []float64{0, -1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		previous, current, start := c.previousCount, c.currentCount, c.currentWindowStart
		if c.AllowN(n) {
			t.Fatalf("AllowN(%v) = true, want false", n)
		}
		if c.previousCount != previous || c.currentCount != current || c.currentWindowStart != start {
			t.Fatalf("AllowN(%v) changed state", n)
		}
	}

	if c.currentWindowStart != time.Unix(0, 0) {
		t.Fatal("invalid costs advanced the window")
	}
}

func TestSlidingWindowFractionalAndMultiTokenCosts(t *testing.T) {
	c := NewSlidingWindowCounterWithClock(3, time.Minute, newFakeClock().Now)
	if !c.AllowN(1.5) || !c.AllowN(1.5) {
		t.Fatal("expected two fractional costs to reach the limit")
	}
	if c.AllowN(0.1) {
		t.Fatal("expected deny after fractional costs reached the limit")
	}
}

func TestSlidingWindowClockRollbackDoesNotMoveWindow(t *testing.T) {
	clk := newFakeClock()
	c := NewSlidingWindowCounterWithClock(4, 10*time.Second, clk.Now)
	if !c.AllowN(3) {
		t.Fatal("expected initial request to be allowed")
	}

	clk.Advance(10 * time.Second)
	if c.AllowN(2) {
		t.Fatal("expected previous window to block request at the boundary")
	}
	start := c.currentWindowStart
	previous, current := c.previousCount, c.currentCount

	clk.Set(time.Unix(0, 0))
	if c.AllowN(2) {
		t.Fatal("clock rollback must not reduce the estimate")
	}
	if c.currentWindowStart != start || c.previousCount != previous || c.currentCount != current {
		t.Fatal("clock rollback changed the window state")
	}

	clk.Set(time.Unix(0, 0).Add(15 * time.Second))
	if !c.AllowN(2) {
		t.Fatal("expected request after previous count decayed by half")
	}
}
