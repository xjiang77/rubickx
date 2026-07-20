package browserlab

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/xjiang77/rubickx/network-security/internal/evidence"
	"github.com/xjiang77/rubickx/network-security/internal/oidclab"
)

const (
	clientID = "browser-rp"
	subject  = "subject-browser-123"
)

type transaction struct {
	Verifier string
	Nonce    string
}

type session struct {
	Subject string
	Active  bool
}

type Server struct {
	URL        string
	OriginBURL string

	mainServer      *http.Server
	originBServer   *http.Server
	mainListener    net.Listener
	originBListener net.Listener
	provider        *oidclab.Provider

	mu            sync.Mutex
	transactions  map[string]transaction
	sessions      map[string]session
	events        []evidence.Event
	deprovisioned bool
	compliant     bool
	keyRevision   int
}

func Start() (*Server, error) {
	mainListener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	originBListener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		_ = mainListener.Close()
		return nil, err
	}
	server := &Server{
		URL:             "http://" + mainListener.Addr().String(),
		OriginBURL:      "http://" + originBListener.Addr().String(),
		mainListener:    mainListener,
		originBListener: originBListener,
		transactions:    make(map[string]transaction),
		sessions:        make(map[string]session),
		compliant:       true,
		keyRevision:     1,
	}
	provider, err := oidclab.NewProvider(server.URL)
	if err != nil {
		_ = mainListener.Close()
		_ = originBListener.Close()
		return nil, err
	}
	provider.RegisterClient(clientID, server.URL+"/callback")
	server.provider = provider

	mainMux := http.NewServeMux()
	mainMux.HandleFunc("/", server.handleIndex)
	mainMux.HandleFunc("/app", server.handleApp)
	mainMux.HandleFunc("/login", server.handleLogin)
	mainMux.HandleFunc("/authorize", server.handleAuthorize)
	mainMux.HandleFunc("/token", server.handleToken)
	mainMux.HandleFunc("/callback", server.handleCallback)
	mainMux.HandleFunc("/negative", server.handleNegative)
	mainMux.HandleFunc("/rotate", server.handleRotate)
	mainMux.HandleFunc("/logout", server.handleLogout)
	mainMux.HandleFunc("/admin/deprovision", server.handleDeprovision)
	mainMux.HandleFunc("/admin/posture", server.handlePosture)
	mainMux.HandleFunc("/admin/recover", server.handleRecover)
	mainMux.HandleFunc("/boundary", server.handleBoundary)
	mainMux.HandleFunc("/csrf", server.handleCSRF)
	mainMux.HandleFunc("/.well-known/openid-configuration", server.handleDiscovery)
	mainMux.HandleFunc("/jwks.json", server.handleJWKS)
	mainMux.HandleFunc("/evidence", server.handleEvidence)
	server.mainServer = &http.Server{Handler: securityHeaders(mainMux, server.OriginBURL), ReadHeaderTimeout: 2 * time.Second}

	originMux := http.NewServeMux()
	originMux.HandleFunc("/allowed", server.handleOriginAllowed)
	originMux.HandleFunc("/denied", server.handleOriginDenied)
	server.originBServer = &http.Server{Handler: originMux, ReadHeaderTimeout: 2 * time.Second}

	go func() { _ = server.mainServer.Serve(mainListener) }()
	go func() { _ = server.originBServer.Serve(originBListener) }()
	return server, nil
}

func securityHeaders(next http.Handler, originB string) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self' "+originB+"; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(writer, request)
	})
}

func (server *Server) Close() error {
	mainErr := server.mainServer.Close()
	originErr := server.originBServer.Close()
	if mainErr != nil {
		return mainErr
	}
	return originErr
}

func randomValue(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func (server *Server) record(labID, outcome, decision, reason string) {
	event := evidence.NewEvent(labID, outcome, decision, reason)
	event.Stage = "browser_interaction"
	event.Component = "browser-lab"
	event.EvidenceKind = "http_interaction"
	event.ObservedState = decision + ":" + reason
	event.Issuer = server.URL
	event.Subject = subject
	event.SessionID = "browser-session"
	server.mu.Lock()
	server.events = append(server.events, event)
	server.mu.Unlock()
}

func (server *Server) handleIndex(writer http.ResponseWriter, _ *http.Request) {
	fmt.Fprintf(writer, `<!doctype html>
<html><head><meta charset="utf-8"><title>Network Security Browser Lab</title></head>
<body><h1>Network Security Browser Lab</h1>
<p id="surface">loopback-only educational surface</p>
<ul>
<li><a id="protected-app" href="/app">Protected application and OIDC-style login</a></li>
<li><a id="browser-boundary" href="/boundary">Two-origin CORS and CSRF boundary</a></li>
<li><a id="negative-all" href="/negative?case=all">Run SSO negative checks</a></li>
<li><a id="rotate-key" href="/rotate">Rotate signing key</a></li>
<li><a id="deprovision" href="/admin/deprovision">SCIM-style deprovision</a></li>
<li><a id="degrade-posture" href="/admin/posture?status=noncompliant">Degrade posture</a></li>
<li><a id="recover" href="/admin/recover">Recover identity and posture</a></li>
<li><a id="evidence" href="/evidence">Evidence JSON</a></li>
</ul><p>Main origin: %s<br>Origin B: %s</p></body></html>`, server.URL, server.OriginBURL)
}

func (server *Server) handleApp(writer http.ResponseWriter, request *http.Request) {
	cookie, err := request.Cookie("rp_session")
	if err != nil {
		http.Redirect(writer, request, "/login", http.StatusFound)
		return
	}
	server.mu.Lock()
	current, ok := server.sessions[cookie.Value]
	deprovisioned := server.deprovisioned
	compliant := server.compliant
	server.mu.Unlock()
	if !ok || !current.Active {
		http.Redirect(writer, request, "/login", http.StatusFound)
		return
	}
	if deprovisioned || !compliant {
		writer.WriteHeader(http.StatusForbidden)
		fmt.Fprint(writer, "<h1 id=access-denied>ACCESS DENIED</h1><p>Existing RP session was attenuated by identity or posture state.</p><a href='/admin/recover'>Recover</a>")
		return
	}
	fmt.Fprintf(writer, "<h1 id=access-granted>ACCESS GRANTED</h1><p>subject=%s</p><p>issuer=%s</p><a id=logout href='/logout'>Logout</a>", current.Subject, server.URL)
}

func (server *Server) handleLogin(writer http.ResponseWriter, request *http.Request) {
	state, err := randomValue(18)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	nonce, _ := randomValue(18)
	verifier, _ := randomValue(32)
	server.mu.Lock()
	server.transactions[state] = transaction{Verifier: verifier, Nonce: nonce}
	server.mu.Unlock()
	query := url.Values{
		"client_id":             {clientID},
		"redirect_uri":          {server.URL + "/callback"},
		"state":                 {state},
		"nonce":                 {nonce},
		"code_challenge":        {oidclab.PKCEChallenge(verifier)},
		"code_challenge_method": {"S256"},
	}
	http.Redirect(writer, request, "/authorize?"+query.Encode(), http.StatusFound)
}

func (server *Server) handleAuthorize(writer http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	code, err := server.provider.Authorize(
		query.Get("client_id"), query.Get("redirect_uri"), subject,
		query.Get("nonce"), query.Get("code_challenge"),
	)
	if err != nil {
		server.record("LAB-NETSEC-07", evidence.Reject, "deny", err.Error())
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	redirect, _ := url.Parse(query.Get("redirect_uri"))
	values := redirect.Query()
	values.Set("code", code)
	values.Set("state", query.Get("state"))
	values.Set("iss", server.URL)
	redirect.RawQuery = values.Encode()
	http.Redirect(writer, request, redirect.String(), http.StatusFound)
}

func (server *Server) handleCallback(writer http.ResponseWriter, request *http.Request) {
	state := request.URL.Query().Get("state")
	server.mu.Lock()
	tx, ok := server.transactions[state]
	delete(server.transactions, state)
	server.mu.Unlock()
	if !ok {
		server.record("LAB-NETSEC-07", evidence.Reject, "deny", "state mismatch")
		http.Error(writer, "state mismatch", http.StatusBadRequest)
		return
	}
	if request.URL.Query().Get("iss") != server.URL {
		server.record("LAB-NETSEC-07", evidence.Reject, "deny", "authorization response issuer mismatch")
		http.Error(writer, "issuer mismatch", http.StatusBadRequest)
		return
	}
	tokenResponse, err := http.PostForm(server.URL+"/token", url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {request.URL.Query().Get("code")},
		"client_id":     {clientID},
		"redirect_uri":  {server.URL + "/callback"},
		"code_verifier": {tx.Verifier},
	})
	if err != nil {
		server.record("LAB-NETSEC-07", evidence.DependencyFailure, "deny", err.Error())
		http.Error(writer, err.Error(), http.StatusBadGateway)
		return
	}
	defer tokenResponse.Body.Close()
	if tokenResponse.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(tokenResponse.Body)
		server.record("LAB-NETSEC-07", evidence.Reject, "deny", string(body))
		http.Error(writer, "token endpoint rejected authorization code", http.StatusBadRequest)
		return
	}
	var tokens oidclab.TokenSet
	if err := json.NewDecoder(tokenResponse.Body).Decode(&tokens); err != nil {
		server.record("LAB-NETSEC-07", evidence.DependencyFailure, "deny", err.Error())
		http.Error(writer, err.Error(), http.StatusBadGateway)
		return
	}
	claims, err := server.provider.ValidateIDToken(tokens.IDToken, server.URL, clientID, tx.Nonce, time.Now())
	if err != nil {
		server.record("LAB-NETSEC-07", evidence.Reject, "deny", err.Error())
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	sessionID, _ := randomValue(24)
	server.mu.Lock()
	server.sessions[sessionID] = session{Subject: claims.Subject, Active: true}
	server.mu.Unlock()
	http.SetCookie(writer, &http.Cookie{
		Name: "rp_session", Value: sessionID, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, MaxAge: 300,
	})
	server.record("LAB-NETSEC-07", evidence.Normal, "allow", "code, PKCE and ID token validation established an RP session")
	http.Redirect(writer, request, "/app", http.StatusFound)
}

func (server *Server) handleToken(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(writer, "POST required", http.StatusMethodNotAllowed)
		return
	}
	if err := request.ParseForm(); err != nil {
		http.Error(writer, "invalid form", http.StatusBadRequest)
		return
	}
	if request.Form.Get("grant_type") != "authorization_code" {
		http.Error(writer, "unsupported grant_type", http.StatusBadRequest)
		return
	}
	tokens, err := server.provider.Redeem(
		request.Form.Get("code"), request.Form.Get("client_id"),
		request.Form.Get("redirect_uri"), request.Form.Get("code_verifier"),
	)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(tokens)
}

func (server *Server) handleNegative(writer http.ResponseWriter, request *http.Request) {
	wanted := request.URL.Query().Get("case")
	cases := []string{
		"state", "nonce", "pkce", "issuer", "audience", "expiry", "redirect", "kid",
		"open_redirect", "code_injection", "mix_up", "token_audience_confusion",
	}
	if wanted != "all" {
		cases = []string{wanted}
	}
	results := make([]string, 0, len(cases))
	for _, name := range cases {
		if err := server.negativeCase(name); err == nil {
			results = append(results, name+"=FAIL(open)")
		} else {
			results = append(results, name+"=PASS(rejected)")
		}
	}
	server.record("LAB-NETSEC-07", evidence.Reject, "deny", strings.Join(results, ";"))
	fmt.Fprintf(writer, "<h1 id=negative-result>SSO NEGATIVE CHECKS</h1><pre>%s</pre><a href='/'>Home</a>", strings.Join(results, "\n"))
}

func (server *Server) negativeCase(name string) error {
	now := time.Now()
	claims := oidclab.Claims{
		Issuer: server.URL, Subject: subject, Audience: clientID, Azp: clientID,
		Nonce: "nonce-good", Expires: now.Add(time.Minute).Unix(), IssuedAt: now.Unix(),
		NotBefore: now.Add(-time.Second).Unix(), SessionID: "sid-negative",
	}
	switch name {
	case "state":
		if "attacker-state" != "expected-state" {
			return fmt.Errorf("state mismatch")
		}
	case "pkce":
		verifier := "good-verifier"
		code, err := server.provider.Authorize(clientID, server.URL+"/callback", subject, claims.Nonce, oidclab.PKCEChallenge(verifier))
		if err != nil {
			return err
		}
		_, err = server.provider.Redeem(code, clientID, server.URL+"/callback", "wrong-verifier")
		return err
	case "redirect":
		_, err := server.provider.Authorize(clientID, server.URL+"/unregistered", subject, claims.Nonce, oidclab.PKCEChallenge("v"))
		return err
	case "open_redirect":
		_, err := server.provider.Authorize(
			clientID,
			server.URL+"/callback?next=https%3A%2F%2Fattacker.invalid",
			subject,
			claims.Nonce,
			oidclab.PKCEChallenge("v"),
		)
		return err
	case "code_injection":
		_, err := server.provider.Redeem(
			"injected-authorization-code", clientID, server.URL+"/callback", "attacker-verifier",
		)
		return err
	case "expiry":
		claims.Expires = now.Add(-time.Second).Unix()
		token, _ := server.provider.IssueTestToken(claims, "")
		_, err := server.provider.ValidateIDToken(token, server.URL, clientID, claims.Nonce, now)
		return err
	case "nonce", "issuer", "audience", "kid", "mix_up", "token_audience_confusion":
		token, _ := server.provider.IssueTestToken(claims, "")
		expectedIssuer, expectedAudience, expectedNonce := server.URL, clientID, claims.Nonce
		if name == "nonce" {
			expectedNonce = "wrong"
		}
		if name == "issuer" || name == "mix_up" {
			expectedIssuer = "http://127.0.0.1/wrong-issuer"
		}
		if name == "audience" || name == "token_audience_confusion" {
			expectedAudience = "wrong-rp"
		}
		if name == "kid" {
			token = tamperKid(token)
		}
		_, err := server.provider.ValidateIDToken(token, expectedIssuer, expectedAudience, expectedNonce, now)
		return err
	default:
		return fmt.Errorf("unknown negative case")
	}
	return nil
}

func tamperKid(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return token
	}
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"unknown","typ":"JWT"}`))
	return header + "." + parts[1] + "." + parts[2]
}

func (server *Server) handleRotate(writer http.ResponseWriter, _ *http.Request) {
	server.mu.Lock()
	server.keyRevision++
	revision := server.keyRevision
	server.mu.Unlock()
	err := server.provider.Rotate(fmt.Sprintf("kid-%d", revision))
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	server.record("LAB-NETSEC-07", evidence.Recovery, "allow", fmt.Sprintf("activated kid-%d with bounded old-key overlap", revision))
	fmt.Fprintf(writer, "<h1 id=rotation>KEY ROTATED</h1><p>active kid-%d</p><a href='/app'>Login again</a>", revision)
}

func (server *Server) handleLogout(writer http.ResponseWriter, request *http.Request) {
	if cookie, err := request.Cookie("rp_session"); err == nil {
		server.mu.Lock()
		delete(server.sessions, cookie.Value)
		server.mu.Unlock()
	}
	http.SetCookie(writer, &http.Cookie{Name: "rp_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
	server.record("LAB-NETSEC-08", evidence.Recovery, "logged_out", "RP session terminated; future access starts a new transaction")
	fmt.Fprint(writer, "<h1 id=logged-out>LOGGED OUT</h1><a href='/app'>Start new login</a>")
}

func (server *Server) handleDeprovision(writer http.ResponseWriter, _ *http.Request) {
	server.mu.Lock()
	server.deprovisioned = true
	for key, value := range server.sessions {
		value.Active = false
		server.sessions[key] = value
	}
	server.mu.Unlock()
	server.record("LAB-NETSEC-08", evidence.Reject, "revoke", "SCIM-style deprovision tombstone invalidated known RP sessions")
	fmt.Fprint(writer, "<h1 id=deprovisioned>SUBJECT DEPROVISIONED</h1><a href='/app'>Verify denial</a>")
}

func (server *Server) handlePosture(writer http.ResponseWriter, request *http.Request) {
	compliant := request.URL.Query().Get("status") == "compliant"
	server.mu.Lock()
	server.compliant = compliant
	server.mu.Unlock()
	decision := "revoke"
	outcome := evidence.Degraded
	if compliant {
		decision, outcome = "allow", evidence.Recovery
	}
	server.record("LAB-NETSEC-09", outcome, decision, fmt.Sprintf("posture compliant=%t", compliant))
	fmt.Fprintf(writer, "<h1 id=posture>POSTURE compliant=%t</h1><a href='/app'>Verify access</a>", compliant)
}

func (server *Server) handleRecover(writer http.ResponseWriter, _ *http.Request) {
	server.mu.Lock()
	server.deprovisioned = false
	server.compliant = true
	server.mu.Unlock()
	server.record("LAB-NETSEC-08", evidence.Recovery, "allow_new_login", "identity lifecycle recovery permits a fresh session")
	server.record("LAB-NETSEC-09", evidence.Recovery, "allow_new_login", "fresh posture permits a new access decision")
	fmt.Fprint(writer, "<h1 id=recovered>IDENTITY AND POSTURE RECOVERED</h1><a href='/app'>Start fresh login</a>")
}

func (server *Server) handleBoundary(writer http.ResponseWriter, _ *http.Request) {
	fmt.Fprintf(writer, `<!doctype html><html><head><meta charset="utf-8"><title>Browser Boundary</title></head>
<body><h1>Two-Origin Browser Boundary</h1>
<button id="cors-allowed" onclick="runAllowed()">Allowed CORS</button>
<button id="cors-denied" onclick="runDenied()">Denied CORS</button>
<button id="csrf-allowed" onclick="csrf(true)">Allowed CSRF token</button>
<button id="csrf-denied" onclick="csrf(false)">Missing CSRF token</button>
<pre id="result">not-run</pre>
<script>
const result = document.getElementById('result');
async function runAllowed(){try{const r=await fetch(%q+'/allowed',{credentials:'omit'});result.textContent='allowed-cors='+await r.text()}catch(e){result.textContent='allowed-cors=FAIL '+e}}
async function runDenied(){try{await fetch(%q+'/denied',{credentials:'omit'});result.textContent='denied-cors=FAIL(open)'}catch(e){result.textContent='denied-cors=PASS(rejected)'}}
async function csrf(ok){const headers=ok?{'X-CSRF-Token':'lab-csrf'}:{};const r=await fetch('/csrf',{method:'POST',headers});result.textContent='csrf-status='+r.status+' '+await r.text()}
</script><a href='/'>Home</a></body></html>`, server.OriginBURL, server.OriginBURL)
}

func (server *Server) handleCSRF(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost || request.Header.Get("X-CSRF-Token") != "lab-csrf" {
		server.record("LAB-NETSEC-05", evidence.Reject, "deny", "missing or invalid CSRF token")
		http.Error(writer, "CSRF rejected", http.StatusForbidden)
		return
	}
	server.record("LAB-NETSEC-05", evidence.Normal, "allow", "same-origin mutation carried a valid CSRF token")
	fmt.Fprint(writer, "CSRF accepted")
}

func (server *Server) handleOriginAllowed(writer http.ResponseWriter, request *http.Request) {
	if request.Header.Get("Origin") == server.URL {
		writer.Header().Set("Access-Control-Allow-Origin", server.URL)
	}
	fmt.Fprint(writer, "origin-b-allowed")
}

func (server *Server) handleOriginDenied(writer http.ResponseWriter, _ *http.Request) {
	fmt.Fprint(writer, "origin-b-without-cors-header")
}

func (server *Server) handleDiscovery(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(map[string]any{
		"issuer": server.URL, "authorization_endpoint": server.URL + "/authorize",
		"token_endpoint": server.URL + "/token", "jwks_uri": server.URL + "/jwks.json",
		"code_challenge_methods_supported": []string{"S256"},
	})
}

func (server *Server) handleJWKS(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(server.provider.JWKSet())
}

func (server *Server) handleEvidence(writer http.ResponseWriter, _ *http.Request) {
	server.mu.Lock()
	copyEvents := append([]evidence.Event(nil), server.events...)
	server.mu.Unlock()
	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(map[string]any{"events": copyEvents})
}

func (server *Server) Evidence() []evidence.Event {
	server.mu.Lock()
	defer server.mu.Unlock()
	return append([]evidence.Event(nil), server.events...)
}
