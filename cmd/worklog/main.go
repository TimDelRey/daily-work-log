package main

import (
	"fmt"
	"os"

	"github.com/TimDelRey/daily-work-log/internal/cli"
)

func main() {
	app := cli.NewApp(os.Stdout, os.Stderr)
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "worklog: %v\n", err)
		os.Exit(1)
	}
}
