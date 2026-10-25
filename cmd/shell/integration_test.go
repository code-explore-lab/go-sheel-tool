package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"shell/internal/editor"
	"shell/internal/search"
)

type integrationRunner struct {
	lookupPath string
	lookupErr  error
	runErr     error

	lookupCalls []string
	runCalls    int
	executable  string
	args        []string
	stdin       io.Reader
	stdout      io.Writer
	stderr      io.Writer
}

func (r *integrationRunner) LookPath(file string) (string, error) {
	r.lookupCalls = append(r.lookupCalls, file)
	return r.lookupPath, r.lookupErr
}

func (r *integrationRunner) Run(executable string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	r.runCalls++
	r.executable = executable
	r.args = append([]string(nil), args...)
	r.stdin = stdin
	r.stdout = stdout
	r.stderr = stderr
	return r.runErr
}

type integrationExitError struct {
	code int
}

func (e integrationExitError) Error() string {
	return fmt.Sprintf("fake nvim exited with status %d", e.code)
}

func (e integrationExitError) ExitCode() int {
	return e.code
}

func TestRunIntegrationSearchesAndOpensSpacedPath(t *testing.T) {
	root := t.TempDir()
	wanted := filepath.Join(root, "space dir", "Notes With Spaces.md")
	writeIntegrationFile(t, wanted)
	writeIntegrationFile(t, filepath.Join(root, "other.txt"))

	runner := &integrationRunner{lookupPath: `C:\tools\nvim.exe`}
	stderr := new(strings.Builder)
	deps := integrationDeps(root, runner, stderr)
	deps.selectResult = func([]string) (string, bool, error) {
		t.Fatal("selector called for one search result")
		return "", false, nil
	}

	if got := run([]string{"NOTES"}, deps); got != exitSuccess {
		t.Fatalf("run() exit code = %d, want %d; stderr=%q", got, exitSuccess, stderr.String())
	}
	if !reflect.DeepEqual(runner.lookupCalls, []string{"nvim"}) {
		t.Fatalf("LookPath calls = %#v, want [nvim]", runner.lookupCalls)
	}
	if runner.runCalls != 1 {
		t.Fatalf("Run calls = %d, want 1", runner.runCalls)
	}
	if runner.executable != runner.lookupPath {
		t.Fatalf("executable = %q, want %q", runner.executable, runner.lookupPath)
	}
	if !reflect.DeepEqual(runner.args, []string{wanted}) {
		t.Fatalf("nvim args = %#v, want one exact spaced path %q", runner.args, wanted)
	}
	if runner.stdin == nil || runner.stdout == nil || runner.stderr == nil {
		t.Fatal("editor runner did not receive terminal streams")
	}
}

func TestRunIntegrationUsesSelectorBeforeOpeningSecondMatch(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first.txt")
	second := filepath.Join(root, "nested dir", "second.txt")
	writeIntegrationFile(t, first)
	writeIntegrationFile(t, second)

	runner := &integrationRunner{lookupPath: "nvim"}
	stderr := new(strings.Builder)
	deps := integrationDeps(root, runner, stderr)
	var selectedPaths []string
	deps.selectResult = func(paths []string) (string, bool, error) {
		selectedPaths = append([]string(nil), paths...)
		return second, true, nil
	}

	if got := run([]string{".TXT"}, deps); got != exitSuccess {
		t.Fatalf("run() exit code = %d, want %d; stderr=%q", got, exitSuccess, stderr.String())
	}
	wantPaths := []string{first, second}
	if !reflect.DeepEqual(selectedPaths, wantPaths) {
		t.Fatalf("selector paths = %#v, want %#v", selectedPaths, wantPaths)
	}
	if !reflect.DeepEqual(runner.args, []string{second}) {
		t.Fatalf("nvim args = %#v, want selected path %q", runner.args, second)
	}
}

func TestRunIntegrationReportsMissingNvim(t *testing.T) {
	root := t.TempDir()
	writeIntegrationFile(t, filepath.Join(root, "notes.md"))

	lookupErr := errors.New("executable file not found")
	runner := &integrationRunner{lookupErr: lookupErr}
	stderr := new(strings.Builder)
	deps := integrationDeps(root, runner, stderr)

	if got := run([]string{"notes"}, deps); got != exitError {
		t.Fatalf("run() exit code = %d, want %d", got, exitError)
	}
	if runner.runCalls != 0 {
		t.Fatal("fake nvim ran after PATH lookup failed")
	}
	message := stderr.String()
	if !strings.Contains(message, "editor error") || !strings.Contains(message, "PATH") {
		t.Fatalf("stderr = %q, want actionable PATH error", message)
	}
	if !strings.Contains(message, "nvim") {
		t.Fatalf("stderr = %q, want nvim context", message)
	}
}

func TestRunIntegrationReportsNvimExitStatus(t *testing.T) {
	root := t.TempDir()
	writeIntegrationFile(t, filepath.Join(root, "notes.md"))

	runner := &integrationRunner{
		lookupPath: "nvim",
		runErr:     integrationExitError{code: 23},
	}
	stderr := new(strings.Builder)
	deps := integrationDeps(root, runner, stderr)

	if got := run([]string{"notes"}, deps); got != exitError {
		t.Fatalf("run() exit code = %d, want %d", got, exitError)
	}
	if !strings.Contains(stderr.String(), "status 23") {
		t.Fatalf("stderr = %q, want nvim exit status", stderr.String())
	}
}

func integrationDeps(root string, runner *integrationRunner, stderr io.Writer) cliDeps {
	return cliDeps{
		getwd:  func() (string, error) { return root, nil },
		search: search.Search,
		open:   editor.New(runner).Open,
		stderr: stderr,
	}
}

func writeIntegrationFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte("integration test"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}
