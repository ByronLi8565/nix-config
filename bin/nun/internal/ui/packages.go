package ui

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"nun/internal/config"
)

type packageItem struct {
	entry config.PackageEntry
}

func (i packageItem) FilterValue() string { return i.entry.Name + " " + i.entry.Set }

type packageDelegate struct{}

func (d packageDelegate) Height() int                         { return 1 }
func (d packageDelegate) Spacing() int                        { return 0 }
func (d packageDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }
func (d packageDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	pkg, ok := item.(packageItem)
	if !ok {
		return
	}
	name := pkg.entry.Name
	set := subtleStyle.Render(pkg.entry.Set)
	if index == m.Index() {
		fmt.Fprintf(w, "%s %s %s", selectedStyle.Render(">"), selectedStyle.Render(name), set)
		return
	}
	fmt.Fprintf(w, "  %s %s", name, set)
}

type packageModel struct {
	list list.Model
}

func BrowsePackages(entries []config.PackageEntry) error {
	items := make([]list.Item, len(entries))
	for i, entry := range entries {
		items[i] = packageItem{entry: entry}
	}
	l := list.New(items, packageDelegate{}, 0, 0)
	l.Title = "nun packages"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(true)
	l.SetShowPagination(true)
	l.Styles.Title = titleStyle
	_, err := tea.NewProgram(packageModel{list: l}).Run()
	return err
}

func (m packageModel) Init() tea.Cmd {
	return nil
}

func (m packageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height)
	case tea.KeyPressMsg:
		switch strings.ToLower(msg.String()) {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m packageModel) View() tea.View {
	return tea.NewView(m.list.View())
}

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	subtleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)
