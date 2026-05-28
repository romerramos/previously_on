package modelselect

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/romerramos/previously_on/internal/ai"
)

type Result struct {
	Model string
	Saved bool
}

type model struct {
	provider ai.ProviderInfo
	current  string
	input    textinput.Model
	err      string
	result   Result
}

func Run(provider ai.ProviderInfo, current string) (Result, error) {
	input := textinput.New()
	input.Prompt = "Model: "
	input.SetValue(current)
	input.Focus()

	final, err := tea.NewProgram(model{provider: provider, current: current, input: input}).Run()
	if err != nil {
		return Result{}, err
	}
	if m, ok := final.(model); ok {
		return m.result, nil
	}
	return Result{}, nil
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			value := strings.TrimSpace(m.input.Value())
			if value == "" {
				m.err = "Model name cannot be empty."
				return m, nil
			}
			m.result = Result{Model: value, Saved: true}
			return m, tea.Quit
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	case tea.PasteMsg:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) View() tea.View {
	title := lipgloss.NewStyle().Bold(true).Render("Choose model")
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

	body := fmt.Sprintf("%s\n\nProvider: %s\nCurrent model: %s\n\nWrite the model name to use for %s.\n%s\n\n%s", title, m.provider.Name, m.current, m.provider.Name, muted.Render("Double-check the model name in your provider docs/dashboard; Previously on... does not validate it yet."), m.input.View())
	if m.err != "" {
		body += "\n" + errorStyle.Render(m.err)
	}
	body += "\n\n" + muted.Render("Enter save · Esc cancel") + "\n"
	return tea.NewView(body)
}
