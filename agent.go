package main

import (
	"context"
	"fmt"
	"github.com/anthropics/anthropic-sdk-go"
	anthropicOption "github.com/anthropics/anthropic-sdk-go/option"
	"github.com/openai/openai-go"
	openaiOption "github.com/openai/openai-go/option"
	"google.golang.org/genai"
)

const (
	openAIAgent      = "openai"
	anthropicAIAgent = "anthropic"
	geminiAIAgent    = "gemini"
	defaultMaxTokens = 1024
)

// Gemini models
const (
	geminiFlashLite = "gemini-2.5-flash-lite"
	geminiFlash     = "gemini-2.5-flash"
)

type Assister interface {
	GetTerminalCommand(ctx context.Context, userMessage string) (string, error)
}

var _ Assister = (*OpenAIAssister)(nil)
var _ Assister = (*AnthropicAIAssister)(nil)
var _ Assister = (*GeminiAIAssister)(nil)

type OpenAIAssister struct {
	AiParameters
}

func (o *OpenAIAssister) GetTerminalCommand(ctx context.Context, userMessage string) (string, error) {
	apiKey := o.AiParameters.apiKey
	var client openai.Client
	if apiKey != "" {
		client = openai.NewClient(openaiOption.WithAPIKey(apiKey))
	} else {
		client = openai.NewClient()
	}
	chatCompletion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(userMessage),
			openai.SystemMessage(o.AiParameters.systemPrompt),
		},
		Model: o.AiParameters.model,
	})
	if err != nil {
		return "", err
	}
	return chatCompletion.Choices[0].Message.Content, nil
}

type AnthropicAIAssister struct {
	AiParameters
}

func (c *AnthropicAIAssister) GetTerminalCommand(ctx context.Context, userMessage string) (string, error) {
	apiKey := c.AiParameters.apiKey
	maxTokens := c.AiParameters.maxTokens
	if maxTokens == 0 {
		maxTokens = defaultMaxTokens
	}
	var client anthropic.Client
	if apiKey != "" {
		client = anthropic.NewClient(anthropicOption.WithAPIKey(apiKey))
	} else {
		client = anthropic.NewClient()
	}
	message, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		MaxTokens: maxTokens,
		System: []anthropic.TextBlockParam{
			{Text: c.AiParameters.systemPrompt},
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
	AiParameters
}

func (g *GeminiAIAssister) GetTerminalCommand(ctx context.Context, userMessage string) (string, error) {
	apiKey := g.AiParameters.apiKey
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
				{Text: g.AiParameters.systemPrompt},
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
	GetAssister(parameters AiParameters) (Assister, error)
}

var _ AssisterCreator = (*defaultAIAssisterCreator)(nil)

type defaultAIAssisterCreator struct{}

func (d *defaultAIAssisterCreator) GetAssister(p AiParameters) (Assister, error) {
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
