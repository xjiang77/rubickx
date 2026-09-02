package browserlab

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

func RunEvidenceFixture(ctx context.Context, server *Server) error {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return err
	}
	client := &http.Client{Jar: jar, Timeout: 3 * time.Second}
	do := func(method, path string, headers map[string]string, wantStatus int) (string, error) {
		request, err := http.NewRequestWithContext(ctx, method, server.URL+path, nil)
		if err != nil {
			return "", err
		}
		for key, value := range headers {
			request.Header.Set(key, value)
		}
		response, err := client.Do(request)
		if err != nil {
			return "", err
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return "", err
		}
		if response.StatusCode != wantStatus {
			return "", fmt.Errorf("%s %s status=%d body=%s", method, path, response.StatusCode, body)
		}
		return string(body), nil
	}
	body, err := do(http.MethodGet, "/app", nil, http.StatusOK)
	if err != nil || !strings.Contains(body, "ACCESS GRANTED") {
		return fmt.Errorf("OIDC fixture login body=%q: %w", body, err)
	}
	if _, err := do(http.MethodGet, "/negative?case=all", nil, http.StatusOK); err != nil {
		return err
	}
	if _, err := do(http.MethodPost, "/csrf", map[string]string{"X-CSRF-Token": "lab-csrf"}, http.StatusOK); err != nil {
		return err
	}
	if _, err := do(http.MethodPost, "/csrf", nil, http.StatusForbidden); err != nil {
		return err
	}
	if _, err := do(http.MethodGet, "/admin/deprovision", nil, http.StatusOK); err != nil {
		return err
	}
	if _, err := do(http.MethodGet, "/admin/posture?status=noncompliant", nil, http.StatusOK); err != nil {
		return err
	}
	if _, err := do(http.MethodGet, "/admin/recover", nil, http.StatusOK); err != nil {
		return err
	}
	return nil
}
