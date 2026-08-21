package git

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func parseBranchRefs(output []byte) ([]BranchRef, error) {
	fields := splitNULFields(output)
	if len(fields)%3 != 0 {
		return nil, fmt.Errorf("unexpected field count: %d", len(fields))
	}

	refs := make([]BranchRef, 0, len(fields)/3)
	for i := 0; i < len(fields); i += 3 {
		refs = append(refs, BranchRef{
			Name:     strings.TrimPrefix(fields[i], "\n"),
			FullName: fields[i+1],
			Upstream: fields[i+2],
		})
	}
	return refs, nil
}

func parseReflog(reference string, output []byte) ([]ReflogEntry, error) {
	fields := splitNULFields(output)
	if len(fields)%4 != 0 {
		return nil, fmt.Errorf("unexpected field count: %d", len(fields))
	}

	entries := make([]ReflogEntry, 0, len(fields)/4)
	for i := 0; i < len(fields); i += 4 {
		occurredAt, err := parseReflogSelectorTime(fields[i+2])
		if err != nil {
			return nil, fmt.Errorf("parse reflog date %q: %w", fields[i+2], err)
		}
		entries = append(entries, ReflogEntry{
			Ref:        reference,
			NewOID:     strings.TrimPrefix(fields[i], "\n"),
			Selector:   fields[i+1],
			OccurredAt: occurredAt,
			Subject:    fields[i+3],
		})
	}

	for i := 0; i+1 < len(entries); i++ {
		entries[i].OldOID = entries[i+1].NewOID
	}
	return entries, nil
}

func parseReflogSelectorTime(selector string) (time.Time, error) {
	start := strings.LastIndex(selector, "@{")
	if start == -1 || !strings.HasSuffix(selector, "}") {
		return time.Time{}, errors.New("date is missing from reflog selector")
	}
	return time.Parse(time.RFC3339, selector[start+2:len(selector)-1])
}

func parseAheadBehind(output []byte) (ahead, behind int, err error) {
	fields := strings.Fields(string(output))
	if len(fields) != 2 {
		return 0, 0, fmt.Errorf("unexpected field count: %d", len(fields))
	}
	ahead, err = strconv.Atoi(fields[0])
	if err != nil {
		return 0, 0, fmt.Errorf("parse ahead count %q: %w", fields[0], err)
	}
	behind, err = strconv.Atoi(fields[1])
	if err != nil {
		return 0, 0, fmt.Errorf("parse behind count %q: %w", fields[1], err)
	}
	return ahead, behind, nil
}

func parseCommits(output []byte) ([]Commit, error) {
	fields := splitNULFields(output)
	if len(fields)%3 != 0 {
		return nil, fmt.Errorf("unexpected field count: %d", len(fields))
	}

	commits := make([]Commit, 0, len(fields)/3)
	for i := 0; i < len(fields); i += 3 {
		authoredAt, err := time.Parse(time.RFC3339, fields[i+1])
		if err != nil {
			return nil, fmt.Errorf("parse authored date %q: %w", fields[i+1], err)
		}

		commits = append(commits, Commit{
			Hash:       strings.TrimPrefix(fields[i], "\n"),
			AuthoredAt: authoredAt,
			Message:    fields[i+2],
		})
	}

	return commits, nil
}

func parseStashes(output []byte) ([]Stash, error) {
	fields := splitNULFields(output)
	if len(fields)%3 != 0 {
		return nil, fmt.Errorf("unexpected field count: %d", len(fields))
	}

	stashes := make([]Stash, 0, len(fields)/3)
	for i := 0; i < len(fields); i += 3 {
		createdAt, err := time.Parse(time.RFC3339, fields[i+1])
		if err != nil {
			return nil, fmt.Errorf("parse stash date %q: %w", fields[i+1], err)
		}

		branch, message := parseStashSubject(fields[i+2])
		stashes = append(stashes, Stash{
			Reference: strings.TrimPrefix(fields[i], "\n"),
			Branch:    branch,
			Message:   message,
			CreatedAt: createdAt,
		})
	}

	return stashes, nil
}

func parseStashSubject(subject string) (branch, message string) {
	for _, prefix := range []string{"WIP on ", "On "} {
		if !strings.HasPrefix(subject, prefix) {
			continue
		}

		rest := strings.TrimPrefix(subject, prefix)
		branch, message, found := strings.Cut(rest, ": ")
		if found {
			return branch, message
		}
	}

	return "", subject
}

func parseStatus(output []byte) ([]string, error) {
	entries := splitNULFields(output)
	files := make([]string, 0, len(entries))

	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		if len(entry) < 4 || entry[2] != ' ' {
			return nil, fmt.Errorf("unexpected status entry %q", entry)
		}

		files = append(files, entry[3:])
		if entry[0] == 'R' || entry[0] == 'C' || entry[1] == 'R' || entry[1] == 'C' {
			i++
			if i >= len(entries) {
				return nil, fmt.Errorf("missing original path for %q", entry)
			}
		}
	}

	return files, nil
}

func parseNULPaths(output []byte) []string {
	return splitNULFields(output)
}

func splitNULFields(output []byte) []string {
	if len(output) == 0 {
		return nil
	}

	rawFields := strings.Split(string(output), "\x00")
	if rawFields[len(rawFields)-1] == "" {
		rawFields = rawFields[:len(rawFields)-1]
	}
	if len(rawFields) > 0 && rawFields[len(rawFields)-1] == "\n" {
		rawFields = rawFields[:len(rawFields)-1]
	}

	return rawFields
}
