package main

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/grantcarthew/start/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		red := color.New(color.FgRed)
		red.Fprint(os.Stderr, "Error: ")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
