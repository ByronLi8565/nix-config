package config

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDotfilePlanItemsMentionsBackups(t *testing.T) {
	items := DotfilePlanItems([]DotfileLink{{
		Source: "/repo/dotfiles/nvim/init.lua",
		Target: filepath.Join(homeForTest(t), ".config/nvim/init.lua"),
		Action: "backup and link",
		Backup: filepath.Join(homeForTest(t), ".config/nvim/init.lua.backup"),
	}})
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if want := "backup existing file"; !strings.Contains(items[0], want) {
		t.Fatalf("plan item %q does not contain %q", items[0], want)
	}
}

func TestParseLinksNix(t *testing.T) {
	content := `
# Dotfile symlinks managed by nun
[
  { source = "aerospace.toml"; target = ".aerospace.toml"; }
  { source = "skhdrc"; target = ".skhdrc"; }
  { source = "ghostty/config"; target = ".config/ghostty/config"; }
]
`
	specs, err := ParseLinksNix(content)
	if err != nil {
		t.Fatalf("ParseLinksNix error: %v", err)
	}
	if len(specs) != 3 {
		t.Fatalf("len(specs) = %d, want 3", len(specs))
	}

	expected := []LinkSpec{
		{Source: "aerospace.toml", Target: ".aerospace.toml"},
		{Source: "skhdrc", Target: ".skhdrc"},
		{Source: "ghostty/config", Target: ".config/ghostty/config"},
	}

	for i, spec := range specs {
		if spec.Source != expected[i].Source {
			t.Errorf("specs[%d].Source = %q, want %q", i, spec.Source, expected[i].Source)
		}
		if spec.Target != expected[i].Target {
			t.Errorf("specs[%d].Target = %q, want %q", i, spec.Target, expected[i].Target)
		}
	}
}

func TestAddLinkToLinksNix(t *testing.T) {
	content := `[
  { source = "aerospace.toml"; target = ".aerospace.toml"; }
]`

	newContent, err := AddLinkToLinksNix(content, "newfile", ".newfile")
	if err != nil {
		t.Fatalf("AddLinkToLinksNix error: %v", err)
	}

	if !strings.Contains(newContent, `{ source = "newfile"; target = ".newfile"; }`) {
		t.Errorf("newContent does not contain new entry: %s", newContent)
	}

	if !strings.Contains(newContent, `{ source = "aerospace.toml"; target = ".aerospace.toml"; }`) {
		t.Errorf("newContent does not contain old entry: %s", newContent)
	}
}

func TestAddLinkToLinksNixDuplicate(t *testing.T) {
	content := `[
  { source = "aerospace.toml"; target = ".aerospace.toml"; }
]`

	newContent, err := AddLinkToLinksNix(content, "aerospace.toml", ".different")
	if err != nil {
		t.Fatalf("AddLinkToLinksNix error: %v", err)
	}

	// Should not add duplicate
	if newContent != content {
		t.Errorf("content changed when adding duplicate: %s", newContent)
	}
}

func TestPlanIngestUsesHomeRelativePortablePaths(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	source := filepath.Join(home, ".config", "zellij", "config.kdl")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("config\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "dotfiles"), 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := (App{Root: root}).PlanIngest(source)
	if err != nil {
		t.Fatal(err)
	}
	if result.RelSource != ".config/zellij/config.kdl" {
		t.Fatalf("RelSource = %q", result.RelSource)
	}
	if result.RelTarget != ".config/zellij/config.kdl" {
		t.Fatalf("RelTarget = %q", result.RelTarget)
	}
	if result.DotfilesPath != filepath.Join(root, "dotfiles", ".config", "zellij", "config.kdl") {
		t.Fatalf("DotfilesPath = %q", result.DotfilesPath)
	}
}

func TestApplyIngestCreatesDotfilesParentAndMovesSource(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	source := filepath.Join(home, ".config", "zellij", "config.kdl")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("config\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "links.nix"), []byte("[\n]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	app := App{Root: root}
	result, err := app.PlanIngest(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.ApplyIngest(result, io.Discard); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Lstat(result.DotfilesPath); err != nil {
		t.Fatalf("dotfiles destination missing: %v", err)
	}
	target, err := os.Readlink(source)
	if err != nil {
		t.Fatalf("source was not recreated as symlink: %v", err)
	}
	if target != result.DotfilesPath {
		t.Fatalf("symlink target = %q, want %q", target, result.DotfilesPath)
	}
	content, err := os.ReadFile(filepath.Join(root, "links.nix"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), `{ source = ".config/zellij/config.kdl"; target = ".config/zellij/config.kdl"; }`) {
		t.Fatalf("links.nix missing portable entry:\n%s", content)
	}
}

func homeForTest(t *testing.T) string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	return home
}
