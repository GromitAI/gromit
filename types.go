package main

import "io"

type systemInfo struct {
	operatingSystem string
	currentShell    string
	delimiter       string
	kernelInfo      string
	pathContent     []string
}

type messagePrinter struct {
	w            io.Writer
	promptPrefix string
	delimiter    string
}

type configuration struct {
	AiParameters
	promptPrefix       string
	w                  io.Writer
	askForConfirmation bool
	systemInfo
}

type userConfirmation struct {
	confirmed bool
}

type ConfigurationModifier func(*configuration) error

type AiParameters struct {
	systemPrompt string
	agent        string
	model        string
	apiKey       string
	maxTokens    int64
}

type AiResponse struct {
	Command  string `json:"command"`
	Response string `json:"response"`
	Exit     bool   `json:"exit"`
}

type Conversation struct {
	Role    Role
	Message string
}

type Role int

const (
	SystemRole Role = iota
	UserRole
	AssistantRole
)
