package server

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRunAPIAndCatalog(t *testing.T) {
	t.Parallel()

	app := NewApp(AppConfig{})

	catalogRequest := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
	catalogRecorder := httptest.NewRecorder()
	app.ServeHTTP(catalogRecorder, catalogRequest)
	if catalogRecorder.Code != http.StatusOK {
		t.Fatalf("GET /api/catalog status = %d, want 200; body=%s", catalogRecorder.Code, catalogRecorder.Body.String())
	}
	var catalog Catalog
	if err := json.Unmarshal(catalogRecorder.Body.Bytes(), &catalog); err != nil {
		t.Fatalf("decode catalog: %v", err)
	}
	if len(catalog.Algorithms) != 5 || len(catalog.Languages) != 4 {
		t.Fatalf("catalog has %d algorithms and %d languages", len(catalog.Algorithms), len(catalog.Languages))
	}

	body, err := json.Marshal(RunRequest{
		ScenarioID:      "burst-capacity",
		Algorithm:       "token-bucket",
		Language:        "go",
		Config:          map[string]float64{"capacity": 2, "ratePerSecond": 1},
		RequestTimeline: points(0, 0, 0),
		StoreMode:       "memory",
	})
	if err != nil {
		t.Fatal(err)
	}
	runRequest := httptest.NewRequest(http.MethodPost, "/api/runs", bytes.NewReader(body))
	runRecorder := httptest.NewRecorder()
	app.ServeHTTP(runRecorder, runRequest)
	if runRecorder.Code != http.StatusOK {
		t.Fatalf("POST /api/runs status = %d, want 200; body=%s", runRecorder.Code, runRecorder.Body.String())
	}
	var response RunResponse
	if err := json.Unmarshal(runRecorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode run response: %v", err)
	}
	if response.RunID == "" || len(response.Events) == 0 || len(response.Decisions) != 3 {
		t.Fatalf("incomplete run response: %+v", response)
	}
}

func TestDemoUsesPerKeyFixedWindowAndReturnsRateLimitHeaders(t *testing.T) {
	t.Parallel()

	now := time.UnixMilli(10_100)
	app := NewApp(AppConfig{
		DemoLimit:  2,
		DemoWindow: time.Second,
		Now:        func() time.Time { return now },
	})

	for i, wantStatus := range []int{http.StatusOK, http.StatusOK, http.StatusTooManyRequests} {
		request := httptest.NewRequest(http.MethodGet, "/demo/search", nil)
		request.Header.Set("X-RateLimit-Key", "alice")
		recorder := httptest.NewRecorder()
		app.ServeHTTP(recorder, request)
		if recorder.Code != wantStatus {
			t.Fatalf("request %d status = %d, want %d; body=%s", i, recorder.Code, wantStatus, recorder.Body.String())
		}
		if recorder.Header().Get("RateLimit-Limit") != "2" || recorder.Header().Get("RateLimit-Remaining") == "" {
			t.Fatalf("request %d missing rate limit headers: %v", i, recorder.Header())
		}
		if wantStatus == http.StatusTooManyRequests && recorder.Header().Get("Retry-After") != "1" {
			t.Fatalf("Retry-After = %q, want 1", recorder.Header().Get("Retry-After"))
		}
	}

	other := httptest.NewRequest(http.MethodGet, "/demo/search", nil)
	other.Header.Set("X-RateLimit-Key", "bob")
	recorder := httptest.NewRecorder()
	app.ServeHTTP(recorder, other)
	if recorder.Code != http.StatusOK {
		t.Fatalf("independent key status = %d, want 200", recorder.Code)
	}
}

func TestDemoKeepsRetryAfterInTheFinalSubMillisecond(t *testing.T) {
	t.Parallel()
	now := time.Unix(1, 999_500_000)
	app := NewApp(AppConfig{DemoLimit: 1, DemoWindow: time.Second, Now: func() time.Time { return now }})

	first := httptest.NewRecorder()
	app.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/demo/search", nil))
	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d, want 200", first.Code)
	}

	denied := httptest.NewRecorder()
	app.ServeHTTP(denied, httptest.NewRequest(http.MethodGet, "/demo/search", nil))
	if denied.Code != http.StatusTooManyRequests || denied.Header().Get("Retry-After") != "1" {
		t.Fatalf("final sub-millisecond denial = %d headers=%v body=%s", denied.Code, denied.Header(), denied.Body.String())
	}
}

func TestRunAPIRejectsFractionalTimelineMilliseconds(t *testing.T) {
	app := NewApp(AppConfig{})
	body := `{"scenarioId":"burst-capacity","algorithm":"token-bucket","language":"go","config":{"capacity":2,"ratePerSecond":1},"requestTimeline":[{"atMs":0.5,"cost":1,"key":"alice"}]}`
	request := httptest.NewRequest(http.MethodPost, "/api/runs", bytes.NewBufferString(body))
	recorder := httptest.NewRecorder()
	app.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("fractional atMs status = %d, want 400; body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestRunAPIRejectsTimestampBeyondCrossLanguageSafeInteger(t *testing.T) {
	app := NewApp(AppConfig{})
	body := `{"algorithm":"token-bucket","language":"go","config":{"capacity":2,"ratePerSecond":1},"requestTimeline":[{"atMs":9007199254740992,"cost":1,"key":"alice"}]}`
	request := httptest.NewRequest(http.MethodPost, "/api/runs", bytes.NewBufferString(body))
	recorder := httptest.NewRecorder()
	app.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "safe integer") {
		t.Fatalf("unsafe atMs response = %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestRunAPIDistinguishesMissingAndEmptyTimeline(t *testing.T) {
	app := NewApp(AppConfig{})
	for name, body := range map[string]string{
		"missing": `{"algorithm":"token-bucket","language":"go","config":{"capacity":2,"ratePerSecond":1}}`,
		"null":    `{"algorithm":"token-bucket","language":"go","config":{"capacity":2,"ratePerSecond":1},"requestTimeline":null}`,
	} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/runs", bytes.NewBufferString(body))
			recorder := httptest.NewRecorder()
			app.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}
	empty := `{"algorithm":"token-bucket","language":"go","config":{"capacity":2,"ratePerSecond":1},"requestTimeline":[]}`
	request := httptest.NewRequest(http.MethodPost, "/api/runs", bytes.NewBufferString(empty))
	recorder := httptest.NewRecorder()
	app.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("explicit empty timeline status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestRunAPIRequiresTimelineCostAndKey(t *testing.T) {
	app := NewApp(AppConfig{})
	for name, body := range map[string]string{
		"missing cost": `{"algorithm":"token-bucket","language":"go","config":{"capacity":2,"ratePerSecond":1},"requestTimeline":[{"atMs":0,"key":"alice"}]}`,
		"missing key":  `{"algorithm":"token-bucket","language":"go","config":{"capacity":2,"ratePerSecond":1},"requestTimeline":[{"atMs":0,"cost":1}]}`,
		"empty key":    `{"algorithm":"token-bucket","language":"go","config":{"capacity":2,"ratePerSecond":1},"requestTimeline":[{"atMs":0,"cost":1,"key":""}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/runs", bytes.NewBufferString(body))
			recorder := httptest.NewRecorder()
			app.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestPolicyCompositionIdentifiesRejectingPolicy(t *testing.T) {
	now := time.UnixMilli(10_100)
	app := NewApp(AppConfig{DemoLimit: 2, DemoWindow: time.Second, Now: func() time.Time { return now }})
	requestPolicy := func(key string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodGet, "/demo/policy-composition?limit=2&window_ms=1000", nil)
		request.Header.Set("X-RateLimit-Key", key)
		recorder := httptest.NewRecorder()
		app.ServeHTTP(recorder, request)
		return recorder
	}

	_ = requestPolicy("alice")
	_ = requestPolicy("alice")
	clientRejected := requestPolicy("alice")
	if clientRejected.Code != http.StatusTooManyRequests || clientRejected.Header().Get("RateLimit-Limit") != "2" {
		t.Fatalf("client rejection status/headers = %d %v; body=%s", clientRejected.Code, clientRejected.Header(), clientRejected.Body.String())
	}
	var clientBody struct {
		Policies   []any    `json:"policies"`
		RejectedBy []string `json:"rejectedBy"`
	}
	if err := json.Unmarshal(clientRejected.Body.Bytes(), &clientBody); err != nil {
		t.Fatal(err)
	}
	if len(clientBody.Policies) != 2 || len(clientBody.RejectedBy) != 1 || clientBody.RejectedBy[0] != "per-client" {
		t.Fatalf("client rejection body = %s", clientRejected.Body.String())
	}

	_ = requestPolicy("bob")
	endpointRejected := requestPolicy("charlie")
	if endpointRejected.Code != http.StatusTooManyRequests || endpointRejected.Header().Get("RateLimit-Limit") != "4" {
		t.Fatalf("endpoint rejection status/headers = %d %v; body=%s", endpointRejected.Code, endpointRejected.Header(), endpointRejected.Body.String())
	}
	var endpointBody struct {
		RejectedBy []string `json:"rejectedBy"`
	}
	if err := json.Unmarshal(endpointRejected.Body.Bytes(), &endpointBody); err != nil {
		t.Fatal(err)
	}
	if len(endpointBody.RejectedBy) != 1 || endpointBody.RejectedBy[0] != "endpoint-wide" {
		t.Fatalf("endpoint rejection body = %s", endpointRejected.Body.String())
	}
}

func TestDemoValidatesOverridesAndExposesFailurePolicy(t *testing.T) {
	app := NewApp(AppConfig{DemoLimit: 2, DemoWindow: time.Second, Now: func() time.Time { return time.UnixMilli(10_100) }})
	for _, path := range []string{
		"/demo/search?store=unknown",
		"/demo/search?limit=0",
		"/demo/search?window_ms=3600001",
		"/demo/search?failure=unknown",
	} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		recorder := httptest.NewRecorder()
		app.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("GET %s status = %d, want 400", path, recorder.Code)
		}
	}

	open := httptest.NewRequest(http.MethodGet, "/demo/search?store=redis&failure=fail-open", nil)
	openRecorder := httptest.NewRecorder()
	app.ServeHTTP(openRecorder, open)
	if openRecorder.Code != http.StatusOK || openRecorder.Header().Get("RateLimit-Policy") != "bypass" || openRecorder.Header().Get("X-RateLimit-Degraded") != "true" {
		t.Fatalf("fail-open response = %d %v %s", openRecorder.Code, openRecorder.Header(), openRecorder.Body.String())
	}

	closed := httptest.NewRequest(http.MethodGet, "/demo/search?store=redis&failure=fail-closed", nil)
	closedRecorder := httptest.NewRecorder()
	app.ServeHTTP(closedRecorder, closed)
	if closedRecorder.Code != http.StatusServiceUnavailable || closedRecorder.Header().Get("X-RateLimit-Degraded") != "true" || closedRecorder.Header().Get("RateLimit-Limit") != "2" {
		t.Fatalf("fail-closed response = %d %v %s", closedRecorder.Code, closedRecorder.Header(), closedRecorder.Body.String())
	}

	localOpen := httptest.NewRequest(http.MethodGet, "/demo/local-vs-shared?store=redis&failure=fail-open&replica=b", nil)
	localRecorder := httptest.NewRecorder()
	app.ServeHTTP(localRecorder, localOpen)
	var localBody struct {
		Replica string `json:"replica"`
		Scope   string `json:"scope"`
	}
	if err := json.Unmarshal(localRecorder.Body.Bytes(), &localBody); err != nil {
		t.Fatal(err)
	}
	if localRecorder.Code != http.StatusOK || localBody.Replica != "b" || localBody.Scope != "shared" {
		t.Fatalf("local fail-open response = %d %s", localRecorder.Code, localRecorder.Body.String())
	}
}

func TestLocalVsSharedMemoryReplicasHaveIndependentQuota(t *testing.T) {
	app := NewApp(AppConfig{DemoLimit: 1, DemoWindow: time.Second, Now: func() time.Time { return time.UnixMilli(10_100) }})
	requestReplica := func(replica string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodGet, "/demo/local-vs-shared?store=memory&replica="+replica, nil)
		request.Header.Set("X-RateLimit-Key", "alice")
		recorder := httptest.NewRecorder()
		app.ServeHTTP(recorder, request)
		return recorder
	}
	if first := requestReplica("a"); first.Code != http.StatusOK {
		t.Fatalf("replica a first status = %d; body=%s", first.Code, first.Body.String())
	}
	if denied := requestReplica("a"); denied.Code != http.StatusTooManyRequests {
		t.Fatalf("replica a second status = %d, want 429; body=%s", denied.Code, denied.Body.String())
	}
	independent := requestReplica("b")
	if independent.Code != http.StatusOK {
		t.Fatalf("replica b first status = %d, want independent 200; body=%s", independent.Code, independent.Body.String())
	}
	var body struct {
		Replica string `json:"replica"`
		Scope   string `json:"scope"`
	}
	if err := json.Unmarshal(independent.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Replica != "b" || body.Scope != "replica-local" {
		t.Fatalf("local-vs-shared body = %s", independent.Body.String())
	}

	invalid := requestReplica("c")
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid replica status = %d, want 400; body=%s", invalid.Code, invalid.Body.String())
	}
}

func TestLocalVsSharedRedisReplicasUseOneSharedQuota(t *testing.T) {
	shared := NewMemoryFixedWindowStore()
	app := NewApp(AppConfig{RedisStore: shared, DemoLimit: 1, DemoWindow: time.Second, Now: func() time.Time { return time.UnixMilli(10_100) }})
	requestReplica := func(replica string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodGet, "/demo/local-vs-shared?store=redis&replica="+replica, nil)
		request.Header.Set("X-RateLimit-Key", "alice")
		recorder := httptest.NewRecorder()
		app.ServeHTTP(recorder, request)
		return recorder
	}
	if first := requestReplica("a"); first.Code != http.StatusOK {
		t.Fatalf("replica a first status = %d; body=%s", first.Code, first.Body.String())
	}
	sharedDenial := requestReplica("b")
	if sharedDenial.Code != http.StatusTooManyRequests {
		t.Fatalf("replica b status = %d, want shared 429; body=%s", sharedDenial.Code, sharedDenial.Body.String())
	}
	var body struct {
		Replica string `json:"replica"`
		Scope   string `json:"scope"`
	}
	if err := json.Unmarshal(sharedDenial.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Replica != "b" || body.Scope != "shared" {
		t.Fatalf("shared body = %s", sharedDenial.Body.String())
	}
}

func TestHotKeyShardingReturnsStableRoutingEvidence(t *testing.T) {
	app := NewApp(AppConfig{DemoLimit: 10, DemoWindow: time.Second, Now: func() time.Time { return time.UnixMilli(10_100) }})
	type routingBody struct {
		Routing struct {
			Strategy string `json:"strategy"`
			Shard    uint32 `json:"shard"`
			Key      string `json:"key"`
		} `json:"routing"`
	}
	route := func(key string) routingBody {
		request := httptest.NewRequest(http.MethodGet, "/demo/hot-key-sharding", nil)
		request.Header.Set("X-RateLimit-Key", key)
		recorder := httptest.NewRecorder()
		app.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("route %q status = %d; body=%s", key, recorder.Code, recorder.Body.String())
		}
		var body routingBody
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return body
	}
	aliceFirst, aliceSecond, bob := route("alice"), route("alice"), route("bob")
	if aliceFirst.Routing.Strategy != "hash(key)%4" || aliceFirst.Routing.Key != "alice" || aliceFirst.Routing.Shard != 3 {
		t.Fatalf("alice routing = %+v", aliceFirst.Routing)
	}
	if aliceSecond.Routing.Shard != aliceFirst.Routing.Shard {
		t.Fatalf("same key changed shards: first=%d second=%d", aliceFirst.Routing.Shard, aliceSecond.Routing.Shard)
	}
	if bob.Routing.Shard != 0 || bob.Routing.Shard == aliceFirst.Routing.Shard {
		t.Fatalf("bob routing = %+v; alice shard=%d", bob.Routing, aliceFirst.Routing.Shard)
	}
}

func TestRunAPIRejectsUnsafeQuantities(t *testing.T) {
	app := NewApp(AppConfig{})
	bodies := []string{
		`{"algorithm":"fixed-window","language":"go","config":{"limit":1e308,"windowMs":1000},"requestTimeline":[{"atMs":0,"cost":1,"key":"alice"}]}`,
		`{"algorithm":"token-bucket","language":"go","config":{"capacity":1e308,"ratePerSecond":1},"requestTimeline":[{"atMs":0,"cost":1,"key":"alice"}]}`,
		`{"algorithm":"token-bucket","language":"go","config":{"capacity":1,"ratePerSecond":5e-324},"requestTimeline":[{"atMs":0,"cost":1,"key":"alice"}]}`,
		`{"algorithm":"token-bucket","language":"go","config":{"capacity":1,"ratePerSecond":1e308},"requestTimeline":[{"atMs":0,"cost":1,"key":"alice"}]}`,
		`{"algorithm":"token-bucket","language":"go","config":{"capacity":1,"ratePerSecond":1},"requestTimeline":[{"atMs":0,"cost":1e308,"key":"alice"}]}`,
		`{"algorithm":"token-bucket","language":"go","config":{"capacity":1,"ratePerSecond":1},"requestTimeline":[{"atMs":0,"cost":1,"key":"` + strings.Repeat("x", 129) + `"}]}`,
	}
	for index, body := range bodies {
		request := httptest.NewRequest(http.MethodPost, "/api/runs", bytes.NewBufferString(body))
		recorder := httptest.NewRecorder()
		app.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("unsafe quantity case %d status = %d, want 400; body=%s", index, recorder.Code, recorder.Body.String())
		}
	}
}

type nonFiniteRunner struct{}

func (nonFiniteRunner) Run(context.Context, RunRequest) (RunResponse, error) {
	return RunResponse{Events: []TraceEvent{}, Decisions: []Decision{{Allowed: true, Remaining: math.Inf(1)}}}, nil
}

func TestRunAPIReportsJSONEncodingFailureBeforeWritingSuccess(t *testing.T) {
	root, err := LabRoot()
	if err != nil {
		t.Fatal(err)
	}
	runners := NewRunnerRegistry(root)
	runners.Register(LanguageGo, nonFiniteRunner{})
	app := NewApp(AppConfig{Runners: runners})
	body := `{"algorithm":"token-bucket","language":"go","config":{"capacity":1,"ratePerSecond":1},"requestTimeline":[]}`
	request := httptest.NewRequest(http.MethodPost, "/api/runs", bytes.NewBufferString(body))
	recorder := httptest.NewRecorder()
	app.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusInternalServerError || !strings.Contains(recorder.Body.String(), "response_encode_failed") {
		t.Fatalf("encoding failure response = %d %s", recorder.Code, recorder.Body.String())
	}
}
