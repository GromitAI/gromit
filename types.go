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
	promptPrefix       string
	w                  io.Writer
	askForConfirmation bool
	systemInfo
}

type userConfirmation struct {
	confirmed bool
}

type ConfigurationModifier func(*configuration) error
