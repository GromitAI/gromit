package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExecuteCommand(t *testing.T) {
	g, err := NewGromit()
	require.NoError(t, err)
	var buff bytes.Buffer
	g.Writer = &buff
	err = g.executeCommand("ls")
	require.NoError(t, err)
	require.Contains(t, buff.String(), "gromit_test.go")
}

func TestConfigurationEmoji(t *testing.T) {
	g, err := NewGromit(WithPromptPrefix("🏝️"))
	require.NoError(t, err)
	var buff bytes.Buffer
	g.Writer = &buff
	g.Run(t.Context(), []string{})
	require.Equal(t, "🏝️ Please specify which linux command you need help with!\n", buff.String())
}

func TestOpenAIFindingCorrectCommand(t *testing.T) {
	g, err := NewGromit()
	require.NoError(t, err)
	var buff bytes.Buffer
	g.Writer = &buff
	g.Run(t.Context(), []string{"I", "want", "to", "list", "all", "files", "in", "current", "directory", "including", "hidden", "files"})
	result := buff.String()
	require.Contains(t, result, "🐶 In order to do that, you need to run")
	require.Contains(t, result, "🐶 ls -a")
	require.Contains(t, result, "🐶 Would you like to run this command?")
	require.Contains(t, result, "🐶 You didn't specify whether you want to run this command!")
}
