package format

import (
	"testing"
	"time"

	"github.com/TimDelRey/daily-work-log/internal/domain"
)

func TestTextFormat(t *testing.T) {
	report := domain.Report{Branches: []domain.BranchActivity{
		{
			Name:                 "feature/z-last",
			CurrentlyUncommitted: []domain.File{{Path: "z.go"}},
		},
		{
			Name: "feature/report",
			Commits: []domain.Commit{
				{Hash: "bbbbbbb222", Message: "second", AuthoredAt: at("10:42")},
				{Hash: "aaaaaaa111", Message: "first", AuthoredAt: at("09:15")},
			},
			Stashes: []domain.Stash{
				{Reference: "stash@{0}", CreatedAt: at("14:20"), Files: []domain.File{{Path: "stash.go"}}},
			},
			CurrentlyUncommitted: []domain.File{{Path: "working.go"}},
		},
	}}

	want := "\x1b[1mfeature/report\x1b[0m" + `
Commits:
  09:15 aaaaaaa first
  10:42 bbbbbbb second

Stashes:
  14:20 stash@{0}
    stash.go

Currently uncommitted:
    working.go

` + "\x1b[1mfeature/z-last\x1b[0m" + `
Currently uncommitted:
    z.go`

	if got := (Text{}).Format(report); got != want {
		t.Fatalf("Format() =\n%s\n\nwant:\n%s", got, want)
	}
}

func TestTextFormatEmptyReport(t *testing.T) {
	if got := (Text{}).Format(domain.Report{}); got != NoActivityMessage {
		t.Fatalf("Format() = %q", got)
	}
}

func TestTextFormatDoesNotMutateReport(t *testing.T) {
	report := domain.Report{Branches: []domain.BranchActivity{
		{Name: "z"},
		{Name: "a"},
	}}

	_ = (Text{}).Format(report)
	if report.Branches[0].Name != "z" {
		t.Fatalf("Format() changed branch order: %#v", report.Branches)
	}
}

func at(clock string) time.Time {
	value, err := time.Parse("15:04", clock)
	if err != nil {
		panic(err)
	}
	return value
}
