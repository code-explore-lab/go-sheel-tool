// Package search provides deterministic filename search for the shell CLI.
package search

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// Warning describes a non-fatal filesystem error encountered while walking a
// search tree. The path is kept separately so callers can present an
// actionable warning without having to parse an error string.
type Warning struct {
	Path string
	Err  error
}

// Error implements error for convenient display of a traversal warning.
func (w Warning) Error() string {
	if w.Err == nil {
		return w.Path
	}
	return fmt.Sprintf("%s: %v", w.Path, w.Err)
}

// Unwrap exposes the underlying filesystem error to errors.Is and errors.As.
func (w Warning) Unwrap() error {
	return w.Err
}

var ignoredDirectories = map[string]struct{}{
	".git":         {},
	"node_modules": {},
	"vendor":       {},
}

// Search recursively walks root and returns files whose names contain query,
// ignoring case. Results are sorted by path. Errors while opening a child
// directory are returned as warnings and do not prevent the rest of the tree
// from being searched. An error for root itself is returned as the fatal error.
//
// WalkDir does not follow symbolic links, so linked directories are never
// traversed. A symbolic-link entry may still be returned when its own name
// matches query.
func Search(root, query string) (paths []string, warnings []Warning, err error) {
	needle := strings.ToLower(query)

	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if filepath.Clean(path) == filepath.Clean(root) {
				return fmt.Errorf("walk root %q: %w", root, walkErr)
			}
			warnings = append(warnings, Warning{Path: path, Err: walkErr})
			return nil
		}

		if entry.IsDir() {
			if isIgnoredDirectory(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.Contains(strings.ToLower(entry.Name()), needle) {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, sortWarnings(warnings), err
	}

	sort.Strings(paths)
	return paths, sortWarnings(warnings), nil
}

func isIgnoredDirectory(name string) bool {
	for ignored := range ignoredDirectories {
		if strings.EqualFold(name, ignored) {
			return true
		}
	}
	return false
}

func sortWarnings(warnings []Warning) []Warning {
	sort.SliceStable(warnings, func(i, j int) bool {
		if warnings[i].Path != warnings[j].Path {
			return warnings[i].Path < warnings[j].Path
		}
		return warnings[i].Error() < warnings[j].Error()
	})
	return warnings
}
