package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"nun/internal/nixexpr"
)

// HomebrewPackage represents a brew or cask package
type HomebrewPackage struct {
	Name       string
	Kind       InstallKind // InstallBrew or InstallCask
	Host       string      // Host name (empty for global)
	IsGlobal   bool
}

// AllPackages returns all packages (nix + homebrew) for the current configuration
func (a App) AllPackages() ([]PackageEntry, []HomebrewPackage, error) {
	nixPackages, err := a.ReadPackageSets()
	if err != nil {
		return nil, nil, err
	}

	root, err := a.repoRoot()
	if err != nil {
		return nil, nil, err
	}

	host, _ := os.Hostname()
	hosts, _ := a.HostNames()

	var brewPackages []HomebrewPackage

	// Read host homebrew files
	for _, h := range hosts {
		homebrewPath := filepath.Join(root, "hosts", h, "homebrew.nix")
		data, err := os.ReadFile(homebrewPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, nil, err
		}
		source := string(data)

		// Parse brews
		brews, _, err := nixexpr.ParseNamedList(source, "brews")
		if err == nil {
			for _, name := range brews {
				brewPackages = append(brewPackages, HomebrewPackage{
					Name:     name,
					Kind:     InstallBrew,
					Host:     h,
					IsGlobal: false,
				})
			}
		}

		// Parse casks
		casks, _, err := nixexpr.ParseNamedList(source, "casks")
		if err == nil {
			for _, name := range casks {
				brewPackages = append(brewPackages, HomebrewPackage{
					Name:     name,
					Kind:     InstallCask,
					Host:     h,
					IsGlobal: false,
				})
			}
		}
	}

	// Read global darwin.nix homebrew lists
	darwinPath := filepath.Join(root, "package-sets", "darwin.nix")
	if data, err := os.ReadFile(darwinPath); err == nil {
		source := string(data)

		brews, _, err := nixexpr.ParseNamedList(source, "homebrewBrews")
		if err == nil {
			for _, name := range brews {
				brewPackages = append(brewPackages, HomebrewPackage{
					Name:     name,
					Kind:     InstallBrew,
					Host:     "",
					IsGlobal: true,
				})
			}
		}

		casks, _, err := nixexpr.ParseNamedList(source, "homebrewCasks")
		if err == nil {
			for _, name := range casks {
				brewPackages = append(brewPackages, HomebrewPackage{
					Name:     name,
					Kind:     InstallCask,
					Host:     "",
					IsGlobal: true,
				})
			}
		}
	}

	_ = host
	return nixPackages, brewPackages, nil
}

type RemoveTarget struct {
	Kind       InstallKind
	Package    string
	PackageSet string
	Host       string
	IsGlobal   bool
}

type RemovePlan struct {
	Targets []RemoveTarget
	Writes  []PendingWrite
}

type RemoveRequest struct {
	Packages    []string
	Interactive bool
}

// PlanRemove creates a plan to remove packages from package lists
func (a App) PlanRemove(req RemoveRequest) (RemovePlan, error) {
	root, err := a.repoRoot()
	if err != nil {
		return RemovePlan{}, err
	}

	// Get all packages to find where they are located
	nixEntries, brewPackages, err := a.AllPackages()
	if err != nil {
		return RemovePlan{}, err
	}

	// Build lookup map
	type packageKey struct {
		set  string
		name string
	}
	
	// Map from "set/name" or just "name" to target info
	lookup := make(map[string]struct {
		kind     InstallKind
		pkgSet   string
		host     string
		isGlobal bool
	})

	// Add nix packages
	for _, entry := range nixEntries {
		key := entry.Set + "/" + entry.Name
		lookup[key] = struct {
			kind     InstallKind
			pkgSet   string
			host     string
			isGlobal bool
		}{InstallNix, entry.Set, "", false}
		// Also by name only
		if _, exists := lookup[entry.Name]; !exists {
			lookup[entry.Name] = struct {
				kind     InstallKind
				pkgSet   string
				host     string
				isGlobal bool
			}{InstallNix, entry.Set, "", false}
		}
	}

	// Add brew packages
	for _, bp := range brewPackages {
		key := bp.Name
		if bp.IsGlobal {
			key = "darwin/" + bp.Name
		} else if bp.Host != "" {
			key = bp.Host + "/" + bp.Name
		}
		lookup[key] = struct {
			kind     InstallKind
			pkgSet   string
			host     string
			isGlobal bool
		}{bp.Kind, "", bp.Host, bp.IsGlobal}
		// Also by name only
		if _, exists := lookup[bp.Name]; !exists {
			lookup[bp.Name] = struct {
				kind     InstallKind
				pkgSet   string
				host     string
				isGlobal bool
			}{bp.Kind, "", bp.Host, bp.IsGlobal}
		}
	}

	var targets []RemoveTarget
	for _, pkg := range req.Packages {
		info, found := lookup[pkg]
		if !found {
			return RemovePlan{}, fmt.Errorf("package %q not found in any package set or homebrew config", pkg)
		}

		target := RemoveTarget{
			Kind:     info.kind,
			Package:  pkg,
			Host:     info.host,
			IsGlobal: info.isGlobal,
		}
		
		if info.kind == InstallNix {
			target.PackageSet = info.pkgSet
			// Extract just the name if fully qualified
			if idx := strings.LastIndex(pkg, "/"); idx >= 0 {
				target.Package = pkg[idx+1:]
			}
		}
		
		targets = append(targets, target)
	}

	return a.buildRemovePlan(root, targets)
}

// buildRemovePlan creates the write operations needed to remove targets
func (a App) buildRemovePlan(root string, targets []RemoveTarget) (RemovePlan, error) {
	files := map[string]string{}
	var writes []PendingWrite

	for _, target := range targets {
		path, rel, err := a.removeLocation(root, target)
		if err != nil {
			return RemovePlan{}, err
		}

		source, ok := files[path]
		if !ok {
			data, err := os.ReadFile(path)
			if err != nil {
				return RemovePlan{}, err
			}
			source = string(data)
		}

		next, err := a.removePackageFromSource(source, target)
		if err != nil {
			return RemovePlan{}, fmt.Errorf("removing %s from %s: %w", target.Package, rel, err)
		}

		files[path] = next
	}

	// Sort paths for consistent output
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

	return RemovePlan{Targets: targets, Writes: writes}, nil
}

// removeLocation determines which file a package should be removed from
func (a App) removeLocation(root string, target RemoveTarget) (path, rel string, err error) {
	switch target.Kind {
	case InstallNix:
		if target.PackageSet != "" {
			rel = filepath.Join("package-sets", target.PackageSet+".nix")
			return filepath.Join(root, rel), rel, nil
		}
		return "", "", fmt.Errorf("package set must be specified for nix package removal")
	case InstallBrew, InstallCask:
		if target.IsGlobal {
			rel = filepath.Join("package-sets", "darwin.nix")
			return filepath.Join(root, rel), rel, nil
		}
		if target.Host != "" {
			rel = filepath.Join("hosts", target.Host, "homebrew.nix")
			return filepath.Join(root, rel), rel, nil
		}
		return "", "", fmt.Errorf("host must be specified for brew/cask removal")
	default:
		return "", "", fmt.Errorf("unknown install kind %q", target.Kind)
	}
}

// removePackageFromSource removes a package from Nix source code
func (a App) removePackageFromSource(source string, target RemoveTarget) (string, error) {
	switch target.Kind {
	case InstallNix:
		if target.PackageSet == "" {
			return "", fmt.Errorf("cannot remove from host packages yet")
		}
		return nixexpr.RemoveListItem(source, target.Package)
	case InstallBrew:
		if target.IsGlobal {
			return nixexpr.RemoveNamedListItem(source, "homebrewBrews", target.Package)
		}
		return nixexpr.RemoveNamedListItem(source, "brews", target.Package)
	case InstallCask:
		if target.IsGlobal {
			return nixexpr.RemoveNamedListItem(source, "homebrewCasks", target.Package)
		}
		return nixexpr.RemoveNamedListItem(source, "casks", target.Package)
	default:
		return "", fmt.Errorf("unknown install kind %q", target.Kind)
	}
}

// ApplyRemove applies the removal plan
func (a App) ApplyRemove(plan RemovePlan, out io.Writer) error {
	return a.ApplyWrites(plan.Writes, out)
}

// FindPackage searches for a package across all package sets
func (a App) FindPackage(name string) ([]PackageEntry, error) {
	entries, err := a.ReadPackageSets()
	if err != nil {
		return nil, err
	}

	var matches []PackageEntry
	for _, entry := range entries {
		if entry.Name == name {
			matches = append(matches, entry)
		}
	}
	return matches, nil
}
