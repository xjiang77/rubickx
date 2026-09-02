package server

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type FixedWindowStore interface {
	Allow(context.Context, string, int, time.Duration, int, time.Time) (Decision, error)
}

type memoryWindow struct {
	start time.Time
	used  int
}

type MemoryFixedWindowStore struct {
	mu      sync.Mutex
	windows map[string]memoryWindow
}

func NewMemoryFixedWindowStore() *MemoryFixedWindowStore {
	return &MemoryFixedWindowStore{windows: map[string]memoryWindow{}}
}

func (m *MemoryFixedWindowStore) Allow(_ context.Context, key string, limit int, window time.Duration, cost int, now time.Time) (Decision, error) {
	if limit <= 0 || window <= 0 || cost <= 0 {
		return Decision{}, fmt.Errorf("limit, window and cost must be positive")
	}
	windowMs := window.Milliseconds()
	startMs := (now.UnixMilli() / windowMs) * windowMs
	start := time.UnixMilli(startMs)
	m.mu.Lock()
	defer m.mu.Unlock()
	state := m.windows[key]
	if !state.start.Equal(start) {
		state = memoryWindow{start: start}
	}
	allowed := state.used+cost <= limit
	if allowed {
		state.used += cost
	}
	m.windows[key] = state
	remaining := maxInt(0, limit-state.used)
	retry := float64(0)
	reason := "fixed window capacity available"
	if !allowed {
		if cost > limit {
			reason = "cost_exceeds_limit"
		} else {
			retry = float64(maxInt64(1, start.Add(window).UnixMilli()-now.UnixMilli()))
			reason = "fixed window limit exceeded"
		}
	}
	return normalizeDecision(Decision{
		Allowed:      allowed,
		Remaining:    float64(remaining),
		RetryAfterMs: retry,
		ResetAtMs:    float64(start.Add(window).UnixMilli()),
		Reason:       reason,
	}), nil
}

type RedisFixedWindowStore struct {
	Addr        string
	Prefix      string
	DialTimeout time.Duration
}

func NewRedisFixedWindowStore(addr, prefix string) *RedisFixedWindowStore {
	return &RedisFixedWindowStore{Addr: addr, Prefix: prefix, DialTimeout: time.Second}
}

const redisFixedWindowLua = `
local cost = tonumber(ARGV[1])
local expires_in_ms = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local current = tonumber(redis.call('GET', KEYS[1]) or '0')
local allowed = 0
if cost <= limit and current + cost <= limit then
  current = redis.call('INCRBY', KEYS[1], cost)
  allowed = 1
  if current == cost then
    redis.call('PEXPIRE', KEYS[1], expires_in_ms)
  end
end
local ttl = redis.call('PTTL', KEYS[1])
if ttl < 0 then
  ttl = expires_in_ms
end
return {allowed, current, ttl}
`

func (r *RedisFixedWindowStore) Allow(ctx context.Context, key string, limit int, window time.Duration, cost int, now time.Time) (Decision, error) {
	if r.Addr == "" {
		return Decision{}, fmt.Errorf("Redis address is not configured")
	}
	if limit <= 0 || window <= 0 || cost <= 0 {
		return Decision{}, fmt.Errorf("limit, window and cost must be positive")
	}
	windowMs := window.Milliseconds()
	bucket := now.UnixMilli() / windowMs
	redisKey := fmt.Sprintf("%s%s:%d", r.Prefix, key, bucket)
	expiresInMs := windowMs - floorMod(now.UnixMilli(), windowMs)
	result, err := r.eval(ctx, redisFixedWindowLua, redisKey, strconv.Itoa(cost), strconv.FormatInt(expiresInMs, 10), strconv.Itoa(limit))
	if err != nil {
		return Decision{}, err
	}
	values, ok := result.([]any)
	if !ok || len(values) != 3 {
		return Decision{}, fmt.Errorf("unexpected Redis Lua result: %#v", result)
	}
	allowedRaw, okAllowed := values[0].(int64)
	current, okCurrent := values[1].(int64)
	ttl, okTTL := values[2].(int64)
	if !okAllowed || !okCurrent || !okTTL {
		return Decision{}, fmt.Errorf("unexpected Redis Lua value types: %#v", result)
	}
	if ttl < 0 {
		ttl = windowMs
	}
	allowed := allowedRaw == 1
	remaining := maxInt64(0, int64(limit)-current)
	retry := float64(0)
	reason := "shared fixed window capacity available"
	if !allowed {
		if cost > limit {
			reason = "cost_exceeds_limit"
		} else {
			retry = float64(ttl)
			reason = "shared fixed window limit exceeded"
		}
	}
	return normalizeDecision(Decision{
		Allowed:      allowed,
		Remaining:    float64(remaining),
		RetryAfterMs: retry,
		ResetAtMs:    float64(now.UnixMilli() + ttl),
		Reason:       reason,
	}), nil
}

func floorMod(value, divisor int64) int64 {
	result := value % divisor
	if result < 0 {
		result += divisor
	}
	return result
}

func (r *RedisFixedWindowStore) eval(ctx context.Context, script, key string, args ...string) (any, error) {
	timeout := r.DialTimeout
	if timeout <= 0 {
		timeout = time.Second
	}
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", r.Addr)
	if err != nil {
		return nil, fmt.Errorf("dial Redis: %w", err)
	}
	defer conn.Close()
	deadline := time.Now().Add(timeout)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	_ = conn.SetDeadline(deadline)
	parts := []string{"EVAL", script, "1", key}
	parts = append(parts, args...)
	if err := writeRESPArray(conn, parts); err != nil {
		return nil, err
	}
	value, err := readRESP(bufio.NewReader(conn))
	if err != nil {
		return nil, fmt.Errorf("read Redis response: %w", err)
	}
	return value, nil
}

func writeRESPArray(writer io.Writer, parts []string) error {
	var builder strings.Builder
	fmt.Fprintf(&builder, "*%d\r\n", len(parts))
	for _, part := range parts {
		fmt.Fprintf(&builder, "$%d\r\n%s\r\n", len(part), part)
	}
	_, err := io.WriteString(writer, builder.String())
	return err
}

func readRESP(reader *bufio.Reader) (any, error) {
	prefix, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	line, err := readRESPLine(reader)
	if err != nil {
		return nil, err
	}
	switch prefix {
	case '+':
		return line, nil
	case '-':
		return nil, errors.New(line)
	case ':':
		return strconv.ParseInt(line, 10, 64)
	case '$':
		length, parseErr := strconv.Atoi(line)
		if parseErr != nil {
			return nil, parseErr
		}
		if length < 0 {
			return nil, nil
		}
		buffer := make([]byte, length+2)
		if _, err := io.ReadFull(reader, buffer); err != nil {
			return nil, err
		}
		return string(buffer[:length]), nil
	case '*':
		count, parseErr := strconv.Atoi(line)
		if parseErr != nil {
			return nil, parseErr
		}
		values := make([]any, 0, count)
		for i := 0; i < count; i++ {
			value, itemErr := readRESP(reader)
			if itemErr != nil {
				return nil, itemErr
			}
			values = append(values, value)
		}
		return values, nil
	default:
		return nil, fmt.Errorf("unsupported RESP prefix %q", prefix)
	}
}

func readRESPLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	if !strings.HasSuffix(line, "\r\n") {
		return "", fmt.Errorf("invalid RESP line")
	}
	return strings.TrimSuffix(line, "\r\n"), nil
}

func maxInt(a, b int) int {
	return int(math.Max(float64(a), float64(b)))
}
