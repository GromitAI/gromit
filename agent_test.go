package main

import (
	"os"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/openai/openai-go"
	"github.com/stretchr/testify/require"
)

type assisterTestCase struct {
	name                   string
	inputAgent, inputModel string
	expectedModel          string
	err                    string
}

func TestGetOpenAIAssister(t *testing.T) {
	tests := []assisterTestCase{
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
			creator := defaultAIAssisterCreator{}
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

func TestGetAnthropicAssister(t *testing.T) {
	tests := []assisterTestCase{
		{
			name:          "Anthropic agent with no model should default to Claude 3.5 Haiku latest",
			inputAgent:    anthropicAIAgent,
			inputModel:    "",
			expectedModel: string(anthropic.ModelClaude3_5HaikuLatest),
			err:           "",
		},
		{
			name:          "Anthropic agent with given model should create correct assister",
			inputAgent:    anthropicAIAgent,
			inputModel:    string(anthropic.ModelClaudeOpus4_0),
			expectedModel: string(anthropic.ModelClaudeOpus4_0),
			err:           "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Run(test.name, func(t *testing.T) {
				creator := defaultAIAssisterCreator{}
				assister, err := creator.GetAssister(test.inputAgent, test.inputModel)
				if test.err != "" {
					require.EqualError(t, err, test.err)
				} else {
					require.NoError(t, err)
					if anthropicAssister, ok := assister.(*AnthropicAIAssister); ok {
						require.Equal(t, test.expectedModel, anthropicAssister.model)
					} else {
						require.Fail(t, "Expected Anthropic AI assister")
					}
				}
			})
		})
	}
}

func TestGetTerminalCommand(t *testing.T) {
	tests := []struct {
		name  string
		agent string
	}{
		{
			name:  "Should create correct command when calling OpenAI llm",
			agent: openAIAgent,
		},
		{
			name:  "Should create correct command when calling Anthropic llm",
			agent: anthropicAIAgent,
		},
	}
	if _, isset := os.LookupEnv("CI"); isset { //skip this test in GitHub CI
		t.Skip("Skipping external AI API calls in CI")
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var assister Assister
			switch test.agent {
			case openAIAgent:
				assister = &OpenAIAssister{
					model: openai.ChatModelGPT4o,
				}
			case anthropicAIAgent:
				assister = &AnthropicAIAssister{
					model: string(anthropic.ModelClaude3_5HaikuLatest),
				}
			}
			command, err := assister.GetTerminalCommand(t.Context(), "I want to list all files in current directory", systemPrompt)
			require.NoError(t, err)
			require.Contains(t, command, "ls")
		})
	}
}
