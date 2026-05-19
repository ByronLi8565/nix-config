package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"nun/internal/nixexpr"
)

type InstallKind string

const (
	InstallNix  InstallKind = "nix"
	InstallBrew InstallKind = "brew"
	InstallCask InstallKind = "cask"
)

type InstallRequest struct {
	Kind        InstallKind
	Packages    []string
	Targets     []InstallTarget
	PackageSet  string
	Interactive bool
	FromTryList bool
}

type InstallTarget struct {
	Kind       InstallKind
	Package    string
	PackageSet string
}

type InstallDefaults struct {
	CurrentHost     string
	PackageSetNames []string
	TryPackages     []InstallTarget
}

type InstallPlan struct {
	Targets []InstallTarget
	Writes  []PendingWrite
	Notes   []string
}

func (a App) InstallDefaults() (InstallDefaults, error) {
	root, err := a.repoRoot()
	if err != nil {
		return InstallDefaults{}, err
	}
	host, _ := os.Hostname()
	sets, err := nixexpr.ReadPackageSetNames(filepath.Join(root, "package-sets"))
	if err != nil {
		return InstallDefaults{}, err
	}
	try, err := a.ReadTryList()
	if err != nil {
		return InstallDefaults{}, err
	}
	return InstallDefaults{CurrentHost: host, PackageSetNames: sets, TryPackages: try}, nil
}

func (a App) TryPackages(kind InstallKind, packages []string) error {
	targets := make([]InstallTarget, len(packages))
	for i, pkg := range packages {
		targets[i] = InstallTarget{Kind: kind, Package: pkg}
	}
	if err := a.installTemporary(targets); err != nil {
		return err
	}
	return a.AddTryPackages(targets)
}

func (a App) TryProfile(host string) error {
	root, err := a.repoRoot()
	if err != nil {
		return err
	}
	hosts, err := a.HostNames()
	if err != nil {
		return err
	}
	found := false
	for _, name := range hosts {
		if name == host {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("unknown host %q", host)
	}
	return run(root, nil, ProfileDryRunArgs(root, host)...)
}

func ProfileDryRunArgs(root, host string) []string {
	return []string{
		"nix",
		"build",
		"--dry-run",
		"path:" + root + "#darwinConfigurations." + host + ".system",
		"--accept-flake-config",
		"--extra-experimental-features",
		"pipe-operators",
	}
}

func (a App) PlanInstall(req InstallRequest) (InstallPlan, error) {
	root, err := a.repoRoot()
	if err != nil {
		return InstallPlan{}, err
	}
	targets := make([]InstallTarget, 0, len(req.Packages))
	if len(req.Targets) > 0 {
		targets = append(targets, req.Targets...)
	} else if req.FromTryList {
		try, err := a.ReadTryList()
		if err != nil {
			return InstallPlan{}, err
		}
		targets = append(targets, try...)
	} else {
		for _, pkg := range req.Packages {
			targets = append(targets, InstallTarget{Kind: req.Kind, Package: pkg, PackageSet: req.PackageSet})
		}
	}
	if len(targets) == 0 {
		return InstallPlan{}, fmt.Errorf("no packages to install")
	}

	files := map[string]string{}
	var writes []PendingWrite
	var notes []string
	for _, target := range targets {
		path, rel, listName, err := a.installLocation(root, target)
		if err != nil {
			return InstallPlan{}, err
		}
		source, ok := files[path]
		if !ok {
			data, err := os.ReadFile(path)
			if err != nil {
				return InstallPlan{}, err
			}
			source = string(data)
		}
		next := source
		switch target.Kind {
		case InstallNix:
			if target.PackageSet == "" {
				next, err = addHostPackage(source, target.Package)
			} else {
				next, err = nixexpr.AddListItem(source, target.Package)
			}
		case InstallBrew, InstallCask:
			next, err = nixexpr.AddNamedListItem(source, listName, nixString(target.Package))
			if err == nil {
				var tapNotes []string
				tapNotes, err = ensureHomebrewTap(root, files, target.Package)
				notes = append(notes, tapNotes...)
			}
		default:
			err = fmt.Errorf("unknown install kind %q", target.Kind)
		}
		if err != nil {
			return InstallPlan{}, err
		}
		files[path] = next
		_ = rel
	}

	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		writes = append(writes, PendingWrite{
			Path:         path,
			RelativePath: mustRel(root, path),
			Content:      files[path],
		})
	}
	return InstallPlan{Targets: targets, Writes: writes, Notes: unique(notes)}, nil
}

func (a App) ApplyInstall(plan InstallPlan, out io.Writer) error {
	if err := a.installTemporary(plan.Targets); err != nil {
		return err
	}
	if err := a.ApplyWrites(plan.Writes, out); err != nil {
		return err
	}
	return a.RemoveTryPackages(plan.Targets)
}

func PrintInstallPlan(out io.Writer, plan InstallPlan) {
	fmt.Fprintln(out, "\nPlan:")
	fmt.Fprintln(out, "  Temporarily install:")
	for _, target := range plan.Targets {
		fmt.Fprintf(out, "    %s %s\n", target.Kind, target.Package)
	}
	fmt.Fprintln(out, "  Files to modify:")
	for _, write := range plan.Writes {
		fmt.Fprintf(out, "    %s\n", write.RelativePath)
	}
	for _, note := range plan.Notes {
		fmt.Fprintf(out, "  Note: %s\n", note)
	}
	fmt.Fprintln(out)
}

func (a App) installLocation(root string, target InstallTarget) (path, rel, listName string, err error) {
	switch target.Kind {
	case InstallNix:
		if target.PackageSet != "" {
			rel = filepath.Join("package-sets", target.PackageSet+".nix")
			return filepath.Join(root, rel), rel, "", nil
		}
		host, _ := os.Hostname()
		rel = filepath.Join("hosts", host, "packages.nix")
		return filepath.Join(root, rel), rel, "", nil
	case InstallBrew:
		if target.PackageSet == "darwin" {
			rel = filepath.Join("package-sets", "darwin.nix")
			return filepath.Join(root, rel), rel, "homebrewBrews", nil
		}
		host, _ := os.Hostname()
		rel = filepath.Join("hosts", host, "homebrew.nix")
		return filepath.Join(root, rel), rel, "brews", nil
	case InstallCask:
		if target.PackageSet == "darwin" {
			rel = filepath.Join("package-sets", "darwin.nix")
			return filepath.Join(root, rel), rel, "homebrewCasks", nil
		}
		host, _ := os.Hostname()
		rel = filepath.Join("hosts", host, "homebrew.nix")
		return filepath.Join(root, rel), rel, "casks", nil
	}
	return "", "", "", fmt.Errorf("unknown install kind %q", target.Kind)
}

func (a App) installTemporary(targets []InstallTarget) error {
	for _, target := range targets {
		switch target.Kind {
		case InstallNix:
			env := []string{"NIXPKGS_ALLOW_UNFREE=1"}
			if err := run("", env, "nix", "profile", "install", "nixpkgs#"+target.Package, "--accept-flake-config", "--impure"); err != nil {
				return err
			}
		case InstallBrew:
			if err := run("", nil, "brew", "install", target.Package); err != nil {
				return err
			}
		case InstallCask:
			if err := run("", nil, "brew", "install", "--cask", target.Package); err != nil {
				return err
			}
		}
	}
	return nil
}

func addHostPackage(source, pkg string) (string, error) {
	if strings.Contains(source, "(with pkgs; [") {
		return nixexpr.AddListItemAfter(source, "(with pkgs; [", pkg)
	}
	idx := strings.Index(source, "home.packages =")
	if idx < 0 {
		return "", fmt.Errorf("could not find host home.packages assignment")
	}
	lineEnd := strings.Index(source[idx:], "\n")
	if lineEnd < 0 {
		return "", fmt.Errorf("could not find host home.packages assignment end")
	}
	assignEnd := idx + lineEnd + 1
	stmtEnd := strings.Index(source[assignEnd:], ";")
	if stmtEnd < 0 {
		return "", fmt.Errorf("could not find host home.packages assignment semicolon")
	}
	expr := strings.TrimSpace(source[assignEnd : assignEnd+stmtEnd])
	replacement := "    builtins.concatLists [\n      " + expr + "\n      (with pkgs; [\n        " + pkg + "\n      ])\n    ]"
	return source[:assignEnd] + replacement + source[assignEnd+stmtEnd:], nil
}

func hostUserFromSource(source string) string {
	const prefix = "home-manager.users."
	idx := strings.Index(source, prefix)
	if idx < 0 {
		return ""
	}
	rest := source[idx+len(prefix):]
	if dot := strings.Index(rest, ".home.packages"); dot >= 0 {
		return rest[:dot]
	}
	return ""
}

func (a App) tryListPath() (string, error) {
	root, err := a.repoRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "nun-trials.json"), nil
}

func (a App) ReadTryList() ([]InstallTarget, error) {
	path, err := a.tryListPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var targets []InstallTarget
	if err := json.Unmarshal(data, &targets); err != nil {
		return nil, err
	}
	return targets, nil
}

func (a App) AddTryPackages(targets []InstallTarget) error {
	existing, err := a.ReadTryList()
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, target := range existing {
		seen[targetKey(target)] = true
	}
	for _, target := range targets {
		if !seen[targetKey(target)] {
			existing = append(existing, target)
		}
	}
	return a.writeTryList(existing)
}

func (a App) RemoveTryPackages(targets []InstallTarget) error {
	existing, err := a.ReadTryList()
	if err != nil {
		return err
	}
	remove := map[string]bool{}
	for _, target := range targets {
		remove[targetKey(target)] = true
	}
	var kept []InstallTarget
	for _, target := range existing {
		if !remove[targetKey(target)] {
			kept = append(kept, target)
		}
	}
	return a.writeTryList(kept)
}

func (a App) writeTryList(targets []InstallTarget) error {
	path, err := a.tryListPath()
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	data, err := json.MarshalIndent(targets, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func targetKey(target InstallTarget) string {
	return string(target.Kind) + "\x00" + target.Package
}

func nixString(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}

func homebrewTap(pkg string) string {
	parts := strings.Split(pkg, "/")
	if len(parts) < 3 {
		return ""
	}
	return parts[0] + "/" + parts[1]
}

func ensureHomebrewTap(root string, files map[string]string, pkg string) ([]string, error) {
	tap := homebrewTap(pkg)
	if tap == "" || tap == "homebrew/homebrew-core" || tap == "homebrew/homebrew-cask" {
		return nil, nil
	}
	inputName := tapInputName(tap)

	flakePath := filepath.Join(root, "flake.nix")
	flakeSource, err := sourceFor(files, flakePath)
	if err != nil {
		return nil, err
	}
	if !strings.Contains(flakeSource, inputName+" = {") {
		next, err := addFlakeTapInput(flakeSource, inputName, tap)
		if err != nil {
			return nil, err
		}
		files[flakePath] = next
	}

	modulePath := filepath.Join(root, "modules", "darwin", "homebrew.nix")
	moduleSource, err := sourceFor(files, modulePath)
	if err != nil {
		return nil, err
	}
	if !strings.Contains(moduleSource, `"`+tap+`" = `+inputName) {
		next, err := addHomebrewTapArg(moduleSource, inputName)
		if err != nil {
			return nil, err
		}
		next, err = addHomebrewTapBinding(next, tap, inputName)
		if err != nil {
			return nil, err
		}
		files[modulePath] = next
	}
	return []string{"custom tap " + tap + " will be wired into flake inputs and nix-homebrew taps"}, nil
}

func sourceFor(files map[string]string, path string) (string, error) {
	if source, ok := files[path]; ok {
		return source, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func tapInputName(tap string) string {
	return strings.NewReplacer("/", "-", ".", "-").Replace(tap)
}

func tapRepoURL(tap string) string {
	parts := strings.Split(tap, "/")
	if len(parts) != 2 {
		return "github:" + tap
	}
	repo := parts[1]
	if !strings.HasPrefix(repo, "homebrew-") {
		repo = "homebrew-" + repo
	}
	return "github:" + parts[0] + "/" + repo
}

func addFlakeTapInput(source, inputName, tap string) (string, error) {
	insert := "\n    " + inputName + ` = {
      url = "` + tapRepoURL(tap) + `";
      flake = false;
    };
`
	marker := "\n  };\n\n  outputs ="
	idx := strings.Index(source, marker)
	if idx < 0 {
		return "", fmt.Errorf("could not find flake inputs block")
	}
	return source[:idx] + insert + source[idx:], nil
}

func addHomebrewTapArg(source, inputName string) (string, error) {
	headerEnd := strings.Index(source, "}: let")
	if headerEnd < 0 {
		return "", fmt.Errorf("could not find homebrew module argument list")
	}
	if strings.Contains(source[:headerEnd], inputName+",") {
		return source, nil
	}
	insertAt := strings.Index(source, "  config,\n")
	if insertAt < 0 || insertAt > headerEnd {
		return "", fmt.Errorf("could not find insertion point in homebrew module arguments")
	}
	return source[:insertAt] + "  " + inputName + ",\n" + source[insertAt:], nil
}

func addHomebrewTapBinding(source, tap, inputName string) (string, error) {
	if strings.Contains(source, `"`+tap+`" = `+inputName) {
		return source, nil
	}
	marker := `      "homebrew/homebrew-core" = homebrew-core;`
	idx := strings.Index(source, marker)
	if idx < 0 {
		return "", fmt.Errorf("could not find nix-homebrew taps block")
	}
	lineEnd := strings.Index(source[idx:], "\n")
	if lineEnd < 0 {
		return "", fmt.Errorf("could not find nix-homebrew taps insertion point")
	}
	insertAt := idx + lineEnd + 1
	return source[:insertAt] + `      "` + tap + `" = ` + inputName + ";\n" + source[insertAt:], nil
}

func mustRel(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}

func unique(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}
