package ai

type ProviderInfo struct {
	ID           string
	Name         string
	Description  string
	EnvVar       string
	DefaultModel string
	RequiresKey  bool
	BaseURL      string
}

func Providers() []ProviderInfo {
	return []ProviderInfo{
		{ID: "openai", Name: "OpenAI", Description: "GPT models using your OpenAI API key", EnvVar: "OPENAI_API_KEY", DefaultModel: "gpt-4.1-mini", RequiresKey: true},
		{ID: "anthropic", Name: "Anthropic", Description: "Claude models using your Anthropic API key", EnvVar: "ANTHROPIC_API_KEY", DefaultModel: "claude-3-5-haiku-latest", RequiresKey: true},
		{ID: "gemini", Name: "Gemini", Description: "Google AI Gemini models using your Gemini API key", EnvVar: "GEMINI_API_KEY", DefaultModel: "gemini-2.5-flash", RequiresKey: true},
		{ID: "openrouter", Name: "OpenRouter", Description: "OpenAI-compatible access to many hosted models", EnvVar: "OPENROUTER_API_KEY", DefaultModel: "openai/gpt-4o-mini", RequiresKey: true, BaseURL: "https://openrouter.ai/api/v1"},
		{ID: "ollama", Name: "Ollama", Description: "Local models through Ollama; no API key required", DefaultModel: "llama3.2", RequiresKey: false, BaseURL: "http://localhost:11434"},
	}
}

func FindProvider(id string) (ProviderInfo, bool) {
	for _, provider := range Providers() {
		if provider.ID == id {
			return provider, true
		}
	}
	return ProviderInfo{}, false
}
