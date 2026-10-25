# Implementation Tasks

These tasks implement the confirmed MVP for the `shell` CLI. Each MVP task should normally map to one Issue and one focused PR. The repository is currently empty, so project initialization is intentionally the first task.

## Dependency map

```text
TASK-001
  ├── TASK-002 (search)
  ├── TASK-003 (editor)
  └── TASK-004 (TUI)
           ↓
      TASK-005 (CLI integration)
           ↓
      TASK-006 (integration and platform verification)
           ↓
      TASK-007 (release documentation and PATH verification)

TASK-008 and TASK-009 are optional post-MVP enhancements.
```

TASK-002, TASK-003, and TASK-004 may run in parallel after TASK-001. TASK-005 must wait for all three. TASK-006 and TASK-007 are serial.

## MVP tasks

### [x] TASK-001: Initialize the standalone Go project

**Goal**

Create the independent `shell` Go module without touching the unrelated `DI` project.

**具体工作**

- Create `go.mod` for the `shell` module.
- Add `cmd/shell` and `internal/search`, `internal/editor`, and `internal/tui` directories.
- Add a minimal executable entry point and project README skeleton.
- Record the supported environment: Windows, Git Bash, Go, and Neovim.

**前置依赖**

- None.

**交付物**

- Standalone Go module.
- Buildable command skeleton.
- Initial project documentation.

**验收标准**

- `go test ./...` runs from the new project directory.
- `go build ./cmd/shell` succeeds.
- No file under the existing `DI` project is changed.

**测试要求**

- Run `gofmt`, `go vet ./...`, `go test ./...`, and a Windows build.

### [x] TASK-002: Implement case-insensitive filename search

**Goal**

Implement deterministic recursive search using `filepath.WalkDir`.

**具体工作**

- Search the current working directory and descendants.
- Match file names with case-insensitive substring matching.
- Skip directories named `.git`, `node_modules`, and `vendor`.
- Do not follow or recursively traverse symbolic links.
- Return sorted results and non-fatal traversal warnings separately from fatal root errors.
- Avoid reading file contents and unnecessary metadata calls.

**前置依赖**

- TASK-001.

**交付物**

- `internal/search` implementation.
- Focused search unit tests.

**验收标准**

- Matching is based on file name, not directory name.
- Matching ignores case and supports spaces and Unicode names.
- Ignored directories are not traversed.
- An inaccessible child does not prevent other accessible paths from being returned.
- A root access error is reported as fatal.
- Results are stable across repeated runs.

**测试要求**

- Use `t.TempDir()`.
- Cover zero, one, and many matches, case differences, spaces, ignored directories, nested paths, and symlink behavior where supported.
- Do not depend on the user's real directory.

### [x] TASK-003: Implement the Neovim process adapter

**Goal**

Open exactly one selected path with Neovim safely and report its exit status.

**具体工作**

- Resolve `nvim` with `exec.LookPath` when an open is requested.
- Execute `nvim` with the path as a separate argument.
- Connect stdin, stdout, and stderr to the current terminal.
- Return actionable errors for missing Neovim and failed processes.
- Preserve a non-zero Neovim exit code when available.

**前置依赖**

- TASK-001.

**交付物**

- `internal/editor` implementation.
- Testable command-runner seam or equivalent unit-test mechanism.

**验收标准**

- A path containing spaces is passed as one argument.
- No shell command string is constructed.
- Missing `nvim` produces a clear PATH error.
- Neovim failure is not reported as success.

**测试要求**

- Use an injected/fake runner or test executable; do not require a real Neovim installation.
- Cover missing executable, success, non-zero exit, and spaced paths.

### [x] TASK-004: Implement the multiple-result TUI

**Goal**

Provide a small Bubble Tea selector for multiple search results.

**具体工作**

- Render the result list and current selection.
- Support Up, Down, Enter, Esc, `q`, and Ctrl-C.
- Keep the selection within bounds.
- Scroll long result lists without allocating a widget framework beyond Bubble Tea.
- Use alternate screen mode and ensure normal exit returns control to the terminal.

**前置依赖**

- TASK-001.

**交付物**

- `internal/tui` model and rendering code.
- Model-level tests that do not require a live terminal.

**验收标准**

- Enter returns exactly the selected path.
- Esc, `q`, and Ctrl-C return without selecting a path.
- Up/Down behavior is deterministic at list boundaries.
- Long lists and long paths remain usable.
- No Neovim process is started from inside the TUI model.

**测试要求**

- Test key messages and model state transitions.
- Manual terminal restoration is deferred to TASK-006.

### [x] TASK-005: Integrate the CLI workflow

**Goal**

Connect argument parsing, search, TUI selection, editor launch, output, and exit codes.

**具体工作**

- Accept `shell <one non-empty query>`.
- Handle usage errors clearly.
- Print a clear no-match message.
- Open one result directly.
- Send multiple results through the TUI, then launch Neovim after TUI exit.
- Print traversal warnings without hiding successful results.
- Define and implement the agreed exit-code behavior.

**前置依赖**

- TASK-002, TASK-003, TASK-004.

**交付物**

- Working `cmd/shell/main.go` orchestration.
- CLI behavior tests where practical.

**验收标准**

- Zero, one, and many results follow the confirmed workflows.
- User cancellation does not launch Neovim.
- Paths with spaces remain intact end to end.
- Search and editor errors are distinguishable and actionable.
- The tool performs no file write, move, delete, or rename operation.

**测试要求**

- Cover missing argument, empty query, no match, one match, multiple matches, cancellation, and editor failure.
- Use temporary directories and fake process execution.

### [x] TASK-006: Run integration and Windows Git Bash verification

**Goal**

Verify the complete workflow at the filesystem, process, and terminal boundaries.

**具体工作**

- Add or complete integration tests using temporary trees.
- Use a fake `nvim` executable or runner to capture arguments and exit codes.
- Build the Windows executable.
- Manually run the binary from Git Bash.
- Verify raw mode, alternate screen restoration, arrow keys, Enter, Esc, `q`, Ctrl-C, paths with spaces, and Neovim return behavior.

**前置依赖**

- TASK-005.

**交付物**

- Integration test coverage.
- Manual verification record with actual commands and results.
- List of any platform limitations or follow-up Issues.

**验收标准**

- `go test ./...` passes without a real user directory or real Neovim.
- `go vet ./...` and formatting checks pass.
- Git Bash remains usable after TUI and Neovim exit.
- The executable works when its directory is on PATH.
- Missing `nvim` and non-zero editor exit are visibly reported.

**测试要求**

- Do not claim manual checks were completed without recording actual results.
- If a terminal-specific check cannot run, mark it as an explicit limitation rather than weakening the test.

### [x] TASK-007: Complete release and installation documentation

**Goal**

Document repeatable build, PATH installation, usage, limitations, and support expectations.

**具体工作**

- Document Windows build commands.
- Document adding the executable directory to Git Bash PATH.
- Document `shell <query>` behavior and exit outcomes.
- Document default ignored directories and symlink policy.
- Document the requirement that `nvim.exe` be discoverable through PATH.
- Update task status only to reflect completed work.

**前置依赖**

- TASK-006.

**交付物**

- Completed README and release notes or installation section.
- Final task checklist status.

**验收标准**

- A new user can build, put the executable on PATH, and run the documented command.
- Documentation does not promise unsupported `.gitignore` parsing, symlink following, or shell aliases.
- All documented test results are real and reproducible.

**测试要求**

- Follow the documented build and installation steps from a clean shell session.

## Optional post-MVP enhancements

### [ ] TASK-008: Add configurable ignored directories

**Goal**

Allow users to add or override ignored directory names without changing code.

**具体工作**

- Choose and document a minimal interface such as `--ignore-dir`.
- Preserve the safe default behavior.
- Add parsing, validation, and help text.

**前置依赖**

- TASK-007.

**交付物**

- CLI option, tests, and documentation.

**验收标准**

- Custom ignored directories are pruned deterministically.
- Existing MVP behavior remains unchanged when the option is absent.

**测试要求**

- Unit-test option parsing and integration-test traversal.

### [ ] TASK-009: Evaluate optional symbolic-link following

**Goal**

Assess whether an explicit opt-in symlink mode is useful without compromising safety.

**具体工作**

- Document cycle, duplicate, broken-link, and permission risks.
- Prototype only if a concrete use case exists.
- Add cycle detection and bounded behavior before enabling it.

**前置依赖**

- TASK-007.

**交付物**

- Decision record; implementation only if approved.

**验收标准**

- The default remains no symlink following.
- Any opt-in mode has deterministic tests for cycles and broken links.

**测试要求**

- Use isolated temporary trees and platform-specific skips where symlink creation is unavailable.
