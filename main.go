package main

import (
	"os"
)

func main() {
	// Initialize repository
	repository := NewGitHubAPIRepository()

	// Initialize service
	service := NewActivityService(repository)

	// Initialize CLI
	cli := NewCLI(service)

	// Run CLI and exit with appropriate code
	exitCode := cli.Run(os.Args)
	os.Exit(exitCode)
}
