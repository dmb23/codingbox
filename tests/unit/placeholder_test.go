package unit

import (
	"strings"
	"testing"

	"github.com/mischa/codingbox/internal/config"
)

func TestGeneratePlaceholder_Deterministic(t *testing.T) {
	a := config.GeneratePlaceholder("ANTHROPIC_API_KEY")
	b := config.GeneratePlaceholder("ANTHROPIC_API_KEY")
	if a != b {
		t.Errorf("not deterministic: %q != %q", a, b)
	}
}

func TestGeneratePlaceholder_UniquePerName(t *testing.T) {
	a := config.GeneratePlaceholder("ANTHROPIC_API_KEY")
	b := config.GeneratePlaceholder("GITHUB_TOKEN")
	if a == b {
		t.Errorf("same placeholder for different env names: %q", a)
	}
}

func TestGeneratePlaceholder_Format(t *testing.T) {
	p := config.GeneratePlaceholder("MY_SECRET")
	if !strings.HasPrefix(p, "__CODINGBOX_MY_SECRET_") {
		t.Errorf("bad prefix: %q", p)
	}
	if !strings.HasSuffix(p, "__") {
		t.Errorf("bad suffix: %q", p)
	}
	if len(p) < 30 {
		t.Errorf("placeholder too short: %q", p)
	}
}

func TestGeneratePlaceholder_ContainsEnvName(t *testing.T) {
	p := config.GeneratePlaceholder("OPENAI_KEY")
	if !strings.Contains(p, "OPENAI_KEY") {
		t.Errorf("placeholder should contain env name: %q", p)
	}
}
