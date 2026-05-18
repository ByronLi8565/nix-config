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
