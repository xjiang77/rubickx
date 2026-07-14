package server

import (
	"fmt"
	"math"
	"runtime"
	"sort"
	"unicode/utf8"
)

type traceEmitter struct {
	events []TraceEvent
}

func (e *traceEmitter) emit(stepID, actor string, timestampMs int64, before, after map[string]any, decision *Decision, reason string) {
	_, file, line, _ := runtime.Caller(1)
	e.events = append(e.events, TraceEvent{
		Seq:         len(e.events) + 1,
		StepID:      stepID,
		Actor:       actor,
		TimestampMs: timestampMs,
		Before:      before,
		After:       after,
		Decision:    decision,
		Reason:      reason,
		Source:      SourceAnchor{Path: labRelativePath(file), Line: line},
	})
}

type algorithm interface {
	Allow(RequestPoint) (Decision, error)
}

const maxSafeIntegerMilliseconds int64 = 9_007_199_254_740_991

func newAlgorithm(name string, config map[string]float64, emitter *traceEmitter) (algorithm, error) {
	switch name {
	case AlgorithmFixedWindow:
		limit, window, err := windowConfig(config)
		if err != nil {
			return nil, err
		}
		return &fixedWindow{limit: limit, windowMs: window, states: map[string]fixedWindowState{}, trace: emitter}, nil
	case AlgorithmSlidingWindowLog:
		limit, window, err := windowConfig(config)
		if err != nil {
			return nil, err
		}
		return &slidingWindowLog{limit: limit, windowMs: window, states: map[string][]logEntry{}, trace: emitter}, nil
	case AlgorithmSlidingWindowCount:
		limit, window, err := windowConfig(config)
		if err != nil {
			return nil, err
		}
		return &slidingWindowCounter{limit: limit, windowMs: window, states: map[string]counterState{}, trace: emitter}, nil
	case AlgorithmTokenBucket:
		capacity, rate, err := rateConfig(config)
		if err != nil {
			return nil, err
		}
		return &tokenBucket{capacity: capacity, ratePerMs: rate / 1000, states: map[string]tokenState{}, trace: emitter}, nil
	case AlgorithmLeakyBucket:
		capacity, rate, err := rateConfig(config)
		if err != nil {
			return nil, err
		}
		return &leakyBucket{capacity: capacity, ratePerMs: rate / 1000, states: map[string]leakyState{}, trace: emitter}, nil
	default:
		return nil, fmt.Errorf("unsupported algorithm %q", name)
	}
}

func windowConfig(config map[string]float64) (float64, int64, error) {
	limit, ok := config["limit"]
	if !ok || !positiveSafeQuantity(limit) {
		return 0, 0, fmt.Errorf("config.limit must be a positive finite safe quantity")
	}
	windowRaw, ok := config["windowMs"]
	if !ok || windowRaw <= 0 || windowRaw != math.Trunc(windowRaw) || windowRaw > float64(maxSafeIntegerMilliseconds) {
		return 0, 0, fmt.Errorf("config.windowMs must be a positive safe integer in milliseconds")
	}
	return limit, int64(windowRaw), nil
}

func rateConfig(config map[string]float64) (float64, float64, error) {
	capacity, ok := config["capacity"]
	if !ok || !positiveSafeQuantity(capacity) {
		return 0, 0, fmt.Errorf("config.capacity must be a positive finite safe quantity")
	}
	rate, ok := config["ratePerSecond"]
	if !ok || !positiveSafeQuantity(rate) {
		return 0, 0, fmt.Errorf("config.ratePerSecond must be a positive finite safe quantity")
	}
	ratePerMs := rate / 1000
	recoveryMs := capacity / ratePerMs
	if ratePerMs <= 0 || math.IsNaN(recoveryMs) || math.IsInf(recoveryMs, 0) || recoveryMs > float64(maxSafeIntegerMilliseconds) {
		return 0, 0, fmt.Errorf("bucket full recovery must fit in safe integer milliseconds")
	}
	return capacity, rate, nil
}

func positiveSafeQuantity(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0) && value <= float64(maxSafeIntegerMilliseconds)
}

func validatePoint(point RequestPoint, lastTimestamp int64, initialized bool) (RequestPoint, error) {
	if point.Key == "" {
		return point, fmt.Errorf("request key must be a non-empty string")
	}
	if len(point.Key) > 128 || !utf8.ValidString(point.Key) {
		return point, fmt.Errorf("request key must be valid UTF-8 containing at most 128 bytes")
	}
	if !positiveSafeQuantity(point.Cost) {
		return point, fmt.Errorf("request cost must be a positive finite safe quantity")
	}
	if point.AtMs < 0 {
		return point, fmt.Errorf("request atMs must not be negative")
	}
	if point.AtMs > maxSafeIntegerMilliseconds {
		return point, fmt.Errorf("request atMs must be a safe integer in milliseconds")
	}
	if initialized && point.AtMs < lastTimestamp {
		return point, fmt.Errorf("request timeline must be monotonic")
	}
	return point, nil
}

type fixedWindowState struct {
	start int64
	count float64
	set   bool
}

type fixedWindow struct {
	limit    float64
	windowMs int64
	states   map[string]fixedWindowState
	trace    *traceEmitter
	last     int64
	seen     bool
}

func (f *fixedWindow) Allow(input RequestPoint) (Decision, error) {
	point, err := validatePoint(input, f.last, f.seen)
	if err != nil {
		return Decision{}, err
	}
	f.last, f.seen = point.AtMs, true
	state := f.states[point.Key]
	windowStart := (point.AtMs / f.windowMs) * f.windowMs
	if !state.set {
		state = fixedWindowState{start: windowStart, set: true}
	}
	before := map[string]any{"windowStartMs": state.start, "count": round6(state.count)}
	if !state.set || state.start != windowStart {
		state = fixedWindowState{start: windowStart, set: true}
	}
	selected := map[string]any{"windowStartMs": state.start, "count": round6(state.count)}
	// @step:fixed.locate-window
	f.trace.emit("fixed.locate-window", point.Key, point.AtMs, before, selected, nil, "window_selected")
	allowed := state.count+point.Cost <= f.limit
	if allowed {
		state.count += point.Cost
	}
	remaining := math.Max(0, f.limit-state.count)
	retry := float64(0)
	reason := "within_limit"
	if !allowed {
		if point.Cost > f.limit {
			reason = "cost_exceeds_limit"
		} else {
			retry = float64(state.start + f.windowMs - point.AtMs)
			reason = "limit_exceeded"
		}
	}
	decision := normalizeDecision(Decision{Allowed: allowed, Remaining: remaining, RetryAfterMs: retry, ResetAtMs: float64(state.start + f.windowMs), Reason: reason})
	f.states[point.Key] = state
	// @step:fixed.decision
	f.trace.emit("fixed.decision", point.Key, point.AtMs, selected, map[string]any{"windowStartMs": state.start, "count": round6(state.count)}, &decision, reason)
	return decision, nil
}

type logEntry struct {
	atMs int64
	cost float64
}

type slidingWindowLog struct {
	limit    float64
	windowMs int64
	states   map[string][]logEntry
	trace    *traceEmitter
	last     int64
	seen     bool
}

func (s *slidingWindowLog) Allow(input RequestPoint) (Decision, error) {
	point, err := validatePoint(input, s.last, s.seen)
	if err != nil {
		return Decision{}, err
	}
	s.last, s.seen = point.AtMs, true
	entries := s.states[point.Key]
	beforeUsed := logUsed(entries)
	before := map[string]any{"entries": logSnapshot(entries), "used": round6(beforeUsed)}
	cutoff := point.AtMs - s.windowMs
	first := sort.Search(len(entries), func(i int) bool { return entries[i].atMs > cutoff })
	entries = append([]logEntry(nil), entries[first:]...)
	used := logUsed(entries)
	afterEvict := map[string]any{"entries": logSnapshot(entries), "used": round6(used)}
	// @step:sliding-log.evict
	s.trace.emit("sliding-log.evict", point.Key, point.AtMs, before, afterEvict, nil, "expired_entries_removed")
	allowed := used+point.Cost <= s.limit
	if allowed {
		entries = append(entries, logEntry{atMs: point.AtMs, cost: point.Cost})
		used += point.Cost
	}
	retry := float64(0)
	reason := "within_limit"
	if !allowed {
		if point.Cost > s.limit {
			reason = "cost_exceeds_limit"
		} else {
			reason = "limit_exceeded"
			requiredRelease := used + point.Cost - s.limit
			released := 0.0
			for _, entry := range entries {
				released += entry.cost
				if released+1e-9 >= requiredRelease {
					retry = math.Max(0, float64(entry.atMs+s.windowMs-point.AtMs))
					break
				}
			}
		}
	}
	resetAt := float64(point.AtMs) + retry
	if allowed {
		resetAt = float64(point.AtMs)
		if len(entries) > 0 {
			resetAt = float64(entries[0].atMs + s.windowMs)
		}
	}
	decision := normalizeDecision(Decision{Allowed: allowed, Remaining: math.Max(0, s.limit-used), RetryAfterMs: retry, ResetAtMs: resetAt, Reason: reason})
	s.states[point.Key] = entries
	after := map[string]any{"entries": logSnapshot(entries), "used": round6(used)}
	// @step:sliding-log.decision
	s.trace.emit("sliding-log.decision", point.Key, point.AtMs, afterEvict, after, &decision, reason)
	return decision, nil
}

func logUsed(entries []logEntry) float64 {
	used := 0.0
	for _, entry := range entries {
		used += entry.cost
	}
	return used
}

func logSnapshot(entries []logEntry) []map[string]any {
	result := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		result = append(result, map[string]any{"atMs": entry.atMs, "cost": round6(entry.cost)})
	}
	return result
}

type counterState struct {
	start    int64
	current  float64
	previous float64
	set      bool
}

type slidingWindowCounter struct {
	limit    float64
	windowMs int64
	states   map[string]counterState
	trace    *traceEmitter
	last     int64
	seen     bool
}

func (s *slidingWindowCounter) Allow(input RequestPoint) (Decision, error) {
	point, err := validatePoint(input, s.last, s.seen)
	if err != nil {
		return Decision{}, err
	}
	s.last, s.seen = point.AtMs, true
	state := s.states[point.Key]
	windowStart := (point.AtMs / s.windowMs) * s.windowMs
	if !state.set {
		state = counterState{start: windowStart, set: true}
	}
	before := counterMap(state)
	if windowStart != state.start {
		windowsPassed := (windowStart - state.start) / s.windowMs
		if windowsPassed == 1 {
			state.previous = state.current
		} else {
			state.previous = 0
		}
		state.current = 0
		state.start = windowStart
	}
	// @step:sliding-counter.rotate
	s.trace.emit("sliding-counter.rotate", point.Key, point.AtMs, before, counterMap(state), nil, "windows_rotated")
	elapsed := float64(point.AtMs-state.start) / float64(s.windowMs)
	weight := math.Max(0, 1-elapsed)
	estimated := state.current + state.previous*weight
	estimateState := counterEstimateMap(state, weight, estimated)
	// @step:sliding-counter.estimate
	s.trace.emit("sliding-counter.estimate", point.Key, point.AtMs, counterMap(state), estimateState, nil, "weighted_count_estimated")
	allowed := estimated+point.Cost <= s.limit
	if allowed {
		state.current += point.Cost
		estimated += point.Cost
	}
	retry := float64(0)
	reason := "within_limit"
	if !allowed {
		if point.Cost > s.limit {
			reason = "cost_exceeds_limit"
		} else {
			reason = "limit_exceeded"
			retry = slidingCounterRetry(state, estimated, point.Cost, s.limit, s.windowMs, point.AtMs)
		}
	}
	resetAt := float64(state.start + s.windowMs)
	if !allowed && point.Cost <= s.limit {
		resetAt = float64(point.AtMs) + retry
	}
	decision := normalizeDecision(Decision{Allowed: allowed, Remaining: math.Max(0, s.limit-estimated), RetryAfterMs: retry, ResetAtMs: resetAt, Reason: reason})
	s.states[point.Key] = state
	// @step:sliding-counter.decision
	s.trace.emit("sliding-counter.decision", point.Key, point.AtMs, estimateState, counterEstimateMap(state, weight, estimated), &decision, reason)
	return decision, nil
}

func slidingCounterRetry(state counterState, estimated, cost, limit float64, windowMs, atMs int64) float64 {
	remainingToBoundary := float64(state.start + windowMs - atMs)
	requiredDrop := estimated + cost - limit
	if state.previous > 0 {
		withinCurrent := requiredDrop * float64(windowMs) / state.previous
		if withinCurrent <= remainingToBoundary+1e-9 {
			return math.Max(0, withinCurrent)
		}
	}
	if state.current+cost <= limit || state.current <= 0 {
		return math.Max(0, remainingToBoundary)
	}
	afterBoundary := (state.current + cost - limit) * float64(windowMs) / state.current
	return math.Max(0, remainingToBoundary+afterBoundary)
}

func counterMap(state counterState) map[string]any {
	return map[string]any{"currentWindowStartMs": state.start, "currentCount": round6(state.current), "previousCount": round6(state.previous)}
}

func counterEstimateMap(state counterState, weight, estimated float64) map[string]any {
	result := counterMap(state)
	result["previousWeight"] = round6(weight)
	result["estimatedCount"] = round6(estimated)
	return result
}

type tokenState struct {
	tokens float64
	lastMs int64
	set    bool
}

type tokenBucket struct {
	capacity  float64
	ratePerMs float64
	states    map[string]tokenState
	trace     *traceEmitter
	last      int64
	seen      bool
}

func (t *tokenBucket) Allow(input RequestPoint) (Decision, error) {
	point, err := validatePoint(input, t.last, t.seen)
	if err != nil {
		return Decision{}, err
	}
	t.last, t.seen = point.AtMs, true
	state := t.states[point.Key]
	if !state.set {
		state = tokenState{tokens: t.capacity, lastMs: point.AtMs, set: true}
	}
	before := map[string]any{"tokens": round6(state.tokens), "lastRefillMs": state.lastMs}
	state.tokens = math.Min(t.capacity, state.tokens+float64(point.AtMs-state.lastMs)*t.ratePerMs)
	state.lastMs = point.AtMs
	refilled := map[string]any{"tokens": round6(state.tokens), "lastRefillMs": state.lastMs}
	// @step:token.refill
	t.trace.emit("token.refill", point.Key, point.AtMs, before, refilled, nil, "tokens_refilled")
	allowed := state.tokens+1e-9 >= point.Cost
	if allowed {
		state.tokens -= point.Cost
	}
	retry := float64(0)
	reason := "token_available"
	if !allowed {
		if point.Cost > t.capacity {
			reason = "cost_exceeds_capacity"
		} else {
			retry = (point.Cost - state.tokens) / t.ratePerMs
			reason = "insufficient_tokens"
		}
	}
	resetAt := float64(point.AtMs) + (t.capacity-state.tokens)/t.ratePerMs
	decision := normalizeDecision(Decision{Allowed: allowed, Remaining: state.tokens, RetryAfterMs: retry, ResetAtMs: resetAt, Reason: reason})
	t.states[point.Key] = state
	// @step:token.decision
	t.trace.emit("token.decision", point.Key, point.AtMs, refilled, map[string]any{"tokens": round6(state.tokens), "lastRefillMs": state.lastMs}, &decision, reason)
	return decision, nil
}

type leakyState struct {
	water  float64
	lastMs int64
	set    bool
}

type leakyBucket struct {
	capacity  float64
	ratePerMs float64
	states    map[string]leakyState
	trace     *traceEmitter
	last      int64
	seen      bool
}

func (l *leakyBucket) Allow(input RequestPoint) (Decision, error) {
	point, err := validatePoint(input, l.last, l.seen)
	if err != nil {
		return Decision{}, err
	}
	l.last, l.seen = point.AtMs, true
	state := l.states[point.Key]
	if !state.set {
		state = leakyState{lastMs: point.AtMs, set: true}
	}
	before := map[string]any{"water": round6(state.water), "lastLeakMs": state.lastMs}
	state.water = math.Max(0, state.water-float64(point.AtMs-state.lastMs)*l.ratePerMs)
	state.lastMs = point.AtMs
	drained := map[string]any{"water": round6(state.water), "lastLeakMs": state.lastMs}
	// @step:leaky.drain
	l.trace.emit("leaky.drain", point.Key, point.AtMs, before, drained, nil, "queued_work_drained")
	allowed := state.water+point.Cost <= l.capacity+1e-9
	if allowed {
		state.water += point.Cost
	}
	retry := float64(0)
	reason := "queue_has_capacity"
	if !allowed {
		if point.Cost > l.capacity {
			reason = "cost_exceeds_capacity"
		} else {
			retry = (state.water + point.Cost - l.capacity) / l.ratePerMs
			reason = "queue_full"
		}
	}
	resetAt := float64(point.AtMs) + state.water/l.ratePerMs
	decision := normalizeDecision(Decision{Allowed: allowed, Remaining: math.Max(0, l.capacity-state.water), RetryAfterMs: retry, ResetAtMs: resetAt, Reason: reason})
	l.states[point.Key] = state
	// @step:leaky.decision
	l.trace.emit("leaky.decision", point.Key, point.AtMs, drained, map[string]any{"water": round6(state.water), "lastLeakMs": state.lastMs}, &decision, reason)
	return decision, nil
}

func normalizeDecision(decision Decision) Decision {
	decision.Remaining = round6(math.Max(0, decision.Remaining))
	decision.RetryAfterMs = round6(decision.RetryAfterMs)
	decision.ResetAtMs = round6(decision.ResetAtMs)
	if decision.RetryAfterMs < 0 {
		decision.RetryAfterMs = 0
	}
	return decision
}

func round6(value float64) float64 {
	return math.Round(value*1_000_000) / 1_000_000
}

func boolCost(ok bool, cost float64) float64 {
	if ok {
		return cost
	}
	return 0
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
