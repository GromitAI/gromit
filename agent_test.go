package main

import (
	"testing"

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
			expectedModel: "",
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
			creator := defaultAssisterCreator{}
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
