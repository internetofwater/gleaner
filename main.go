package main

import (
	cmd "github.com/gleanerio/gleaner/cmd"
)

func main() {
	// The CLI is separated into a different function so it can be e2e tested
	// since in golang you cannot import the main package
	cmd.Execute()
}
