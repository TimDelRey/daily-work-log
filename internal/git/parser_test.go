package git

import (
	"slices"
	"testing"
	"time"
)

func TestParseBranchRefs(t *testing.T) {
	output := []byte("feature/one\x00refs/heads/feature/one\x00origin/feature/one\x00\nmaster\x00refs/heads/master\x00\x00\n")

	refs, err := parseBranchRefs(output)
	if err != nil {
		t.Fatalf("parseBranchRefs() вернул ошибку: %v", err)
	}
	if len(refs) != 2 || refs[0].Upstream != "origin/feature/one" || refs[1].Name != "master" {
		t.Fatalf("parseBranchRefs() = %#v", refs)
	}
}

func TestParseReflogDerivesOldOIDFromPreviousEntry(t *testing.T) {
	output := []byte("new\x00origin/main@{2026-07-16T12:00:00+03:00}\x00origin/main@{2026-07-16T12:00:00+03:00}\x00update by push\x00\nold\x00origin/main@{2026-07-16T11:00:00+03:00}\x00origin/main@{2026-07-16T11:00:00+03:00}\x00fetch origin: fast-forward\x00\n")

	entries, err := parseReflog("refs/remotes/origin/main", output)
	if err != nil {
		t.Fatalf("parseReflog() вернул ошибку: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("parseReflog() вернул %d записей: %#v", len(entries), entries)
	}
	if entries[0].OldOID != "old" || entries[0].NewOID != "new" || entries[0].Subject != "update by push" {
		t.Fatalf("первая запись = %#v", entries[0])
	}
	if entries[0].OccurredAt.Format(time.RFC3339) != "2026-07-16T12:00:00+03:00" {
		t.Fatalf("OccurredAt = %s", entries[0].OccurredAt.Format(time.RFC3339))
	}
	if entries[1].OldOID != "" {
		t.Fatalf("OldOID самой старой доступной записи = %q", entries[1].OldOID)
	}
}

func TestParseAheadBehind(t *testing.T) {
	ahead, behind, err := parseAheadBehind([]byte("2\t3\n"))
	if err != nil {
		t.Fatalf("parseAheadBehind() вернул ошибку: %v", err)
	}
	if ahead != 2 || behind != 3 {
		t.Fatalf("parseAheadBehind() = (%d, %d), ожидалось (2, 3)", ahead, behind)
	}
}

func TestParseStatus(t *testing.T) {
	output := []byte(" M file with spaces.txt\x00R  renamed.txt\x00original.txt\x00?? line\nbreak.txt\x00")

	files, err := parseStatus(output)
	if err != nil {
		t.Fatalf("parseStatus() вернул ошибку: %v", err)
	}
	want := []string{"file with spaces.txt", "renamed.txt", "line\nbreak.txt"}
	if !slices.Equal(files, want) {
		t.Fatalf("parseStatus() = %#v, ожидалось %#v", files, want)
	}
}

func TestParseStashSubject(t *testing.T) {
	tests := []struct {
		subject     string
		wantBranch  string
		wantMessage string
	}{
		{subject: "WIP on feature/report: abc123 message", wantBranch: "feature/report", wantMessage: "abc123 message"},
		{subject: "On feature/report: custom message", wantBranch: "feature/report", wantMessage: "custom message"},
		{subject: "unknown metadata", wantMessage: "unknown metadata"},
	}

	for _, tt := range tests {
		t.Run(tt.subject, func(t *testing.T) {
			branch, message := parseStashSubject(tt.subject)
			if branch != tt.wantBranch || message != tt.wantMessage {
				t.Fatalf("parseStashSubject() = (%q, %q), ожидалось (%q, %q)",
					branch, message, tt.wantBranch, tt.wantMessage)
			}
		})
	}
}

func TestParseCommitsPreservesEmptyMessage(t *testing.T) {
	output := []byte("abc123\x002026-07-16T10:00:00Z\x00\x00\n")

	commits, err := parseCommits(output)
	if err != nil {
		t.Fatalf("parseCommits() вернул ошибку: %v", err)
	}
	if len(commits) != 1 || commits[0].Message != "" {
		t.Fatalf("parseCommits() = %#v", commits)
	}
}
