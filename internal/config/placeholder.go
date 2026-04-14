package config

import (
	"crypto/sha256"
	"fmt"
)

// GeneratePlaceholder creates a deterministic placeholder string for an env var name.
// Format: __CODINGBOX_<ENV_NAME>_<sha256[:8]>__
func GeneratePlaceholder(envName string) string {
	hash := sha256.Sum256([]byte(envName))
	return fmt.Sprintf("__CODINGBOX_%s_%x__", envName, hash[:4])
}
