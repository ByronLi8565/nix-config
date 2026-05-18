package ui

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"charm.land/bubbles/v2/cursor"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"nun/internal/config"
	"nun/internal/nixexpr"
)

type NewHostResult struct {
	Name        string
	User        string
	System      string
	PackageSets []string
	Cancelled   bool
	Aborted     bool
}

type hostStep int

const (
	hostFields hostStep = iota
	hostPackageSets
)

type hostModel struct {
	step          hostStep
	focusIndex    int
	inputs        []textinput.Model
	existingHosts map[string]bool
	packageList   list.Model
	selectedSets  map[string]bool
	cursorMode    cursor.Mode
	err           string
	result        NewHostResult
}

type packageSetItem string

func (i packageSetItem) FilterValue() string { return string(i) }

func NewHost(defaults config.NewHostDefaults) (NewHostResult, error) {
	m := initialHostModel(defaults)
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return NewHostResult{}, err
	}
	return final.(hostModel).result, nil
}

func initialHostModel(defaults config.NewHostDefaults) hostModel {
	inputs := make([]textinput.Model, 3)
	labels := []struct {
		placeholder string
		value       string
		limit       int
	}{
		{"Host name", defaults.DefaultName, 64},
		{"Primary user", defaults.DefaultUser, 64},
		{"System", defaults.DefaultSystem, 64},
	}
	for i, label := range labels {
		t := textinput.New()
		t.Placeholder = label.placeholder
		t.CharLimit = label.limit
		t.SetValue(label.value)
		s := t.Styles()
		s.Cursor.Color = lipgloss.Color("86")
		s.Focused.Prompt = selectedStyle
		s.Focused.Text = selectedStyle
		s.Blurred.Prompt = subtleStyle
		t.SetStyles(s)
		if i == 0 {
			t.Focus()
		}
		inputs[i] = t
	}

	items := make([]list.Item, len(defaults.PackageSetNames))
	for i, name := range defaults.PackageSetNames {
		items[i] = packageSetItem(name)
	}
	selected := map[string]bool{}
	l := list.New(items, packageSetDelegate{selected: selected}, 0, 0)
	l.Title = "host package sets"
	l.SetFilteringEnabled(true)
	l.SetShowStatusBar(false)
	l.SetShowPagination(true)
	l.Styles.Title = titleStyle

	existing := make(map[string]bool, len(defaults.ExistingHosts))
	for _, host := range defaults.ExistingHosts {
		existing[host] = true
	}
	return hostModel{
		step:          hostFields,
		inputs:        inputs,
		existingHosts: existing,
		packageList:   l,
		selectedSets:  selected,
		cursorMode:    cursor.CursorBlink,
	}
}

func (m hostModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m hostModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.step {
	case hostFields:
		return m.updateFields(msg)
	case hostPackageSets:
		return m.updatePackageSets(msg)
	default:
		return m, nil
	}
}

func (m hostModel) updateFields(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.packageList.SetSize(msg.Width, msg.Height-4)
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.result.Cancelled = true
			return m, tea.Quit
		case "ctrl+r":
			m.cursorMode++
			if m.cursorMode > cursor.CursorHide {
				m.cursorMode = cursor.CursorBlink
			}
		case "tab", "shift+tab", "enter", "up", "down":
			if msg.String() == "enter" && m.focusIndex == len(m.inputs) {
				if err := m.validateFields(); err != nil {
					m.err = err.Error()
					return m, nil
				}
				m.err = ""
				m.step = hostPackageSets
				for i := range m.inputs {
					m.inputs[i].Blur()
				}
				return m, nil
			}
			if msg.String() == "up" || msg.String() == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}
			if m.focusIndex > len(m.inputs) {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs)
			}
			cmds := make([]tea.Cmd, len(m.inputs))
			for i := range m.inputs {
				if i == m.focusIndex {
					cmds[i] = m.inputs[i].Focus()
				} else {
					m.inputs[i].Blur()
				}
			}
			return m, tea.Batch(cmds...)
		}
	}
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return m, tea.Batch(cmds...)
}

func (m hostModel) updatePackageSets(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.packageList.SetSize(msg.Width, msg.Height-4)
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			m.result.Cancelled = true
			return m, tea.Quit
		case "esc":
			if m.packageList.SettingFilter() {
				break
			}
			m.step = hostFields
			m.focusIndex = len(m.inputs)
			return m, nil
		case "space":
			if item, ok := m.packageList.SelectedItem().(packageSetItem); ok {
				name := string(item)
				m.selectedSets[name] = !m.selectedSets[name]
			}
			return m, nil
		case "enter":
			m.result = NewHostResult{
				Name:        strings.TrimSpace(m.inputs[0].Value()),
				User:        strings.TrimSpace(m.inputs[1].Value()),
				System:      strings.TrimSpace(m.inputs[2].Value()),
				PackageSets: m.sortedSelectedSets(),
			}
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.packageList, cmd = m.packageList.Update(msg)
	return m, cmd
}

func (m hostModel) validateFields() error {
	name := strings.TrimSpace(m.inputs[0].Value())
	if !nixexpr.ValidIdentifier(name) {
		return fmt.Errorf("host name may only contain letters, numbers, '_' and '-'")
	}
	if m.existingHosts[name] {
		return fmt.Errorf("host %q already exists", name)
	}
	if strings.TrimSpace(m.inputs[1].Value()) == "" || strings.TrimSpace(m.inputs[2].Value()) == "" {
		return fmt.Errorf("user and system cannot be empty")
	}
	return nil
}

func (m hostModel) sortedSelectedSets() []string {
	var sets []string
	for name, selected := range m.selectedSets {
		if selected {
			sets = append(sets, name)
		}
	}
	sort.Strings(sets)
	return sets
}

func (m hostModel) View() tea.View {
	if m.step == hostPackageSets {
		header := titleStyle.Render("Select package sets") + "\n" + subtleStyle.Render("space toggles, / filters, enter accepts, esc returns") + "\n\n"
		return tea.NewView(header + m.packageList.View())
	}

	var b strings.Builder
	var c *tea.Cursor
	b.WriteString(titleStyle.Render("New host"))
	b.WriteString("\n\n")
	for i, input := range m.inputs {
		b.WriteString(input.View())
		b.WriteRune('\n')
		if input.Focused() && m.cursorMode != cursor.CursorHide {
			c = input.Cursor()
			if c != nil {
				c.Y += i + 2
			}
		}
	}
	button := subtleStyle.Render("[ Continue ]")
	if m.focusIndex == len(m.inputs) {
		button = selectedStyle.Render("[ Continue ]")
	}
	b.WriteString("\n")
	b.WriteString(button)
	b.WriteString("\n\n")
	b.WriteString(subtleStyle.Render("tab/enter moves focus, esc cancels"))
	if m.err != "" {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(m.err))
	}
	v := tea.NewView(b.String())
	v.Cursor = c
	return v
}

type packageSetDelegate struct {
	selected map[string]bool
}

func (d packageSetDelegate) Height() int                         { return 1 }
func (d packageSetDelegate) Spacing() int                        { return 0 }
func (d packageSetDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }
func (d packageSetDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	set, ok := item.(packageSetItem)
	if !ok {
		return
	}
	name := string(set)
	mark := "[ ]"
	if d.selected[name] {
		mark = "[x]"
	}
	line := mark + " " + name
	if index == m.Index() {
		fmt.Fprint(w, selectedStyle.Render("> "+line))
		return
	}
	fmt.Fprint(w, "  "+line)
}
