package browserlab

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
)

type memoryJar struct {
	mu      sync.Mutex
	cookies map[string]map[string]*http.Cookie
}

func newMemoryJar() *memoryJar {
	return &memoryJar{cookies: make(map[string]map[string]*http.Cookie)}
}

func (jar *memoryJar) SetCookies(target *url.URL, cookies []*http.Cookie) {
	jar.mu.Lock()
	defer jar.mu.Unlock()
	if jar.cookies[target.Host] == nil {
		jar.cookies[target.Host] = make(map[string]*http.Cookie)
	}
	for _, cookie := range cookies {
		if cookie.MaxAge < 0 {
			delete(jar.cookies[target.Host], cookie.Name)
			continue
		}
		copyCookie := *cookie
		jar.cookies[target.Host][cookie.Name] = &copyCookie
	}
}

func (jar *memoryJar) Cookies(target *url.URL) []*http.Cookie {
	jar.mu.Lock()
	defer jar.mu.Unlock()
	var result []*http.Cookie
	for _, cookie := range jar.cookies[target.Host] {
		copyCookie := *cookie
		result = append(result, &copyCookie)
	}
	return result
}

func TestBrowserLoginNegativeAndLifecycle(t *testing.T) {
	t.Parallel()
	server, err := Start()
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	client := &http.Client{Jar: newMemoryJar()}
	response, err := client.Get(server.URL + "/app")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), "ACCESS GRANTED") {
		t.Fatalf("login flow status=%d body=%s", response.StatusCode, body)
	}
	response, err = client.Get(server.URL + "/negative?case=all")
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(response.Body)
	_ = response.Body.Close()
	if strings.Contains(string(body), "FAIL(open)") || !strings.Contains(string(body), "kid=PASS(rejected)") {
		t.Fatalf("negative suite did not fail closed: %s", body)
	}
	_, _ = client.Get(server.URL + "/admin/posture?status=noncompliant")
	response, err = client.Get(server.URL + "/app")
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("non-compliant session status=%d", response.StatusCode)
	}
}

func TestCORSAndCSRFHeaders(t *testing.T) {
	t.Parallel()
	server, err := Start()
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	request, _ := http.NewRequest(http.MethodGet, server.OriginBURL+"/allowed", nil)
	request.Header.Set("Origin", server.URL)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.Header.Get("Access-Control-Allow-Origin") != server.URL {
		t.Fatal("exact CORS origin missing")
	}
	request, _ = http.NewRequest(http.MethodPost, server.URL+"/csrf", nil)
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("CSRF request status=%d", response.StatusCode)
	}
}

func TestBrowserEvidenceUsesExplicitLabAttribution(t *testing.T) {
	t.Parallel()
	server, err := Start()
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	client := &http.Client{}
	for _, target := range []string{
		server.URL + "/negative?case=state",
		server.URL + "/admin/deprovision",
		server.URL + "/admin/posture?status=noncompliant",
	} {
		response, err := client.Get(target)
		if err != nil {
			t.Fatal(err)
		}
		_ = response.Body.Close()
	}
	request, _ := http.NewRequest(http.MethodPost, server.URL+"/csrf", nil)
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	seen := make(map[string]bool)
	for _, event := range server.Evidence() {
		seen[event.LabID] = true
	}
	for _, labID := range []string{"LAB-NETSEC-05", "LAB-NETSEC-07", "LAB-NETSEC-08", "LAB-NETSEC-09"} {
		if !seen[labID] {
			t.Errorf("missing browser evidence attribution for %s: %#v", labID, seen)
		}
	}
}

func TestEvidenceFixtureProducesV2Attribution(t *testing.T) {
	t.Parallel()
	server, err := Start()
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	if err := RunEvidenceFixture(context.Background(), server); err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	for _, event := range server.Evidence() {
		seen[event.LabID] = true
		if event.ScenarioID == "" || event.Stage == "" || event.Component == "" || event.EvidenceKind == "" || event.ObservedState == "" || event.ActionID == "" || event.PreconditionRevision == "" {
			t.Fatalf("incomplete browser v2 event: %#v", event)
		}
	}
	for _, labID := range []string{"LAB-NETSEC-05", "LAB-NETSEC-07", "LAB-NETSEC-08", "LAB-NETSEC-09"} {
		if !seen[labID] {
			t.Errorf("fixture missing %s: %#v", labID, seen)
		}
	}
}
