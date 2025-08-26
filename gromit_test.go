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
	systemInfo := getSystemInfo()
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
		delimiter:    "\r\n",
	}
	p.print("hello")
	require.Equal(t, "✌️ hello \r\n", buff.String())
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

	g.Run(t.Context(), []string{"gromit", "--model", "myModel", "--agent", "myAgent",
		"--apiKey=key1234", "--maxTokens=2000",
		"--systemPrompt", "myPrompt", "I", "want", "to", "list", "all", "files", "in", "current", "directory"})
	result := buff.String()
	require.Contains(t, result, "🐶 In order to do that, you need to run")
	require.Contains(t, result, "🐶 ls")
	require.Contains(t, result, "README.md")
	require.Contains(t, result, "🐶 How can I help?")

	require.Equal(t, "myAgent", m.actualAiParameters.agent)
	require.Equal(t, "myModel", m.actualAiParameters.model)
	require.Equal(t, "key1234", m.actualAiParameters.apiKey)
	require.Equal(t, int64(2000), m.actualAiParameters.maxTokens)
	require.Contains(t, m.actualAiParameters.systemPrompt, "myPrompt")
	require.Contains(t, m.actualAiParameters.systemPrompt, "User's operating system is")
	require.Contains(t, m.actualAiParameters.systemPrompt, "User's current shell is")
	require.Equal(t, "I want to list all files in current directory", m.actualUserMessage)
}

type mockAIProvider struct {
	assisterError error
	commandError  error
	commandResult string

	actualUserMessage string

	actualAiParameters aiParameters
}

func (m *mockAIProvider) GetAssister(p aiParameters) (Assister, error) {
	m.actualAiParameters = p
	if m.assisterError != nil {
		return nil, m.assisterError
	}
	return m, nil
}

func (m *mockAIProvider) GetTerminalCommand(ctx context.Context, userMessage string) (string, error) {
	m.actualUserMessage = userMessage
	if m.commandError != nil {
		return "", m.commandError
	}
	return m.commandResult, nil
}
