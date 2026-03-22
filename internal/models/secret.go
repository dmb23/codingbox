package models

import "time"

// SecretMapping maps a placeholder to a real secret for proxy injection.
type SecretMapping struct {
	ID             string    `json:"id" yaml:"-"`
	Name           string    `json:"name" yaml:"name"`
	TargetHost     string    `json:"target_host" yaml:"host"`
	HeaderName     string    `json:"header_name" yaml:"header"`
	HeaderTemplate string    `json:"header_template" yaml:"template"`
	SecretValue    string    `json:"-" yaml:"value"` // Never serialized to JSON
	CreatedAt      time.Time `json:"created_at" yaml:"-"`
}
