package cli

import (
	"fmt"

	"github.com/TimDelRey/daily-work-log/internal/config"
	"github.com/TimDelRey/daily-work-log/internal/domain"
)

type DomainReportBuilder interface {
	BuildToday(authorEmail string) (domain.Report, error)
	BuildYesterday(authorEmail string) (domain.Report, error)
	BuildStatus() (domain.Report, error)
}

type ReportFormatter interface {
	Format(report domain.Report) string
}

// CommandReportBuilder connects CLI commands to report construction and formatting.
type CommandReportBuilder struct {
	Reports      DomainReportBuilder
	Formatter    ReportFormatter
	Config       config.Config
	AuthorSource config.AuthorEmailSource
}

func (b CommandReportBuilder) BuildToday() (string, error) {
	email, err := b.authorEmail()
	if err != nil {
		return "", err
	}
	report, err := b.Reports.BuildToday(email)
	if err != nil {
		return "", err
	}
	return b.Formatter.Format(report), nil
}

func (b CommandReportBuilder) BuildYesterday() (string, error) {
	email, err := b.authorEmail()
	if err != nil {
		return "", err
	}
	report, err := b.Reports.BuildYesterday(email)
	if err != nil {
		return "", err
	}
	return b.Formatter.Format(report), nil
}

func (b CommandReportBuilder) BuildStatus() (string, error) {
	report, err := b.Reports.BuildStatus()
	if err != nil {
		return "", err
	}
	return b.Formatter.Format(report), nil
}

func (b CommandReportBuilder) authorEmail() (string, error) {
	email, err := config.ResolveAuthorEmail(b.Config, b.AuthorSource)
	if err != nil {
		return "", fmt.Errorf("resolve author email: %w", err)
	}
	return email, nil
}
