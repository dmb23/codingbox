package proxy

import (
	"net/http"
	"strings"

	"github.com/codingbox/codingbox/internal/models"
)

// SecretInjector injects secrets into outbound HTTP requests based on host matching.
type SecretInjector struct {
	mappings []models.SecretMapping
}

// NewSecretInjector creates a new injector with the given secret mappings.
func NewSecretInjector(mappings []models.SecretMapping) *SecretInjector {
	return &SecretInjector{mappings: mappings}
}

// InjectSecrets modifies the request to inject matching secrets. Returns names of injected secrets.
func (si *SecretInjector) InjectSecrets(req *http.Request) []string {
	var injected []string
	host := req.URL.Hostname()

	for _, m := range si.mappings {
		if m.TargetHost == host {
			value := strings.Replace(m.HeaderTemplate, "{secret}", m.SecretValue, 1)
			req.Header.Set(m.HeaderName, value)
			injected = append(injected, m.Name)
		}
	}

	return injected
}

// Mappings returns the configured secret mappings.
func (si *SecretInjector) Mappings() []models.SecretMapping {
	return si.mappings
}
