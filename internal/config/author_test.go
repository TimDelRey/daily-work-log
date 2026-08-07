package config

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestResolveAuthorEmail(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		source    stubAuthorEmailSource
		wantEmail string
		wantError error
		wantCalls int
	}{
		{
			name:      "явная настройка имеет приоритет",
			config:    Config{AuthorEmail: " explicit@example.com "},
			source:    stubAuthorEmailSource{email: "git@example.com"},
			wantEmail: "explicit@example.com",
			wantCalls: 0,
		},
		{
			name:      "email берётся из Git",
			source:    stubAuthorEmailSource{email: " git@example.com\n"},
			wantEmail: "git@example.com",
			wantCalls: 1,
		},
		{
			name:      "email не найден",
			source:    stubAuthorEmailSource{},
			wantError: ErrAuthorEmailNotFound,
			wantCalls: 1,
		},
		{
			name:      "ошибка источника возвращается с контекстом",
			source:    stubAuthorEmailSource{err: errGitUnavailable},
			wantError: errGitUnavailable,
			wantCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := tt.source

			got, err := ResolveAuthorEmail(tt.config, &source)

			if tt.wantError != nil {
				if !errors.Is(err, tt.wantError) {
					t.Fatalf("ожидалась ошибка %v, получена %v", tt.wantError, err)
				}
			} else if err != nil {
				t.Fatalf("получена неожиданная ошибка: %v", err)
			}

			if got != tt.wantEmail {
				t.Errorf("email = %q, ожидался %q", got, tt.wantEmail)
			}
			if source.calls != tt.wantCalls {
				t.Errorf("источник вызван %d раз, ожидалось %d", source.calls, tt.wantCalls)
			}
		})
	}
}

func TestGitConfigSourceReadsRepositoryEmail(t *testing.T) {
	repository := initTestRepository(t)
	runGit(t, repository, "config", "--local", "user.email", "developer@example.com")

	email, err := (GitConfigSource{Directory: repository}).AuthorEmail()
	if err != nil {
		t.Fatalf("AuthorEmail вернул ошибку: %v", err)
	}
	if email != "developer@example.com" {
		t.Fatalf("email = %q, ожидался %q", email, "developer@example.com")
	}
}

func TestGitConfigSourceReturnsEmptyEmailWhenNotConfigured(t *testing.T) {
	repository := initTestRepository(t)

	email, err := (GitConfigSource{Directory: repository}).AuthorEmail()
	if err != nil {
		t.Fatalf("AuthorEmail вернул ошибку: %v", err)
	}
	if email != "" {
		t.Fatalf("email = %q, ожидалась пустая строка", email)
	}
}

var errGitUnavailable = errors.New("git unavailable")

type stubAuthorEmailSource struct {
	email string
	err   error
	calls int
}

func (s *stubAuthorEmailSource) AuthorEmail() (string, error) {
	s.calls++
	if s.err != nil {
		return "", s.err
	}

	return s.email, nil
}

func initTestRepository(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	repository := filepath.Join(t.TempDir(), "repository")
	if err := os.Mkdir(repository, 0o755); err != nil {
		t.Fatalf("не удалось создать тестовый репозиторий: %v", err)
	}

	t.Setenv("HOME", home)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	runGit(t, repository, "init", "--quiet")

	return repository
}

func runGit(t *testing.T, directory string, args ...string) {
	t.Helper()

	command := exec.Command("git", args...)
	command.Dir = directory
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v завершился с ошибкой: %v: %s", args, err, output)
	}
}
