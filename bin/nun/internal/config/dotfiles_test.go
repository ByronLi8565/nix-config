package config

import (
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

func homeForTest(t *testing.T) string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	return home
}
