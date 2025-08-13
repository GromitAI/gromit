package main

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
)

const openAIAgent = "openai"

const openAIModelGpt4o = "gpt-4o"

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

type AssisterCreator interface {
	GetAssister(agent string, model string) (Assister, error)
}

var _ AssisterCreator = (*defaultAssisterCreator)(nil)

type defaultAssisterCreator struct{}

func (d *defaultAssisterCreator) GetAssister(agent, model string) (Assister, error) {
	if agent == "" || agent == openAIAgent {
		return &OpenAIAssister{
			model: model,
		}, nil
	}
	return nil, fmt.Errorf("cannot create AI agent for %s and model %s", agent, model)
}
