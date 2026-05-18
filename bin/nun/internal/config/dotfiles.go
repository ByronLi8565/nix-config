package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type DotfileLink struct {
	Source string
	Target string
	Action string
	Backup string
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
	specs := []struct {
		source string
		target string
	}{
		{"dotfiles/aerospace.toml", ".aerospace.toml"},
		{"dotfiles/skhdrc", ".skhdrc"},
		{"dotfiles/yabairc", ".yabairc"},
		{"dotfiles/ghostty/config", ".config/ghostty/config"},
		{"dotfiles/nvim/init.lua", ".config/nvim/init.lua"},
	}
	links := make([]DotfileLink, 0, len(specs))
	for _, spec := range specs {
		source := filepath.Join(root, spec.source)
		target := filepath.Join(home, spec.target)
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
