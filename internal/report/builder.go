package report

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/TimDelRey/daily-work-log/internal/activity"
	"github.com/TimDelRey/daily-work-log/internal/domain"
	filefilter "github.com/TimDelRey/daily-work-log/internal/filter"
	gitadapter "github.com/TimDelRey/daily-work-log/internal/git"
)

type GitClient interface {
	CurrentBranch() (string, error)
	Branches() ([]string, error)
	Commits(branch, authorEmail string, start, end time.Time) ([]gitadapter.Commit, error)
	Stashes() ([]gitadapter.Stash, error)
	StashFiles(reference string) ([]string, error)
	StatusFiles() ([]string, error)
}

type FileFilter interface {
	Include(path string) bool
}

type ActivityGitClient interface {
	BranchRefs() ([]gitadapter.BranchRef, error)
	RemoteRefs() ([]gitadapter.BranchRef, error)
	Reflog(reference string) ([]gitadapter.ReflogEntry, error)
	UpstreamState(branch, upstream string) (gitadapter.UpstreamState, error)
	IsAncestor(olderOID, newerOID string) (bool, error)
}

type Builder struct {
	Git    GitClient
	Filter FileFilter
	Now    func() time.Time
}

const detachedBranchName = "detached/unknown"

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
	branchName = normalizedBranchName(branchName)

	paths, err := b.Git.StatusFiles()
	if err != nil {
		return domain.Report{}, fmt.Errorf("get current status: %w", err)
	}

	paths = b.includedPaths(paths)
	report := domain.Report{}
	branch := domain.BranchActivity{Name: branchName}
	addCurrentFiles(&branch, paths)
	if activityGit, ok := b.Git.(ActivityGitClient); ok {
		if err := addCurrentUpstream(activityGit, &branch); err != nil {
			return domain.Report{}, err
		}
	}
	report.Branches = append(report.Branches, branch)

	return report, nil
}

func (b Builder) buildDay(authorEmail string, dateRange domain.DateRange, includeStatus bool) (domain.Report, error) {
	branchName, err := b.Git.CurrentBranch()
	if err != nil {
		return domain.Report{}, fmt.Errorf("get current branch: %w", err)
	}
	branchName = normalizedBranchName(branchName)

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

	gitBranches, err := b.Git.Branches()
	if err != nil {
		return domain.Report{}, fmt.Errorf("get branches: %w", err)
	}
	sortBranchesForCommitOwnership(gitBranches)
	seenCommits := make(map[string]struct{})
	for _, gitBranch := range gitBranches {
		commits, err := b.Git.Commits(gitBranch, authorEmail, dateRange.Start, dateRange.End)
		if err != nil {
			return domain.Report{}, fmt.Errorf("get commits for branch %s: %w", gitBranch, err)
		}
		for _, source := range commits {
			if _, seen := seenCommits[source.Hash]; seen {
				continue
			}
			seenCommits[source.Hash] = struct{}{}
			commit := domain.Commit{
				Hash:       source.Hash,
				Message:    source.Message,
				AuthoredAt: source.AuthoredAt,
			}
			branch := getBranch(gitBranch)
			branch.Commits = append(branch.Commits, commit)
			branch.Actions = append(branch.Actions, commitAction(commit))
		}
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
		branch.Actions = append(branch.Actions, domain.Action{
			Type: domain.ActionStash, OccurredAt: stash.CreatedAt, Summary: stash.Message,
			Source: domain.SourceStashLog, Confidence: domain.ConfidenceExact, Files: stash.Files,
		})
	}

	if activityGit, ok := b.Git.(ActivityGitClient); ok {
		if err := b.addReflogActivity(activityGit, dateRange, includeStatus, getBranch); err != nil {
			return domain.Report{}, err
		}
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
		branch := branches[name]
		if len(branch.Actions) == 0 && len(branch.Commits) == 0 && len(branch.Stashes) == 0 && len(branch.CurrentlyUncommitted) == 0 {
			continue
		}
		report.Branches = append(report.Branches, *branch)
	}

	return report, nil
}

func (b Builder) addReflogActivity(git ActivityGitClient, dateRange domain.DateRange, includeState bool, getBranch func(string) *domain.BranchActivity) error {
	refs, err := git.BranchRefs()
	if err != nil {
		return fmt.Errorf("get branch refs: %w", err)
	}
	classifier := activity.Classifier{Graph: git}
	upstreamBranches := make(map[string]string)
	for _, ref := range refs {
		branch := getBranch(ref.Name)
		branch.Upstream = ref.Upstream
		if ref.Upstream != "" {
			upstreamBranches[ref.Upstream] = ref.Name
		}
		entries, err := git.Reflog(ref.FullName)
		if err != nil {
			return fmt.Errorf("get reflog for %s: %w", ref.Name, err)
		}
		actions, err := classifier.Reflog(entriesInRange(entries, dateRange))
		if err != nil {
			return err
		}
		for _, action := range actions {
			if action.Type == domain.ActionCommit || action.Type == domain.ActionRevert || action.Type == domain.ActionPush || action.Type == domain.ActionForcePush {
				continue
			}
			branch.Actions = append(branch.Actions, action)
			removePlainCommitAction(branch, action.CommitHash)
		}
		if includeState && ref.Upstream != "" {
			state, err := git.UpstreamState(ref.Name, ref.Upstream)
			if err != nil {
				return fmt.Errorf("get upstream state for %s: %w", ref.Name, err)
			}
			branch.SyncState = syncState(state)
		}
	}

	remoteRefs, err := git.RemoteRefs()
	if err != nil {
		return fmt.Errorf("get remote refs: %w", err)
	}
	for _, ref := range remoteRefs {
		branchName, ok := upstreamBranches[ref.Name]
		if !ok {
			continue
		}
		entries, err := git.Reflog(ref.FullName)
		if err != nil {
			return fmt.Errorf("get reflog for %s: %w", ref.Name, err)
		}
		actions, err := classifier.Reflog(entriesInRange(entries, dateRange))
		if err != nil {
			return err
		}
		for _, action := range actions {
			if action.Type == domain.ActionPush || action.Type == domain.ActionForcePush {
				getBranch(branchName).Actions = append(getBranch(branchName).Actions, action)
			}
		}
	}
	return nil
}

func addCurrentUpstream(git ActivityGitClient, branch *domain.BranchActivity) error {
	refs, err := git.BranchRefs()
	if err != nil {
		return fmt.Errorf("get branch refs: %w", err)
	}
	for _, ref := range refs {
		if ref.Name != branch.Name || ref.Upstream == "" {
			continue
		}
		state, err := git.UpstreamState(ref.Name, ref.Upstream)
		if err != nil {
			return fmt.Errorf("get upstream state for %s: %w", ref.Name, err)
		}
		branch.Upstream = ref.Upstream
		branch.SyncState = syncState(state)
	}
	return nil
}

func syncState(state gitadapter.UpstreamState) *domain.SyncState {
	if !state.Known {
		return &domain.SyncState{}
	}
	return &domain.SyncState{Ahead: state.Ahead, Behind: state.Behind, Synchronized: state.Ahead == 0 && state.Behind == 0}
}

func entriesInRange(entries []gitadapter.ReflogEntry, dateRange domain.DateRange) []gitadapter.ReflogEntry {
	result := make([]gitadapter.ReflogEntry, 0, len(entries))
	for _, entry := range entries {
		if dateRange.Contains(entry.OccurredAt) {
			result = append(result, entry)
		}
	}
	return result
}

func commitAction(commit domain.Commit) domain.Action {
	actionType := domain.ActionCommit
	if strings.HasPrefix(commit.Message, "Revert ") {
		actionType = domain.ActionRevert
	}
	return domain.Action{Type: actionType, OccurredAt: commit.AuthoredAt, Summary: commit.Message, CommitHash: commit.Hash, Source: domain.SourceCommitLog, Confidence: domain.ConfidenceExact}
}

func removePlainCommitAction(branch *domain.BranchActivity, hash string) {
	branch.Actions = slices.DeleteFunc(branch.Actions, func(action domain.Action) bool {
		return action.CommitHash == hash && action.Type == domain.ActionCommit
	})
}

func sortBranchesForCommitOwnership(branches []string) {
	slices.SortFunc(branches, func(a, b string) int {
		aBase := isBaseBranch(a)
		bBase := isBaseBranch(b)
		if aBase != bBase {
			if aBase {
				return -1
			}
			return 1
		}
		return strings.Compare(a, b)
	})
}

func isBaseBranch(branch string) bool {
	switch branch {
	case "main", "master", "develop", "development":
		return true
	default:
		return false
	}
}

func normalizedBranchName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return detachedBranchName
	}
	return name
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
