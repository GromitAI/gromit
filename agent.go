package main

import (
	"context"
	"fmt"
	"github.com/anthropics/anthropic-sdk-go"
	anthropicOption "github.com/anthropics/anthropic-sdk-go/option"
	"github.com/openai/openai-go"
	openaiOption "github.com/openai/openai-go/option"
	"google.golang.org/genai"
	"os"
)

const (
	openAIAgent      = "openai"
	anthropicAIAgent = "anthropic"
	geminiAIAgent    = "gemini"
)

// Gemini models
const (
	geminiFlashLite = "gemini-2.5-flash-lite"
	geminiFlash     = "gemini-2.5-flash"
)

const (
	defaultMaxTokens = 1024
)

type Assister interface {
	GetTerminalCommand(ctx context.Context, userMessage string) (string, error)
}

var _ Assister = (*OpenAIAssister)(nil)
var _ Assister = (*AnthropicAIAssister)(nil)
var _ Assister = (*GeminiAIAssister)(nil)

type OpenAIAssister struct {
	aiParameters
}

func (o *OpenAIAssister) GetTerminalCommand(ctx context.Context, userMessage string) (string, error) {
	apiKey := o.aiParameters.apiKey
	if apiKey == "" {
		apiKey, _ = os.LookupEnv("OPENAI_API_KEY")
	}
	client := openai.NewClient(openaiOption.WithAPIKey(apiKey))
	chatCompletion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(userMessage),
			openai.SystemMessage(o.aiParameters.systemPrompt),
		},
		Model: o.aiParameters.model,
	})
	if err != nil {
		return "", err
	}
	return chatCompletion.Choices[0].Message.Content, nil
}

type AnthropicAIAssister struct {
	aiParameters
}

func (c *AnthropicAIAssister) GetTerminalCommand(ctx context.Context, userMessage string) (string, error) {
	apiKey := c.aiParameters.apiKey
	if apiKey == "" {
		apiKey, _ = os.LookupEnv("ANTHROPIC_API_KEY")
	}
	maxTokens := c.aiParameters.maxTokens
	if maxTokens == 0 {
		maxTokens = defaultMaxTokens
	}
	client := anthropic.NewClient(anthropicOption.WithAPIKey(apiKey))
	message, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		MaxTokens: maxTokens,
		System: []anthropic.TextBlockParam{
			{Text: c.aiParameters.systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userMessage)),
		},
		Model: anthropic.Model(c.model),
	})
	if err != nil {
		return "", err
	}
	var response string
	for _, content := range message.Content {
		switch block := content.AsAny().(type) {
		case anthropic.TextBlock:
			response = block.Text
		}
	}
	return response, nil
}

type GeminiAIAssister struct {
	aiParameters
}

func (g *GeminiAIAssister) GetTerminalCommand(ctx context.Context, userMessage string) (string, error) {
	apiKey := g.aiParameters.apiKey
	if apiKey == "" {
		apiKey, _ = os.LookupEnv("GEMINI_API_KEY")
	}
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  apiKey,
	})
	if err != nil {
		return "", err
	}
	config := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{
				{Text: g.aiParameters.systemPrompt},
			},
		},
	}
	chat, err := client.Chats.Create(ctx, g.model, config, nil)
	if err != nil {
		return "", err
	}
	result, err := chat.SendMessage(ctx, genai.Part{Text: userMessage})
	if err != nil {
		return "", err
	}
	return result.Candidates[0].Content.Parts[0].Text, nil
}

type AssisterCreator interface {
	GetAssister(parameters aiParameters) (Assister, error)
}

var _ AssisterCreator = (*defaultAIAssisterCreator)(nil)

type defaultAIAssisterCreator struct{}

func (d *defaultAIAssisterCreator) GetAssister(p aiParameters) (Assister, error) {
	switch {
	case p.agent == "" || p.agent == openAIAgent:
		if p.model == "" {
			p.model = openai.ChatModelGPT4o
		}
		return &OpenAIAssister{p}, nil

	case p.agent == anthropicAIAgent:
		if p.model == "" {
			p.model = string(anthropic.ModelClaude3_5HaikuLatest)
		}
		return &AnthropicAIAssister{p}, nil
	case p.agent == geminiAIAgent:
		if p.model == "" {
			p.model = geminiFlashLite
		}
		return &GeminiAIAssister{p}, nil
	default:
		return nil, fmt.Errorf("cannot create AI agent for %s and model %s", p.agent, p.model)
	}
}
