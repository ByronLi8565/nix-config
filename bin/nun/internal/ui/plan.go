package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

type PlanAction string

const (
	PlanApply       PlanAction = "apply"
	PlanCancel      PlanAction = "cancel"
	PlanInteractive PlanAction = "interactive"
)

type PlanSection struct {
	Title string
	Items []string
}

type PlanView struct {
	Title    string
	Summary  string
	Sections []PlanSection
	Notes    []string
	Actions  []PlanAction
}

type planModel struct {
	plan   PlanView
	cursor int
	action PlanAction
}

func ShowPlan(plan PlanView) (PlanAction, error) {
	if len(plan.Actions) == 0 {
		plan.Actions = []PlanAction{PlanApply, PlanCancel}
	}
	final, err := tea.NewProgram(planModel{plan: plan}).Run()
	if err != nil {
		return PlanCancel, err
	}
	if model, ok := final.(planModel); ok && model.action != "" {
		return model.action, nil
	}
	return PlanCancel, nil
}

func (m planModel) Init() tea.Cmd {
	return nil
}

func (m planModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.action = PlanCancel
			return m, tea.Quit
		case "left", "h", "shift+tab":
			m.cursor--
			if m.cursor < 0 {
				m.cursor = len(m.plan.Actions) - 1
			}
		case "right", "l", "tab":
			m.cursor++
			if m.cursor >= len(m.plan.Actions) {
				m.cursor = 0
			}
		case "enter", "space":
			m.action = m.plan.Actions[m.cursor]
			return m, tea.Quit
		case "y":
			if actionIndex(m.plan.Actions, PlanApply) >= 0 {
				m.action = PlanApply
				return m, tea.Quit
			}
		case "n":
			m.action = PlanCancel
			return m, tea.Quit
		case "i":
			if actionIndex(m.plan.Actions, PlanInteractive) >= 0 {
				m.action = PlanInteractive
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m planModel) View() tea.View {
	var b strings.Builder
	b.WriteString(titleStyle.Render(m.plan.Title))
	b.WriteString("\n")
	if m.plan.Summary != "" {
		b.WriteString(subtleStyle.Render(m.plan.Summary))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	for _, section := range m.plan.Sections {
		if len(section.Items) == 0 {
			continue
		}
		b.WriteString(selectedStyle.Render(section.Title))
		b.WriteString("\n")
		for _, item := range section.Items {
			b.WriteString("  ")
			b.WriteString(item)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	if len(m.plan.Notes) > 0 {
		b.WriteString(selectedStyle.Render("Notes"))
		b.WriteString("\n")
		for _, note := range m.plan.Notes {
			b.WriteString("  ")
			b.WriteString(note)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	for i, action := range m.plan.Actions {
		label := actionLabel(action)
		if i == m.cursor {
			b.WriteString(selectedStyle.Render("[ " + label + " ]"))
		} else {
			b.WriteString(subtleStyle.Render("[ " + label + " ]"))
		}
		if i < len(m.plan.Actions)-1 {
			b.WriteString("  ")
		}
	}
	b.WriteString("\n\n")
	b.WriteString(subtleStyle.Render("tab/h/l move  enter/space select  y apply  n cancel"))
	if actionIndex(m.plan.Actions, PlanInteractive) >= 0 {
		b.WriteString(subtleStyle.Render("  i interactive"))
	}
	b.WriteString("\n")
	return tea.NewView(b.String())
}

func actionLabel(action PlanAction) string {
	switch action {
	case PlanApply:
		return "Apply"
	case PlanInteractive:
		return "Interactive"
	default:
		return "Cancel"
	}
}

func actionIndex(actions []PlanAction, needle PlanAction) int {
	for i, action := range actions {
		if action == needle {
			return i
		}
	}
	return -1
}
