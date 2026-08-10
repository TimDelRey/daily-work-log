package report

import (
	"fmt"
	"time"

	"github.com/TimDelRey/daily-work-log/internal/domain"
	filefilter "github.com/TimDelRey/daily-work-log/internal/filter"
	gitadapter "github.com/TimDelRey/daily-work-log/internal/git"
)

type GitClient interface {
	CurrentBranch() (string, error)
	Commits(authorEmail string, start, end time.Time) ([]gitadapter.Commit, error)
	CommitFiles(hash string) ([]string, error)
	Stashes() ([]gitadapter.Stash, error)
	StashFiles(reference string) ([]string, error)
	StatusFiles() ([]string, error)
}

type FileFilter interface {
	Include(path string) bool
}

type Builder struct {
	Git    GitClient
	Filter FileFilter
	Now    func() time.Time
}

func (b Builder) BuildToday(authorEmail string) (domain.Report, error) {
	return b.buildDay(authorEmail, domain.TodayRange(b.now()), true)
}

func (b Builder) BuildYesterday(authorEmail string) (domain.Report, error) {
	return b.buildDay(authorEmail, domain.YesterdayRange(b.now()), false)
}

func (b Builder) BuildStatus() (domain.Report, error) {
	branchName, err := b.Git.CurrentBranch()
	if err != nil {
		return domain.Report{}, fmt.Errorf("get current branch: %w", err)
	}

	paths, err := b.Git.StatusFiles()
	if err != nil {
		return domain.Report{}, fmt.Errorf("get current status: %w", err)
	}

	paths = b.includedPaths(paths)
	report := domain.Report{}
	if len(paths) == 0 {
		return report, nil
	}

	branch := domain.BranchActivity{Name: branchName}
	addCurrentFiles(&branch, paths)
	report.Branches = append(report.Branches, branch)

	return report, nil
}

func (b Builder) buildDay(authorEmail string, dateRange domain.DateRange, includeStatus bool) (domain.Report, error) {
	branchName, err := b.Git.CurrentBranch()
	if err != nil {
		return domain.Report{}, fmt.Errorf("get current branch: %w", err)
	}

	report := domain.Report{DateRange: dateRange}
	branches := make(map[string]*domain.BranchActivity)
	branchOrder := make([]string, 0)
	getBranch := func(name string) *domain.BranchActivity {
		if branch, ok := branches[name]; ok {
			return branch
		}

		branch := &domain.BranchActivity{Name: name}
		branches[name] = branch
		branchOrder = append(branchOrder, name)
		return branch
	}

	commits, err := b.Git.Commits(authorEmail, dateRange.Start, dateRange.End)
	if err != nil {
		return domain.Report{}, fmt.Errorf("get commits: %w", err)
	}
	for _, source := range commits {
		paths, err := b.Git.CommitFiles(source.Hash)
		if err != nil {
			return domain.Report{}, fmt.Errorf("get files for commit %s: %w", source.Hash, err)
		}

		commit := domain.Commit{
			Hash:       source.Hash,
			Message:    source.Message,
			AuthoredAt: source.AuthoredAt,
		}
		addCommitFiles(&commit, b.includedPaths(paths))
		branch := getBranch(branchName)
		branch.Commits = append(branch.Commits, commit)
	}

	stashes, err := b.Git.Stashes()
	if err != nil {
		return domain.Report{}, fmt.Errorf("get stashes: %w", err)
	}
	for _, source := range stashes {
		if !dateRange.Contains(source.CreatedAt) {
			continue
		}

		paths, err := b.Git.StashFiles(source.Reference)
		if err != nil {
			return domain.Report{}, fmt.Errorf("get files for stash %s: %w", source.Reference, err)
		}

		stash := domain.Stash{
			Reference: source.Reference,
			Message:   source.Message,
			CreatedAt: source.CreatedAt,
		}
		addStashFiles(&stash, b.includedPaths(paths))
		branch := getBranch(source.Branch)
		branch.Stashes = append(branch.Stashes, stash)
	}

	if includeStatus {
		paths, err := b.Git.StatusFiles()
		if err != nil {
			return domain.Report{}, fmt.Errorf("get current status: %w", err)
		}
		paths = b.includedPaths(paths)
		if len(paths) > 0 {
			addCurrentFiles(getBranch(branchName), paths)
		}
	}

	for _, name := range branchOrder {
		report.Branches = append(report.Branches, *branches[name])
	}

	return report, nil
}

func (b Builder) now() time.Time {
	if b.Now != nil {
		return b.Now()
	}
	return time.Now()
}

func (b Builder) includedPaths(paths []string) []string {
	matcher := b.Filter
	if matcher == nil {
		matcher = filefilter.Default()
	}

	included := make([]string, 0, len(paths))
	for _, path := range paths {
		if matcher.Include(path) {
			included = append(included, path)
		}
	}

	return included
}

func addCommitFiles(commit *domain.Commit, paths []string) {
	for _, path := range paths {
		commit.AddFile(domain.File{Path: path})
	}
}

func addStashFiles(stash *domain.Stash, paths []string) {
	for _, path := range paths {
		stash.AddFile(domain.File{Path: path})
	}
}

func addCurrentFiles(branch *domain.BranchActivity, paths []string) {
	for _, path := range paths {
		branch.AddCurrentlyUncommitted(domain.File{Path: path})
	}
}
