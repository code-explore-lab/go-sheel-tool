# shell

`shell` is a standalone Windows/Git Bash command-line tool for finding files
and opening a selected result in Neovim. The MVP workflow is implemented in
the command and the supporting search, editor, and selector packages.

## Supported environment

- Windows
- Git Bash
- Go 1.23 or newer
- Neovim with `nvim.exe` discoverable through `PATH`

## Build and install

From PowerShell or Git Bash, build the Windows executable in the project
directory:

```sh
go build -o shell.exe ./cmd/shell
```

Do not add the executable to the repository. To use it from Git Bash for the
current session, put its directory on `PATH`:

```sh
export PATH="/e/Desktop/go-shell-tool:$PATH"
command -v shell.exe
```

To make that PATH change persistent, add the same `export PATH=...` line to
your Git Bash `~/.bashrc`, then start a new shell. PowerShell and Git Bash may
have different PATH values; verify `nvim.exe` in the shell where `shell` will
run:

```sh
command -v nvim
nvim --version
```

The tool uses `exec.LookPath`, so Bash aliases and functions do not satisfy
the `nvim.exe` requirement.

## Usage

```sh
shell <query>
```

On Windows Git Bash, `shell.exe <query>` is equivalent when the executable
extension is not resolved automatically.

The query is one non-empty argument. The tool recursively searches the
current working directory by file name, using a case-insensitive substring
match. It skips directories named `.git`, `node_modules`, and `vendor` and
does not parse `.gitignore` or follow symbolic links.

- Zero matches prints a message and exits with status `1`.
- One match opens that path directly in Neovim.
- Multiple matches open the Bubble Tea selector. Enter opens the selection;
  Esc, `q`, or Ctrl-C cancels without starting Neovim.
- Non-fatal traversal warnings are printed to stderr while successful
  matches remain usable.

## Exit statuses

| Status | Meaning |
| ---: | --- |
| `0` | Successful open or user cancellation. |
| `1` | No match, search failure, selector failure, or Neovim failure. |
| `2` | Invalid usage: missing, empty, or multiple query arguments. |

The tool itself does not write, move, rename, or delete user files. Neovim
may edit a selected file only after the user explicitly chooses to open it.

## Verification

Run the repository checks from the project directory:

```sh
gofmt -l .
go vet ./...
go test ./...
go build ./cmd/shell
```

The integration and Windows/Git Bash boundary checks, including their actual
commands and results, are recorded in
[`docs/TASK-006-VERIFICATION.md`](docs/TASK-006-VERIFICATION.md).
