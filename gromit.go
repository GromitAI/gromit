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

const systemPrompt = `You are an assistant providing terminal commands based on user's questions. 
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

func getSystemInfo() systemInfo {
	o := runtime.GOOS
	var eol, shell, kernelInfo string
	var err error
	if strings.Contains(strings.ToLower(o), "windows") {
		eol = "\r\n"
		kernelInfo, err = runCommand("cmd", "/C", "ver")
	} else {
		eol = "\n"
		shell = os.Getenv("SHELL")
		kernelInfo, err = runCommand("uname", "-a")
	}
	if err != nil {
		fmt.Println("Error retrieving runtime information: ", err)
	}
	return systemInfo{
		operatingSystem: o,
		currentShell:    shell,
		delimiter:       eol,
		kernelInfo:      kernelInfo,
	}
}

func (m *messagePrinter) print(s string) {
	fmt.Fprintf(m.w, "%s %s %s", m.promptPrefix, s, m.delimiter)
}

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
	prompt := g.String("systemPrompt")
	if prompt == "" {
		prompt = systemPrompt
	}
	prompt = addEnvironmentInfo(g.configuration.systemInfo, prompt)
	g.configuration.aiParameters = aiParameters{
		maxTokens:    g.Int64("maxToken"),
		apiKey:       g.String("apiKey"),
		agent:        g.String("agent"),
		model:        g.String("model"),
		systemPrompt: prompt,
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
	assister, err := g.AssisterCreator.GetAssister(g.configuration.aiParameters)
	if err != nil {
		return err
	}
	exeCommand, err := assister.GetTerminalCommand(ctx, query)
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
func addEnvironmentInfo(systemInfo systemInfo, systemPrompt string) string {
	result := fmt.Sprintf("%s. User's operating system is %s", systemPrompt, systemInfo.operatingSystem)
	if systemInfo.kernelInfo != "" {
		result = fmt.Sprintf("%s. User's kernel info is %s", result, systemInfo.kernelInfo)
	}
	if systemInfo.currentShell != "" {
		result = fmt.Sprintf("%s. User's current shell is %s", result, systemInfo.currentShell)
	}
	return result
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
	output, err := runCommand(command)
	if err != nil {
		g.print(fmt.Sprintf("error running the command: %s", err.Error()))
		return err
	} else {
		const lineWidth = 50
		g.print("Command output:")
		g.print(strings.Repeat("-", lineWidth))
		g.print(output)
		g.print(strings.Repeat("-", lineWidth))
		return nil
	}
}

func runCommand(command string, args ...string) (string, error) {
	allArgs := append([]string{"-c", command}, args...)
	c := exec.Command("sh", allArgs...)
	output, err := c.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
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
		&cli.StringFlag{
			Name:  "apiKey",
			Usage: "The API key to use for given AI agent. By default it is read from environment variables.",
		},
		&cli.Int32Flag{
			Name:  "maxTokens",
			Usage: "Maximum number of tokens for AI agents to generate",
		},
	}
	config := configuration{
		promptPrefix:       "⚡️🐶",
		w:                  os.Stdout,
		askForConfirmation: true,
		systemInfo:         getSystemInfo(),
		aiParameters:       aiParameters{},
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
		delimiter:    gromit.configuration.systemInfo.delimiter,
	}

	return &gromit, nil
}
