package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestClientCurrentBranch(t *testing.T) {
	repository := initRepository(t)
	runGit(t, repository, "checkout", "-q", "-b", "feature/git-adapter")

	branch, err := (Client{Directory: repository}).CurrentBranch()
	if err != nil {
		t.Fatalf("CurrentBranch() вернул ошибку: %v", err)
	}
	if branch != "feature/git-adapter" {
		t.Fatalf("CurrentBranch() = %q, ожидалось %q", branch, "feature/git-adapter")
	}
}

func TestClientBranchesReturnsLocalBranches(t *testing.T) {
	repository := initRepository(t)
	writeFile(t, repository, "initial.txt", "initial")
	commit(t, repository, "Worklog User", "developer@example.com", "2026-07-16T08:00:00Z", "initial")
	runGit(t, repository, "branch", "feature/two")
	runGit(t, repository, "branch", "feature/one")

	branches, err := (Client{Directory: repository}).Branches()
	if err != nil {
		t.Fatalf("Branches() вернул ошибку: %v", err)
	}
	assertPaths(t, branches, []string{"master", "feature/one", "feature/two"})
}

func TestClientCommitsFiltersAuthorAndDateRange(t *testing.T) {
	repository := initRepository(t)
	writeFile(t, repository, "before.txt", "before")
	commit(t, repository, "Worklog User", "developer+worklog@example.com", "2026-07-15T23:59:59+03:00", "before range")
	writeFile(t, repository, "other.txt", "other")
	commit(t, repository, "Other Developer", "other@example.com", "2026-07-16T09:00:00+03:00", "other commit")
	writeFile(t, repository, "own.txt", "own")
	commit(t, repository, "Worklog User", "developer+worklog@example.com", "2026-07-16T10:00:00+03:00", "own commit")
	writeFile(t, repository, "at-end.txt", "at end")
	commit(t, repository, "Worklog User", "developer+worklog@example.com", "2026-07-17T00:00:00+03:00", "at range end")

	location := time.FixedZone("test", 3*60*60)
	start := time.Date(2026, time.July, 16, 0, 0, 0, 0, location)
	end := start.AddDate(0, 0, 1)

	commits, err := (Client{Directory: repository}).Commits("master", "developer+worklog@example.com", start, end)
	if err != nil {
		t.Fatalf("Commits() вернул ошибку: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("Commits() вернул %d коммитов, ожидался 1: %#v", len(commits), commits)
	}
	if commits[0].Message != "own commit" {
		t.Errorf("Message = %q, ожидалось %q", commits[0].Message, "own commit")
	}
	if commits[0].AuthoredAt.Format(time.RFC3339) != "2026-07-16T10:00:00+03:00" {
		t.Errorf("AuthoredAt = %s", commits[0].AuthoredAt.Format(time.RFC3339))
	}
}

func TestClientStashesAndFiles(t *testing.T) {
	repository := initRepository(t)
	runGit(t, repository, "checkout", "-q", "-b", "feature/stash")
	writeFile(t, repository, "tracked.txt", "initial")
	commit(t, repository, "Worklog User", "developer@example.com", "2026-07-16T10:00:00Z", "initial")
	writeFile(t, repository, "tracked.txt", "changed")
	writeFile(t, repository, "untracked file.txt", "new")
	runGitWithEnv(t, repository, []string{"GIT_COMMITTER_DATE=2026-07-16T12:30:00+03:00"},
		"stash", "push", "-q", "--include-untracked", "-m", "saved work")

	client := Client{Directory: repository}
	stashes, err := client.Stashes()
	if err != nil {
		t.Fatalf("Stashes() вернул ошибку: %v", err)
	}
	if len(stashes) != 1 {
		t.Fatalf("Stashes() вернул %d записей, ожидалась 1: %#v", len(stashes), stashes)
	}
	stash := stashes[0]
	if stash.Reference != "stash@{0}" || stash.Branch != "feature/stash" || stash.Message != "saved work" {
		t.Errorf("неожиданная metadata stash: %#v", stash)
	}
	if stash.CreatedAt.Format(time.RFC3339) != "2026-07-16T12:30:00+03:00" {
		t.Errorf("CreatedAt = %s", stash.CreatedAt.Format(time.RFC3339))
	}

	files, err := client.StashFiles(stash.Reference)
	if err != nil {
		t.Fatalf("StashFiles() вернул ошибку: %v", err)
	}
	assertPaths(t, files, []string{"tracked.txt", "untracked file.txt"})
}

func TestClientStatusFilesIncludesCurrentPathForRename(t *testing.T) {
	repository := initRepository(t)
	writeFile(t, repository, "old.txt", "old")
	writeFile(t, repository, "modified.txt", "initial")
	commit(t, repository, "Worklog User", "developer@example.com", "2026-07-16T10:00:00Z", "initial")
	runGit(t, repository, "mv", "old.txt", "new name.txt")
	writeFile(t, repository, "modified.txt", "changed")
	writeFile(t, repository, "untracked.txt", "new")

	files, err := (Client{Directory: repository}).StatusFiles()
	if err != nil {
		t.Fatalf("StatusFiles() вернул ошибку: %v", err)
	}
	assertPaths(t, files, []string{"modified.txt", "new name.txt", "untracked.txt"})
}

func initRepository(t *testing.T) string {
	t.Helper()

	repository := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	runGit(t, repository, "init", "-q")
	runGit(t, repository, "config", "user.name", "Worklog User")
	runGit(t, repository, "config", "user.email", "developer@example.com")

	return repository
}

func writeFile(t *testing.T, repository, path, contents string) {
	t.Helper()

	fullPath := filepath.Join(repository, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("создать директорию для %q: %v", path, err)
	}
	if err := os.WriteFile(fullPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("записать %q: %v", path, err)
	}
}

func commit(t *testing.T, repository, name, email, date, message string) {
	t.Helper()

	runGit(t, repository, "add", "--all")
	runGitWithEnv(t, repository, []string{
		"GIT_AUTHOR_NAME=" + name,
		"GIT_AUTHOR_EMAIL=" + email,
		"GIT_AUTHOR_DATE=" + date,
		"GIT_COMMITTER_NAME=" + name,
		"GIT_COMMITTER_EMAIL=" + email,
		"GIT_COMMITTER_DATE=" + date,
	}, "commit", "-q", "-m", message)
}

func runGit(t *testing.T, repository string, args ...string) string {
	t.Helper()
	return runGitWithEnv(t, repository, nil, args...)
}

func runGitWithEnv(t *testing.T, repository string, environment []string, args ...string) string {
	t.Helper()

	command := exec.Command("git", args...)
	command.Dir = repository
	command.Env = append(os.Environ(), environment...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v завершился с ошибкой: %v: %s", args, err, output)
	}

	return string(output)
}

func assertPaths(t *testing.T, got, want []string) {
	t.Helper()

	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("пути = %#v, ожидались %#v", got, want)
	}
}
