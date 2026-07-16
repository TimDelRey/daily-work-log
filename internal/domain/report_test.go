package domain

import "testing"

func TestCommitAddFileDeduplicatesPaths(t *testing.T) {
	commit := Commit{}

	commit.AddFile(File{Path: "internal/domain/report.go"})
	commit.AddFile(File{Path: "internal/domain/report.go"})
	commit.AddFile(File{Path: "README.md"})

	assertFilePaths(t, commit.Files, []string{"internal/domain/report.go", "README.md"})
}

func TestStashAddFileDeduplicatesPaths(t *testing.T) {
	stash := Stash{}

	stash.AddFile(File{Path: "internal/domain/report.go"})
	stash.AddFile(File{Path: "internal/domain/report.go"})

	assertFilePaths(t, stash.Files, []string{"internal/domain/report.go"})
}

func TestBranchAddCurrentlyUncommittedDeduplicatesPaths(t *testing.T) {
	branch := BranchActivity{}

	branch.AddCurrentlyUncommitted(File{Path: "internal/domain/report.go"})
	branch.AddCurrentlyUncommitted(File{Path: "internal/domain/report.go"})
	branch.AddCurrentlyUncommitted(File{Path: "go.mod"})

	assertFilePaths(t, branch.CurrentlyUncommitted, []string{"internal/domain/report.go", "go.mod"})
}

func TestFilesAreDeduplicatedOnlyWithinTheirLogicalGroup(t *testing.T) {
	file := File{Path: "shared.go"}
	commit := Commit{}
	stash := Stash{}

	commit.AddFile(file)
	stash.AddFile(file)

	assertFilePaths(t, commit.Files, []string{"shared.go"})
	assertFilePaths(t, stash.Files, []string{"shared.go"})
}

func assertFilePaths(t *testing.T, files []File, want []string) {
	t.Helper()

	if len(files) != len(want) {
		t.Fatalf("expected %d files, got %d: %#v", len(want), len(files), files)
	}

	for i := range want {
		if files[i].Path != want[i] {
			t.Errorf("file %d: expected path %q, got %q", i, want[i], files[i].Path)
		}
	}
}
