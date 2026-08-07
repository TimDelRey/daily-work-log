package config

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

var ErrAuthorEmailNotFound = errors.New("author email is not configured")

type AuthorEmailSource interface {
	AuthorEmail() (string, error)
}

func ResolveAuthorEmail(cfg Config, source AuthorEmailSource) (string, error) {
	if email := strings.TrimSpace(cfg.AuthorEmail); email != "" {
		return email, nil
	}

	email, err := source.AuthorEmail()
	if err != nil {
		return "", fmt.Errorf("read author email from git config: %w", err)
	}

	email = strings.TrimSpace(email)
	if email == "" {
		return "", ErrAuthorEmailNotFound
	}

	return email, nil
}

type GitConfigSource struct {
	Directory string
}

func (s GitConfigSource) AuthorEmail() (string, error) {
	command := exec.Command("git", "config", "--get", "user.email")
	command.Dir = s.Directory

	output, err := command.Output()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) && exitError.ExitCode() == 1 {
			return "", nil
		}

		return "", fmt.Errorf("run git config user.email: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}
