package ai

import (
	"context"
	"fmt"

	genai "github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/anthropic"
	"github.com/firebase/genkit/go/plugins/compat_oai"
	oai "github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/firebase/genkit/go/plugins/ollama"
)

type GenkitProvider struct {
	Info  ProviderInfo
	Key   string
	Model string
}

func (p GenkitProvider) Generate(ctx context.Context, payload Payload) (sections GeneratedSections, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("genkit provider %s failed: %v", p.Info.ID, recovered)
		}
	}()

	model := p.Model
	if model == "" {
		model = p.Info.DefaultModel
	}
	prompt := SystemPrompt() + "\n\n" + UserPrompt(payload)

	switch p.Info.ID {
	case "openai":
		plugin := &oai.OpenAI{APIKey: p.Key}
		g := genkit.Init(ctx, genkit.WithPlugins(plugin))
		return parseGeneratedText(genkit.GenerateText(ctx, g, genai.WithModel(plugin.Model(g, model)), genai.WithPrompt(prompt)))
	case "anthropic":
		g := genkit.Init(ctx, genkit.WithPlugins(&anthropic.Anthropic{APIKey: p.Key}))
		return parseGeneratedText(genkit.GenerateText(ctx, g, genai.WithModelName("anthropic/"+model), genai.WithPrompt(prompt)))
	case "gemini":
		g := genkit.Init(ctx, genkit.WithPlugins(&googlegenai.GoogleAI{APIKey: p.Key}))
		return parseGeneratedText(genkit.GenerateText(ctx, g, genai.WithModelName("googleai/"+model), genai.WithPrompt(prompt)))
	case "openrouter":
		g := genkit.Init(ctx, genkit.WithPlugins(&compat_oai.OpenAICompatible{Provider: "openrouter", APIKey: p.Key, BaseURL: p.Info.BaseURL}), genkit.WithDefaultModel("openrouter/"+model))
		return parseGeneratedText(genkit.GenerateText(ctx, g, genai.WithPrompt(prompt)))
	case "ollama":
		g := genkit.Init(ctx, genkit.WithPlugins(&ollama.Ollama{ServerAddress: p.Info.BaseURL}))
		return parseGeneratedText(genkit.GenerateText(ctx, g, genai.WithModelName("ollama/"+model), genai.WithPrompt(prompt)))
	default:
		return GeneratedSections{}, fmt.Errorf("unsupported AI provider %q", p.Info.ID)
	}
}

func parseGeneratedText(text string, err error) (GeneratedSections, error) {
	if err != nil {
		return GeneratedSections{}, err
	}
	return ParseGeneratedSections(text)
}
