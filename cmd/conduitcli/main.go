package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli"
)

// fatal exits the process and prints out error information
func fatal(err error) {
	fmt.Fprintf(os.Stderr, "[conduitcli] %v\n", err)
	os.Exit(1)
}

// main is the main entry point for the conduit CLI tool
func main() {
	app := cli.NewApp()
	app.Name = "conduitcli"
	app.Usage = "Control panel for the Conduit Plugin Manager (conduit)"
	app.Commands = []cli.Command{
		testCommand,
	}
	if err := app.Run(os.Args); err != nil {
		fatal(err)
	}
}
