package domain

import "time"

type Report struct {
	DateRange DateRange
	Branches  []BranchActivity
}

type BranchActivity struct {
	Name                 string
	Upstream             string
	Actions              []Action
	SyncState            *SyncState
	Commits              []Commit
	Stashes              []Stash
	CurrentlyUncommitted []File
}

type ActionType string

const (
	ActionCommit     ActionType = "commit"
	ActionRebase     ActionType = "rebase"
	ActionMerge      ActionType = "merge"
	ActionCherryPick ActionType = "cherry_pick"
	ActionRevert     ActionType = "revert"
	ActionPush       ActionType = "push"
	ActionForcePush  ActionType = "force_push"
	ActionStash      ActionType = "stash"
)

type ActionSource string

const (
	SourceCommitLog ActionSource = "commit_log"
	SourceReflog    ActionSource = "reflog"
	SourceStashLog  ActionSource = "stash_log"
)

type Confidence string

const (
	ConfidenceExact        Confidence = "exact"
	ConfidenceInferred     Confidence = "inferred"
	ConfidenceCurrentState Confidence = "current_state"
)

type Action struct {
	Type       ActionType
	OccurredAt time.Time
	Summary    string
	CommitHash string
	Source     ActionSource
	Confidence Confidence
	Files      []File
}

type SyncState struct {
	Ahead        int
	Behind       int
	Synchronized bool
}

type Commit struct {
	Hash       string
	Message    string
	AuthoredAt time.Time
}

type Stash struct {
	Reference string
	Message   string
	CreatedAt time.Time
	Files     []File
}

func (s *Stash) AddFile(file File) {
	s.Files = addUniqueFile(s.Files, file)
}

type File struct {
	Path string
}

func (b *BranchActivity) AddCurrentlyUncommitted(file File) {
	b.CurrentlyUncommitted = addUniqueFile(b.CurrentlyUncommitted, file)
}

func addUniqueFile(files []File, file File) []File {
	for _, existing := range files {
		if existing.Path == file.Path {
			return files
		}
	}

	return append(files, file)
}
