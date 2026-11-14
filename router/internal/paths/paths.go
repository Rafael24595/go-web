package paths

import (
	"cmp"
	"slices"
	"strings"
)

// HasPrefix reports whether path is prefix or lies below it, matching
// whole segments: /api is a prefix of /api and /api/items, not of /apiary.
func HasPrefix(path, prefix string) bool {
	if !strings.HasPrefix(path, prefix) {
		return false
	}

	return len(path) == len(prefix) ||
		strings.HasSuffix(prefix, "/") ||
		path[len(prefix)] == '/'
}

// Groups returns the groups path belongs to, the outer ones first
// (/api before /api/admin), so that inner groups act after outer ones.
func Groups(path string, groups []string) []string {
	matching := slices.DeleteFunc(slices.Clone(groups), func(group string) bool {
		return !HasPrefix(path, group)
	})

	slices.SortStableFunc(matching, func(a, b string) int {
		return cmp.Compare(len(a), len(b))
	})

	return matching
}