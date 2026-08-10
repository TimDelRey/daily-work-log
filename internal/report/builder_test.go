package report

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/TimDelRey/daily-work-log/internal/domain"
	gitadapter "github.com/TimDelRey/daily-work-log/internal/git"
)

func TestBuildTodayCombinesAllSources(t *testing.T) {
	now := time.Date(2026, time.July, 16, 15, 0, 0, 0, time.FixedZone("test", 3*60*60))
	git := &stubGit{
		branch:      "feature/current",
		commits:     []gitadapter.Commit{{Hash: "abc123", Message: "own work", AuthoredAt: now.Add(-time.Hour)}},
		commitFiles: map[string][]string{"abc123": {"shared.go", "commit.go", "shared.go"}},
		stashes: []gitadapter.Stash{
			{Reference: "stash@{0}", Branch: "feature/stashed", Message: "saved", CreatedAt: now.Add(-2 * time.Hour)},
			{Reference: "stash@{1}", Branch: "feature/old", Message: "old", CreatedAt: now.AddDate(0, 0, -1)},
		},
		stashFiles:  map[string][]string{"stash@{0}": {"stash.go"}},
		statusFiles: []string{"working.go", "working.go"},
	}

	report, err := (Builder{Git: git, Now: func() time.Time { return now }}).BuildToday("me@example.com")
	if err != nil {
		t.Fatalf("BuildToday() returned error: %v", err)
	}

	assertRange(t, report.DateRange, "2026-07-16T00:00:00+03:00", "2026-07-17T00:00:00+03:00")
	if git.authorEmail != "me@example.com" {
		t.Fatalf("author email = %q", git.authorEmail)
	}
	if len(report.Branches) != 2 {
		t.Fatalf("branches = %#v", report.Branches)
	}
	current := report.Branches[0]
	if current.Name != "feature/current" || len(current.Commits) != 1 {
		t.Fatalf("current branch = %#v", current)
	}
	assertFiles(t, current.Commits[0].Files, []string{"shared.go", "commit.go"})
	assertFiles(t, current.CurrentlyUncommitted, []string{"working.go"})
	stashed := report.Branches[1]
	if stashed.Name != "feature/stashed" || len(stashed.Stashes) != 1 {
		t.Fatalf("stash branch = %#v", stashed)
	}
	assertFiles(t, stashed.Stashes[0].Files, []string{"stash.go"})
	if !slices.Equal(git.requestedStashes, []string{"stash@{0}"}) {
		t.Fatalf("requested stash files = %#v", git.requestedStashes)
	}
}

func TestBuildYesterdayExcludesStatusAndOtherDayStashes(t *testing.T) {
	now := time.Date(2026, time.July, 16, 9, 0, 0, 0, time.UTC)
	git := &stubGit{
		branch: "feature/report",
		stashes: []gitadapter.Stash{
			{Reference: "stash@{0}", Branch: "feature/report", CreatedAt: time.Date(2026, time.July, 15, 23, 59, 59, 0, time.UTC)},
			{Reference: "stash@{1}", Branch: "feature/report", CreatedAt: time.Date(2026, time.July, 16, 0, 0, 0, 0, time.UTC)},
		},
		stashFiles:  map[string][]string{"stash@{0}": {"yesterday.go"}},
		statusFiles: []string{"must-not-appear.go"},
	}

	report, err := (Builder{Git: git, Now: func() time.Time { return now }}).BuildYesterday("me@example.com")
	if err != nil {
		t.Fatalf("BuildYesterday() returned error: %v", err)
	}

	assertRange(t, report.DateRange, "2026-07-15T00:00:00Z", "2026-07-16T00:00:00Z")
	if git.statusCalls != 0 {
		t.Fatalf("StatusFiles() called %d times", git.statusCalls)
	}
	if len(report.Branches) != 1 || len(report.Branches[0].Stashes) != 1 {
		t.Fatalf("report = %#v", report)
	}
}

func TestBuildStatusReturnsOnlyCurrentChanges(t *testing.T) {
	git := &stubGit{branch: "feature/status", statusFiles: []string{"one.go", "two.go"}}

	report, err := (Builder{Git: git}).BuildStatus()
	if err != nil {
		t.Fatalf("BuildStatus() returned error: %v", err)
	}

	if len(report.Branches) != 1 || report.Branches[0].Name != "feature/status" {
		t.Fatalf("report = %#v", report)
	}
	assertFiles(t, report.Branches[0].CurrentlyUncommitted, []string{"one.go", "two.go"})
	if git.commitCalls != 0 || git.stashCalls != 0 {
		t.Fatalf("historical data was requested: commits=%d stashes=%d", git.commitCalls, git.stashCalls)
	}
}

func TestBuildDayOmitsBranchesWithoutActivity(t *testing.T) {
	git := &stubGit{branch: "feature/empty"}

	report, err := (Builder{Git: git, Now: func() time.Time { return time.Now() }}).BuildToday("me@example.com")
	if err != nil {
		t.Fatalf("BuildToday() returned error: %v", err)
	}
	if len(report.Branches) != 0 {
		t.Fatalf("branches = %#v", report.Branches)
	}
}

func TestBuildDayWrapsSourceErrors(t *testing.T) {
	sourceErr := errors.New("git failed")
	git := &stubGit{branch: "main", commitsErr: sourceErr}

	_, err := (Builder{Git: git, Now: time.Now}).BuildToday("me@example.com")
	if !errors.Is(err, sourceErr) {
		t.Fatalf("BuildToday() error = %v", err)
	}
}

type stubGit struct {
	branch           string
	commits          []gitadapter.Commit
	commitFiles      map[string][]string
	stashes          []gitadapter.Stash
	stashFiles       map[string][]string
	statusFiles      []string
	commitsErr       error
	authorEmail      string
	requestedStashes []string
	statusCalls      int
	commitCalls      int
	stashCalls       int
}

func (g *stubGit) CurrentBranch() (string, error) { return g.branch, nil }
func (g *stubGit) Commits(authorEmail string, _, _ time.Time) ([]gitadapter.Commit, error) {
	g.commitCalls++
	g.authorEmail = authorEmail
	return g.commits, g.commitsErr
}
func (g *stubGit) CommitFiles(hash string) ([]string, error) { return g.commitFiles[hash], nil }
func (g *stubGit) Stashes() ([]gitadapter.Stash, error) {
	g.stashCalls++
	return g.stashes, nil
}
func (g *stubGit) StashFiles(reference string) ([]string, error) {
	g.requestedStashes = append(g.requestedStashes, reference)
	return g.stashFiles[reference], nil
}
func (g *stubGit) StatusFiles() ([]string, error) {
	g.statusCalls++
	return g.statusFiles, nil
}

func assertRange(t *testing.T, got domain.DateRange, start, end string) {
	t.Helper()
	if got.Start.Format(time.RFC3339) != start || got.End.Format(time.RFC3339) != end {
		t.Fatalf("range = [%s, %s), want [%s, %s)", got.Start.Format(time.RFC3339), got.End.Format(time.RFC3339), start, end)
	}
}

func assertFiles(t *testing.T, got []domain.File, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("files = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i].Path != want[i] {
			t.Fatalf("file %d = %q, want %q", i, got[i].Path, want[i])
		}
	}
}
