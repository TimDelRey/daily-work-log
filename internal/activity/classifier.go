package activity

import (
	"fmt"
	"strings"

	"github.com/TimDelRey/daily-work-log/internal/domain"
	gitadapter "github.com/TimDelRey/daily-work-log/internal/git"
)

type AncestorChecker interface {
	IsAncestor(olderOID, newerOID string) (bool, error)
}

type Classifier struct {
	Graph AncestorChecker
}

func (c Classifier) Reflog(entries []gitadapter.ReflogEntry) ([]domain.Action, error) {
	actions := make([]domain.Action, 0, len(entries))
	for _, entry := range entries {
		action, include, err := c.classifyReflogEntry(entry)
		if err != nil {
			return nil, err
		}
		if include {
			actions = append(actions, action)
		}
	}
	return actions, nil
}

func (c Classifier) classifyReflogEntry(entry gitadapter.ReflogEntry) (domain.Action, bool, error) {
	action := domain.Action{
		OccurredAt: entry.OccurredAt,
		CommitHash: entry.NewOID,
		Source:     domain.SourceReflog,
		Confidence: domain.ConfidenceExact,
	}

	switch {
	case entry.Subject == "update by push":
		action.Type = domain.ActionPush
		if entry.OldOID != "" && c.Graph != nil {
			ancestor, err := c.Graph.IsAncestor(entry.OldOID, entry.NewOID)
			if err != nil {
				return domain.Action{}, false, fmt.Errorf("classify push %s..%s: %w", entry.OldOID, entry.NewOID, err)
			}
			if !ancestor {
				action.Type = domain.ActionForcePush
			}
		}
		return action, true, nil
	case strings.HasPrefix(entry.Subject, "cherry-pick: "):
		action.Type = domain.ActionCherryPick
		action.Summary = strings.TrimPrefix(entry.Subject, "cherry-pick: ")
		return action, true, nil
	case strings.HasPrefix(entry.Subject, "merge "):
		action.Type = domain.ActionMerge
		action.Summary = entry.Subject
		return action, true, nil
	case strings.HasPrefix(entry.Subject, "rebase (finish):"):
		action.Type = domain.ActionRebase
		action.Summary = strings.TrimSpace(strings.TrimPrefix(entry.Subject, "rebase (finish):"))
		return action, true, nil
	case strings.HasPrefix(entry.Subject, "commit (initial): "):
		action.Type = domain.ActionCommit
		action.Summary = strings.TrimPrefix(entry.Subject, "commit (initial): ")
		return action, true, nil
	case strings.HasPrefix(entry.Subject, "commit: "):
		action.Summary = strings.TrimPrefix(entry.Subject, "commit: ")
		if strings.HasPrefix(action.Summary, "Revert ") {
			action.Type = domain.ActionRevert
		} else {
			action.Type = domain.ActionCommit
		}
		return action, true, nil
	default:
		return domain.Action{}, false, nil
	}
}
