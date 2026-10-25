package editor

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

const executableName = "nvim"

// Runner abstracts PATH lookup and process execution so editor behavior can be
// tested without requiring Neovim to be installed.
type Runner interface {
	LookPath(file string) (string, error)
	Run(executable string, args []string, stdin io.Reader, stdout, stderr io.Writer) error
}

// Editor launches Neovim with the selected path.
type Editor struct {
	runner Runner
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

// New returns an Editor backed by runner. A nil runner uses the operating
// system's PATH lookup and process execution.
func New(runner Runner) *Editor {
	if runner == nil {
		runner = systemRunner{}
	}

	return &Editor{
		runner: runner,
		stdin:  os.Stdin,
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
}

// Open resolves nvim and opens path in it. The path is passed as one process
// argument, so spaces and other shell-sensitive characters remain intact.
func Open(path string) error {
	return New(nil).Open(path)
}

// Open resolves nvim and opens path in it using this Editor's runner.
func (e *Editor) Open(path string) error {
	if e == nil {
		return errors.New("editor: cannot open a path with a nil editor")
	}
	if path == "" {
		return errors.New("editor: path must not be empty")
	}

	runner := e.runner
	if runner == nil {
		runner = systemRunner{}
	}
	stdin, stdout, stderr := e.stdin, e.stdout, e.stderr
	if stdin == nil {
		stdin = os.Stdin
	}
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}

	executable, err := runner.LookPath(executableName)
	if err != nil {
		return fmt.Errorf("editor: nvim was not found on PATH: %w", err)
	}

	if err := runner.Run(executable, []string{path}, stdin, stdout, stderr); err != nil {
		var exitCoder interface{ ExitCode() int }
		if errors.As(err, &exitCoder) {
			return fmt.Errorf("editor: nvim exited with status %d while opening %q: %w", exitCoder.ExitCode(), path, err)
		}
		return fmt.Errorf("editor: failed to open %q with nvim: %w", path, err)
	}

	return nil
}

type systemRunner struct{}

func (systemRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (systemRunner) Run(executable string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	command := exec.Command(executable, args...)
	command.Stdin = stdin
	command.Stdout = stdout
	command.Stderr = stderr
	return command.Run()
}
