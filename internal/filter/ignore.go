package filter

import "strings"

// Matcher decides whether repository paths should be included in a report.
type Matcher struct {
	rules []rule
}

type rule func(string) bool

// Default returns the ignore rules used by worklog.
func Default() Matcher {
	return Matcher{rules: []rule{
		hasSuffix(".rbi"),
		isInside("sorbet/rbi"),
	}}
}

// Include reports whether path is useful for a worklog report.
func (m Matcher) Include(path string) bool {
	for _, ignore := range m.rules {
		if ignore(path) {
			return false
		}
	}

	return true
}

func hasSuffix(suffix string) rule {
	return func(path string) bool {
		return strings.HasSuffix(path, suffix)
	}
}

func isInside(directory string) rule {
	prefix := strings.TrimSuffix(directory, "/") + "/"
	return func(path string) bool {
		return strings.HasPrefix(path, prefix)
	}
}
