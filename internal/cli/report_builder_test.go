package cli

import (
	"errors"
	"testing"

	"github.com/TimDelRey/daily-work-log/internal/config"
	"github.com/TimDelRey/daily-work-log/internal/domain"
)

func TestCommandReportBuilderBuildsHistoricalReportsWithResolvedAuthor(t *testing.T) {
	reports := &stubDomainReports{report: domain.Report{Branches: []domain.BranchActivity{{Name: "main"}}}}
	builder := CommandReportBuilder{
		Reports:      reports,
		Formatter:    stubFormatter{},
		AuthorSource: stubAuthorSource{email: "developer@example.com"},
	}

	got, err := builder.BuildToday()
	if err != nil {
		t.Fatalf("BuildToday() returned error: %v", err)
	}
	if got != "formatted main" || reports.todayEmail != "developer@example.com" {
		t.Fatalf("result = %q, email = %q", got, reports.todayEmail)
	}

	_, err = builder.BuildYesterday()
	if err != nil {
		t.Fatalf("BuildYesterday() returned error: %v", err)
	}
	if reports.yesterdayEmail != "developer@example.com" {
		t.Fatalf("yesterday email = %q", reports.yesterdayEmail)
	}
}

func TestCommandReportBuilderStatusDoesNotRequireAuthor(t *testing.T) {
	reports := &stubDomainReports{report: domain.Report{}}
	builder := CommandReportBuilder{
		Reports:      reports,
		Formatter:    stubFormatter{},
		AuthorSource: stubAuthorSource{err: errors.New("must not be called")},
	}

	got, err := builder.BuildStatus()
	if err != nil {
		t.Fatalf("BuildStatus() returned error: %v", err)
	}
	if got != "formatted empty" || reports.statusCalls != 1 {
		t.Fatalf("result = %q, status calls = %d", got, reports.statusCalls)
	}
}

func TestCommandReportBuilderPrefersConfiguredAuthor(t *testing.T) {
	reports := &stubDomainReports{}
	builder := CommandReportBuilder{
		Reports:      reports,
		Formatter:    stubFormatter{},
		Config:       config.Config{AuthorEmail: "configured@example.com"},
		AuthorSource: stubAuthorSource{err: errors.New("must not be called")},
	}

	_, err := builder.BuildToday()
	if err != nil {
		t.Fatalf("BuildToday() returned error: %v", err)
	}
	if reports.todayEmail != "configured@example.com" {
		t.Fatalf("today email = %q", reports.todayEmail)
	}
}

type stubDomainReports struct {
	report         domain.Report
	todayEmail     string
	yesterdayEmail string
	statusCalls    int
}

func (s *stubDomainReports) BuildToday(email string) (domain.Report, error) {
	s.todayEmail = email
	return s.report, nil
}

func (s *stubDomainReports) BuildYesterday(email string) (domain.Report, error) {
	s.yesterdayEmail = email
	return s.report, nil
}

func (s *stubDomainReports) BuildStatus() (domain.Report, error) {
	s.statusCalls++
	return s.report, nil
}

type stubFormatter struct{}

func (stubFormatter) Format(report domain.Report) string {
	if len(report.Branches) == 0 {
		return "formatted empty"
	}
	return "formatted " + report.Branches[0].Name
}

type stubAuthorSource struct {
	email string
	err   error
}

func (s stubAuthorSource) AuthorEmail() (string, error) {
	return s.email, s.err
}
