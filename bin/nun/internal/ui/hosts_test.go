package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"nun/internal/config"
)

func TestHostsNewSpaceTogglesPackageSet(t *testing.T) {
	m := initialHostModel(config.NewHostDefaults{
		DefaultName:     "test-host",
		DefaultUser:     "spheal",
		DefaultSystem:   "aarch64-darwin",
		PackageSetNames: []string{"development", "global"},
	})
	m.step = hostPackageSets

	updated, _ := m.updatePackageSets(tea.KeyPressMsg{Code: tea.KeySpace})
	got := updated.(hostModel)
	if !got.selectedSets["development"] {
		t.Fatalf("space did not toggle selected package set")
	}

	updated, _ = got.updatePackageSets(tea.KeyPressMsg{Code: tea.KeySpace})
	got = updated.(hostModel)
	if got.selectedSets["development"] {
		t.Fatalf("second space did not untoggle selected package set")
	}
}
