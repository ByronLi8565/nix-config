package nixexpr

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseTopLevelList(t *testing.T) {
	source := `pkgs:
with pkgs; [
  jq
  # comment
  (callPackage ../bin/nun {})
  ripgrep
]`

	got, list, err := ParseTopLevelList(source)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"jq", "(callPackage ../bin/nun {})", "ripgrep"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("packages = %#v, want %#v", got, want)
	}
	if list.OpenLine != 1 || list.CloseLine != 6 || list.Indent != "  " {
		t.Fatalf("list range = %#v", list)
	}
}

func TestAddListItem(t *testing.T) {
	source := "pkgs:\nwith pkgs; [\n  jq\n]\n"
	got, err := AddListItem(source, "ripgrep")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "\n  ripgrep\n]\n") {
		t.Fatalf("updated source did not insert before close bracket:\n%s", got)
	}
}

func TestRemoveListItem(t *testing.T) {
	source := "pkgs:\nwith pkgs; [\n  jq\n  ripgrep\n]\n"
	got, err := RemoveListItem(source, "jq")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "\n  jq\n") {
		t.Fatalf("updated source still contains removed package:\n%s", got)
	}
}

func TestRemoveNamedListItem(t *testing.T) {
	source := `{
  homebrew = {
    casks = [
      "docker-desktop"
      "unity-hub"
      "steam"
    ];

    brews = [
      "tcl-tk"
    ];
  };
}`
	got, err := RemoveNamedListItem(source, "casks", "unity-hub")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, `"unity-hub"`) {
		t.Fatalf("updated source still contains removed cask:\n%s", got)
	}
	if !strings.Contains(got, `"docker-desktop"`) {
		t.Fatalf("updated source removed wrong cask:\n%s", got)
	}
	if !strings.Contains(got, `"steam"`) {
		t.Fatalf("updated source removed wrong cask:\n%s", got)
	}
	if !strings.Contains(got, `"tcl-tk"`) {
		t.Fatalf("updated source affected brews:\n%s", got)
	}
}

func TestRemoveListItemAfter(t *testing.T) {
	source := `(with pkgs; [
  jq
  ripgrep
])`
	got, err := RemoveListItemAfter(source, "(with pkgs; [", "jq")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "jq") {
		t.Fatalf("updated source still contains removed package:\n%s", got)
	}
	if !strings.Contains(got, "ripgrep") {
		t.Fatalf("updated source removed wrong package:\n%s", got)
	}
}

func TestRemoveListItemWithComma(t *testing.T) {
	source := `pkgs:
with pkgs; [
  jq
  ripgrep
  bat
]`
	got, err := RemoveListItem(source, "ripgrep")
	if err != nil {
		t.Fatal(err)
	}
	// Check that ripgrep is removed
	if strings.Contains(got, "ripgrep") {
		t.Fatalf("updated source still contains removed package:\n%s", got)
	}
	// Check that other packages are still there
	if !strings.Contains(got, "jq") {
		t.Fatalf("updated source removed jq:\n%s", got)
	}
	if !strings.Contains(got, "bat") {
		t.Fatalf("updated source removed bat:\n%s", got)
	}
}
