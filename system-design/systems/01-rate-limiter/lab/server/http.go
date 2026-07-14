package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"io/fs"
	"math"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"rubickx/system-design/systems/01-rate-limiter/lab/server/internal/webui"
)

type AppConfig struct {
	LabRoot       string
	Runners       *RunnerRegistry
	Debug         *DelveSessionManager
	MemoryStore   FixedWindowStore
	RedisStore    FixedWindowStore
	RedisAddr     string
	DemoLimit     int
	DemoWindow    time.Duration
	FailurePolicy string
	Now           func() time.Time
}

type App struct {
	config        AppConfig
	mux           *http.ServeMux
	runners       *RunnerRegistry
	debug         *DelveSessionManager
	memory        FixedWindowStore
	redis         FixedWindowStore
	localReplicas map[string]FixedWindowStore
	nextRun       atomic.Uint64
}

func NewApp(config AppConfig) *App {
	if config.LabRoot == "" {
		config.LabRoot, _ = LabRoot()
	}
	if config.DemoLimit <= 0 {
		config.DemoLimit = 5
	}
	if config.DemoWindow <= 0 {
		config.DemoWindow = time.Second
	}
	if config.FailurePolicy == "" {
		config.FailurePolicy = "fail-open"
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.Runners == nil {
		config.Runners = NewRunnerRegistry(config.LabRoot)
	}
	if config.Debug == nil {
		config.Debug = NewDelveSessionManager(config.LabRoot)
	}
	if config.MemoryStore == nil {
		config.MemoryStore = NewMemoryFixedWindowStore()
	}
	if config.RedisStore == nil && config.RedisAddr != "" {
		config.RedisStore = NewRedisFixedWindowStore(config.RedisAddr, "rate-limiter-lab:")
	}
	app := &App{
		config: config, mux: http.NewServeMux(), runners: config.Runners, debug: config.Debug,
		memory: config.MemoryStore, redis: config.RedisStore,
		localReplicas: map[string]FixedWindowStore{"a": NewMemoryFixedWindowStore(), "b": NewMemoryFixedWindowStore()},
	}
	app.routes()
	return app
}

func (a *App) routes() {
	a.mux.HandleFunc("/api/health", onlyMethod(http.MethodGet, a.handleHealth))
	a.mux.HandleFunc("/api/catalog", onlyMethod(http.MethodGet, a.handleCatalog))
	a.mux.HandleFunc("/api/runs", onlyMethod(http.MethodPost, a.handleRun))
	a.mux.HandleFunc("/api/debug/sessions", onlyMethod(http.MethodPost, a.handleCreateDebugSession))
	a.mux.HandleFunc("/api/debug/sessions/", a.handleDebugSessionPath)
	a.mux.HandleFunc("/demo/", a.handleDemo)
	assets := webui.FS()
	fileServer := http.FileServer(http.FS(assets))
	a.mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		path := strings.TrimPrefix(request.URL.Path, "/")
		if path != "" {
			if _, err := fs.Stat(assets, path); err == nil {
				fileServer.ServeHTTP(writer, request)
				return
			}
		}
		request.URL.Path = "/"
		fileServer.ServeHTTP(writer, request)
	})
}

func onlyMethod(method string, handler http.HandlerFunc) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != method {
			writer.Header().Set("Allow", method)
			writeAPIError(writer, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		handler(writer, request)
	}
}

func (a *App) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	a.mux.ServeHTTP(writer, request)
}

func (a *App) Close() error {
	if a.debug != nil {
		return a.debug.CloseAll()
	}
	return nil
}

func (a *App) handleHealth(writer http.ResponseWriter, _ *http.Request) {
	_, dlvErr := exec.LookPath("dlv")
	writeJSON(writer, http.StatusOK, map[string]any{
		"status":          "ok",
		"redisConfigured": a.redis != nil,
		"debugAvailable":  dlvErr == nil,
	})
}

func (a *App) handleCatalog(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, DefaultCatalog())
}

func (a *App) handleRun(writer http.ResponseWriter, request *http.Request) {
	var runRequest RunRequest
	if err := decodeJSON(request, &runRequest); err != nil {
		writeAPIError(writer, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if len(runRequest.RequestTimeline) > 100 {
		writeAPIError(writer, http.StatusBadRequest, "invalid_request", "requestTimeline must contain at most 100 items")
		return
	}
	if runRequest.Language == "" {
		runRequest.Language = LanguageGo
	}
	if scenario, ok := catalogScenario(runRequest.ScenarioID); ok && scenario.Tier == "system" && runRequest.Language != LanguageGo {
		writeAPIError(writer, http.StatusBadRequest, "go_required", "system scenarios are implemented by the Go end-to-end path")
		return
	}
	runner, err := a.runners.Runner(runRequest.Language)
	if err != nil {
		writeAPIError(writer, http.StatusBadRequest, "unsupported_language", err.Error())
		return
	}
	response, err := runner.Run(request.Context(), runRequest)
	if err != nil {
		writeAPIError(writer, http.StatusBadRequest, "run_failed", err.Error())
		return
	}
	response.RunID = fmt.Sprintf("run-%d", a.nextRun.Add(1))
	response.Language = runRequest.Language
	response.Algorithm = runRequest.Algorithm
	writeJSON(writer, http.StatusOK, response)
}

func catalogScenario(id string) (CatalogItem, bool) {
	for _, scenario := range DefaultCatalog().Scenarios {
		if scenario.ID == id {
			return scenario, true
		}
	}
	return CatalogItem{}, false
}

func (a *App) handleDemo(writer http.ResponseWriter, request *http.Request) {
	endpoint := strings.TrimPrefix(request.URL.Path, "/demo/")
	if endpoint == "" || len(endpoint) > 128 || strings.Contains(endpoint, "..") {
		writeAPIError(writer, http.StatusNotFound, "not_found", "demo endpoint not found")
		return
	}
	key := request.Header.Get("X-RateLimit-Key")
	if key == "" {
		key = "anonymous"
	}
	if len(key) > 128 {
		writeAPIError(writer, http.StatusBadRequest, "invalid_key", "X-RateLimit-Key must contain at most 128 bytes")
		return
	}
	storeName := request.URL.Query().Get("store")
	if storeName == "" {
		storeName = "memory"
	}
	store := a.memory
	if storeName == "redis" {
		store = a.redis
	} else if storeName != "memory" {
		writeAPIError(writer, http.StatusBadRequest, "invalid_store", "store must be memory or redis")
		return
	}
	replica := ""
	scope := ""
	if endpoint == "local-vs-shared" {
		replica = request.URL.Query().Get("replica")
		if replica == "" {
			replica = "a"
		}
		if replica != "a" && replica != "b" {
			writeAPIError(writer, http.StatusBadRequest, "invalid_replica", "replica must be a or b")
			return
		}
		if storeName == "memory" {
			store = a.localReplicas[replica]
			scope = "replica-local"
		} else {
			scope = "shared"
		}
	}
	metadata := map[string]any{}
	if endpoint == "local-vs-shared" {
		metadata["replica"] = replica
		metadata["scope"] = scope
	}
	if endpoint == "hot-key-sharding" {
		metadata["routing"] = map[string]any{"strategy": "hash(key)%4", "shard": shardForKey(key), "key": key}
	}
	limit, window, err := demoOverrides(request, a.config.DemoLimit, a.config.DemoWindow)
	if err != nil {
		writeAPIError(writer, http.StatusBadRequest, "invalid_demo_config", err.Error())
		return
	}
	policy, err := failurePolicy(request.URL.Query().Get("failure"), a.config.FailurePolicy)
	if err != nil {
		writeAPIError(writer, http.StatusBadRequest, "invalid_failure_policy", err.Error())
		return
	}
	now := a.config.Now()
	if store == nil {
		a.handleStoreFailure(writer, fmt.Errorf("%s store is not configured", storeName), policy, limit, now, metadata)
		return
	}
	if endpoint == "policy-composition" {
		a.handlePolicyComposition(writer, request, store, storeName, key, limit, window, policy, now)
		return
	}
	decision, err := store.Allow(request.Context(), endpoint+":"+key, limit, window, 1, now)
	if err != nil {
		a.handleStoreFailure(writer, err, policy, limit, now, metadata)
		return
	}
	setRateLimitHeaders(writer.Header(), limit, decision, now)
	writer.Header().Set("RateLimit-Policy", fmt.Sprintf("%d;w=%d", limit, maxInt64(1, int64(math.Ceil(window.Seconds())))))
	status := http.StatusOK
	if !decision.Allowed {
		status = http.StatusTooManyRequests
	}
	response := map[string]any{"endpoint": endpoint, "key": key, "store": storeName, "decision": decision}
	for name, value := range metadata {
		response[name] = value
	}
	writeJSON(writer, status, response)
}

func shardForKey(key string) uint32 {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(key))
	return hash.Sum32() % 4
}

type policyResult struct {
	ID       string   `json:"id"`
	Limit    int      `json:"limit"`
	Decision Decision `json:"decision"`
}

func (a *App) handlePolicyComposition(writer http.ResponseWriter, request *http.Request, store FixedWindowStore, storeName, key string, clientLimit int, window time.Duration, failurePolicy string, now time.Time) {
	endpointLimit := clientLimit * 2
	endpointDecision, err := store.Allow(request.Context(), "policy:endpoint:policy-composition", endpointLimit, window, 1, now)
	if err != nil {
		a.handleStoreFailure(writer, err, failurePolicy, endpointLimit, now)
		return
	}
	clientDecision, err := store.Allow(request.Context(), "policy:client:policy-composition:"+key, clientLimit, window, 1, now)
	if err != nil {
		a.handleStoreFailure(writer, err, failurePolicy, clientLimit, now)
		return
	}
	policies := []policyResult{
		{ID: "endpoint-wide", Limit: endpointLimit, Decision: endpointDecision},
		{ID: "per-client", Limit: clientLimit, Decision: clientDecision},
	}
	rejectedBy := []string{}
	selectedLimit := clientLimit
	final := clientDecision
	if !endpointDecision.Allowed {
		rejectedBy = append(rejectedBy, "endpoint-wide")
		selectedLimit, final = endpointLimit, endpointDecision
	}
	if !clientDecision.Allowed {
		rejectedBy = append(rejectedBy, "per-client")
		if final.Allowed || clientDecision.RetryAfterMs > final.RetryAfterMs {
			selectedLimit, final = clientLimit, clientDecision
		}
	}
	status := http.StatusOK
	if len(rejectedBy) == 0 {
		final.Reason = "all_policies_allowed"
	} else {
		status = http.StatusTooManyRequests
		final.Allowed = false
		final.Reason = "policy_rejected:" + strings.Join(rejectedBy, ",")
	}
	setRateLimitHeaders(writer.Header(), selectedLimit, final, now)
	windowSeconds := maxInt64(1, int64(math.Ceil(window.Seconds())))
	writer.Header().Set("RateLimit-Policy", fmt.Sprintf("%d;w=%d, %d;w=%d", clientLimit, windowSeconds, endpointLimit, windowSeconds))
	writeJSON(writer, status, map[string]any{
		"endpoint": "policy-composition", "key": key, "store": storeName,
		"decision": final, "policies": policies, "rejectedBy": rejectedBy,
	})
}

func (a *App) handleStoreFailure(writer http.ResponseWriter, storeErr error, policy string, limit int, now time.Time, metadata ...map[string]any) {
	writer.Header().Set("X-RateLimit-Degraded", "true")
	if policy == "fail-open" {
		decision := Decision{Allowed: true, Remaining: float64(limit), Reason: "storage_unavailable_fail_open"}
		writer.Header().Set("RateLimit-Policy", "bypass")
		setRateLimitHeaders(writer.Header(), limit, decision, now)
		body := map[string]any{"degraded": true, "decision": decision}
		mergeMetadata(body, metadata)
		writeJSON(writer, http.StatusOK, body)
		return
	}
	decision := Decision{Allowed: false, Remaining: 0, Reason: "storage_unavailable_fail_closed"}
	writer.Header().Set("RateLimit-Policy", "enforced")
	setRateLimitHeaders(writer.Header(), limit, decision, now)
	body := map[string]any{
		"error":    map[string]string{"code": "rate_limit_store_unavailable", "message": storeErr.Error()},
		"decision": decision,
	}
	mergeMetadata(body, metadata)
	writeJSON(writer, http.StatusServiceUnavailable, body)
}

func mergeMetadata(body map[string]any, values []map[string]any) {
	if len(values) == 0 {
		return
	}
	for name, value := range values[0] {
		body[name] = value
	}
}

func demoOverrides(request *http.Request, defaultLimit int, defaultWindow time.Duration) (int, time.Duration, error) {
	limit := defaultLimit
	window := defaultWindow
	if raw := request.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 10_000 {
			return 0, 0, fmt.Errorf("limit must be an integer between 1 and 10000")
		}
		limit = parsed
	}
	if raw := request.URL.Query().Get("window_ms"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 3_600_000 {
			return 0, 0, fmt.Errorf("window_ms must be an integer between 1 and 3600000")
		}
		window = time.Duration(parsed) * time.Millisecond
	}
	return limit, window, nil
}

func failurePolicy(value, fallback string) (string, error) {
	if value == "" {
		value = fallback
	}
	switch value {
	case "fail-open", "open":
		return "fail-open", nil
	case "fail-closed", "closed":
		return "fail-closed", nil
	default:
		return "", fmt.Errorf("failure must be fail-open or fail-closed")
	}
}

func setRateLimitHeaders(header http.Header, limit int, decision Decision, now time.Time) {
	header.Set("RateLimit-Limit", strconv.Itoa(limit))
	header.Set("RateLimit-Remaining", strconv.FormatInt(int64(math.Floor(decision.Remaining)), 10))
	resetDelay := math.Max(0, decision.ResetAtMs-float64(now.UnixMilli()))
	header.Set("RateLimit-Reset", strconv.FormatInt(int64(math.Ceil(resetDelay/1000)), 10))
	header.Set("X-RateLimit-Limit", strconv.Itoa(limit))
	header.Set("X-RateLimit-Remaining", strconv.FormatInt(int64(math.Floor(decision.Remaining)), 10))
	header.Set("X-RateLimit-Reset", strconv.FormatInt(int64(math.Ceil(decision.ResetAtMs/1000)), 10))
	if !decision.Allowed && decision.RetryAfterMs > 0 {
		header.Set("Retry-After", strconv.FormatInt(maxInt64(1, int64(math.Ceil(decision.RetryAfterMs/1000))), 10))
	}
}

func decodeJSON(request *http.Request, value any) error {
	defer request.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("request body must contain exactly one JSON value")
	}
	return nil
}

func writeAPIError(writer http.ResponseWriter, status int, code, message string) {
	writeJSON(writer, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	payload, err := json.Marshal(value)
	if err != nil {
		status = http.StatusInternalServerError
		payload = []byte(`{"error":{"code":"response_encode_failed","message":"response contains a non-JSON value"}}`)
	}
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_, _ = writer.Write(append(payload, '\n'))
}
