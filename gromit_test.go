package main

import (
	"bytes"
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExecuteCommand(t *testing.T) {
	var buff bytes.Buffer
	g, err := NewGromit(&mockAIProvider{}, WithWriter(&buff))
	require.NoError(t, err)
	err = g.executeCommand("ls")
	require.NoError(t, err)
	require.Contains(t, buff.String(), "gromit_test.go")
}

func TestGetOperatingSystemInfo(t *testing.T) {
	systemInfo := getOperatingSystemInfo()
	require.Equal(t, runtime.GOOS, systemInfo.operatingSystem)
	require.Equal(t, "\n", systemInfo.delimiter)
	acceptableShells := []string{"zsh", "bash"}
	for _, s := range acceptableShells {
		if strings.Contains(systemInfo.currentShell, s) {
			return
		}
	}
	t.Fail()
}

func TestMessagePrinter(t *testing.T) {
	var buff bytes.Buffer
	p := messagePrinter{
		promptPrefix: "✌️",
		w:            &buff,
	}
	p.print("hello")
	require.Equal(t, "✌️ hello\n", buff.String())
}

func TestConfigurationPromptPrefix(t *testing.T) {
	var buff bytes.Buffer
	g, err := NewGromit(&mockAIProvider{}, WithPromptPrefix("🏝️"), WithWriter(&buff))
	require.NoError(t, err)
	g.Run(t.Context(), []string{})
	require.Contains(t, buff.String(), "🏝️ Please run ./gromit --help to see usage")
}

func TestWhenAIProviderFailsToCreateAssister(t *testing.T) {
	m := &mockAIProvider{
		assisterError: errors.New("Unable to create assister"),
	}
	g, err := NewGromit(m)
	require.NoError(t, err)
	err = g.handleUserQuery(t.Context(), "some query")
	require.EqualError(t, err, "Unable to create assister")
}

func TestWhenAIProviderFailsToFindTheCommand(t *testing.T) {
	m := &mockAIProvider{
		commandError: errors.New("unable to find the correct command"),
	}
	g, err := NewGromit(m)
	require.NoError(t, err)
	err = g.Run(t.Context(), []string{"Find", "some", "commmand"})
	require.EqualError(t, err, "unable to find the correct command")
}

func TestAIAssisterFindingCorrectCommand(t *testing.T) {
	var buff bytes.Buffer
	m := &mockAIProvider{
		commandResult: "ls",
	}
	g, err := NewGromit(m, WithWriter(&buff), WithPromptPrefix("🐶"), WithAskForConfirmation(false))
	require.NoError(t, err)

	g.Run(t.Context(), []string{"gromit", "--model", "myModel", "--agent", "myAgent", "--systemPrompt", "myPrompt", "I", "want", "to", "list", "all", "files", "in", "current", "directory"})
	result := buff.String()
	require.Contains(t, result, "🐶 In order to do that, you need to run")
	require.Contains(t, result, "🐶 ls")
	require.Contains(t, result, "README.md")
	require.Contains(t, result, "🐶 How can I help?")

	require.Equal(t, "myAgent", m.actualAgent)
	require.Equal(t, "myModel", m.actualModel)
	require.Equal(t, "myPrompt", m.actualSystemMessage)
	require.Equal(t, "I want to list all files in current directory", m.actualUserMessage)
}

type mockAIProvider struct {
	assisterError error
	commandError  error
	commandResult string

	actualAgent         string
	actualModel         string
	actualSystemMessage string
	actualUserMessage   string
}

func (m *mockAIProvider) GetAssister(agent string, model string) (Assister, error) {
	m.actualAgent = agent
	m.actualModel = model
	if m.assisterError != nil {
		return nil, m.assisterError
	}
	return m, nil
}

func (m *mockAIProvider) GetTerminalCommand(ctx context.Context, userMessage string, systemMessage string) (string, error) {
	m.actualSystemMessage = systemMessage
	m.actualUserMessage = userMessage
	if m.commandError != nil {
		return "", m.commandError
	}
	return m.commandResult, nil
}
