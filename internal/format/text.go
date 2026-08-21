package format

import (
	"fmt"
	"sort"
	"strings"

	"github.com/TimDelRey/daily-work-log/internal/domain"
)

const NoActivityMessage = "No activity found."

type Text struct{}

func (Text) Format(report domain.Report) string {
	if len(report.Branches) == 0 {
		return NoActivityMessage
	}

	branches := append([]domain.BranchActivity(nil), report.Branches...)
	sort.SliceStable(branches, func(i, j int) bool {
		return branches[i].Name < branches[j].Name
	})

	sections := make([]string, 0, len(branches))
	for _, branch := range branches {
		sections = append(sections, formatBranch(branch))
	}

	return strings.Join(sections, "\n\n")
}

func formatBranch(branch domain.BranchActivity) string {
	var sections []string

	if len(branch.Actions) > 0 {
		sections = append(sections, formatActions(branch.Actions))
	} else if len(branch.Commits) > 0 {
		commits := append([]domain.Commit(nil), branch.Commits...)
		sort.SliceStable(commits, func(i, j int) bool {
			if commits[i].AuthoredAt.Equal(commits[j].AuthoredAt) {
				return commits[i].Hash < commits[j].Hash
			}
			return commits[i].AuthoredAt.Before(commits[j].AuthoredAt)
		})

		lines := []string{"Commits:"}
		for _, commit := range commits {
			line := "  " + commit.AuthoredAt.Format("15:04") + " " + shortHash(commit.Hash)
			if commit.Message != "" {
				line += " " + commit.Message
			}
			lines = append(lines, line)
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}

	if len(branch.Actions) == 0 && len(branch.Stashes) > 0 {
		stashes := append([]domain.Stash(nil), branch.Stashes...)
		sort.SliceStable(stashes, func(i, j int) bool {
			if stashes[i].CreatedAt.Equal(stashes[j].CreatedAt) {
				return stashes[i].Reference < stashes[j].Reference
			}
			return stashes[i].CreatedAt.Before(stashes[j].CreatedAt)
		})

		lines := []string{"Stashes:"}
		for _, stash := range stashes {
			lines = append(lines, "  "+stash.CreatedAt.Format("15:04")+" "+stash.Reference)
			lines = append(lines, formatFiles(stash.Files)...)
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}

	if len(branch.CurrentlyUncommitted) > 0 {
		lines := []string{"Currently uncommitted:"}
		lines = append(lines, formatFiles(branch.CurrentlyUncommitted)...)
		sections = append(sections, strings.Join(lines, "\n"))
	}

	if branch.SyncState != nil {
		sections = append(sections, formatSyncState(branch.Upstream, *branch.SyncState))
	}

	if len(sections) == 0 {
		return bold(branch.Name)
	}
	return bold(branch.Name) + "\n" + strings.Join(sections, "\n\n")
}

func formatActions(actions []domain.Action) string {
	ordered := append([]domain.Action(nil), actions...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].OccurredAt.Equal(ordered[j].OccurredAt) {
			return ordered[i].Type < ordered[j].Type
		}
		return ordered[i].OccurredAt.Before(ordered[j].OccurredAt)
	})
	lines := make([]string, 0, len(ordered))
	for _, action := range ordered {
		line := fmt.Sprintf("  %s %-12s", action.OccurredAt.Format("15:04"), action.Type)
		if action.Summary != "" {
			line += " " + action.Summary
		}
		lines = append(lines, strings.TrimRight(line, " "))
		lines = append(lines, formatFiles(action.Files)...)
	}
	return strings.Join(lines, "\n")
}

func formatSyncState(upstream string, state domain.SyncState) string {
	switch {
	case state.Synchronized:
		return "synchronized with " + upstream
	case state.Ahead > 0 && state.Behind > 0:
		return fmt.Sprintf("ahead %d, behind %d of %s", state.Ahead, state.Behind, upstream)
	case state.Ahead > 0:
		return fmt.Sprintf("ahead %d of %s", state.Ahead, upstream)
	case state.Behind > 0:
		return fmt.Sprintf("behind %d of %s", state.Behind, upstream)
	default:
		return "upstream state unknown"
	}
}

func bold(value string) string {
	return fmt.Sprintf("\x1b[1m%s\x1b[0m", value)
}

func formatFiles(files []domain.File) []string {
	paths := make([]string, 0, len(files))
	seen := make(map[string]struct{}, len(files))
	for _, file := range files {
		if _, exists := seen[file.Path]; exists {
			continue
		}
		seen[file.Path] = struct{}{}
		paths = append(paths, file.Path)
	}
	sort.Strings(paths)

	for i := range paths {
		paths[i] = "    " + paths[i]
	}
	return paths
}

func shortHash(hash string) string {
	const length = 7
	if len(hash) <= length {
		return hash
	}
	return hash[:length]
}
