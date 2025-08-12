package main

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	AssisterCreator
	messagePrinter
}

type messagePrinter struct {
	config *configuration
	w      *io.Writer
}

type configuration struct {
	promptPrefix string
}

func (m *messagePrinter) print(s string) {
	fmt.Fprintf(*m.w, "%s %s\n", m.config.promptPrefix, s)
}

type ConfigurationModifier func(*configuration) error

func WithPromptPrefix(prefix string) ConfigurationModifier {
	return func(c *configuration) error {
		c.promptPrefix = prefix
		return nil
	}
}

func (g *Gromit) actionGromit(ctx context.Context, command *cli.Command) error {
	assister, err := g.AssisterCreator.GetAssister(g.String("agent"), g.String("model"))
	if err != nil {
		return err
	}
	commandArgs := command.Args().Slice()
	query := strings.Join(commandArgs, " ")
	if query == "" {
		g.print("Please specify which linux command you need help with!")
		return nil
	}
	exeCommand, err := assister.GetTerminalCommand(ctx, query, systemPrompt)
	if err != nil {
		return err
	}
	g.print("In order to do that, you need to run:")
	g.print(exeCommand)
	g.print("Would you like to run this command?")
	var userResponse string
	n, err := fmt.Scanln(&userResponse)
	userResponse = strings.ToLower(userResponse)
	switch {
	case n == 0:
		g.print("You didn't specify whether you want to run this command!")
		return nil
	case err != nil:
		g.print("Error reading your response")
		return err
	case userResponse == "yes" || userResponse == "y":
		g.print("Running the command...")
		err := g.executeCommand(exeCommand)
		if err != nil {
			g.print("error running the command")
			return err
		} else {
			g.print("Done!")
		}
	case userResponse == "no" || userResponse == "n":
		g.print("You chose not to execute this command.")
	}
	return nil
}

func (g *Gromit) executeCommand(command string) error {
	c := exec.Command("sh", "-c", command)
	output, err := c.CombinedOutput()
	if err != nil {
		return err
	} else {
		g.print("Command output:")
		g.print(string(output))
		return nil
	}
}

func NewGromit(a AssisterCreator, mods ...ConfigurationModifier) (*Gromit, error) {
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
			Name:  "systemPrompt",
			Usage: "The system prompt for the AI agent. Defaults to command line helper in a linux environment.",
		},
	}
	gromit := Gromit{
		AssisterCreator: a,
		Command: cli.Command{
			Usage: "A command line helper that uses generative AI to generate commands based on user input.",
			Name:  "gromit",
			Flags: flags,
		},
	}
	gromit.Action = gromit.actionGromit
	gromit.messagePrinter = messagePrinter{
		w: &gromit.Writer,
		config: &configuration{
			promptPrefix: "🐶",
		},
	}
	for _, apply := range mods {
		if err := apply(gromit.messagePrinter.config); err != nil {
			return nil, err
		}
	}
	return &gromit, nil
}
