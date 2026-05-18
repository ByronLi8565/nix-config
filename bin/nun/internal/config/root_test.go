package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRepoRootPrefersEnv(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NUN_CONFIG_ROOT", root)
	got, err := (App{}).repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	if got != root {
		t.Fatalf("repoRoot = %q, want %q", got, root)
	}
}

func TestRepoRootFallsBackToHomeNixConfig(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(original)
	})
	t.Setenv("HOME", home)
	t.Setenv("NUN_CONFIG_ROOT", "")
	configRoot := filepath.Join(home, "nix-config")
	if err := os.MkdirAll(configRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configRoot, "flake.nix"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := (App{}).repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	if got != configRoot {
		t.Fatalf("repoRoot = %q, want %q", got, configRoot)
	}
}
