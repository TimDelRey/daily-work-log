package git

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type Commit struct {
	Hash       string
	Message    string
	AuthoredAt time.Time
}

type Stash struct {
	Reference string
	Branch    string
	Message   string
	CreatedAt time.Time
}

type Client struct {
	Directory string
}

func (c Client) CurrentBranch() (string, error) {
	output, err := c.run("branch", "--show-current")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

func (c Client) Commits(authorEmail string, start, end time.Time) ([]Commit, error) {
	authorEmail = strings.TrimSpace(authorEmail)
	if authorEmail == "" {
		return nil, errors.New("author email is empty")
	}
	if !start.Before(end) {
		return nil, errors.New("commit date range must have start before end")
	}

	authorPattern := regexp.QuoteMeta("<" + authorEmail + ">")
	output, err := c.run(
		"log",
		"--since=@"+fmt.Sprint(start.Unix()),
		"--until=@"+fmt.Sprint(end.Add(-time.Second).Unix()),
		"--extended-regexp",
		"--author="+authorPattern,
		"--format=%H%x00%aI%x00%s%x00",
	)
	if err != nil {
		return nil, err
	}

	commits, err := parseCommits(output)
	if err != nil {
		return nil, fmt.Errorf("parse git log: %w", err)
	}

	return commits, nil
}

func (c Client) CommitFiles(hash string) ([]string, error) {
	if strings.TrimSpace(hash) == "" {
		return nil, errors.New("commit hash is empty")
	}

	output, err := c.run("show", "--name-only", "--format=", "-z", hash)
	if err != nil {
		return nil, err
	}

	return parseNULPaths(output), nil
}

func (c Client) Stashes() ([]Stash, error) {
	output, err := c.run("stash", "list", "--format=%gd%x00%cI%x00%gs%x00")
	if err != nil {
		return nil, err
	}

	stashes, err := parseStashes(output)
	if err != nil {
		return nil, fmt.Errorf("parse git stash list: %w", err)
	}

	return stashes, nil
}

func (c Client) StashFiles(reference string) ([]string, error) {
	if strings.TrimSpace(reference) == "" {
		return nil, errors.New("stash reference is empty")
	}

	output, err := c.run(
		"stash", "show", "--name-only", "--format=", "-z", "--include-untracked", reference,
	)
	if err != nil {
		return nil, err
	}

	return parseNULPaths(output), nil
}

func (c Client) StatusFiles() ([]string, error) {
	output, err := c.run("status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return nil, err
	}

	files, err := parseStatus(output)
	if err != nil {
		return nil, fmt.Errorf("parse git status: %w", err)
	}

	return files, nil
}

func (c Client) run(args ...string) ([]byte, error) {
	command := exec.Command("git", args...)
	command.Dir = c.Directory

	output, err := command.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return nil, fmt.Errorf("run git %s: %w", strings.Join(args, " "), err)
		}

		return nil, fmt.Errorf("run git %s: %w: %s", strings.Join(args, " "), err, message)
	}

	return output, nil
}
