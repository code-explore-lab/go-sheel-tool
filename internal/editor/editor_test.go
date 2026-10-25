package editor

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

type fakeRunner struct {
	lookupPath    string
	lookupErr     error
	runErr        error
	lookupCalls   []string
	runExecutable string
	runArgs       []string
	runStdin      io.Reader
	runStdout     io.Writer
	runStderr     io.Writer
}

func (f *fakeRunner) LookPath(file string) (string, error) {
	f.lookupCalls = append(f.lookupCalls, file)
	return f.lookupPath, f.lookupErr
}

func (f *fakeRunner) Run(executable string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	f.runExecutable = executable
	f.runArgs = append([]string(nil), args...)
	f.runStdin = stdin
	f.runStdout = stdout
	f.runStderr = stderr
	return f.runErr
}

type fakeExitError struct {
	code int
}

func (e fakeExitError) Error() string {
	return fmt.Sprintf("process exited with code %d", e.code)
}

func (e fakeExitError) ExitCode() int {
	return e.code
}

func TestOpenPassesPathAsOneArgumentAndConnectsTerminal(t *testing.T) {
	runner := &fakeRunner{lookupPath: `C:\Program Files\Neovim\bin\nvim.exe`}
	path := `C:\work dir\notes with spaces.md`

	if err := New(runner).Open(path); err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	if got, want := strings.Join(runner.lookupCalls, ","), executableName; got != want {
		t.Fatalf("LookPath calls = %q, want %q", got, want)
	}
	if runner.runExecutable != runner.lookupPath {
		t.Fatalf("Run executable = %q, want %q", runner.runExecutable, runner.lookupPath)
	}
	if len(runner.runArgs) != 1 || runner.runArgs[0] != path {
		t.Fatalf("Run args = %#v, want one exact path argument %q", runner.runArgs, path)
	}
	if runner.runStdin != os.Stdin || runner.runStdout != os.Stdout || runner.runStderr != os.Stderr {
		t.Fatal("Run did not receive the current terminal streams")
	}
}

func TestOpenReportsMissingNvim(t *testing.T) {
	lookupErr := errors.New("executable file not found")
	runner := &fakeRunner{lookupErr: lookupErr}

	err := New(runner).Open("notes.md")
	if err == nil {
		t.Fatal("Open() error = nil, want missing nvim error")
	}
	if !strings.Contains(err.Error(), "nvim") || !strings.Contains(err.Error(), "PATH") {
		t.Fatalf("Open() error = %q, want nvim and PATH context", err)
	}
	if !errors.Is(err, lookupErr) {
		t.Fatalf("Open() error does not wrap lookup error: %v", err)
	}
	if runner.runExecutable != "" {
		t.Fatal("Run was called after PATH lookup failed")
	}
}

func TestOpenSuccess(t *testing.T) {
	runner := &fakeRunner{lookupPath: "nvim"}

	if err := New(runner).Open("notes.md"); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
}

func TestOpenReportsAndPreservesNonZeroExit(t *testing.T) {
	processErr := fakeExitError{code: 17}
	runner := &fakeRunner{lookupPath: "nvim", runErr: processErr}

	err := New(runner).Open("notes.md")
	if err == nil {
		t.Fatal("Open() error = nil, want process failure")
	}
	if !strings.Contains(err.Error(), "status 17") {
		t.Fatalf("Open() error = %q, want exit status", err)
	}
	var exitErr fakeExitError
	if !errors.As(err, &exitErr) || exitErr.code != processErr.code {
		t.Fatalf("Open() error does not preserve process error: %v", err)
	}
}

func TestOpenRejectsEmptyPath(t *testing.T) {
	runner := &fakeRunner{lookupPath: "nvim"}

	err := New(runner).Open("")
	if err == nil {
		t.Fatal("Open() error = nil, want empty path error")
	}
	if !strings.Contains(err.Error(), "path must not be empty") {
		t.Fatalf("Open() error = %q, want empty path context", err)
	}
	if len(runner.lookupCalls) != 0 {
		t.Fatal("LookPath was called for an empty path")
	}
}
