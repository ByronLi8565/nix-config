package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanInstallHostPackageConvertsImportToHostList(t *testing.T) {
	root := t.TempDir()
	host, _ := os.Hostname()
	hostDir := filepath.Join(root, "hosts", host)
	if err := os.MkdirAll(hostDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hostDir, "packages.nix"), []byte(`{pkgs, ...}: {
  home-manager.users.spheal.home.packages =
    import ../../package-sets/development.nix pkgs;
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := (App{Root: root}).PlanInstall(InstallRequest{
		Kind:     InstallNix,
		Packages: []string{"ripgrep"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Writes) != 1 {
		t.Fatalf("writes = %d, want 1", len(plan.Writes))
	}
	got := plan.Writes[0].Content
	for _, want := range []string{"builtins.concatLists", "import ../../package-sets/development.nix pkgs", "(with pkgs; [", "ripgrep"} {
		if !strings.Contains(got, want) {
			t.Fatalf("planned host packages missing %q:\n%s", want, got)
		}
	}
}

func TestPlanInstallGlobalCaskAddsCustomTapWiring(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "package-sets", "darwin.nix"), `pkgs: {
  homebrewBrews = [];
  homebrewCasks = [
  ];
}

func TestTryListPathUsesNunTrialsJSON(t *testing.T) {
	root := t.TempDir()
	path, err := (App{Root: root}).tryListPath()
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(root, "nun-trials.json") {
		t.Fatalf("try list path = %q", path)
	}
}

func TestProfileDryRunArgs(t *testing.T) {
	got := ProfileDryRunArgs("/tmp/nix-config", "spheal-mbp")
	want := []string{
		"darwin-rebuild",
		"build",
		"--dry-run",
		"--flake",
		"/tmp/nix-config#spheal-mbp",
	}
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("args = %#v, want %#v", got, want)
	}
}
`)
	mustWrite(t, filepath.Join(root, "flake.nix"), `{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  };

  outputs = inputs: {};
}
`)
	mustWrite(t, filepath.Join(root, "modules", "darwin", "homebrew.nix"), `{
  config,
  homebrew-cask,
  homebrew-core,
  ...
}: let
in {
  nix-homebrew = {
    taps = {
      "homebrew/homebrew-core" = homebrew-core;
      "homebrew/homebrew-cask" = homebrew-cask;
    };
  };
}
`)

	plan, err := (App{Root: root}).PlanInstall(InstallRequest{
		Kind:       InstallCask,
		PackageSet: "darwin",
		Packages:   []string{"owner/tools/example"},
	})
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{}
	for _, write := range plan.Writes {
		contents[write.RelativePath] = write.Content
	}
	if !strings.Contains(contents["package-sets/darwin.nix"], `"owner/tools/example"`) {
		t.Fatalf("darwin package set missing cask:\n%s", contents["package-sets/darwin.nix"])
	}
	if !strings.Contains(contents["flake.nix"], `owner-tools = {`) || !strings.Contains(contents["flake.nix"], `github:owner/homebrew-tools`) {
		t.Fatalf("flake missing tap input:\n%s", contents["flake.nix"])
	}
	if !strings.Contains(contents["modules/darwin/homebrew.nix"], `"owner/tools" = owner-tools;`) {
		t.Fatalf("homebrew module missing tap binding:\n%s", contents["modules/darwin/homebrew.nix"])
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
