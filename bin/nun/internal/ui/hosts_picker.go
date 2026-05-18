package ui

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

type hostItem string

func (i hostItem) FilterValue() string { return string(i) }

type hostDelegate struct{}

func (d hostDelegate) Height() int                         { return 1 }
func (d hostDelegate) Spacing() int                        { return 0 }
func (d hostDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }
func (d hostDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	host, ok := item.(hostItem)
	if !ok {
		return
	}
	name := string(host)
	if index == m.Index() {
		fmt.Fprintf(w, "%s %s", selectedStyle.Render(">"), selectedStyle.Render(name))
		return
	}
	fmt.Fprintf(w, "  %s", name)
}

type hostPickerModel struct {
	list      list.Model
	selected  string
	cancelled bool
}

func PickHost(hosts []string) (string, error) {
	items := make([]list.Item, len(hosts))
	for i, host := range hosts {
		items[i] = hostItem(host)
	}
	l := list.New(items, hostDelegate{}, 0, 0)
	l.Title = "Select host profile"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(true)
	l.SetShowPagination(true)
	l.Styles.Title = titleStyle
	final, err := tea.NewProgram(hostPickerModel{list: l}).Run()
	if err != nil {
		return "", err
	}
	model := final.(hostPickerModel)
	if model.cancelled {
		return "", nil
	}
	return model.selected, nil
}

func (m hostPickerModel) Init() tea.Cmd {
	return nil
}

func (m hostPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height)
	case tea.KeyPressMsg:
		switch strings.ToLower(msg.String()) {
		case "ctrl+c", "q", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "enter":
			if item, ok := m.list.SelectedItem().(hostItem); ok {
				m.selected = string(item)
			}
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m hostPickerModel) View() tea.View {
	return tea.NewView(m.list.View())
}
