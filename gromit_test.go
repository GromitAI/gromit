package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteCommand(t *testing.T) {
	g, err := NewGromit()
	var buff bytes.Buffer
	g.Writer = &buff
	require.NoError(t, err)
	err = g.executeCommand("ls")
	require.NoError(t, err)
	assert.Contains(t, buff.String(), "gromit_test.go")
}

func TestConfigurationEmoji(t *testing.T) {
	gromit, err := NewGromit(WithEmoji("🏝️"))
	assert.NoError(t, err)
	var buff bytes.Buffer
	gromit.Writer = &buff
	gromit.Run(context.Background(), []string{})
	assert.Equal(t, "🏝️ Please specify which linux command you need help with!\n", buff.String())
}
