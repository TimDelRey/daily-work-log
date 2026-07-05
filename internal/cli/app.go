package cli

import (
	"errors"
	"fmt"
	"io"
)

const usage = `Usage:
  worklog today
  worklog yesterday
  worklog status

Commands:
  today      Show today's worklog report
  yesterday  Show yesterday's worklog report
  status     Show current uncommitted files
`

const (
	commandToday     = "today"
	commandYesterday = "yesterday"
	commandStatus    = "status"

	placeholderTodayReport     = "No report data yet. The today report builder will be added in the next stages."
	placeholderYesterdayReport = "No report data yet. The yesterday report builder will be added in the next stages."
	placeholderStatusReport    = "No report data yet. The status report builder will be added in the next stages."
)

var ErrUnknownCommand = errors.New("unknown command")

type ReportBuilder interface {
	BuildToday() (string, error)
	BuildYesterday() (string, error)
	BuildStatus() (string, error)
}

type App struct {
	out     io.Writer
	errOut  io.Writer
	builder ReportBuilder
}

func NewApp(out, errOut io.Writer) App {
	return App{
		out:     out,
		errOut:  errOut,
		builder: placeholderBuilder{},
	}
}

func NewAppWithBuilder(out, errOut io.Writer, builder ReportBuilder) App {
	return App{
		out:     out,
		errOut:  errOut,
		builder: builder,
	}
}

func (a App) Run(args []string) error {
	if len(args) == 0 {
		a.printUsage(a.errOut)
		return fmt.Errorf("%w: command is required", ErrUnknownCommand)
	}

	if len(args) > 1 {
		a.printUsage(a.errOut)
		return fmt.Errorf("%w: too many arguments", ErrUnknownCommand)
	}

	switch args[0] {
	case commandToday:
		return a.printReport(a.builder.BuildToday)
	case commandYesterday:
		return a.printReport(a.builder.BuildYesterday)
	case commandStatus:
		return a.printReport(a.builder.BuildStatus)
	case "-h", "--help", "help":
		a.printUsage(a.out)
		return nil
	default:
		a.printUsage(a.errOut)
		return fmt.Errorf("%w: %s", ErrUnknownCommand, args[0])
	}
}

func (a App) printReport(build func() (string, error)) error {
	report, err := build()
	if err != nil {
		return fmt.Errorf("build report: %w", err)
	}

	if report == "" {
		return nil
	}

	_, err = fmt.Fprintln(a.out, report)
	if err != nil {
		return fmt.Errorf("write report: %w", err)
	}

	return nil
}

func (a App) printUsage(w io.Writer) {
	fmt.Fprint(w, usage)
}

type placeholderBuilder struct{}

func (placeholderBuilder) BuildToday() (string, error) {
	return placeholderTodayReport, nil
}

func (placeholderBuilder) BuildYesterday() (string, error) {
	return placeholderYesterdayReport, nil
}

func (placeholderBuilder) BuildStatus() (string, error) {
	return placeholderStatusReport, nil
}
