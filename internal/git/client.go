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

type ReflogEntry struct {
	Ref        string
	Selector   string
	OldOID     string
	NewOID     string
	OccurredAt time.Time
	Subject    string
}

type BranchRef struct {
	Name     string
	FullName string
	Upstream string
}

type UpstreamState struct {
	Branch   string
	Upstream string
	Ahead    int
	Behind   int
	Known    bool
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

func (c Client) Branches() ([]string, error) {
	output, err := c.run("for-each-ref", "--sort=refname", "--format=%(refname:short)", "refs/heads")
	if err != nil {
		return nil, err
	}

	return strings.Fields(string(output)), nil
}

func (c Client) BranchRefs() ([]BranchRef, error) {
	output, err := c.run(
		"for-each-ref", "--sort=refname",
		"--format=%(refname:short)%00%(refname)%00%(upstream:short)%00",
		"refs/heads",
	)
	if err != nil {
		return nil, err
	}

	refs, err := parseBranchRefs(output)
	if err != nil {
		return nil, fmt.Errorf("parse local branch refs: %w", err)
	}
	return refs, nil
}

func (c Client) RemoteRefs() ([]BranchRef, error) {
	output, err := c.run(
		"for-each-ref", "--sort=refname",
		"--format=%(refname:short)%00%(refname)%00%00",
		"refs/remotes",
	)
	if err != nil {
		return nil, err
	}

	refs, err := parseBranchRefs(output)
	if err != nil {
		return nil, fmt.Errorf("parse remote refs: %w", err)
	}
	return refs, nil
}

func (c Client) Reflog(reference string) ([]ReflogEntry, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return nil, errors.New("reflog reference is empty")
	}

	output, err := c.run(
		"reflog", "show", "--date=iso-strict",
		"--format=%H%x00%gD%x00%gD%x00%gs%x00", reference,
	)
	if err != nil {
		return nil, err
	}

	entries, err := parseReflog(reference, output)
	if err != nil {
		return nil, fmt.Errorf("parse reflog for %s: %w", reference, err)
	}
	return entries, nil
}

func (c Client) UpstreamState(branch, upstream string) (UpstreamState, error) {
	branch = strings.TrimSpace(branch)
	upstream = strings.TrimSpace(upstream)
	if branch == "" {
		return UpstreamState{}, errors.New("branch is empty")
	}
	if upstream == "" {
		return UpstreamState{}, errors.New("upstream is empty")
	}
	state := UpstreamState{Branch: branch, Upstream: upstream}
	for _, reference := range []string{branch, upstream} {
		exists, err := c.commitExists(reference)
		if err != nil {
			return UpstreamState{}, err
		}
		if !exists {
			return state, nil
		}
	}

	output, err := c.run("rev-list", "--left-right", "--count", branch+"..."+upstream)
	if err != nil {
		return UpstreamState{}, err
	}

	ahead, behind, err := parseAheadBehind(output)
	if err != nil {
		return UpstreamState{}, fmt.Errorf("parse upstream state: %w", err)
	}
	state.Ahead = ahead
	state.Behind = behind
	state.Known = true
	return state, nil
}

func (c Client) commitExists(reference string) (bool, error) {
	command := exec.Command("git", "rev-parse", "--verify", "--quiet", reference+"^{commit}")
	command.Dir = c.Directory
	output, err := command.CombinedOutput()
	if err == nil {
		return true, nil
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) && exitError.ExitCode() == 1 {
		return false, nil
	}
	return false, fmt.Errorf("resolve commit %s: %w: %s", reference, err, strings.TrimSpace(string(output)))
}

func (c Client) IsAncestor(olderOID, newerOID string) (bool, error) {
	olderOID = strings.TrimSpace(olderOID)
	newerOID = strings.TrimSpace(newerOID)
	if olderOID == "" || newerOID == "" {
		return false, errors.New("both object IDs are required")
	}

	command := exec.Command("git", "merge-base", "--is-ancestor", olderOID, newerOID)
	command.Dir = c.Directory
	output, err := command.CombinedOutput()
	if err == nil {
		return true, nil
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) && exitError.ExitCode() == 1 {
		return false, nil
	}
	return false, fmt.Errorf("check whether %s is ancestor of %s: %w: %s",
		olderOID, newerOID, err, strings.TrimSpace(string(output)))
}

func (c Client) Commits(branch, authorEmail string, start, end time.Time) ([]Commit, error) {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return nil, errors.New("branch is empty")
	}
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
		branch,
		"--",
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
