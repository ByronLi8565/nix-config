package config

import (
	"strings"
	"testing"
)

func TestRenderHostPackagesParenthesizesImports(t *testing.T) {
	got := RenderHostPackages("byron", []string{"darwin", "global"})
	for _, want := range []string{
		"(import ../../package-sets/darwin.nix pkgs).nixPackages",
		"(import ../../package-sets/global.nix pkgs)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("rendered packages missing %q:\n%s", want, got)
		}
	}
}

func TestRenderHostDefaultIncludesConfigRoot(t *testing.T) {
	got := RenderHostDefault(HostDefaultInput{
		Name:       "work-mac",
		User:       "byron",
		System:     "aarch64-darwin",
		ConfigRoot: "/Users/byron/nix-config",
	})
	if !strings.Contains(got, `configRoot = "/Users/byron/nix-config";`) {
		t.Fatalf("rendered host missing configRoot:\n%s", got)
	}
}
