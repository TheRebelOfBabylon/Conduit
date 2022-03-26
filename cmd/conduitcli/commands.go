package main

import (
	"fmt"

	"github.com/urfave/cli"
)

var testCommand = cli.Command{
	Name:      "test",
	Usage:     "Test command",
	ArgsUsage: "string",
	Description: `
	A test command which prints an inputted string to the console`,
	Action: test,
}

// testCommand accepts a string as an argument and prints it to the console
func test(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return cli.ShowCommandHelp(ctx, "test")
	}
	str := ctx.Args().First()
	fmt.Println(str)
	return nil
}
