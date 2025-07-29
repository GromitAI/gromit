package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/urfave/cli"
)

const systemPrompt = `You are a command line helper in a linux environment. 
	You will be given a question about how to do something in the CLI environment. 
	You then will find what the command is to execute and provide the command. 
	Do not provide any additional information or context, just the linux command.`

type Gromit struct {
	cli.Command
}

func NewGromit() (*Gromit, error) {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:  "agent",
			Usage: "The AI agent to use for processing requests. Defaults to 'OpenAI'. Currently supported agents: OpenAI.",
			Value: openAIAgent,
			Action: func(ctx context.Context, command *cli.Command, s string) error {
				switch s {
				case openAIAgent:
					return nil
				default:
					return fmt.Errorf("Unsupported AI agent %s", s)
				}
			},
		},
		&cli.StringFlag{
			Name:  "model",
			Usage: "The model to use for AI agent; for example, gpt-4o",
			Value: openAIModelGpt4o,
			Action: func(ctx context.Context, command *cli.Command, s string) error {
				if s == "" {
					return errors.New("model cannot be empty")
				}
			},
		},
		&cli.StringFlag{
			Name:  "apiKey",
			Usage: "API key for the AI service. Defaults to environment variable <AI provider>_API_KEY, for example OPENAI_API_KEY",
		},
		&cli.StringFlag{
			Name:  "systemPrompt",
			Usage: "The system prompt for the AI agent. Defaults to command line helper in a linux environment.",
		},
	}
	gromit := Gromit{
		Command: cli.Command{
			Usage: "A command line helper that uses generative AI to generate commands based on user input.",
			Name:  "gromit",
			Flags: flags,
		},
	}
	return &gromit, nil
}
