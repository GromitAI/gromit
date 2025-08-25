package main

import "io"

type systemInfo struct {
	operatingSystem string
	currentShell    string
	delimiter       string
	kernelInfo      string
}

type messagePrinter struct {
	w            io.Writer
	promptPrefix string
	delimiter    string
}

type configuration struct {
	aiParameters
	promptPrefix       string
	w                  io.Writer
	askForConfirmation bool
	systemInfo
}

type userConfirmation struct {
	confirmed bool
}

type ConfigurationModifier func(*configuration) error

type aiParameters struct {
	systemPrompt string
	agent        string
	model        string
	apiKey       string
	maxTokens    int64
}
