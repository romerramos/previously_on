package connect

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/romerramos/previously_on/internal/ai"
	"github.com/romerramos/previously_on/internal/config"
	"github.com/romerramos/previously_on/internal/secrets"
)

type Result struct {
	Provider ai.ProviderInfo
	Model    string
	Saved    bool
}

type step int

const (
	stepProvider step = iota
	stepKey
	stepDone
)

type model struct {
	cfg       config.Config
	store     secrets.Store
	providers []ai.ProviderInfo
	cursor    int
	step      step
	input     textinput.Model
	err       string
	result    Result
}

func Run(cfg config.Config, store secrets.Store, initialProvider string) (Result, error) {
	m := newModel(cfg, store, initialProvider)
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return Result{}, err
	}
	if m, ok := final.(model); ok {
		return m.result, nil
	}
	return Result{}, nil
}

func newModel(cfg config.Config, store secrets.Store, initialProvider string) model {
	input := textinput.New()
	input.Prompt = "API key: "
	input.EchoMode = textinput.EchoNone
	input.Focus()

	providers := ai.Providers()
	cursor := 0
	initialStep := stepProvider
	for i, provider := range providers {
		if provider.ID == initialProvider {
			cursor = i
			initialStep = stepKey
			if !provider.RequiresKey {
				initialStep = stepProvider
			}
			break
		}
	}

	return model{cfg: cfg, store: store, providers: providers, cursor: cursor, step: initialStep, input: input}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		}

		switch m.step {
		case stepProvider:
			switch msg.String() {
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.providers)-1 {
					m.cursor++
				}
			case "enter":
				provider := m.providers[m.cursor]
				if !provider.RequiresKey {
					return m.save(provider, "")
				}
				m.step = stepKey
				m.err = ""
			}
		case stepKey:
			if msg.String() == "enter" {
				key := strings.TrimSpace(m.input.Value())
				if key == "" {
					m.err = "API key cannot be empty."
					return m, nil
				}
				return m.save(m.providers[m.cursor], key)
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		case stepDone:
			return m, tea.Quit
		}
	case tea.PasteMsg:
		if m.step == stepKey {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
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
	providerStyle := lipgloss.NewStyle().Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	title := titleStyle.Render("Connect an AI provider")
	help := helpStyle.Render("↑/↓ choose · Enter continue · Esc cancel")

	switch m.step {
	case stepProvider:
		var b strings.Builder
		fmt.Fprintf(&b, "%s\n\n", title)
		for i, provider := range m.providers {
			cursor := " "
			current := " "
			name := providerStyle.Render(provider.Name)
			if i == m.cursor {
				cursor = ">"
				name = selectedStyle.Render(provider.Name)
			}
			if provider.ID == m.cfg.AI.Provider && m.cfg.AI.Enabled {
				current = "✓"
			}
			fmt.Fprintf(&b, "%s %s %s\n    %s\n\n", cursor, current, name, descStyle.Render(provider.Description))
		}
		fmt.Fprintf(&b, "%s\n", help)
		return b.String()
	case stepKey:
		provider := m.providers[m.cursor]
		body := fmt.Sprintf("%s\n\n%s API key\n\nPaste your API key below. Input is hidden and will be stored in your OS keychain, not in config files.\n\n%s", title, provider.Name, m.input.View())
		if m.err != "" {
			body += "\n" + errorStyle.Render(m.err)
		}
		return body + "\n\n" + helpStyle.Render("Enter save · Esc cancel") + "\n"
	case stepDone:
		if m.result.Provider.RequiresKey {
			return fmt.Sprintf("Connected %s. API key stored outside config.\n", m.result.Provider.Name)
		}
		return fmt.Sprintf("Connected %s. No API key required.\n", m.result.Provider.Name)
	}
	return ""
}

func (m model) save(provider ai.ProviderInfo, key string) (tea.Model, tea.Cmd) {
	if provider.RequiresKey {
		if err := m.store.Set(provider.ID, key); err != nil {
			m.err = err.Error()
			return m, nil
		}
	}
	m.cfg.AI.Enabled = true
	m.cfg.AI.Provider = provider.ID
	m.cfg.AI.Model = provider.DefaultModel
	if m.cfg.AI.Models == nil {
		m.cfg.AI.Models = map[string]string{}
	}
	m.cfg.AI.Models[provider.ID] = provider.DefaultModel
	if err := config.Save(m.cfg); err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.input.SetValue("")
	m.result = Result{Provider: provider, Model: provider.DefaultModel, Saved: true}
	m.step = stepDone
	return m, tea.Quit
}
