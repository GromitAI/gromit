package main

import (
	"os"
	"testing"

	"github.com/openai/openai-go"
	"github.com/stretchr/testify/require"
)

func TestGetOpenAIAssister(t *testing.T) {
	tests := []struct {
		name                   string
		inputAgent, inputModel string
		expectedModel          string
		err                    string
	}{
		{
			name:          "no agent and model should default to openAI assister",
			inputAgent:    "",
			inputModel:    "",
			expectedModel: openai.ChatModelGPT4o,
			err:           "",
		},
		{
			name:          "OpenAI agent with given model should create correct assister",
			inputAgent:    openAIAgent,
			inputModel:    "gpt-5o-mini",
			expectedModel: "gpt-5o-mini",
			err:           "",
		},
		{
			name:       "Unknown agent should result in error",
			inputAgent: "Unknown agent",
			inputModel: "unknown model",
			err:        "cannot create AI agent for Unknown agent and model unknown model",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			creator := openAIAssisterCreator{}
			assister, err := creator.GetAssister(test.inputAgent, test.inputModel)
			if test.err != "" {
				require.EqualError(t, err, test.err)
			} else {
				require.NoError(t, err)
				if openAIAssister, ok := assister.(*OpenAIAssister); ok {
					require.Equal(t, test.expectedModel, openAIAssister.model)
				} else {
					require.Fail(t, "Expected OpenAIAssister")
				}
			}
		})
	}
}

func TestGetTerminalCommand(t *testing.T) {
	if _, isset := os.LookupEnv("CI"); isset { //skip this test in GitHub CI
		t.Skip("Skipping external AI API calls in CI")
	}
	assister := OpenAIAssister{
		model: openai.ChatModelGPT4o,
	}
	command, err := assister.GetTerminalCommand(t.Context(), "I want to list all files in current directory", systemPrompt)
	require.NoError(t, err)
	require.Contains(t, command, "ls")
}
