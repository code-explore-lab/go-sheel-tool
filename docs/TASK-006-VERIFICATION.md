# TASK-006 verification record

Date: 2026-10-25 (Asia/Shanghai)

## Environment

The checks below used the installed Git Bash executable, not a PowerShell
compatibility shell:

```text
shell=/usr/bin/bash
git version 2.54.0.windows.1
go version go1.27.0 windows/amd64
/d/neovim/nvim-win64/bin/nvim
NVIM v0.12.3
```

The command used to collect this information was:

```powershell
& 'C:\Program Files\Git\bin\bash.exe' -lc 'cd /e/Desktop/go-shell-tool && printf "shell=%s\n" "$SHELL" && git --version && go version && command -v nvim && nvim --version | head -n 2'
```

## Automated integration checks

`cmd/shell/integration_test.go` builds temporary directory trees with
`t.TempDir()`, uses the real search implementation, and injects a fake
Neovim runner. It verifies spaced paths, selector-to-editor flow, missing
`nvim`, and a non-zero editor exit status without touching a user directory
or requiring a real editor process.

The following commands all passed:

```text
gofmt -l .
go vet ./...
go test -count=1 ./...
go test -race ./...
go mod verify
```

`go build ./cmd/shell` also succeeded on `windows/amd64`.

## Git Bash and PTY checks

For the manual checks, a temporary `nvim.cmd` wrapper captured its single
argument and returned a configurable exit status. The binary directory was
placed at the front of `PATH`.

| Scenario | Input | Result |
| --- | --- | --- |
| PATH lookup | `command -v shell.exe` | Resolved `/e/Desktop/go-shell-tool/manual-task006/shell.exe`. |
| One result with spaces | `shell.exe spaces` | Exit `0`; wrapper captured one argument ending in `notes with spaces.txt`. |
| Missing editor | `PATH=/does-not-exist ./shell.exe spaces` | Exit `1`; reported `nvim was not found on PATH`. |
| Editor failure | `NVIM_EXIT=23 ./shell.exe spaces` | Exit `1`; reported Neovim status `23`. |
| Down + Enter | `shell.exe notes`, then Down and Enter | Exit `0`; wrapper received `second notes.txt`. |
| Up + Enter | `shell.exe notes`, then Down, Up, Enter | Exit `0`; wrapper received `notes with spaces.txt`. |
| `q` | `shell.exe notes`, then `q` | Exit `0`; no editor invocation. |
| Esc | `shell.exe notes`, then Esc | Exit `0`; no editor invocation. |
| Ctrl-C | `shell.exe notes`, then Ctrl-C | Exit `0`; no editor invocation. |
| Real Neovim | `shell.exe notes`, Enter, then `:q` + Enter in NVIM v0.12.3 | NVIM and the CLI both exited `0`; the Git Bash session and alternate screen were restored. |

The PTY output contained Bubble Tea's alternate-screen enter/leave sequences
(`?1049h` / `?1049l`) and cursor/raw-mode cleanup. Each process returned to
the invoking Git Bash session, and subsequent commands ran normally.

The documented build and PATH steps were also run in a clean Git Bash session
(`--noprofile --norc`):

```text
shell_path=/e/Desktop/go-shell-tool/shell.exe
nvim_path=/d/neovim/nvim-win64/bin/nvim
NVIM v0.12.3
no matches for "__definitely_no_match__"
cli_exit=1
```

## Limitations

- The wrapper was used for deterministic argument and exit-code assertions.
  A separate PTY run also opened the real `nvim.exe` (NVIM v0.12.3) on a
  spaced path and exited with `:q`; both the editor and CLI returned `0`.
- The interactive key checks used a real PTY and recorded process results;
  terminal rendering can vary with terminal emulator dimensions.
