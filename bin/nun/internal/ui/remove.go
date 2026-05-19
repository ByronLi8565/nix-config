package ui

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"nun/internal/config"
)

// RemovePackageEntry represents either a nix package or a homebrew package for removal
type RemovePackageEntry struct {
	Name     string
	Set      string // For nix: package set name; For brew: "brew" or "cask"
	Kind     string // "nix", "brew", or "cask"
	Host     string // For host-specific brew packages
	IsGlobal bool   // For global brew packages in darwin.nix
}

type removeItem struct {
	entry RemovePackageEntry
}

func (i removeItem) FilterValue() string { return i.entry.Name + " " + i.entry.Set }

type removeDelegate struct {
	selected map[string]bool
}

func (d removeDelegate) Height() int                         { return 1 }
func (d removeDelegate) Spacing() int                        { return 0 }
func (d removeDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }
func (d removeDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	pkg, ok := item.(removeItem)
	if !ok {
		return
	}
	name := pkg.entry.Name
	
	// Build display label
	var setLabel string
	if pkg.entry.Kind == "nix" {
		setLabel = pkg.entry.Set
	} else if pkg.entry.IsGlobal {
		setLabel = "darwin/" + pkg.entry.Kind
	} else if pkg.entry.Host != "" {
		setLabel = pkg.entry.Host + "/" + pkg.entry.Kind
	} else {
		setLabel = pkg.entry.Kind
	}
	
	set := subtleStyle.Render(setLabel)
	key := makeRemoveKey(pkg.entry)
	mark := "[ ]"
	if d.selected[key] {
		mark = "[x]"
	}
	line := mark + " " + name + " " + set
	if index == m.Index() {
		fmt.Fprint(w, selectedStyle.Render("> "+line))
		return
	}
	fmt.Fprint(w, "  "+line)
}

func makeRemoveKey(entry RemovePackageEntry) string {
	if entry.Kind == "nix" {
		return entry.Set + "/" + entry.Name
	}
	if entry.IsGlobal {
		return "darwin/" + entry.Name
	}
	if entry.Host != "" {
		return entry.Host + "/" + entry.Name
	}
	return entry.Kind + "/" + entry.Name
}

type removeModel struct {
	list      list.Model
	selected  map[string]bool
	cancelled bool
	aborted   bool
}

// RemoveResult contains the selected packages to remove
type RemoveResult struct {
	Selected  []RemovePackageEntry
	Cancelled bool
	Aborted   bool
}

// SelectPackagesToRemove shows an interactive picker for selecting packages to remove
func SelectPackagesToRemove(nixPackages []config.PackageEntry, brewPackages []config.HomebrewPackage) (RemoveResult, error) {
	// Convert to unified entries
	var allEntries []RemovePackageEntry
	
	for _, entry := range nixPackages {
		allEntries = append(allEntries, RemovePackageEntry{
			Name: entry.Name,
			Set:  entry.Set,
			Kind: "nix",
		})
	}
	
	for _, bp := range brewPackages {
		kind := "brew"
		if bp.Kind == config.InstallCask {
			kind = "cask"
		}
		allEntries = append(allEntries, RemovePackageEntry{
			Name:     bp.Name,
			Set:      kind,
			Kind:     kind,
			Host:     bp.Host,
			IsGlobal: bp.IsGlobal,
		})
	}
	
	// Sort entries
	sort.Slice(allEntries, func(i, j int) bool {
		if allEntries[i].Kind != allEntries[j].Kind {
			return allEntries[i].Kind < allEntries[j].Kind
		}
		if allEntries[i].Set != allEntries[j].Set {
			return allEntries[i].Set < allEntries[j].Set
		}
		return allEntries[i].Name < allEntries[j].Name
	})
	
	items := make([]list.Item, len(allEntries))
	for i, entry := range allEntries {
		items[i] = removeItem{entry: entry}
	}

	selected := make(map[string]bool)
	l := list.New(items, removeDelegate{selected: selected}, 0, 0)
	l.Title = "Select packages to remove"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(true)
	l.SetShowPagination(true)
	l.Styles.Title = titleStyle

	m := removeModel{
		list:     l,
		selected: selected,
	}

	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return RemoveResult{}, err
	}

	model := final.(removeModel)
	if model.aborted {
		return RemoveResult{Aborted: true}, nil
	}
	if model.cancelled {
		return RemoveResult{Cancelled: true}, nil
	}

	// Collect selected items
	var selectedEntries []RemovePackageEntry
	for _, entry := range allEntries {
		key := makeRemoveKey(entry)
		if model.selected[key] {
			selectedEntries = append(selectedEntries, entry)
		}
	}

	return RemoveResult{Selected: selectedEntries}, nil
}

func (m removeModel) Init() tea.Cmd {
	return nil
}

func (m removeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height)
	case tea.KeyPressMsg:
		switch strings.ToLower(msg.String()) {
		case "ctrl+c":
			m.aborted = true
			return m, tea.Quit
		case "q", "esc":
			if m.list.SettingFilter() {
				break
			}
			m.cancelled = true
			return m, tea.Quit
		case "space":
			if item, ok := m.list.SelectedItem().(removeItem); ok {
				key := makeRemoveKey(item.entry)
				m.selected[key] = !m.selected[key]
			}
			return m, nil
		case "enter":
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m removeModel) View() tea.View {
	header := titleStyle.Render("Select packages to remove") + "\n" +
		subtleStyle.Render("space toggles, / filters, enter confirms, esc/q cancels") + "\n\n"
	return tea.NewView(header + m.list.View())
}
