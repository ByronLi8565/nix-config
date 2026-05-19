package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type DotfileLink struct {
	Source string
	Target string
	Action string
	Backup string
}

type LinkSpec struct {
	Source string
	Target string
}

// ParseLinksNix reads the links.nix file and extracts source/target pairs
func ParseLinksNix(content string) ([]LinkSpec, error) {
	var specs []LinkSpec
	
	// Regex to match { source = "..."; target = "..."; }
	entryRe := regexp.MustCompile(`\{\s*source\s*=\s*"([^"]+)"\s*;\s*target\s*=\s*"([^"]+)"\s*;\s*\}`)
	
	matches := entryRe.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) == 3 {
			specs = append(specs, LinkSpec{
				Source: match[1],
				Target: match[2],
			})
		}
	}
	
	return specs, nil
}

// ReadLinksNix reads the links.nix file from the repo root
func (a App) ReadLinksNix() ([]LinkSpec, string, error) {
	root, err := a.repoRoot()
	if err != nil {
		return nil, "", err
	}
	
	linksPath := filepath.Join(root, "links.nix")
	content, err := os.ReadFile(linksPath)
	if err != nil {
		return nil, "", fmt.Errorf("reading links.nix: %w", err)
	}
	
	specs, err := ParseLinksNix(string(content))
	if err != nil {
		return nil, "", fmt.Errorf("parsing links.nix: %w", err)
	}
	
	return specs, string(content), nil
}

func (a App) PlanDotfileLinks() ([]DotfileLink, error) {
	root, err := a.repoRoot()
	if err != nil {
		return nil, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	
	specs, _, err := a.ReadLinksNix()
	if err != nil {
		return nil, err
	}
	
	links := make([]DotfileLink, 0, len(specs))
	for _, spec := range specs {
		source := filepath.Join(root, "dotfiles", spec.Source)
		target := filepath.Join(home, spec.Target)
		action := "link"
		backup := ""
		if current, err := os.Readlink(target); err == nil {
			if current == source {
				action = "already linked"
			} else {
				action = "replace symlink"
				backup = target + ".backup"
			}
		} else if _, err := os.Lstat(target); err == nil {
			action = "backup and link"
			backup = target + ".backup"
		} else if !os.IsNotExist(err) {
			return nil, err
		}
		links = append(links, DotfileLink{
			Source: source,
			Target: target,
			Action: action,
			Backup: backup,
		})
	}
	return links, nil
}

func (a App) ApplyDotfileLinks(links []DotfileLink, out io.Writer) error {
	suffix := time.Now().Format("20060102-150405")
	for _, link := range links {
		if link.Action == "already linked" {
			fmt.Fprintf(out, "unchanged %s\n", displayHomePath(link.Target))
			continue
		}
		if err := os.MkdirAll(filepath.Dir(link.Target), 0o755); err != nil {
			return err
		}
		if link.Backup != "" {
			backup := link.Backup + "-" + suffix
			if err := os.Rename(link.Target, backup); err != nil {
				return err
			}
			fmt.Fprintf(out, "backed up %s -> %s\n", displayHomePath(link.Target), displayHomePath(backup))
		}
		if err := os.Symlink(link.Source, link.Target); err != nil {
			return err
		}
		fmt.Fprintf(out, "linked %s -> %s\n", displayHomePath(link.Target), link.Source)
	}
	return nil
}

func DotfilePlanItems(links []DotfileLink) []string {
	items := make([]string, len(links))
	for i, link := range links {
		item := link.Action + " " + displayHomePath(link.Target) + " -> " + link.Source
		if link.Backup != "" {
			item += " (backup existing file)"
		}
		items[i] = item
	}
	return items
}

func displayHomePath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == home {
		return "~"
	}
	prefix := home + string(os.PathSeparator)
	if strings.HasPrefix(path, prefix) {
		return "~/" + strings.TrimPrefix(path, prefix)
	}
	return path
}

// IngestResult contains the result of an ingest operation
type IngestResult struct {
	SourcePath    string
	DotfilesPath  string
	TargetPath    string
	LinksNixPath  string
	BackupPath    string
}

// PlanIngest plans the ingestion of a file into the dotfiles directory
func (a App) PlanIngest(filePath string) (IngestResult, error) {
	root, err := a.repoRoot()
	if err != nil {
		return IngestResult{}, err
	}
	
	home, err := os.UserHomeDir()
	if err != nil {
		return IngestResult{}, err
	}
	
	// Resolve absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return IngestResult{}, fmt.Errorf("resolving path: %w", err)
	}
	
	// Check file exists
	info, err := os.Stat(absPath)
	if err != nil {
		return IngestResult{}, fmt.Errorf("accessing file: %w", err)
	}
	if info.IsDir() {
		return IngestResult{}, fmt.Errorf("cannot ingest directories, only files")
	}
	
	// Determine the relative path within home
	if !strings.HasPrefix(absPath, home) {
		return IngestResult{}, fmt.Errorf("file must be within home directory: %s", absPath)
	}
	
	relPath := strings.TrimPrefix(absPath, home+string(filepath.Separator))
	
	// Calculate destination in dotfiles
	dotfilesPath := filepath.Join(root, "dotfiles", relPath)
	
	// Calculate the target path (where the symlink will be created)
	targetPath := absPath
	
	// Check if already in dotfiles
	if strings.HasPrefix(absPath, filepath.Join(root, "dotfiles")) {
		return IngestResult{}, fmt.Errorf("file is already in dotfiles directory")
	}
	
	return IngestResult{
		SourcePath:   absPath,
		DotfilesPath: dotfilesPath,
		TargetPath:   targetPath,
		LinksNixPath: filepath.Join(root, "links.nix"),
	}, nil
}

// AddLinkToLinksNix adds a new link entry to links.nix
func AddLinkToLinksNix(content, source, target string) (string, error) {
	// Find the closing bracket of the list
	closeIdx := strings.LastIndex(content, "]")
	if closeIdx < 0 {
		return "", fmt.Errorf("could not find list closing bracket")
	}
	
	// Create the new entry
	indent := "  "
	newEntry := fmt.Sprintf("%s{ source = \"%s\"; target = \"%s\"; }", indent, source, target)
	
	// Check if entry already exists
	if strings.Contains(content, fmt.Sprintf(`source = "%s"`, source)) {
		return content, nil
	}
	
	// Insert before the closing bracket
	before := content[:closeIdx]
	after := content[closeIdx:]
	
	// Add newline before entry if there isn't one
	if !strings.HasSuffix(before, "\n") {
		before += "\n"
	}
	
	return before + newEntry + "\n" + after, nil
}

// ApplyIngest applies the ingest operation
func (a App) ApplyIngest(result IngestResult, out io.Writer) error {
	// 1. Read current links.nix content
	linksContent, err := os.ReadFile(result.LinksNixPath)
	if err != nil {
		return fmt.Errorf("reading links.nix: %w", err)
	}
	
	// 2. Calculate relative paths
	root, err := a.repoRoot()
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	
	relSource := strings.TrimPrefix(result.DotfilesPath, filepath.Join(root, "dotfiles")+string(filepath.Separator))
	relTarget := strings.TrimPrefix(result.TargetPath, home+string(filepath.Separator))
	
	// 3. Backup existing file if needed
	if _, err := os.Lstat(result.TargetPath); err == nil {
		suffix := time.Now().Format("20060102-150405")
		backupPath := result.TargetPath + ".backup-" + suffix
		if err := os.Rename(result.TargetPath, backupPath); err != nil {
			return fmt.Errorf("backing up existing file: %w", err)
		}
		fmt.Fprintf(out, "backed up %s -> %s\n", displayHomePath(result.TargetPath), displayHomePath(backupPath))
	}
	
	// 4. Create dotfiles subdirectory if needed
	dotfilesDir := filepath.Dir(result.DotfilesPath)
	if err := os.MkdirAll(dotfilesDir, 0o755); err != nil {
		return fmt.Errorf("creating dotfiles directory: %w", err)
	}
	
	// 5. Move file to dotfiles
	if err := os.Rename(result.SourcePath, result.DotfilesPath); err != nil {
		return fmt.Errorf("moving file to dotfiles: %w", err)
	}
	fmt.Fprintf(out, "moved %s -> %s\n", displayHomePath(result.SourcePath), result.DotfilesPath)
	
	// 6. Update links.nix
	newContent, err := AddLinkToLinksNix(string(linksContent), relSource, relTarget)
	if err != nil {
		return fmt.Errorf("updating links.nix: %w", err)
	}
	if err := os.WriteFile(result.LinksNixPath, []byte(newContent), 0o644); err != nil {
		return fmt.Errorf("writing links.nix: %w", err)
	}
	fmt.Fprintf(out, "updated links.nix: added { source = \"%s\"; target = \"%s\"; }\n", relSource, relTarget)
	
	// 7. Create symlink
	if err := os.Symlink(result.DotfilesPath, result.TargetPath); err != nil {
		return fmt.Errorf("creating symlink: %w", err)
	}
	fmt.Fprintf(out, "linked %s -> %s\n", displayHomePath(result.TargetPath), result.DotfilesPath)
	
	return nil
}
