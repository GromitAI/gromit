package main

import (
	"os"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/openai/openai-go"
	"github.com/stretchr/testify/require"
)

type assisterTestCase struct {
	name              string
	inputAiParameters AiParameters
	expectedModel     string
	err               string
}

func TestGetOpenAIAssister(t *testing.T) {
	tests := []assisterTestCase{
		{
			name:          "no agent and model should default to openAI assister",
			expectedModel: openai.ChatModelGPT4o,
		},
		{
			name: "OpenAI agent with given model should create correct assister",
			inputAiParameters: AiParameters{
				agent:     openAIAgent,
				model:     "gpt-5o-mini",
				apiKey:    "key1234",
				maxTokens: defaultMaxTokens,
			},
			expectedModel: "gpt-5o-mini",
		},
		{
			name: "Unknown agent should result in error",
			inputAiParameters: AiParameters{
				agent:     "Unknown agent",
				model:     "unknown model",
				apiKey:    "key1234",
				maxTokens: defaultMaxTokens,
			},
			err: "cannot create AI agent for Unknown agent and model unknown model",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var creator defaultAIAssisterCreator
			assister, err := creator.GetAssister(test.inputAiParameters)
			if test.err != "" {
				require.EqualError(t, err, test.err)
			} else {
				require.NoError(t, err)
				if openAIAssister, ok := assister.(*OpenAIAssister); ok {
					require.Equal(t, test.expectedModel, openAIAssister.model)
					require.Equal(t, test.inputAiParameters.apiKey, openAIAssister.apiKey)
					require.Equal(t, test.inputAiParameters.maxTokens, openAIAssister.maxTokens)
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
			name: "Anthropic agent with no model should default to Claude 3.5 Haiku latest",
			inputAiParameters: AiParameters{
				agent:     anthropicAIAgent,
				apiKey:    "key1234",
				maxTokens: defaultMaxTokens,
			},
			expectedModel: string(anthropic.ModelClaude3_5HaikuLatest),
		},
		{
			name: "Anthropic agent with given model should create correct assister",
			inputAiParameters: AiParameters{
				agent:     anthropicAIAgent,
				model:     string(anthropic.ModelClaudeOpus4_0),
				apiKey:    "key1234",
				maxTokens: defaultMaxTokens,
			},
			expectedModel: string(anthropic.ModelClaudeOpus4_0),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var creator defaultAIAssisterCreator
			assister, err := creator.GetAssister(test.inputAiParameters)
			if test.err != "" {
				require.EqualError(t, err, test.err)
			} else {
				require.NoError(t, err)
				if anthropicAssister, ok := assister.(*AnthropicAIAssister); ok {
					require.Equal(t, test.expectedModel, anthropicAssister.model)
					require.Equal(t, test.inputAiParameters.apiKey, anthropicAssister.apiKey)
					require.Equal(t, test.inputAiParameters.maxTokens, anthropicAssister.maxTokens)
				} else {
					require.Fail(t, "Expected Anthropic AI assister")
				}
			}
		})
	}
}

func TestGetGeminiAssister(t *testing.T) {
	tests := []assisterTestCase{
		{
			name: "Gemini agent with no model should default to gemini-2.5-flash-lite",
			inputAiParameters: AiParameters{
				agent:     geminiAIAgent,
				apiKey:    "key1234",
				maxTokens: defaultMaxTokens,
			},
			expectedModel: geminiFlashLite,
		},
		{
			name: "Gemini agent with given model should create correct assister",
			inputAiParameters: AiParameters{
				agent:     geminiAIAgent,
				model:     "gemini-2.5-flash-preview-tts",
				apiKey:    "key1234",
				maxTokens: defaultMaxTokens,
			},
			expectedModel: "gemini-2.5-flash-preview-tts",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var creator defaultAIAssisterCreator
			assister, err := creator.GetAssister(test.inputAiParameters)
			if test.err != "" {
				require.EqualError(t, err, test.err)
			} else {
				require.NoError(t, err)
				if geminiAssister, ok := assister.(*GeminiAIAssister); ok {
					require.Equal(t, test.expectedModel, geminiAssister.model)
					require.Equal(t, test.inputAiParameters.apiKey, geminiAssister.apiKey)
					require.Equal(t, test.inputAiParameters.maxTokens, geminiAssister.maxTokens)
				} else {
					require.Fail(t, "Expected Gemini AI assister")
				}
			}
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
		{
			name:  "Should create correct command when calling Gemini llm",
			agent: geminiAIAgent,
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
					AiParameters{
						model:        openai.ChatModelGPT4o,
						systemPrompt: systemPrompt,
					},
				}
			case anthropicAIAgent:
				assister = &AnthropicAIAssister{
					AiParameters{
						model:        string(anthropic.ModelClaude3_5HaikuLatest),
						systemPrompt: systemPrompt,
					},
				}
			case geminiAIAgent:
				assister = &GeminiAIAssister{
					AiParameters{
						model:        geminiFlash,
						systemPrompt: systemPrompt,
					},
				}
			}
			conversations := &[]Conversation{
				{Text: systemPrompt, Role: SystemRole},
				{Text: "I want to list all files in current directory", Role: UserRole},
			}
			command, err := assister.GetTerminalCommand(t.Context(), conversations)
			require.NoError(t, err)
			require.Contains(t, command, "ls")
		})
	}
}
