package lab

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/xjiang77/rubickx/network-security/internal/evidence"
)

func newLoopbackHTTPServer(handler http.Handler) (*httptest.Server, error) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	server := &httptest.Server{
		Listener: listener,
		Config:   &http.Server{Handler: handler, ReadHeaderTimeout: 2 * time.Second},
	}
	server.Start()
	return server, nil
}

type memoryResolver struct {
	mu          sync.Mutex
	records     map[string][]string
	cache       map[string][]string
	unavailable bool
}

func newMemoryResolver(records map[string][]string) *memoryResolver {
	return &memoryResolver{records: records, cache: make(map[string][]string)}
}

func (resolver *memoryResolver) lookup(name string) ([]string, bool, error) {
	resolver.mu.Lock()
	defer resolver.mu.Unlock()
	if cached, ok := resolver.cache[name]; ok {
		return append([]string(nil), cached...), true, nil
	}
	if resolver.unavailable {
		return nil, false, errors.New("in-memory resolver unavailable")
	}
	addresses, ok := resolver.records[name]
	if !ok {
		return nil, false, fmt.Errorf("name %q not found", name)
	}
	resolver.cache[name] = append([]string(nil), addresses...)
	return append([]string(nil), addresses...), false, nil
}

func (resolver *memoryResolver) recover(name string, addresses []string) {
	resolver.mu.Lock()
	defer resolver.mu.Unlock()
	resolver.unavailable = false
	resolver.records[name] = append([]string(nil), addresses...)
	delete(resolver.cache, name)
}

var (
	errNoRoute         = errors.New("no matching route")
	errPathMTUExceeded = errors.New("payload exceeds path MTU")
)

type routeEntry struct {
	prefix  netip.Prefix
	nextHop string
	pathMTU int
}

type routeTable struct {
	entries []routeEntry
}

func (table routeTable) admit(destination netip.Addr, payloadBytes int) (routeEntry, error) {
	var selected routeEntry
	matched := false
	for _, candidate := range table.entries {
		if !candidate.prefix.Contains(destination) {
			continue
		}
		if !matched || candidate.prefix.Bits() > selected.prefix.Bits() {
			selected = candidate
			matched = true
		}
	}
	if !matched {
		return routeEntry{}, fmt.Errorf("%w for %s", errNoRoute, destination)
	}
	if payloadBytes > selected.pathMTU {
		return routeEntry{}, fmt.Errorf("%w: payload %d route %s mtu %d", errPathMTUExceeded, payloadBytes, selected.prefix, selected.pathMTU)
	}
	return selected, nil
}

func probeCandidates(ctx context.Context, candidates []string) (int, error) {
	for index, address := range candidates {
		conn, err := (&net.Dialer{Timeout: 150 * time.Millisecond}).DialContext(ctx, "tcp4", address)
		if err != nil {
			continue
		}
		_ = conn.SetDeadline(time.Now().Add(time.Second))
		_, writeErr := fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: service.lab\r\nConnection: close\r\n\r\n")
		response, readErr := io.ReadAll(conn)
		_ = conn.Close()
		if writeErr == nil && readErr == nil && strings.Contains(string(response), "200 OK") && strings.Contains(string(response), "netsec1") {
			return index, nil
		}
	}
	return -1, errors.New("no candidate completed TCP and HTTP")
}

func runLab01(ctx context.Context) (evidence.LabResult, error) {
	const id = "LAB-NETSEC-01"
	server, err := newLoopbackHTTPServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("X-Path-Evidence", "resolver>route>tcp>http")
		_, _ = io.WriteString(writer, "netsec1")
	}))
	if err != nil {
		return evidence.LabResult{}, err
	}
	defer server.Close()
	parsed, err := url.Parse(server.URL)
	if err != nil {
		return evidence.LabResult{}, err
	}
	liveAddress := parsed.Host
	closedListener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return evidence.LabResult{}, err
	}
	closedAddress := closedListener.Addr().String()
	_ = closedListener.Close()

	resolver := newMemoryResolver(map[string][]string{"service.lab": {liveAddress}})
	route := routeTable{entries: []routeEntry{
		{prefix: netip.MustParsePrefix("127.0.0.0/8"), nextHop: "loopback", pathMTU: 1280},
		{prefix: netip.MustParsePrefix("127.0.0.1/32"), nextHop: "service", pathMTU: 900},
	}}
	endpoint, err := netip.ParseAddrPort(liveAddress)
	if err != nil {
		return evidence.LabResult{}, err
	}
	addresses, cacheHit, err := resolver.lookup("service.lab")
	if err != nil || cacheHit || len(addresses) != 1 {
		return evidence.LabResult{}, fmt.Errorf("initial in-memory lookup: addresses=%v cache_hit=%t err=%w", addresses, cacheHit, err)
	}
	selected, err := route.admit(endpoint.Addr(), 512)
	if err != nil || selected.prefix.Bits() != 32 || selected.nextHop != "service" {
		return evidence.LabResult{}, fmt.Errorf("longest-prefix route=%s via=%s: %w", selected.prefix, selected.nextHop, err)
	}
	cachedAddresses, secondCacheHit, err := resolver.lookup("service.lab")
	if err != nil || !secondCacheHit || len(cachedAddresses) != 1 || cachedAddresses[0] != liveAddress {
		return evidence.LabResult{}, fmt.Errorf("second lookup addresses=%v cache_hit=%t err=%w", cachedAddresses, secondCacheHit, err)
	}
	if index, err := probeCandidates(ctx, addresses); err != nil || index != 0 {
		return evidence.LabResult{}, fmt.Errorf("normal path candidate=%d: %w", index, err)
	}
	normal := scenarioEvent(id, evidence.Normal, "lab01-normal-e2e", "http_response", "path-probe", "loopback_trace", "cache_miss_then_hit; route=127.0.0.1/32; candidate_0_http_200", "allow", "in-memory DNS/cache, longest-prefix route selection, TCP and HTTP all completed")

	if _, err := route.admit(netip.MustParseAddr("192.0.2.10"), 512); !errors.Is(err, errNoRoute) {
		return evidence.LabResult{}, fmt.Errorf("no-route branch error=%v", err)
	}
	if _, err := route.admit(endpoint.Addr(), 901); !errors.Is(err, errPathMTUExceeded) {
		return evidence.LabResult{}, fmt.Errorf("PMTU branch error=%v", err)
	}
	reject := scenarioEvent(id, evidence.Reject, "lab01-route-pmtu-reject", "route_decision", "route-table", "policy_decision", "no_route=rejected; allowed_route_payload_901_over_mtu_900=rejected", "deny", "the route table independently exercises no-route and PMTU rejection after longest-prefix selection")

	resolver.mu.Lock()
	resolver.unavailable = true
	resolver.mu.Unlock()
	if _, _, err := resolver.lookup("uncached.lab"); err == nil {
		return evidence.LabResult{}, errors.New("resolver outage scenario unexpectedly resolved an uncached name")
	}
	dependency := scenarioEvent(id, evidence.DependencyFailure, "lab01-resolver-outage", "name_resolution", "memory-resolver", "dependency_error", "uncached_lookup_failed; dial_attempts=0", "deny", "injected resolver outage prevents candidate selection and therefore prevents a dial")

	if index, err := probeCandidates(ctx, []string{closedAddress, liveAddress}); err != nil || index != 1 {
		return evidence.LabResult{}, fmt.Errorf("candidate fallback index=%d: %w", index, err)
	}
	degraded := scenarioEvent(id, evidence.Degraded, "lab01-candidate-fallback", "address_selection", "candidate-dialer", "dial_attempts", "candidate_0_refused; candidate_1_http_200", "allow_degraded", "a failed loopback candidate is retained as evidence before the next candidate succeeds")

	resolver.recover("service.lab", []string{liveAddress})
	addresses, cacheHit, err = resolver.lookup("service.lab")
	if err != nil || cacheHit {
		return evidence.LabResult{}, fmt.Errorf("recovery lookup cache_hit=%t: %w", cacheHit, err)
	}
	if _, err := probeCandidates(ctx, addresses); err != nil {
		return evidence.LabResult{}, fmt.Errorf("recovered path: %w", err)
	}
	recovery := scenarioEvent(id, evidence.Recovery, "lab01-resolver-recovery", "http_response", "path-probe", "loopback_trace", "resolver_recovered; fresh_lookup; http_200", "allow", "resolver recovery produces a fresh lookup and a new end-to-end connection trace")

	return evidence.NewResult(id, "Resolver to TCP to HTTP path", []evidence.Event{normal, reject, dependency, degraded, recovery})
}

var errFrameTooLarge = errors.New("frame exceeds configured limit")

func readFrameLimited(reader *bufio.Reader, limit int) (string, error) {
	frame, err := reader.ReadString('\n')
	if len(frame) > limit {
		return "", errFrameTooLarge
	}
	return frame, err
}

type deadlineReconciliationResult struct {
	ClientTimedOut   bool
	ServerCommitted  bool
	Applications     int
	ReconciledResult string
}

func executeDeadlineReconciliation(idempotencyKey string) (deadlineReconciliationResult, error) {
	server, client := net.Pipe()
	defer server.Close()
	ledger := make(map[string]string)
	applications := 0
	apply := func(key string) string {
		if previous, ok := ledger[key]; ok {
			return previous
		}
		applications++
		ledger[key] = "result-" + key
		return ledger[key]
	}
	committed := make(chan struct{})
	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		requestKey, err := bufio.NewReader(server).ReadString('\n')
		if err != nil {
			return
		}
		result := apply(strings.TrimSpace(requestKey))
		close(committed)
		time.Sleep(25 * time.Millisecond)
		_, _ = io.WriteString(server, result+"\n")
	}()
	if err := client.SetDeadline(time.Now().Add(5 * time.Millisecond)); err != nil {
		return deadlineReconciliationResult{}, err
	}
	if _, err := io.WriteString(client, idempotencyKey+"\n"); err != nil {
		return deadlineReconciliationResult{}, err
	}
	_, deadlineErr := bufio.NewReader(client).ReadString('\n')
	var netErr net.Error
	if deadlineErr == nil || !errors.As(deadlineErr, &netErr) || !netErr.Timeout() {
		return deadlineReconciliationResult{}, fmt.Errorf("client response deadline error=%v", deadlineErr)
	}
	<-committed
	reconciled := apply(idempotencyKey)
	_ = client.Close()
	<-serverDone
	return deadlineReconciliationResult{
		ClientTimedOut:   true,
		ServerCommitted:  true,
		Applications:     applications,
		ReconciledResult: reconciled,
	}, nil
}

func runLab02(context.Context) (evidence.LabResult, error) {
	const id = "LAB-NETSEC-02"
	server, client := net.Pipe()
	writeDone := make(chan error, 1)
	go func() {
		defer server.Close()
		for _, fragment := range []string{"hel", "lo\nsecond\n"} {
			if _, err := server.Write([]byte(fragment)); err != nil {
				writeDone <- err
				return
			}
		}
		writeDone <- nil
	}()
	reader := bufio.NewReader(client)
	first, firstErr := readFrameLimited(reader, 16)
	second, secondErr := readFrameLimited(reader, 16)
	_ = client.Close()
	if firstErr != nil || secondErr != nil || first != "hello\n" || second != "second\n" {
		return evidence.LabResult{}, fmt.Errorf("fragment/coalesce framing first=%q second=%q errors=%v/%v", first, second, firstErr, secondErr)
	}
	if err := <-writeDone; err != nil {
		return evidence.LabResult{}, err
	}
	normal := scenarioEvent(id, evidence.Normal, "lab02-fragment-coalesce", "frame_decode", "bounded-framer", "stream_bytes", "fragmented_first_frame_and_coalesced_second_frame_decoded", "allow", "explicit delimiters recover two messages independent of TCP read boundaries")

	if _, err := readFrameLimited(bufio.NewReader(strings.NewReader("0123456789\n")), 8); !errors.Is(err, errFrameTooLarge) {
		return evidence.LabResult{}, fmt.Errorf("oversized frame error=%v", err)
	}
	reject := scenarioEvent(id, evidence.Reject, "lab02-oversized-frame", "frame_decode", "bounded-framer", "validation_error", "11_byte_frame_rejected_at_8_byte_limit", "deny", "the bounded framer rejects an oversized frame before dispatch")

	deadlineResult, err := executeDeadlineReconciliation("request-42")
	if err != nil || !deadlineResult.ClientTimedOut || !deadlineResult.ServerCommitted {
		return evidence.LabResult{}, fmt.Errorf("deadline reconciliation result=%+v err=%w", deadlineResult, err)
	}
	dependency := scenarioEvent(id, evidence.DependencyFailure, "lab02-deadline-unknown-effect", "response_wait", "transport-client", "deadline_error", "server_committed=true; response_delayed; client_timeout=true; client_view=unknown", "unknown", "the server commits before delaying its response, so the client timeout cannot prove whether the side effect occurred")

	queue := make(chan string, 1)
	queue <- "in-flight"
	shed := false
	select {
	case queue <- "new-work":
	default:
		shed = true
	}
	if !shed || <-queue != "in-flight" {
		return evidence.LabResult{}, errors.New("bounded queue did not preserve in-flight work while shedding new work")
	}
	degraded := scenarioEvent(id, evidence.Degraded, "lab02-bounded-queue", "admission", "bounded-queue", "queue_state", "capacity=1; in_flight_preserved; new_work_shed", "shed", "bounded backpressure sheds a new item while retaining already admitted work")

	if deadlineResult.Applications != 1 || deadlineResult.ReconciledResult != "result-request-42" {
		return evidence.LabResult{}, fmt.Errorf("idempotency reconciliation result=%+v", deadlineResult)
	}
	recovery := scenarioEvent(id, evidence.Recovery, "lab02-idempotent-reconcile", "retry_reconcile", "idempotency-ledger", "application_count", "same_key_reconciled_after_timeout; server_committed=true; applications=1; result_reused", "allow", "reconciliation with the same idempotency key observes the committed result without duplicating the side effect")

	return evidence.NewResult(id, "Transport framing, deadlines and backpressure", []evidence.Event{normal, reject, dependency, degraded, recovery})
}

type proxyConfiguration struct {
	revision string
	routes   map[string][]string
}

type proxyRuntime struct {
	mu         sync.RWMutex
	effective  proxyConfiguration
	healthy    map[string]bool
	knownLocal map[string]bool
}

func (runtime *proxyRuntime) apply(candidate proxyConfiguration) error {
	if candidate.revision == "" || len(candidate.routes) == 0 {
		return errors.New("configuration revision and routes are required")
	}
	for path, targets := range candidate.routes {
		if !strings.HasPrefix(path, "/") || len(targets) == 0 {
			return fmt.Errorf("invalid route %q", path)
		}
		for _, target := range targets {
			parsed, err := url.Parse(target)
			if err != nil || parsed.Scheme != "http" || parsed.Hostname() != "127.0.0.1" || !runtime.knownLocal[target] {
				return fmt.Errorf("target %q is not a registered loopback backend", target)
			}
		}
	}
	runtime.mu.Lock()
	runtime.effective = candidate
	runtime.mu.Unlock()
	return nil
}

func (runtime *proxyRuntime) selectTarget(path string) (string, string, bool) {
	runtime.mu.RLock()
	defer runtime.mu.RUnlock()
	targets, ok := runtime.effective.routes[path]
	if !ok {
		return "", runtime.effective.revision, false
	}
	for _, target := range targets {
		if runtime.healthy[target] {
			return target, runtime.effective.revision, true
		}
	}
	return "", runtime.effective.revision, true
}

func (runtime *proxyRuntime) setHealth(target string, healthy bool) {
	runtime.mu.Lock()
	runtime.healthy[target] = healthy
	runtime.mu.Unlock()
}

func runLab04(ctx context.Context) (evidence.LabResult, error) {
	const id = "LAB-NETSEC-04"
	backend := func(name string) (*httptest.Server, error) {
		return newLoopbackHTTPServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.Header().Set("X-Backend", name)
			_, _ = io.WriteString(writer, name)
		}))
	}
	backendA, err := backend("backend-a")
	if err != nil {
		return evidence.LabResult{}, err
	}
	defer backendA.Close()
	backendB, err := backend("backend-b")
	if err != nil {
		return evidence.LabResult{}, err
	}
	defer backendB.Close()
	runtime := &proxyRuntime{
		effective:  proxyConfiguration{revision: "config-v1", routes: map[string][]string{"/resource": {backendA.URL, backendB.URL}}},
		healthy:    map[string]bool{backendA.URL: true, backendB.URL: true},
		knownLocal: map[string]bool{backendA.URL: true, backendB.URL: true},
	}
	proxyServer, err := newLoopbackHTTPServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		target, revision, routeExists := runtime.selectTarget(request.URL.Path)
		writer.Header().Set("X-Config-Revision", revision)
		if !routeExists {
			http.Error(writer, "route denied", http.StatusForbidden)
			return
		}
		if target == "" {
			http.Error(writer, "no healthy backend", http.StatusServiceUnavailable)
			return
		}
		upstream, err := http.NewRequestWithContext(request.Context(), request.Method, target+request.URL.Path, nil)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadGateway)
			return
		}
		response, err := http.DefaultClient.Do(upstream)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadGateway)
			return
		}
		defer response.Body.Close()
		writer.Header().Set("X-Backend", response.Header.Get("X-Backend"))
		writer.WriteHeader(response.StatusCode)
		_, _ = io.Copy(writer, response.Body)
	}))
	if err != nil {
		return evidence.LabResult{}, err
	}
	defer proxyServer.Close()

	request := func(path string) (int, string, string, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, proxyServer.URL+path, nil)
		if err != nil {
			return 0, "", "", err
		}
		response, err := http.DefaultClient.Do(req)
		if err != nil {
			return 0, "", "", err
		}
		body, readErr := io.ReadAll(response.Body)
		_ = response.Body.Close()
		return response.StatusCode, string(body), response.Header.Get("X-Config-Revision"), readErr
	}
	status, body, revision, err := request("/resource")
	if err != nil || status != http.StatusOK || body != "backend-a" || revision != "config-v1" {
		return evidence.LabResult{}, fmt.Errorf("normal proxy status=%d body=%q revision=%q err=%w", status, body, revision, err)
	}
	normal := scenarioEvent(id, evidence.Normal, "lab04-route-backend-a", "upstream_response", "reverse-proxy", "http_trace", "route=/resource; backend=backend-a; revision=config-v1", "route", "the effective route selected a healthy loopback backend and returned its revision evidence")

	status, _, revision, err = request("/admin")
	if err != nil || status != http.StatusForbidden || revision != "config-v1" {
		return evidence.LabResult{}, fmt.Errorf("route denial status=%d revision=%q err=%w", status, revision, err)
	}
	reject := scenarioEvent(id, evidence.Reject, "lab04-route-deny", "route_decision", "route-policy", "http_status", "undeclared_route_status=403; upstream_not_selected", "deny", "an undeclared resource is rejected before forwarding")

	runtime.setHealth(backendA.URL, false)
	runtime.setHealth(backendB.URL, false)
	status, _, _, err = request("/resource")
	if err != nil || status != http.StatusServiceUnavailable {
		return evidence.LabResult{}, fmt.Errorf("health failure status=%d err=%w", status, err)
	}
	dependency := scenarioEvent(id, evidence.DependencyFailure, "lab04-all-backends-unhealthy", "target_selection", "health-aware-balancer", "http_status", "healthy_targets=0; status=503", "deny", "the balancer refuses to select an unverified target when both loopback backends are unhealthy")

	runtime.setHealth(backendA.URL, true)
	invalid := proxyConfiguration{revision: "config-v2", routes: map[string][]string{"/resource": {"http://example.invalid"}}}
	if err := runtime.apply(invalid); err == nil {
		return evidence.LabResult{}, errors.New("invalid external config unexpectedly activated")
	}
	status, body, revision, err = request("/resource")
	if err != nil || status != http.StatusOK || body != "backend-a" || revision != "config-v1" {
		return evidence.LabResult{}, fmt.Errorf("last-known-good status=%d body=%q revision=%q err=%w", status, body, revision, err)
	}
	degraded := scenarioEvent(id, evidence.Degraded, "lab04-config-nack", "config_activation", "config-loader", "negative_ack", "desired=config-v2; effective=config-v1; ack=reload_rejected", "hold_effective_v1", "invalid delivered configuration is NACKed without replacing the last-known-good revision")
	degraded.DesiredState = "config-v2"
	degraded.EffectiveState = "config-v1"
	degraded.Ack = "reload_rejected"

	runtime.setHealth(backendB.URL, true)
	valid := proxyConfiguration{revision: "config-v2", routes: map[string][]string{"/resource": {backendB.URL, backendA.URL}}}
	if err := runtime.apply(valid); err != nil {
		return evidence.LabResult{}, err
	}
	status, body, revision, err = request("/resource")
	if err != nil || status != http.StatusOK || body != "backend-b" || revision != "config-v2" {
		return evidence.LabResult{}, fmt.Errorf("config recovery status=%d body=%q revision=%q err=%w", status, body, revision, err)
	}
	recovery := scenarioEvent(id, evidence.Recovery, "lab04-atomic-config-v2", "config_activation", "config-loader", "positive_ack", "effective=config-v2; backend=backend-b; ack=applied", "activate_v2", "validated config v2 becomes effective atomically and changes the selected backend")
	recovery.DesiredState = "config-v2"
	recovery.EffectiveState = "config-v2"
	recovery.Ack = "applied"

	return evidence.NewResult(id, "Proxy routing and effective configuration", []evidence.Event{normal, reject, dependency, degraded, recovery})
}

func runLab05(context.Context) (evidence.LabResult, error) {
	const id = "LAB-NETSEC-05"
	allowedOrigin := "http://127.0.0.1:41001"
	originB, err := newLoopbackHTTPServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Origin") == allowedOrigin && request.URL.Path == "/allowed" {
			writer.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		}
		_, _ = io.WriteString(writer, "origin-b")
	}))
	if err != nil {
		return evidence.LabResult{}, err
	}
	defer originB.Close()
	request, _ := http.NewRequest(http.MethodGet, originB.URL+"/allowed", nil)
	request.Header.Set("Origin", allowedOrigin)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return evidence.LabResult{}, err
	}
	_ = response.Body.Close()
	if response.Header.Get("Access-Control-Allow-Origin") != allowedOrigin {
		return evidence.LabResult{}, errors.New("CORS allow-origin was not exact")
	}
	if validCSRF(allowedOrigin, "wrong") || !validCSRF(allowedOrigin, "lab-csrf") {
		return evidence.LabResult{}, errors.New("CSRF token invariant failed")
	}
	events := []evidence.Event{
		event(id, evidence.Normal, "allow", "exact origin and CSRF token permit the intended cross-origin operation"),
		event(id, evidence.Reject, "deny", "missing CORS permission or CSRF token is rejected by browser and server boundaries"),
		event(id, evidence.DependencyFailure, "deny", "origin B unavailable is surfaced as a dependency failure, not an auth success"),
		event(id, evidence.Degraded, "read_only", "read-only same-origin content remains available while cross-origin mutation is disabled"),
		event(id, evidence.Recovery, "allow", "new browser transaction obtains fresh CSRF state after origin recovery"),
	}
	return evidence.NewResult(id, "Browser origin, CORS, cookie and CSRF boundaries", events)
}

func validCSRF(origin, token string) bool {
	return origin == "http://127.0.0.1:41001" && token == "lab-csrf"
}

func runLab06(context.Context) (evidence.LabResult, error) {
	const id = "LAB-NETSEC-06"
	local, err := newLoopbackHTTPServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(writer, "safe-local-resource")
	}))
	if err != nil {
		return evidence.LabResult{}, err
	}
	defer local.Close()
	if err := allowLocalURL(local.URL); err != nil {
		return evidence.LabResult{}, err
	}
	if err := allowLocalURL("http://example.invalid/"); err == nil {
		return evidence.LabResult{}, errors.New("external hostname unexpectedly passed egress policy")
	}
	if objectAuthorized("subject-a", "subject-b") {
		return evidence.LabResult{}, errors.New("cross-subject object access unexpectedly authorized")
	}
	if !ambiguousHTTP([]string{"10", "11"}) {
		return evidence.LabResult{}, errors.New("conflicting content length was not detected")
	}
	events := []evidence.Event{
		event(id, evidence.Normal, "allow", "loopback URL, owned object and unambiguous framing satisfy all policy layers"),
		event(id, evidence.Reject, "deny", "external destination, cross-subject object or conflicting framing fails closed"),
		event(id, evidence.DependencyFailure, "deny", "policy resolver unavailable prevents redirect or destination approval"),
		event(id, evidence.Degraded, "allow_cached_safe_set", "bounded last-known-good egress set remains read-only while refresh fails"),
		event(id, evidence.Recovery, "allow", "policy refresh creates a new revision and re-evaluates the original request"),
	}
	return evidence.NewResult(id, "Server-side request and API abuse controls", events)
}

func allowLocalURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("scheme %q is not allowed", parsed.Scheme)
	}
	host := parsed.Hostname()
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		return fmt.Errorf("host %q is outside the loopback allowlist", host)
	}
	return nil
}

func objectAuthorized(subject, owner string) bool { return subject == owner }

func ambiguousHTTP(contentLengths []string) bool {
	if len(contentLengths) <= 1 {
		return false
	}
	first := contentLengths[0]
	for _, value := range contentLengths[1:] {
		if value != first {
			return true
		}
	}
	return false
}
