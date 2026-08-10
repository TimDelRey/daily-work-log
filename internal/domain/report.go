package domain

import "time"

type Report struct {
	DateRange DateRange
	Branches  []BranchActivity
}

type BranchActivity struct {
	Name                 string
	Commits              []Commit
	Stashes              []Stash
	CurrentlyUncommitted []File
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
