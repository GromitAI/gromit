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

func TestGetAvailablePathExecutables(t *testing.T) {
	result := getAvailablePathExecutables()
	for _, r := range result {
		if strings.Contains(r, "gofmt") {
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

func TestWhenAIProviderFailsToCreateAssister(t *testing.T) {
	m := &mockAIProvider{
		assisterError: errors.New("Unable to create assister"),
	}
	g, err := NewGromit(m)
	require.NoError(t, err)
	_, err = g.extractResponseForQuery(t.Context(), &[]Conversation{})
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
		aiResponse: "json```{\"Command\": \"ls\",\"Response\": \"\",\"Exit\": false}```",
	}
	g, err := NewGromit(m, WithWriter(&buff), WithPromptPrefix("🐶"), WithAskForConfirmation(false))
	require.NoError(t, err)
	g.Reader = strings.NewReader("I want to list all files in current directory\n")
	g.Run(t.Context(), []string{"gromit", "--model", "myModel", "--agent", "myAgent",
		"--apiKey=key1234", "--maxTokens=2000",
		"--systemPrompt", "myPrompt", "hello", "my", "ai", "friend!"})
	result := buff.String()
	require.Contains(t, result, "🐶 In order to do that, you need to run")
	require.Contains(t, result, "🐶 ls")
	require.Contains(t, result, "README.md")

	require.Equal(t, "myAgent", m.actualAiParameters.agent)
	require.Equal(t, "myModel", m.actualAiParameters.model)
	require.Equal(t, "key1234", m.actualAiParameters.apiKey)
	require.Equal(t, int64(2000), m.actualAiParameters.maxTokens)
	require.Contains(t, m.actualAiParameters.systemPrompt, "myPrompt")
	require.Contains(t, m.actualAiParameters.systemPrompt, "User's operating system is")
	require.Contains(t, m.actualAiParameters.systemPrompt, "User's current shell is")
	require.Contains(t, m.actualAiParameters.systemPrompt, "User's available path commands are")

	//_ := []Conversation {
	//	{Role: SystemRole, Text: "myPrompt"},
	//	{Role: UserRole, Text: "hello my ai friend!"},
	//	{Role: UserRole, Text: "hello my ai friend!"},
	//}

	for _, c := range *m.actualConversations {
		if c.Role == SystemRole {
			require.Contains(t, c.Text, "myPrompt")
		} else if c.Role == UserRole {
			require.Contains(t, c.Text, "I want to list all files in current directory")
		} else {
			require.Failf(t, "Unknown conversation %s", c.Text)
		}
	}
}

type mockAIProvider struct {
	assisterError error
	commandError  error
	aiResponse    string

	actualConversations *[]Conversation

	actualAiParameters AiParameters
}

func (m *mockAIProvider) GetAssister(p AiParameters) (Assister, error) {
	m.actualAiParameters = p
	if m.assisterError != nil {
		return nil, m.assisterError
	}
	return m, nil
}

func (m *mockAIProvider) GetTerminalCommand(ctx context.Context, conversations *[]Conversation) (string, error) {
	m.actualConversations = conversations
	if m.commandError != nil {
		return "", m.commandError
	}
	return m.aiResponse, nil
}
