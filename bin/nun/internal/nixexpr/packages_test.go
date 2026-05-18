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
