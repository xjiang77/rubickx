// Package ratelimiter 实现令牌桶限流器。
//
// 令牌桶：以固定速率补充令牌，每次请求取走若干，桶空则拒绝。
// 两个旋钮：capacity（突发上限）、refillRate（长期平均速率，令牌/秒）。
//
// 并发：用 sync.Mutex 保护"读-补-判-扣-写"这段复合操作。
package ratelimiter

import (
	"sync"
	"time"
)

// TokenBucket 是并发安全的令牌桶。
type TokenBucket struct {
	mu         sync.Mutex
	capacity   float64
	refillRate float64
	tokens     float64
	last       time.Time
	now        func() time.Time // 可注入时钟，便于确定性测试
}

// New 用真实时钟构造令牌桶。
func New(capacity, refillRate float64) *TokenBucket {
	return NewWithClock(capacity, refillRate, time.Now)
}

// NewWithClock 用可注入时钟构造（测试用假时钟推进）。
func NewWithClock(capacity, refillRate float64, now func() time.Time) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		refillRate: refillRate,
		tokens:     capacity,
		last:       now(),
		now:        now,
	}
}

// Allow 取 1 个令牌，成功返回 true。
func (b *TokenBucket) Allow() bool { return b.AllowN(1) }

// AllowN 取 n 个令牌，成功返回 true。
func (b *TokenBucket) AllowN(n float64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	t := b.now()
	elapsed := t.Sub(b.last).Seconds()
	b.last = t
	b.tokens = min(b.capacity, b.tokens+elapsed*b.refillRate)

	if b.tokens >= n {
		b.tokens -= n
		return true
	}
	return false
}

// Tokens 返回当前令牌数（主要用于测试/观测）。
func (b *TokenBucket) Tokens() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.tokens
}
