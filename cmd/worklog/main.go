package main

import (
	"fmt"
	"os"

	"github.com/TimDelRey/daily-work-log/internal/cli"
	"github.com/TimDelRey/daily-work-log/internal/config"
	textformat "github.com/TimDelRey/daily-work-log/internal/format"
	gitadapter "github.com/TimDelRey/daily-work-log/internal/git"
	"github.com/TimDelRey/daily-work-log/internal/report"
)

func main() {
	gitClient := gitadapter.Client{Directory: "."}
	commandBuilder := cli.CommandReportBuilder{
		Reports:      report.Builder{Git: gitClient},
		Formatter:    textformat.Text{},
		AuthorSource: config.GitConfigSource{Directory: "."},
	}
	app := cli.NewAppWithBuilder(os.Stdout, os.Stderr, commandBuilder)
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "worklog: %v\n", err)
		os.Exit(1)
	}
}
