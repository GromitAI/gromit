package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/urfave/cli/v3"
)

const systemPrompt = `You are a command line helper in a linux environment. 
	You will be given a question about how to do something in the CLI environment. 
	You will then find out what command to execute and provide the command.
	Do not provide any additional information, explanation or context, just the linux command.
	For example, if question is about listing all files in a directory, respond with "ls".`

type Gromit struct {
	cli.Command
	*configuration
}

type configuration struct{}

type ConfigurationModifier func(*configuration) error

func (g *Gromit) actionGromit(ctx context.Context, command *cli.Command) error {
	apiKey := g.String("apiKey")
	if apiKey == "" && g.String("agent") == openAIAgent {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	assister, err := (&AssisterFactory{}).GetAssister(g.String("agent"), g.String("model"), apiKey)
	if err != nil {
		return err
	}
	commandArgs := command.Args().Slice()
	query := strings.Join(commandArgs, " ")
	if query == "" {
		fmt.Println("🤖⚡️ Please specify which linux command you need help with!")
		return nil
	}
	exeCommand, err := assister.GetTerminalCommand(ctx, query, systemPrompt)
	if err != nil {
		return err
	}
	fmt.Println("🤖⚡️ In order to do that, you need to run:")
	fmt.Println(exeCommand)
	fmt.Println("🤖⚡️ Would you like to run this command?")
	var userResponse string
	n, err := fmt.Scanln(&userResponse)
	userResponse = strings.ToLower(userResponse)
	switch {
	case n == 0:
		fmt.Println("You didn't specify whether you want to run this command!")
		return nil
	case err != nil:
		fmt.Println("Error reading your response: ", err)
	case userResponse == "yes" || userResponse == "y":
		fmt.Println("Running the command:")
		err := executeCommand(exeCommand)
		if err != nil {
			fmt.Println("error running the command:", err)
		} else {
			fmt.Println("🤖⚡️ Done!")
		}
	case userResponse == "no" || userResponse == "n":
		fmt.Println("You chose not to execute this command.")
	}
	return nil
}

func executeCommand(command string) error {
	c := exec.Command("sh", "-c", command)
	output, err := c.CombinedOutput()
	if err != nil {
		return err
	} else {
		fmt.Println("Command output:", string(output))
		return nil
	}
}

func NewGromit(mods ...ConfigurationModifier) (*Gromit, error) {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:  "agent",
			Usage: "The AI agent to use for processing requests. Defaults to 'OpenAI'. Currently supported agents: OpenAI.",
			Value: openAIAgent,
			Action: func(ctx context.Context, command *cli.Command, s string) error {
				switch strings.ToLower(s) {
				case openAIAgent:
					return nil
				default:
					return fmt.Errorf("unsupported AI agent %s", s)
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
				return nil
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
	gromit.Action = gromit.actionGromit
	for _, apply := range mods {
		if err := apply(gromit.configuration); err != nil {
			return nil, err
		}
	}
	return &gromit, nil
}
