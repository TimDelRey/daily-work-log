package domain

import "time"

// Report содержит активность, сгруппированную по веткам Git.
type Report struct {
	DateRange DateRange
	Branches  []BranchActivity
}

// BranchActivity объединяет все поддерживаемые источники активности для одной ветки.
type BranchActivity struct {
	Name                 string
	Commits              []Commit
	Stashes              []Stash
	CurrentlyUncommitted []File
}

// Commit описывает коммит Git и затронутые им файлы.
type Commit struct {
	Hash       string
	Message    string
	AuthoredAt time.Time
	Files      []File
}

// AddFile добавляет путь к файлу, если его ещё нет в этом коммите.
func (c *Commit) AddFile(file File) {
	c.Files = addUniqueFile(c.Files, file)
}

// Stash описывает stash Git и сохранённые в нём файлы.
type Stash struct {
	Reference string
	Message   string
	CreatedAt time.Time
	Files     []File
}

// AddFile добавляет путь к файлу, если его ещё нет в этом stash.
func (s *Stash) AddFile(file File) {
	s.Files = addUniqueFile(s.Files, file)
}

// File определяет файл репозитория по его пути.
type File struct {
	Path string
}

// AddCurrentlyUncommitted добавляет путь, если его ещё нет среди текущих изменений.
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
