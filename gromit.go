package main

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	AssisterCreator
	messagePrinter
}

type messagePrinter struct {
	config *configuration
}

type configuration struct {
	promptPrefix string
	w            io.Writer
}

func (m *messagePrinter) print(s string) {
	fmt.Fprintf(m.config.w, "%s %s\n", m.config.promptPrefix, s)
}

type ConfigurationModifier func(*configuration) error

func WithPromptPrefix(prefix string) ConfigurationModifier {
	return func(c *configuration) error {
		c.promptPrefix = prefix
		return nil
	}
}

func WithWriter(writer io.Writer) ConfigurationModifier {
	return func(c *configuration) error {
		c.w = writer
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
		g.print("Please run ./gromit --help to see usage")
		return nil
	}
	prompt := g.String("systemPrompt")
	if prompt == "" {
		prompt = systemPrompt
	}
	exeCommand, err := assister.GetTerminalCommand(ctx, query, prompt)
	if err != nil {
		return err
	}
	g.print("In order to do that, you need to run:")
	g.print(exeCommand)
	g.print("Would you like to run this command?")

	confirmation, err := g.askConfirmation()
	if err != nil {
		g.print("Error reading your response")
		return err
	}
	if confirmation.confirm {
		g.print("Running the command...")
		err := g.executeCommand(exeCommand)
		if err != nil {
			g.print(fmt.Sprintf("error running the command: %s", err.Error()))
			return err
		} else {
			g.print("Done!")
		}
	} else {
		g.print("You chose not to execute this command.")
	}
	return nil
}

type userConfirmation struct {
	confirm bool
}

func (g *Gromit) askConfirmation() (userConfirmation, error) {
	var userConfirmation userConfirmation
	var userResponse string
	n, err := fmt.Scanln(&userResponse)
	userResponse = strings.ToLower(userResponse)
	switch {
	case n == 0:
		g.print("You didn't confirm your choice! Please reply with yes(y) or no(n).")
		return g.askConfirmation()
	case err != nil:
		g.print("Error reading your response")
		return userConfirmation, err
	case userResponse == "yes" || userResponse == "y":
		userConfirmation.confirm = true
	case userResponse == "no" || userResponse == "n":
		userConfirmation.confirm = false
	}
	return userConfirmation, nil
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
			Action: func(ctx context.Context, command *cli.Command, s string) error {
				if s == "" {
					return errors.New("agent cannot be empty")
				}
				return nil
			},
		},
		&cli.StringFlag{
			Name:  "model",
			Usage: "The model to use for AI agent; for example, gpt-4o",
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
		config: &configuration{
			promptPrefix: "🐶",
			w:            os.Stdout,
		},
	}
	for _, apply := range mods {
		if err := apply(gromit.messagePrinter.config); err != nil {
			return nil, err
		}
	}
	return &gromit, nil
}
