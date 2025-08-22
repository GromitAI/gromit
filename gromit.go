package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/urfave/cli/v3"
)

const systemPrompt = `You are an assistant providing terminal commands for various environments (linux, windows, etc). 
	You will be given a question about how to do something in the CLI environment. 
	You will then find out what command to execute and provide the command.
	Do not provide any additional information, explanation or context, just the linux command.
	For example, if question is about listing all files in a directory for linux, respond with "ls".`

type Gromit struct {
	cli.Command
	AssisterCreator
	messagePrinter
	*configuration
}

type operatingSystemInfo struct {
	operatingSystem string
	currentShell    string
	delimiter       string
}

func getOperatingSystemInfo() operatingSystemInfo {
	o := runtime.GOOS
	var eol string
	var shell string
	if strings.Contains(strings.ToLower(o), "windows") {
		eol = "\r\n"
	} else {
		eol = "\n"
		shell = os.Getenv("SHELL")
	}

	return operatingSystemInfo{
		operatingSystem: o,
		currentShell:    shell,
		delimiter:       eol,
	}
}

type messagePrinter struct {
	w            io.Writer
	promptPrefix string
}

type configuration struct {
	promptPrefix       string
	w                  io.Writer
	askForConfirmation bool
	osInfo             operatingSystemInfo
}

func (m *messagePrinter) print(s string) {
	fmt.Fprintf(m.w, "%s %s\n", m.promptPrefix, s)
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

func WithAskForConfirmation(confirm bool) ConfigurationModifier {
	return func(c *configuration) error {
		c.askForConfirmation = confirm
		return nil
	}
}

func (g *Gromit) actionGromit(ctx context.Context, command *cli.Command) error {
	commandArgs := command.Args().Slice()
	query := strings.Join(commandArgs, " ")
	if query == "" {
		g.print("Please run ./gromit --help to see usage")
		return nil
	}
	err := g.handleUserQuery(ctx, query)
	if err != nil {
		return err
	}
	for ctx.Err() == nil {
		confirmation, err := g.askConfirmation("Can I help you with anything else?")
		if err != nil {
			return err
		}
		if confirmation.confirmed {
			g.print("How can I help?")
			reader := bufio.NewReader(os.Stdin)
			query, err := reader.ReadString('\n')
			if err != nil {
				return err
			}
			if err = g.handleUserQuery(ctx, query); err != nil {
				return err
			}
		} else {
			break
		}
	}
	return nil
}

func (g *Gromit) handleUserQuery(ctx context.Context, query string) error {
	assister, err := g.AssisterCreator.GetAssister(g.String("agent"), g.String("model"))
	if err != nil {
		return err
	}
	prompt := g.String("systemPrompt")
	if prompt == "" {
		prompt = systemPrompt
	}
	prompt = addEnvironmentInfo(g.configuration.osInfo, prompt)
	exeCommand, err := assister.GetTerminalCommand(ctx, query, prompt)
	if err != nil {
		return err
	}
	g.print("In order to do that, you need to run:")
	g.print(exeCommand)

	confirmation, err := g.askConfirmation("Would you like to run this command?")
	if err != nil {
		g.print("Error reading your response")
		return err
	}
	if confirmation.confirmed {
		err = g.executeCommand(exeCommand)
		if err != nil {
			return err
		}
	} else {
		g.print("You chose not to execute this command.")
	}
	return nil
}

// adds environment info such as OS, available shells, etc to the system prompt for the AI
func addEnvironmentInfo(osInfo operatingSystemInfo, systemPrompt string) string {
	result := fmt.Sprintf("%s. User's operating system is %s", systemPrompt, osInfo.operatingSystem)
	if osInfo.currentShell != "" {
		result = fmt.Sprintf("%s. User's current shell is %s", result, osInfo.currentShell)
	}
	return result
}

type userConfirmation struct {
	confirmed bool
}

func (g *Gromit) askConfirmation(message string) (userConfirmation, error) {
	if !g.configuration.askForConfirmation {
		return userConfirmation{
			confirmed: true,
		}, nil
	}
	g.print(message)
	var c userConfirmation
	var userResponse string
	n, err := fmt.Scanln(&userResponse)
	userResponse = strings.ToLower(userResponse)
	switch {
	case n == 0:
		g.print("You didn't confirm your choice! Please reply with yes(y) or no(n).")
		return g.askConfirmation(message)
	case err != nil:
		g.print("Error reading your response")
		return c, err
	case userResponse == "yes" || userResponse == "y":
		c.confirmed = true
	case userResponse == "no" || userResponse == "n":
		c.confirmed = false
	}
	return c, nil
}

func (g *Gromit) executeCommand(command string) error {
	g.print("Running the command...")
	c := exec.Command("sh", "-c", command)
	output, err := c.CombinedOutput()
	if err != nil {
		g.print(fmt.Sprintf("error running the command: %s", err.Error()))
		return err
	} else {
		const lineWidth = 50
		g.print("Command output:")
		g.print(strings.Repeat("-", lineWidth))
		g.print(string(output))
		g.print(strings.Repeat("-", lineWidth))
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
	config := configuration{
		promptPrefix:       "⚡️🐶",
		w:                  os.Stdout,
		askForConfirmation: true,
		osInfo:             getOperatingSystemInfo(),
	}
	gromit := Gromit{
		AssisterCreator: a,
		Command: cli.Command{
			Usage: "A command line helper that uses generative AI to generate commands based on user input.",
			Name:  "gromit",
			Flags: flags,
		},
		configuration: &config,
	}
	for _, apply := range mods {
		if err := apply(gromit.configuration); err != nil {
			return nil, err
		}
	}
	gromit.Action = gromit.actionGromit
	gromit.messagePrinter = messagePrinter{
		promptPrefix: gromit.configuration.promptPrefix,
		w:            gromit.configuration.w,
	}

	return &gromit, nil
}
