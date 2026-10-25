package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"shell/internal/search"
)

func testDeps() (cliDeps, *bytes.Buffer, *bytes.Buffer) {
	stderr := new(bytes.Buffer)
	deps := cliDeps{
		getwd: func() (string, error) { return `C:\work dir`, nil },
		search: func(string, string) ([]string, []search.Warning, error) {
			return nil, nil, nil
		},
		selectResult: func([]string) (string, bool, error) { return "", false, nil },
		open:         func(string) error { return nil },
		stderr:       stderr,
	}
	return deps, nil, stderr
}

func TestRunRejectsInvalidArguments(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "missing query", args: nil},
		{name: "empty query", args: []string{""}},
		{name: "too many arguments", args: []string{"one", "two"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deps, _, stderr := testDeps()
			if got := run(test.args, deps); got != exitUsage {
				t.Fatalf("run() exit code = %d, want %d", got, exitUsage)
			}
			if !strings.Contains(stderr.String(), "usage: shell <non-empty query>") {
				t.Fatalf("stderr = %q, want usage message", stderr.String())
			}
		})
	}
}

func TestRunReportsNoMatchWithoutOpeningEditor(t *testing.T) {
	deps, _, stderr := testDeps()
	opened := false
	deps.search = func(root, query string) ([]string, []search.Warning, error) {
		if root != `C:\work dir` || query != "needle" {
			t.Fatalf("search arguments = (%q, %q)", root, query)
		}
		return nil, nil, nil
	}
	deps.open = func(string) error {
		opened = true
		return nil
	}

	if got := run([]string{"needle"}, deps); got != exitError {
		t.Fatalf("run() exit code = %d, want %d", got, exitError)
	}
	if opened {
		t.Fatal("editor opened after no match")
	}
	if !strings.Contains(stderr.String(), `no matches for "needle"`) {
		t.Fatalf("stderr = %q, want no-match message", stderr.String())
	}
}

func TestRunOpensSingleMatchWithSpaces(t *testing.T) {
	deps, _, stderr := testDeps()
	wantPath := `C:\work dir\notes with spaces.md`
	var openedPath string
	selectorCalled := false
	deps.search = func(string, string) ([]string, []search.Warning, error) {
		return []string{wantPath}, nil, nil
	}
	deps.selectResult = func([]string) (string, bool, error) {
		selectorCalled = true
		return "", false, nil
	}
	deps.open = func(path string) error {
		openedPath = path
		return nil
	}

	if got := run([]string{"notes"}, deps); got != exitSuccess {
		t.Fatalf("run() exit code = %d, want %d", got, exitSuccess)
	}
	if selectorCalled {
		t.Fatal("selector called for a single match")
	}
	if openedPath != wantPath {
		t.Fatalf("opened path = %q, want %q", openedPath, wantPath)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty output", stderr.String())
	}
}

func TestRunSelectsAndOpensMultipleMatches(t *testing.T) {
	deps, _, stderr := testDeps()
	paths := []string{`C:\one.txt`, `C:\dir with spaces\two.txt`}
	selectedPath := paths[1]
	var selectorPaths []string
	var openedPath string
	deps.search = func(string, string) ([]string, []search.Warning, error) {
		return paths, nil, nil
	}
	deps.selectResult = func(got []string) (string, bool, error) {
		selectorPaths = append([]string(nil), got...)
		return selectedPath, true, nil
	}
	deps.open = func(path string) error {
		openedPath = path
		return nil
	}

	if got := run([]string{"txt"}, deps); got != exitSuccess {
		t.Fatalf("run() exit code = %d, want %d", got, exitSuccess)
	}
	if len(selectorPaths) != len(paths) || selectorPaths[1] != paths[1] {
		t.Fatalf("selector paths = %#v, want %#v", selectorPaths, paths)
	}
	if openedPath != selectedPath {
		t.Fatalf("opened path = %q, want %q", openedPath, selectedPath)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty output", stderr.String())
	}
}

func TestRunCancellationDoesNotOpenEditor(t *testing.T) {
	deps, _, stderr := testDeps()
	opened := false
	deps.search = func(string, string) ([]string, []search.Warning, error) {
		return []string{"one", "two"}, nil, nil
	}
	deps.selectResult = func([]string) (string, bool, error) {
		return "", false, nil
	}
	deps.open = func(string) error {
		opened = true
		return nil
	}

	if got := run([]string{"query"}, deps); got != exitSuccess {
		t.Fatalf("run() exit code = %d, want %d", got, exitSuccess)
	}
	if opened {
		t.Fatal("editor opened after cancellation")
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty output", stderr.String())
	}
}

func TestRunPrintsWarningsAndContinues(t *testing.T) {
	deps, _, stderr := testDeps()
	warning := search.Warning{Path: `C:\restricted`, Err: errors.New("access denied")}
	opened := false
	deps.search = func(string, string) ([]string, []search.Warning, error) {
		return []string{`C:\visible.txt`}, []search.Warning{warning}, nil
	}
	deps.open = func(string) error {
		opened = true
		return nil
	}

	if got := run([]string{"txt"}, deps); got != exitSuccess {
		t.Fatalf("run() exit code = %d, want %d", got, exitSuccess)
	}
	if !opened {
		t.Fatal("editor was not opened despite a successful match")
	}
	if !strings.Contains(stderr.String(), "warning: C:\\restricted: access denied") {
		t.Fatalf("stderr = %q, want warning", stderr.String())
	}
}

func TestRunDistinguishesSearchAndEditorFailures(t *testing.T) {
	t.Run("search", func(t *testing.T) {
		deps, _, stderr := testDeps()
		searchErr := errors.New("walk root failed")
		deps.search = func(string, string) ([]string, []search.Warning, error) {
			return nil, nil, searchErr
		}

		if got := run([]string{"query"}, deps); got != exitError {
			t.Fatalf("run() exit code = %d, want %d", got, exitError)
		}
		if !strings.Contains(stderr.String(), "search error: walk root failed") {
			t.Fatalf("stderr = %q, want search error", stderr.String())
		}
	})

	t.Run("editor", func(t *testing.T) {
		deps, _, stderr := testDeps()
		editorErr := errors.New("nvim failed")
		deps.search = func(string, string) ([]string, []search.Warning, error) {
			return []string{"one"}, nil, nil
		}
		deps.open = func(string) error { return editorErr }

		if got := run([]string{"query"}, deps); got != exitError {
			t.Fatalf("run() exit code = %d, want %d", got, exitError)
		}
		if !strings.Contains(stderr.String(), "editor error: nvim failed") {
			t.Fatalf("stderr = %q, want editor error", stderr.String())
		}
	})
}

func TestRunReportsSelectorFailure(t *testing.T) {
	deps, _, stderr := testDeps()
	selectorErr := errors.New("terminal unavailable")
	deps.search = func(string, string) ([]string, []search.Warning, error) {
		return []string{"one", "two"}, nil, nil
	}
	deps.selectResult = func([]string) (string, bool, error) {
		return "", false, selectorErr
	}

	if got := run([]string{"query"}, deps); got != exitError {
		t.Fatalf("run() exit code = %d, want %d", got, exitError)
	}
	if !strings.Contains(stderr.String(), "selector error: terminal unavailable") {
		t.Fatalf("stderr = %q, want selector error", stderr.String())
	}
}

func TestRunReportsWorkingDirectoryFailure(t *testing.T) {
	deps, _, stderr := testDeps()
	getwdErr := errors.New("current directory unavailable")
	deps.getwd = func() (string, error) { return "", getwdErr }

	if got := run([]string{"query"}, deps); got != exitError {
		t.Fatalf("run() exit code = %d, want %d", got, exitError)
	}
	if !strings.Contains(stderr.String(), "search error: cannot determine current directory") {
		t.Fatalf("stderr = %q, want working-directory error", stderr.String())
	}
}
