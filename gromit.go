package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"

	"github.com/urfave/cli/v3"
)

const systemPrompt = `You are an assistant providing terminal commands based on user's questions.
	Also you should keep up the conversation with the user in case there is no terminal command to execute.
	You will be given a question about how to do something in the CLI environment and then find out what command to execute and provide the command.
	Provide your response in the following json format:
	{
		"command": "the command to execute, can be empty if a response is provided",
		"response": "the response to the user, can be empty if a command is provided"
		"exit": "true if the user wants to exit, false otherwise"
	}
	For example, if question is about listing all files in a directory for linux, respond with 
	{
		"command": "ls",
		"response": "",
		"exit": false
	}
	If no question is asked by user, continue the conversation. If they want to exit, respond with 
	{
		"command": "",
		"response": "",
		"exit": true
	}`

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
	isWindows := strings.Contains(strings.ToLower(o), "windows")
	if isWindows {
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
	var pathContent []string
	if !isWindows {
		pathContent = getAvailablePathExecutables()
	}
	return systemInfo{
		operatingSystem: o,
		currentShell:    shell,
		delimiter:       eol,
		kernelInfo:      kernelInfo,
		pathContent:     pathContent,
	}
}

func getAvailablePathExecutables() []string {
	var result []string
	const maxEntries = 300
	const entryPerDirectory = 10
	path := os.Getenv("PATH")
	pathDirectories := strings.Split(path, string(os.PathListSeparator))
	var counter int
	for _, d := range pathDirectories {
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		var fileInfos []os.FileInfo

		for _, entry := range entries {
			if entry.IsDir() || !entry.Type().IsRegular() {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			fileInfos = append(fileInfos, info)
		}

		sort.Slice(fileInfos, func(i, j int) bool { //sort by last modified time, assuming those files are more useful
			return fileInfos[i].ModTime().After(fileInfos[j].ModTime())
		})

		for _, info := range fileInfos {
			if info.Mode()&0111 != 0 { //if file is executable by user, group or others
				result = append(result, info.Name())
				counter++
			}
			if len(result) >= maxEntries {
				return result
			}
			if counter >= entryPerDirectory {
				counter = 0
				break
			}
		}
	}
	return result
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
	prompt := g.String("systemPrompt")
	if prompt == "" {
		prompt = systemPrompt
	}
	prompt = addEnvironmentInfo(g.configuration.systemInfo, prompt)
	g.configuration.AiParameters = AiParameters{
		maxTokens:    g.Int64("maxTokens"),
		apiKey:       g.String("apiKey"),
		agent:        g.String("agent"),
		model:        g.String("model"),
		systemPrompt: prompt,
	}

	for ctx.Err() == nil {
		response, err := g.extractResponseForQuery(ctx, query)
		if err != nil {
			return err
		}
		err = g.handleAiResponse(ctx, response)
		if err != nil {
			return err
		}
		reader := bufio.NewReader(g.Reader)
		query, err = reader.ReadString('\n')
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *Gromit) extractResponseForQuery(ctx context.Context, query string) (AiResponse, error) {
	var result AiResponse
	assister, err := g.AssisterCreator.GetAssister(g.configuration.AiParameters)
	if err != nil {
		return result, err
	}
	if query == "" {
		query = "Can you please introduce yourself or continue the conversation?"
	}
	response, err := assister.GetTerminalCommand(ctx, query)
	if err != nil {
		return result, err
	}
	for _, s := range []string{"json", "```"} {
		response = strings.ReplaceAll(response, s, "")
	}
	strings.ReplaceAll(response, "json", "")
	if !json.Valid([]byte(response)) {
		return result, fmt.Errorf("received invalid json response: %s", response)
	}
	if err = json.Unmarshal([]byte(response), &result); err != nil {
		return result, fmt.Errorf("failed to unmarshal json response: \n %s \n error: %s", response, err.Error())
	}
	return result, nil
}

func (g *Gromit) handleTerminalCommand(ctx context.Context, terminalCommand string) error {
	g.print("In order to do that, you need to run:")
	g.print(terminalCommand)
	confirmation, err := g.askConfirmation("Would you like to run this command?")
	if err != nil {
		g.print("Error reading your response")
		return err
	}
	if confirmation.confirmed {
		err = g.executeCommand(terminalCommand)
		if err != nil {
			return err
		}
	} else {
		g.print("You chose not to execute this command.")
	}
	return nil
}

func (g *Gromit) handleAiResponse(ctx context.Context, aiResponse AiResponse) error {
	if aiResponse.Command != "" {
		err := g.handleTerminalCommand(ctx, aiResponse.Command)
		if err != nil {
			return err
		}
	} else if aiResponse.Response != "" {
		g.print(aiResponse.Response)
	} else if aiResponse.Exit {
		os.Exit(0)
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
	if len(systemInfo.pathContent) > 0 {
		result = fmt.Sprintf("%s. User's available path commands are: %s", result, systemInfo.pathContent)
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
		&cli.Int64Flag{
			Name:  "maxTokens",
			Usage: "Maximum number of tokens for AI agents to generate",
		},
	}
	config := configuration{
		promptPrefix:       "⚡️🐶",
		w:                  os.Stdout,
		askForConfirmation: true,
		systemInfo:         getSystemInfo(),
		AiParameters:       AiParameters{},
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
