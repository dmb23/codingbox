package unit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mischa/codingbox/internal/config"
)

func TestResolveAutoMounts_ExistingPaths(t *testing.T) {
	home := t.TempDir()

	// Create some paths that exist.
	os.WriteFile(filepath.Join(home, ".gitconfig"), []byte("[user]\nname=test"), 0644)
	os.MkdirAll(filepath.Join(home, ".claude"), 0755)
	os.MkdirAll(filepath.Join(home, ".vibe"), 0755)

	mounts := config.ResolveAutoMounts(home)

	// Should find .gitconfig, .claude, .vibe (3 paths that exist).
	if len(mounts) != 3 {
		t.Fatalf("expected 3 mounts, got %d: %+v", len(mounts), mounts)
	}

	// Verify target == source for all.
	for _, m := range mounts {
		if m.Source != m.Target {
			t.Errorf("target should equal source: source=%q target=%q", m.Source, m.Target)
		}
	}
}

func TestResolveAutoMounts_MissingPathsSkipped(t *testing.T) {
	home := t.TempDir()
	// Don't create any paths — all should be skipped.

	mounts := config.ResolveAutoMounts(home)
	if len(mounts) != 0 {
		t.Errorf("expected 0 mounts for empty home, got %d", len(mounts))
	}
}

func TestResolveAutoMounts_CorrectModes(t *testing.T) {
	home := t.TempDir()

	os.WriteFile(filepath.Join(home, ".gitconfig"), []byte(""), 0644)
	os.MkdirAll(filepath.Join(home, ".claude"), 0755)

	mounts := config.ResolveAutoMounts(home)

	for _, m := range mounts {
		switch filepath.Base(m.Source) {
		case ".gitconfig":
			if m.Mode != "ro" {
				t.Errorf(".gitconfig should be ro, got %q", m.Mode)
			}
		case ".claude":
			if m.Mode != "rw" {
				t.Errorf(".claude should be rw, got %q", m.Mode)
			}
		}
	}
}

func TestResolveAutoMounts_AbsolutePaths(t *testing.T) {
	home := t.TempDir()
	os.WriteFile(filepath.Join(home, ".gitconfig"), []byte(""), 0644)

	mounts := config.ResolveAutoMounts(home)
	if len(mounts) == 0 {
		t.Fatal("expected at least 1 mount")
	}

	if !filepath.IsAbs(mounts[0].Source) {
		t.Errorf("source should be absolute: %q", mounts[0].Source)
	}
	if !filepath.IsAbs(mounts[0].Target) {
		t.Errorf("target should be absolute: %q", mounts[0].Target)
	}
}
