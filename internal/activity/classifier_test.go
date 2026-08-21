package activity

import (
	"errors"
	"testing"
	"time"

	"github.com/TimDelRey/daily-work-log/internal/domain"
	gitadapter "github.com/TimDelRey/daily-work-log/internal/git"
)

func TestClassifierReflog(t *testing.T) {
	now := time.Date(2026, time.July, 16, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		subject     string
		oldOID      string
		isAncestor  bool
		wantType    domain.ActionType
		wantSummary string
		wantCount   int
	}{
		{name: "commit", subject: "commit: add report", wantType: domain.ActionCommit, wantSummary: "add report", wantCount: 1},
		{name: "revert", subject: "commit: Revert \"bad change\"", wantType: domain.ActionRevert, wantSummary: "Revert \"bad change\"", wantCount: 1},
		{name: "cherry-pick", subject: "cherry-pick: selected change", wantType: domain.ActionCherryPick, wantSummary: "selected change", wantCount: 1},
		{name: "merge", subject: "merge feature/x: Merge made by the 'ort' strategy.", wantType: domain.ActionMerge, wantSummary: "merge feature/x: Merge made by the 'ort' strategy.", wantCount: 1},
		{name: "rebase finish", subject: "rebase (finish): returning to refs/heads/main", wantType: domain.ActionRebase, wantSummary: "returning to refs/heads/main", wantCount: 1},
		{name: "push", subject: "update by push", oldOID: "old", isAncestor: true, wantType: domain.ActionPush, wantCount: 1},
		{name: "force push", subject: "update by push", oldOID: "old", isAncestor: false, wantType: domain.ActionForcePush, wantCount: 1},
		{name: "fetch ignored", subject: "fetch origin: fast-forward", wantCount: 0},
		{name: "checkout ignored", subject: "checkout: moving from main to feature/x", wantCount: 0},
		{name: "rebase internal step ignored", subject: "rebase (pick): add report", wantCount: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := fakeGraph{ancestor: tt.isAncestor}
			actions, err := (Classifier{Graph: graph}).Reflog([]gitadapter.ReflogEntry{{
				OldOID: tt.oldOID, NewOID: "new", OccurredAt: now, Subject: tt.subject,
			}})
			if err != nil {
				t.Fatalf("Reflog() вернул ошибку: %v", err)
			}
			if len(actions) != tt.wantCount {
				t.Fatalf("Reflog() = %#v, ожидалось %d действий", actions, tt.wantCount)
			}
			if tt.wantCount == 1 && (actions[0].Type != tt.wantType || actions[0].Summary != tt.wantSummary) {
				t.Fatalf("действие = %#v", actions[0])
			}
		})
	}
}

func TestClassifierReturnsAncestorError(t *testing.T) {
	_, err := (Classifier{Graph: fakeGraph{err: errors.New("broken graph")}}).Reflog([]gitadapter.ReflogEntry{{
		OldOID: "old", NewOID: "new", Subject: "update by push",
	}})
	if err == nil {
		t.Fatal("Reflog() не вернул ошибку")
	}
}

type fakeGraph struct {
	ancestor bool
	err      error
}

func (g fakeGraph) IsAncestor(_, _ string) (bool, error) {
	return g.ancestor, g.err
}
