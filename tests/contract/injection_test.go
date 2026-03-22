package contract

import (
	"net/http"
	"testing"

	"github.com/codingbox/codingbox/internal/models"
	"github.com/codingbox/codingbox/internal/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecretInjector_InjectsCorrectHeader(t *testing.T) {
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

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", nil)
	require.NoError(t, err)

	injected := injector.InjectSecrets(req)

	assert.Equal(t, []string{"openai-key"}, injected)
	assert.Equal(t, "Bearer sk-real-secret-123", req.Header.Get("Authorization"))
}

func TestSecretInjector_NoInjectionForNonMatchingHost(t *testing.T) {
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

	req, err := http.NewRequest("GET", "https://api.github.com/repos", nil)
	require.NoError(t, err)

	injected := injector.InjectSecrets(req)

	assert.Empty(t, injected)
	assert.Empty(t, req.Header.Get("Authorization"))
}

func TestSecretInjector_MultipleSecrets(t *testing.T) {
	mappings := []models.SecretMapping{
		{
			ID:             "uuid-1",
			Name:           "openai-key",
			TargetHost:     "api.openai.com",
			HeaderName:     "Authorization",
			HeaderTemplate: "Bearer {secret}",
			SecretValue:    "sk-openai-123",
		},
		{
			ID:             "uuid-2",
			Name:           "github-token",
			TargetHost:     "api.github.com",
			HeaderName:     "Authorization",
			HeaderTemplate: "token {secret}",
			SecretValue:    "ghp-github-456",
		},
	}

	injector := proxy.NewSecretInjector(mappings)

	// Request to OpenAI
	req1, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat", nil)
	injected1 := injector.InjectSecrets(req1)
	assert.Equal(t, []string{"openai-key"}, injected1)
	assert.Equal(t, "Bearer sk-openai-123", req1.Header.Get("Authorization"))

	// Request to GitHub
	req2, _ := http.NewRequest("GET", "https://api.github.com/repos", nil)
	injected2 := injector.InjectSecrets(req2)
	assert.Equal(t, []string{"github-token"}, injected2)
	assert.Equal(t, "token ghp-github-456", req2.Header.Get("Authorization"))
}

func TestSecretInjector_TemplateExpansion(t *testing.T) {
	mappings := []models.SecretMapping{
		{
			ID:             "uuid-1",
			Name:           "api-key",
			TargetHost:     "api.example.com",
			HeaderName:     "X-API-Key",
			HeaderTemplate: "{secret}",
			SecretValue:    "raw-key-value",
		},
	}

	injector := proxy.NewSecretInjector(mappings)

	req, _ := http.NewRequest("GET", "https://api.example.com/data", nil)
	injected := injector.InjectSecrets(req)

	assert.Equal(t, []string{"api-key"}, injected)
	assert.Equal(t, "raw-key-value", req.Header.Get("X-API-Key"))
}

func TestSecretInjector_EmptyMappings(t *testing.T) {
	injector := proxy.NewSecretInjector(nil)

	req, _ := http.NewRequest("GET", "https://api.example.com/data", nil)
	injected := injector.InjectSecrets(req)

	assert.Empty(t, injected)
}
