package report

import (
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/TimDelRey/daily-work-log/internal/domain"
	gitadapter "github.com/TimDelRey/daily-work-log/internal/git"
)

func TestBuildTodayCombinesAllSources(t *testing.T) {
	now := time.Date(2026, time.July, 16, 15, 0, 0, 0, time.FixedZone("test", 3*60*60))
	git := &stubGit{
		branch:   "feature/current",
		branches: []string{"feature/current"},
		commits: map[string][]gitadapter.Commit{
			"feature/current": {{Hash: "abc123", Message: "own work", AuthoredAt: now.Add(-time.Hour)}},
		},
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
		branch:   "feature/report",
		branches: []string{"feature/report"},
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
	git := &stubGit{branch: "feature/empty", branches: []string{"feature/empty"}}

	report, err := (Builder{Git: git, Now: func() time.Time { return time.Now() }}).BuildToday("me@example.com")
	if err != nil {
		t.Fatalf("BuildToday() returned error: %v", err)
	}
	if len(report.Branches) != 0 {
		t.Fatalf("branches = %#v", report.Branches)
	}
}

func TestBuildDayGroupsCommitsAcrossBranchesAndDeduplicatesMergedCommit(t *testing.T) {
	now := time.Date(2026, time.July, 16, 12, 0, 0, 0, time.UTC)
	shared := gitadapter.Commit{Hash: "shared", Message: "feature work", AuthoredAt: now}
	git := &stubGit{
		branch:   "master",
		branches: []string{"master", "feature/two", "feature/one"},
		commits: map[string][]gitadapter.Commit{
			"master":      {shared, {Hash: "master-only", Message: "release", AuthoredAt: now}},
			"feature/one": {shared},
			"feature/two": {{Hash: "second", Message: "other feature", AuthoredAt: now}},
		},
	}

	report, err := (Builder{Git: git, Now: func() time.Time { return now }}).BuildYesterday("me@example.com")
	if err != nil {
		t.Fatalf("BuildYesterday() returned error: %v", err)
	}

	if len(report.Branches) != 2 {
		t.Fatalf("branches = %#v", report.Branches)
	}
	if report.Branches[0].Name != "master" || len(report.Branches[0].Commits) != 2 {
		t.Fatalf("master = %#v", report.Branches[0])
	}
	if report.Branches[0].Commits[0].Hash != "shared" || report.Branches[0].Commits[1].Hash != "master-only" {
		t.Fatalf("master commits = %#v", report.Branches[0].Commits)
	}
	if report.Branches[1].Name != "feature/two" || report.Branches[1].Commits[0].Hash != "second" {
		t.Fatalf("feature/two = %#v", report.Branches[1])
	}
}

func TestBuildYesterdayDoesNotDependOnCurrentBranch(t *testing.T) {
	now := time.Date(2026, time.July, 16, 12, 0, 0, 0, time.UTC)
	git := &stubGit{
		branch:   "master",
		branches: []string{"master", "feature/work"},
		commits: map[string][]gitadapter.Commit{
			"feature/work": {{Hash: "feature", Message: "feature work", AuthoredAt: now}},
		},
	}
	builder := Builder{Git: git, Now: func() time.Time { return now }}

	fromMaster, err := builder.BuildYesterday("me@example.com")
	if err != nil {
		t.Fatalf("BuildYesterday() from master returned error: %v", err)
	}
	git.branch = "feature/work"
	fromFeature, err := builder.BuildYesterday("me@example.com")
	if err != nil {
		t.Fatalf("BuildYesterday() from feature returned error: %v", err)
	}

	if !reflect.DeepEqual(fromMaster, fromFeature) {
		t.Fatalf("reports differ by current branch:\nmaster: %#v\nfeature: %#v", fromMaster, fromFeature)
	}
}

func TestBuilderFiltersStashAndCurrentFiles(t *testing.T) {
	now := time.Date(2026, time.July, 16, 12, 0, 0, 0, time.UTC)
	git := &stubGit{
		branch:   "feature/filter",
		branches: []string{"feature/filter"},
		commits: map[string][]gitadapter.Commit{
			"feature/filter": {{Hash: "abc123", AuthoredAt: now}},
		},
		stashes: []gitadapter.Stash{
			{Reference: "stash@{0}", Branch: "feature/filter", CreatedAt: now},
		},
		stashFiles: map[string][]string{
			"stash@{0}": {"stash.go", "sorbet/rbi/generated.rb"},
		},
		statusFiles: []string{"working.go", "working.rbi"},
	}

	report, err := (Builder{Git: git, Now: func() time.Time { return now }}).BuildToday("me@example.com")
	if err != nil {
		t.Fatalf("BuildToday() returned error: %v", err)
	}

	branch := report.Branches[0]
	assertFiles(t, branch.Stashes[0].Files, []string{"stash.go"})
	assertFiles(t, branch.CurrentlyUncommitted, []string{"working.go"})
}

func TestBuildStatusOmitsBranchWhenAllFilesAreIgnored(t *testing.T) {
	git := &stubGit{branch: "feature/generated", statusFiles: []string{"types.rbi", "sorbet/rbi/cache.rb"}}

	report, err := (Builder{Git: git}).BuildStatus()
	if err != nil {
		t.Fatalf("BuildStatus() returned error: %v", err)
	}
	if len(report.Branches) != 0 {
		t.Fatalf("branches = %#v", report.Branches)
	}
}

func TestBuildDayWrapsSourceErrors(t *testing.T) {
	sourceErr := errors.New("git failed")
	git := &stubGit{branch: "main", branches: []string{"main"}, commitsErr: sourceErr}

	_, err := (Builder{Git: git, Now: time.Now}).BuildToday("me@example.com")
	if !errors.Is(err, sourceErr) {
		t.Fatalf("BuildToday() error = %v", err)
	}
}

type stubGit struct {
	branch           string
	branches         []string
	commits          map[string][]gitadapter.Commit
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
func (g *stubGit) Branches() ([]string, error)    { return g.branches, nil }
func (g *stubGit) Commits(branch, authorEmail string, _, _ time.Time) ([]gitadapter.Commit, error) {
	g.commitCalls++
	g.authorEmail = authorEmail
	return g.commits[branch], g.commitsErr
}
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
