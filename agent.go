package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)


const openAIAgent = "openai"

const openAIModelGpt4o = "gpt-4o"

type Assister interface {
	GetTerminalCommand(ctx context.Context, userMessage string, systemMessage string) (string, error)
}

var _ Assister = (*OpenAIAssister)(nil)

type OpenAIAssister struct {
	apiKey string
	model string
}


func (o *OpenAIAssister) GetTerminalCommand(ctx context.Context, userMessage string, systemMessage string) (string, error) {
	var options []option.RequestOption
	options = append(options, option.WithAPIKey(o.apiKey))
	client := openai.NewClient(
		options...
	)
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
	GetAssister(agent string, model string, apiKey string) (Assister, error)
}

var _ AssisterCreator = (*AssisterFactory)(nil)

type AssisterFactory struct {}

func (a *AssisterFactory) GetAssister(agent, model, apiKey string) (Assister, error) {
	if agent == "" {
		return nil, errors.New("unable to create an ai assister since agent is not specified")
	}
	if model == "" {
		return nil, errors.New("unable to create an ai assister since model is not specified")
	}
	if apiKey == "" {
		return nil, errors.New("unable to create an ai assister since api key is not specified")
	}
	switch true {
	case agent == openAIAgent && model == openAIModelGpt4o:
		return &OpenAIAssister{
			apiKey: apiKey,
			model: model,
		}, nil
		default:
			return nil, fmt.Errorf("cannot create AI agent for %s and model %s", agent, model)
	}
}