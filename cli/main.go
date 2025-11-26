// Package main provides the entry point for the Drata Agent CLI.
// This is a command-line interface version of the Drata Agent that can be
// compiled for any platform and used without a GUI.
package main

import (
	"github.com/drata/drata-agent-cli/cmd"
)

func main() {
	cmd.Execute()
}
