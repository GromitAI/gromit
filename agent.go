package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/openai/openai-go"
	"google.golang.org/genai"
)

const (
	openAIAgent      = "openai"
	anthropicAIAgent = "anthropic"
	geminiAIAgent = "gemini"
)

const gemini2_5_flash_lite = "gemini-2.5-flash-lite"

type Assister interface {
	GetTerminalCommand(ctx context.Context, userMessage string, systemMessage string) (string, error)
}

var _ Assister = (*OpenAIAssister)(nil)

type OpenAIAssister struct {
	model string
}

func (o *OpenAIAssister) GetTerminalCommand(ctx context.Context, userMessage string, systemMessage string) (string, error) {
	client := openai.NewClient()
	chatCompletion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(userMessage),
			openai.SystemMessage(systemMessage),
		},
		Model: o.model,
	})
	if err != nil {
		return "", err
	}
	return chatCompletion.Choices[0].Message.Content, nil
}

type AnthropicAIAssister struct {
	model string
}

func (c *AnthropicAIAssister) GetTerminalCommand(ctx context.Context, userMessage string, systemMessage string) (string, error) {
	client := anthropic.NewClient() //defaults to os.LookupEnv("ANTHROPIC_API_KEY")
	message, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{
			{Text: systemMessage},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userMessage)),
		},
		Model: anthropic.Model(c.model),
	})
	if err != nil {
		return "", err
	}
	var response []string
	for _, content := range message.Content {
		switch block := content.AsAny().(type) {
		case anthropic.TextBlock:
			response = append(response, block.Text)
		}
	}
	return strings.Join(response, "\n"), nil
}

type GeminiAIAssister struct {
	model string
}

func (g *GeminiAIAssister) GetTerminalCommand(ctx context.Context, userMessage string, systemMessage string) (string, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return "", err
	}
	config := &genai.GenerateContentConfig {
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{
				{Text: systemMessage},
			},
		},
	}
	chat, _ := client.Chats.Create(ctx, g.model, config, nil)
	result, err := chat.SendMessage(ctx, genai.Part{Text: userMessage})
	if err != nil {
		return "", err
	}
	return result.Candidates[0].Content.Parts[0].Text, nil
}

type AssisterCreator interface {
	GetAssister(agent string, model string) (Assister, error)
}

var _ AssisterCreator = (*defaultAIAssisterCreator)(nil)

type defaultAIAssisterCreator struct{}

func (d *defaultAIAssisterCreator) GetAssister(agent, model string) (Assister, error) {
	if agent == "" || agent == openAIAgent {
		if model == "" {
			model = openai.ChatModelGPT4o
		}
		return &OpenAIAssister{
			model: model,
		}, nil
	}
	if agent == anthropicAIAgent {
		if model == "" {
			model = string(anthropic.ModelClaude3_5HaikuLatest)
		}
		return &AnthropicAIAssister{
			model: model,
		}, nil
	}
	if agent == geminiAIAgent {
		if model == "" {
			model = gemini2_5_flash_lite
		}
		return &GeminiAIAssister{
			model,
		}, nil
	}
	return nil, fmt.Errorf("cannot create AI agent for %s and model %s", agent, model)
}
