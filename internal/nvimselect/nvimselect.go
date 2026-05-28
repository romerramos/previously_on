package nvimselect

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/romerramos/previously_on/internal/nvim"
)

type Option struct {
	Strategy    nvim.Strategy
	Name        string
	Description string
}

type Result struct {
	Strategy nvim.Strategy
	Selected bool
}

type model struct {
	options []Option
	cursor  int
	result  Result
}

func Run() (Result, error) {
	final, err := tea.NewProgram(newModel()).Run()
	if err != nil {
		return Result{}, err
	}
	if m, ok := final.(model); ok {
		return m.result, nil
	}
	return Result{}, nil
}

func newModel() model {
	return model{options: []Option{
		{Strategy: nvim.StrategyNative, Name: "Native", Description: "Install to Neovim's native package path."},
		{Strategy: nvim.StrategyLazy, Name: "lazy.nvim", Description: "Install native package plus a managed lazy.nvim local spec."},
	}}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case "enter":
			m.result = Result{Strategy: m.options[m.cursor].Strategy, Selected: true}
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() tea.View {
	return tea.NewView(m.view())
}

func (m model) view() string {
	titleStyle := lipgloss.NewStyle().Bold(true)
	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	nameStyle := lipgloss.NewStyle().Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", titleStyle.Render("Choose Neovim package strategy"))
	for i, option := range m.options {
		cursor := " "
		name := nameStyle.Render(option.Name)
		if i == m.cursor {
			cursor = ">"
			name = selectedStyle.Render(option.Name)
		}
		fmt.Fprintf(&b, "%s %s\n    %s\n\n", cursor, name, descStyle.Render(option.Description))
	}
	fmt.Fprintf(&b, "%s\n", helpStyle.Render("↑/↓ choose · Enter install · Esc cancel"))
	return b.String()
}
