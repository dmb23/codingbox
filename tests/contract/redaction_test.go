package contract

import (
	"net/http"
	"strings"
	"testing"

	"github.com/codingbox/codingbox/internal/models"
	"github.com/codingbox/codingbox/internal/proxy"
	"github.com/stretchr/testify/assert"
)

func TestRedactHeaders_ReplacesSecretWithPlaceholder(t *testing.T) {
	mappings := []models.SecretMapping{
		{
			ID:             "placeholder-uuid-1",
			Name:           "openai-key",
			TargetHost:     "api.openai.com",
			HeaderName:     "Authorization",
			HeaderTemplate: "Bearer {secret}",
			SecretValue:    "sk-real-secret-123",
		},
	}

	injector := proxy.NewSecretInjector(mappings)
	logger := proxy.NewRequestLogger(nil, injector)

	headers := http.Header{}
	headers.Set("Authorization", "Bearer sk-real-secret-123")
	headers.Set("Content-Type", "application/json")

	redacted := logger.RedactHeaders(headers)

	// Secret should be replaced with placeholder UUID
	assert.Equal(t, "Bearer placeholder-uuid-1", redacted.Get("Authorization"))
	// Non-secret headers should be unchanged
	assert.Equal(t, "application/json", redacted.Get("Content-Type"))
}

func TestRedactHeaders_MultipleSecrets(t *testing.T) {
	mappings := []models.SecretMapping{
		{
			ID:          "uuid-1",
			Name:        "key1",
			TargetHost:  "api.example.com",
			SecretValue: "secret-aaa",
		},
		{
			ID:          "uuid-2",
			Name:        "key2",
			TargetHost:  "api.example.com",
			SecretValue: "secret-bbb",
		},
	}

	injector := proxy.NewSecretInjector(mappings)
	logger := proxy.NewRequestLogger(nil, injector)

	headers := http.Header{}
	headers.Set("X-Key-1", "secret-aaa")
	headers.Set("X-Key-2", "secret-bbb")

	redacted := logger.RedactHeaders(headers)

	assert.Equal(t, "uuid-1", redacted.Get("X-Key-1"))
	assert.Equal(t, "uuid-2", redacted.Get("X-Key-2"))
}

func TestRedactHeaders_NoSecretsInHeaders(t *testing.T) {
	mappings := []models.SecretMapping{
		{
			ID:          "uuid-1",
			Name:        "key1",
			TargetHost:  "api.example.com",
			SecretValue: "secret-aaa",
		},
	}

	injector := proxy.NewSecretInjector(mappings)
	logger := proxy.NewRequestLogger(nil, injector)

	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	headers.Set("Accept", "text/html")

	redacted := logger.RedactHeaders(headers)

	assert.Equal(t, "application/json", redacted.Get("Content-Type"))
	assert.Equal(t, "text/html", redacted.Get("Accept"))
}

func TestRedactHeaders_SecretNeverInOutput(t *testing.T) {
	secret := "sk-super-secret-value-12345"
	mappings := []models.SecretMapping{
		{
			ID:             "placeholder-uuid",
			Name:           "api-key",
			TargetHost:     "api.example.com",
			HeaderName:     "Authorization",
			HeaderTemplate: "Bearer {secret}",
			SecretValue:    secret,
		},
	}

	injector := proxy.NewSecretInjector(mappings)
	logger := proxy.NewRequestLogger(nil, injector)

	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+secret)
	headers.Set("X-Custom", "contains "+secret+" in middle")

	redacted := logger.RedactHeaders(headers)

	// Verify the secret value never appears in any redacted header
	for key, values := range redacted {
		for _, v := range values {
			assert.False(t, strings.Contains(v, secret),
				"secret found in redacted header %s: %s", key, v)
		}
	}
}

func TestRedactHeaders_NilInjector(t *testing.T) {
	logger := proxy.NewRequestLogger(nil, nil)

	headers := http.Header{}
	headers.Set("Authorization", "Bearer some-token")

	redacted := logger.RedactHeaders(headers)
	assert.Equal(t, "Bearer some-token", redacted.Get("Authorization"))
}
