package main

import (
	"bytes"
	"context"
	"errors"
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

func TestConfigurationPromptPrefix(t *testing.T) {
	var buff bytes.Buffer
	g, err := NewGromit(&mockAIProvider{}, WithPromptPrefix("🏝️"), WithWriter(&buff))
	require.NoError(t, err)
	g.Run(t.Context(), []string{})
	require.Equal(t, "🏝️ Please specify which linux command you need help with!\n", buff.String())
}

func TestWhenAIProviderFailsToCreateAssister(t *testing.T) {
	m := &mockAIProvider{
		assisterError: errors.New("Unable to create assister"),
	}
	g, err := NewGromit(m)
	require.NoError(t, err)
	err = g.Run(t.Context(), []string{})
	require.EqualError(t, err, "Unable to create assister")
}

func TestWhenAIProviderFailsToFindTheCommand(t *testing.T) {
	m := &mockAIProvider{
		commandError: errors.New("Unable to find the correct command"),
	}
	g, err := NewGromit(m)
	require.NoError(t, err)
	err = g.Run(t.Context(), []string{"Find", "some", "commmand"})
	require.EqualError(t, err, "Unable to find the correct command")
}

func TestOpenAIFindingCorrectCommand(t *testing.T) {
	var buff bytes.Buffer
	m := &mockAIProvider{
		commandResult: "ls -la",
	}
	g, err := NewGromit(m, WithWriter(&buff))
	require.NoError(t, err)

	g.Run(t.Context(), []string{"I", "want", "to", "list", "all", "files", "in", "current", "directory"})
	result := buff.String()
	require.Contains(t, result, "🐶 In order to do that, you need to run")
	require.Contains(t, result, "🐶 ls -la")
	require.Contains(t, result, "🐶 Would you like to run this command?")
	require.Contains(t, result, "🐶 You didn't specify whether you want to run this command!")
}

type mockAIProvider struct {
	assisterError error
	commandError  error
	commandResult string
}

func (m *mockAIProvider) GetAssister(agent string, model string) (Assister, error) {
	if m.assisterError != nil {
		return nil, m.assisterError
	}
	return m, nil
}

func (m *mockAIProvider) GetTerminalCommand(ctx context.Context, userMessage string, systemMessage string) (string, error) {
	if m.commandError != nil {
		return "", m.commandError
	}
	return m.commandResult, nil
}
