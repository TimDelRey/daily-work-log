package git

import (
	"slices"
	"testing"
)

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
