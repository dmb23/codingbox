package unit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mischa/codingbox/internal/config"
	"github.com/mischa/codingbox/internal/models"
)

func TestDirectoryConfigStore_SetGetRemove(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "dirs.yaml")
	store := config.NewDirectoryConfigStore(storePath)

	cfg := models.SandboxConfig{Image: "ubuntu:22.04"}
	store.Set("/home/user/project-a", cfg)

	got, ok := store.Get("/home/user/project-a")
	if !ok {
		t.Fatal("Get returned false for existing entry")
	}
	if got.Image != "ubuntu:22.04" {
		t.Errorf("Image = %q, want ubuntu:22.04", got.Image)
	}

	// Update.
	cfg.Image = "alpine:latest"
	store.Set("/home/user/project-a", cfg)
	got, _ = store.Get("/home/user/project-a")
	if got.Image != "alpine:latest" {
		t.Errorf("after update Image = %q, want alpine:latest", got.Image)
	}

	// Remove.
	if !store.Remove("/home/user/project-a") {
		t.Error("Remove returned false for existing entry")
	}
	_, ok = store.Get("/home/user/project-a")
	if ok {
		t.Error("Get returned true after remove")
	}

	// Remove non-existent.
	if store.Remove("/nonexistent") {
		t.Error("Remove returned true for non-existent entry")
	}
}

func TestDirectoryConfigStore_List(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "dirs.yaml")
	store := config.NewDirectoryConfigStore(storePath)

	store.Set("/a", models.SandboxConfig{Image: "a"})
	store.Set("/b", models.SandboxConfig{Image: "b"})

	list := store.List()
	if len(list) != 2 {
		t.Errorf("List returned %d entries, want 2", len(list))
	}
}

func TestDirectoryConfigStore_FindNearest(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "dirs.yaml")
	store := config.NewDirectoryConfigStore(storePath)

	store.Set("/home/user/project", models.SandboxConfig{Image: "project-image"})

	// Exact match.
	cfg, dir, ok := store.FindNearest("/home/user/project")
	if !ok {
		t.Fatal("FindNearest: exact match not found")
	}
	if cfg.Image != "project-image" || dir != "/home/user/project" {
		t.Errorf("FindNearest exact: Image=%q Dir=%q", cfg.Image, dir)
	}

	// Child directory match.
	cfg, dir, ok = store.FindNearest("/home/user/project/src/main")
	if !ok {
		t.Fatal("FindNearest: parent match not found")
	}
	if dir != "/home/user/project" {
		t.Errorf("FindNearest child: matched dir = %q, want /home/user/project", dir)
	}

	// No match.
	_, _, ok = store.FindNearest("/other/path")
	if ok {
		t.Error("FindNearest: should not match /other/path")
	}
}

func TestDirectoryConfigStore_SaveLoad(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "dirs.yaml")
	store := config.NewDirectoryConfigStore(storePath)

	store.Set("/home/user/project", models.SandboxConfig{
		Image: "my-image:latest",
		Secrets: []models.SecretMapping{
			{Env: "API_KEY", ReplaceIn: []string{"headers"}},
		},
	})

	if err := store.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file exists.
	if _, err := os.Stat(storePath); err != nil {
		t.Fatalf("store file not created: %v", err)
	}

	// Load into new store.
	store2 := config.NewDirectoryConfigStore(storePath)
	if err := store2.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	cfg, ok := store2.Get("/home/user/project")
	if !ok {
		t.Fatal("loaded store missing entry")
	}
	if cfg.Image != "my-image:latest" {
		t.Errorf("Image = %q, want my-image:latest", cfg.Image)
	}
	if len(cfg.Secrets) != 1 || cfg.Secrets[0].Env != "API_KEY" {
		t.Errorf("Secrets not round-tripped: %+v", cfg.Secrets)
	}
}

func TestDirectoryConfigStore_LoadMissing(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "nonexistent.yaml")
	store := config.NewDirectoryConfigStore(storePath)

	if err := store.Load(); err != nil {
		t.Fatalf("Load on missing file should not error: %v", err)
	}
	if len(store.List()) != 0 {
		t.Error("should be empty")
	}
}

func TestCanonicalDir(t *testing.T) {
	// Should return absolute path.
	dir, err := config.CanonicalDir(".")
	if err != nil {
		t.Fatalf("CanonicalDir: %v", err)
	}
	if !filepath.IsAbs(dir) {
		t.Errorf("not absolute: %q", dir)
	}
}
