package ratelimiter

import (
	"math"
	"sync"
	"time"
)

// SlidingWindowCounter 是并发安全的滑动窗口计数器。
type SlidingWindowCounter struct {
	mu                 sync.Mutex
	limit              float64
	window             time.Duration
	previousCount      float64
	currentCount       float64
	currentWindowStart time.Time
	now                func() time.Time
}

// NewSlidingWindowCounter 用真实时钟构造滑动窗口计数器。
func NewSlidingWindowCounter(limit float64, window time.Duration) *SlidingWindowCounter {
	return NewSlidingWindowCounterWithClock(limit, window, time.Now)
}

// NewSlidingWindowCounterWithClock 用可注入时钟构造滑动窗口计数器。
func NewSlidingWindowCounterWithClock(limit float64, window time.Duration, now func() time.Time) *SlidingWindowCounter {
	if limit <= 0 || math.IsNaN(limit) || math.IsInf(limit, 0) {
		panic("limit must be positive and finite")
	}
	if window <= 0 {
		panic("window must be positive")
	}
	if now == nil {
		panic("now must not be nil")
	}

	return &SlidingWindowCounter{
		limit:              limit,
		window:             window,
		currentWindowStart: now(),
		now:                now,
	}
}

// Allow 记录一个请求，成功返回 true。
func (c *SlidingWindowCounter) Allow() bool { return c.AllowN(1) }

// AllowN 记录权重为 n 的请求，成功返回 true。
func (c *SlidingWindowCounter) AllowN(n float64) bool {
	if n <= 0 || math.IsNaN(n) || math.IsInf(n, 0) {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	elapsed := c.now().Sub(c.currentWindowStart)
	if elapsed < 0 {
		elapsed = 0
	}
	if elapsed >= c.window {
		windowsElapsed := elapsed / c.window
		if windowsElapsed == 1 {
			c.previousCount = c.currentCount
		} else {
			c.previousCount = 0
		}
		c.currentCount = 0
		c.currentWindowStart = c.currentWindowStart.Add(windowsElapsed * c.window)
		elapsed -= windowsElapsed * c.window
	}

	previousWeight := 1 - float64(elapsed)/float64(c.window)
	estimate := c.currentCount + c.previousCount*previousWeight
	if estimate+n > c.limit {
		return false
	}
	c.currentCount += n
	return true
}
