package unit

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/mischa/codingbox/internal/models"
	"github.com/mischa/codingbox/internal/proxy"
)

func TestReplaceSecretsInRequest_HeadersOnly(t *testing.T) {
	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Scheme: "https", Host: "api.example.com", Path: "/v1/data"},
		Header: http.Header{
			"Authorization": {"Bearer __API_KEY__"},
			"X-Custom":      {"no-placeholder-here"},
		},
		Body: io.NopCloser(bytes.NewReader([]byte(`{"key":"__API_KEY__"}`))),
	}

	secrets := []models.SecretMapping{
		{Placeholder: "__API_KEY__", Value: "real-secret-123", ReplaceIn: []string{models.ReplaceHeaders}},
	}

	result := proxy.ReplaceSecretsInRequest(req, secrets)

	// Header should be replaced.
	if got := result.Header.Get("Authorization"); got != "Bearer real-secret-123" {
		t.Errorf("Authorization header = %q, want 'Bearer real-secret-123'", got)
	}

	// Body should NOT be replaced (only headers configured).
	body, _ := io.ReadAll(result.Body)
	if !bytes.Contains(body, []byte("__API_KEY__")) {
		t.Error("body was replaced despite only headers being configured")
	}
}

func TestReplaceSecretsInRequest_BodyOnly(t *testing.T) {
	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Scheme: "https", Host: "api.example.com", Path: "/v1/data"},
		Header: http.Header{
			"Authorization": {"Bearer __TOKEN__"},
		},
		Body: io.NopCloser(bytes.NewReader([]byte(`{"token":"__TOKEN__"}`))),
	}

	secrets := []models.SecretMapping{
		{Placeholder: "__TOKEN__", Value: "secret-token-456", ReplaceIn: []string{models.ReplaceBody}},
	}

	result := proxy.ReplaceSecretsInRequest(req, secrets)

	// Header should NOT be replaced.
	if got := result.Header.Get("Authorization"); got != "Bearer __TOKEN__" {
		t.Errorf("Authorization header = %q, want 'Bearer __TOKEN__' (should not be replaced)", got)
	}

	// Body should be replaced.
	body, _ := io.ReadAll(result.Body)
	if !bytes.Contains(body, []byte("secret-token-456")) {
		t.Error("body was not replaced despite body being configured")
	}
	if bytes.Contains(body, []byte("__TOKEN__")) {
		t.Error("body still contains placeholder after replacement")
	}
}

func TestReplaceSecretsInRequest_QueryOnly(t *testing.T) {
	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Scheme: "https", Host: "api.example.com", Path: "/v1/data", RawQuery: "api_key=__KEY__&format=json"},
		Header: http.Header{
			"X-Key": {"__KEY__"},
		},
	}

	secrets := []models.SecretMapping{
		{Placeholder: "__KEY__", Value: "real-key-789", ReplaceIn: []string{models.ReplaceQuery}},
	}

	result := proxy.ReplaceSecretsInRequest(req, secrets)

	// Query should be replaced.
	if got := result.URL.RawQuery; got != "api_key=real-key-789&format=json" {
		t.Errorf("RawQuery = %q, want 'api_key=real-key-789&format=json'", got)
	}

	// Header should NOT be replaced.
	if got := result.Header.Get("X-Key"); got != "__KEY__" {
		t.Errorf("X-Key header = %q, want '__KEY__' (should not be replaced)", got)
	}
}

func TestReplaceSecretsInRequest_AllLocations(t *testing.T) {
	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Scheme: "https", Host: "api.example.com", Path: "/v1", RawQuery: "token=__SECRET__"},
		Header: http.Header{
			"Authorization": {"Bearer __SECRET__"},
		},
		Body: io.NopCloser(bytes.NewReader([]byte(`{"auth":"__SECRET__"}`))),
	}

	secrets := []models.SecretMapping{
		{Placeholder: "__SECRET__", Value: "the-real-secret", ReplaceIn: []string{models.ReplaceHeaders, models.ReplaceBody, models.ReplaceQuery}},
	}

	result := proxy.ReplaceSecretsInRequest(req, secrets)

	if got := result.Header.Get("Authorization"); got != "Bearer the-real-secret" {
		t.Errorf("Authorization = %q, want 'Bearer the-real-secret'", got)
	}

	body, _ := io.ReadAll(result.Body)
	if !bytes.Contains(body, []byte("the-real-secret")) {
		t.Error("body not replaced")
	}

	if got := result.URL.RawQuery; got != "token=the-real-secret" {
		t.Errorf("RawQuery = %q, want 'token=the-real-secret'", got)
	}
}

func TestReplaceSecretsInRequest_MultipleSecrets(t *testing.T) {
	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Scheme: "https", Host: "api.example.com", Path: "/v1"},
		Header: http.Header{
			"Authorization": {"Bearer __TOKEN_A__"},
			"X-Api-Key":     {"__TOKEN_B__"},
		},
	}

	secrets := []models.SecretMapping{
		{Placeholder: "__TOKEN_A__", Value: "secret-a", ReplaceIn: []string{models.ReplaceHeaders}},
		{Placeholder: "__TOKEN_B__", Value: "secret-b", ReplaceIn: []string{models.ReplaceHeaders}},
	}

	result := proxy.ReplaceSecretsInRequest(req, secrets)

	if got := result.Header.Get("Authorization"); got != "Bearer secret-a" {
		t.Errorf("Authorization = %q, want 'Bearer secret-a'", got)
	}
	if got := result.Header.Get("X-Api-Key"); got != "secret-b" {
		t.Errorf("X-Api-Key = %q, want 'secret-b'", got)
	}
}

func TestReplaceSecretsInRequest_NoMatchNoChange(t *testing.T) {
	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Scheme: "https", Host: "api.example.com", Path: "/v1", RawQuery: "page=1"},
		Header: http.Header{
			"Accept": {"application/json"},
		},
		Body: io.NopCloser(bytes.NewReader([]byte(`{"data":"value"}`))),
	}

	secrets := []models.SecretMapping{
		{Placeholder: "__NONEXISTENT__", Value: "secret", ReplaceIn: []string{models.ReplaceHeaders, models.ReplaceBody, models.ReplaceQuery}},
	}

	result := proxy.ReplaceSecretsInRequest(req, secrets)

	if got := result.Header.Get("Accept"); got != "application/json" {
		t.Errorf("Accept header changed unexpectedly: %q", got)
	}
	body, _ := io.ReadAll(result.Body)
	if string(body) != `{"data":"value"}` {
		t.Errorf("body changed unexpectedly: %s", body)
	}
	if got := result.URL.RawQuery; got != "page=1" {
		t.Errorf("query changed unexpectedly: %q", got)
	}
}

func TestReplaceSecretsInRequest_NilBody(t *testing.T) {
	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Scheme: "https", Host: "api.example.com", Path: "/v1"},
		Header: http.Header{
			"Authorization": {"Bearer __KEY__"},
		},
		Body: nil,
	}

	secrets := []models.SecretMapping{
		{Placeholder: "__KEY__", Value: "real-key", ReplaceIn: []string{models.ReplaceHeaders, models.ReplaceBody}},
	}

	// Should not panic on nil body.
	result := proxy.ReplaceSecretsInRequest(req, secrets)

	if got := result.Header.Get("Authorization"); got != "Bearer real-key" {
		t.Errorf("Authorization = %q, want 'Bearer real-key'", got)
	}
}

// Response reverse-replacement tests.

func TestReplaceSecretsInResponse_HeadersAndBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: 200,
		Header: http.Header{
			"X-Token": {"real-secret-123"},
		},
		Body: io.NopCloser(bytes.NewReader([]byte(`{"token":"real-secret-123"}`))),
	}

	secrets := []models.SecretMapping{
		{Placeholder: "__API_KEY__", Value: "real-secret-123", ReplaceIn: []string{models.ReplaceHeaders, models.ReplaceBody}},
	}

	result, replaced := proxy.ReplaceSecretsInResponse(resp, secrets)

	if !replaced {
		t.Error("expected replaced=true")
	}

	if got := result.Header.Get("X-Token"); got != "__API_KEY__" {
		t.Errorf("X-Token header = %q, want '__API_KEY__'", got)
	}

	body, _ := io.ReadAll(result.Body)
	if !bytes.Contains(body, []byte("__API_KEY__")) {
		t.Error("body not reverse-replaced")
	}
	if bytes.Contains(body, []byte("real-secret-123")) {
		t.Error("body still contains real secret after reverse replacement")
	}
}

func TestReplaceSecretsInResponse_NoSecretPresent(t *testing.T) {
	resp := &http.Response{
		StatusCode: 200,
		Header: http.Header{
			"Content-Type": {"application/json"},
		},
		Body: io.NopCloser(bytes.NewReader([]byte(`{"status":"ok"}`))),
	}

	secrets := []models.SecretMapping{
		{Placeholder: "__KEY__", Value: "secret-value", ReplaceIn: []string{models.ReplaceHeaders, models.ReplaceBody}},
	}

	_, replaced := proxy.ReplaceSecretsInResponse(resp, secrets)

	if replaced {
		t.Error("expected replaced=false when no secrets are present")
	}
}

func TestReplaceSecretsInResponse_HeadersOnlyConfig(t *testing.T) {
	resp := &http.Response{
		StatusCode: 200,
		Header: http.Header{
			"X-Key": {"secret-val"},
		},
		Body: io.NopCloser(bytes.NewReader([]byte(`{"key":"secret-val"}`))),
	}

	secrets := []models.SecretMapping{
		{Placeholder: "__PH__", Value: "secret-val", ReplaceIn: []string{models.ReplaceHeaders}},
	}

	result, replaced := proxy.ReplaceSecretsInResponse(resp, secrets)

	if !replaced {
		t.Error("expected replaced=true for header replacement")
	}

	// Header should be replaced.
	if got := result.Header.Get("X-Key"); got != "__PH__" {
		t.Errorf("X-Key = %q, want '__PH__'", got)
	}

	// Body should NOT be replaced (only headers configured).
	body, _ := io.ReadAll(result.Body)
	if bytes.Contains(body, []byte("__PH__")) {
		t.Error("body was replaced despite only headers being in replace_in")
	}
}

func TestReplaceSecretsInResponse_NilBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: 204,
		Header: http.Header{
			"X-Token": {"my-secret"},
		},
		Body: nil,
	}

	secrets := []models.SecretMapping{
		{Placeholder: "__TOKEN__", Value: "my-secret", ReplaceIn: []string{models.ReplaceHeaders, models.ReplaceBody}},
	}

	// Should not panic on nil body.
	result, replaced := proxy.ReplaceSecretsInResponse(resp, secrets)

	if !replaced {
		t.Error("expected replaced=true for header replacement")
	}
	if got := result.Header.Get("X-Token"); got != "__TOKEN__" {
		t.Errorf("X-Token = %q, want '__TOKEN__'", got)
	}
}
