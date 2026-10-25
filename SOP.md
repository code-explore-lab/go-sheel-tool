# Go Shell Tool Standard Operating Procedure

## 1. Scope and development principles

This document applies to the `shell` Windows/Git Bash CLI. The MVP searches file names below the current working directory, performs case-insensitive substring matching, opens one result directly with Neovim, and presents a Bubble Tea selector for multiple results.

The tool must not modify, delete, move, rename, or write user files. Launching Neovim is allowed because editing is an explicit user action after the tool has selected a file.

Keep the implementation small and standard-library based. Use Bubble Tea only for terminal interaction. Do not add a framework, command framework, configuration system, file-content index, or concurrency layer unless an approved issue requires it.

Default traversal rules are:

- Search the current working directory recursively.
- Match file names, not directory names, using case-insensitive substring matching.
- Skip `.git`, `node_modules`, and `vendor` directories.
- Do not follow symbolic links in the MVP.
- Do not parse `.gitignore` in the MVP.

## 2. Standard issue/task workflow

Every Issue or Task must follow this sequence:

1. Confirm the task is in scope and identify its acceptance criteria.
2. Read the repository instructions, relevant code, tests, and current Git status.
3. Write down assumptions, affected files, dependencies, and risks before coding.
4. Split unrelated work into separate tasks. Do not mix refactors with feature work.
5. Implement the smallest reviewable change.
6. Add or update tests that prove the acceptance criteria.
7. Run the required checks from this SOP and record actual results.
8. Perform a self-review of the diff, error paths, and platform behavior.
9. Request review through a focused PR.
10. After review, address each comment or document why it is not applicable.
11. Merge only after the Definition of Done is satisfied.

If the task changes public behavior, update `README.md` and `task.md` in the same change when appropriate.

## 3. Analysis before coding

Before editing files, answer these questions in the issue or implementation notes:

- What exact user behavior changes?
- Which package owns the behavior?
- What is explicitly out of scope?
- What happens for empty input, no matches, many matches, inaccessible paths, broken links, and missing `nvim`?
- What is the expected exit status?
- Does the change affect Windows, Git Bash, terminal state, process execution, or paths with spaces?
- What tests can run without a real user directory or real Neovim installation?
- Which existing files are user-modified and must be preserved?

For `filepath.WalkDir`, remember that it does not follow symbolic links in the tree. A traversal error is delivered to the callback; root access failure is fatal, while an inaccessible child directory should normally produce a warning and allow other paths to be searched. Use `filepath.SkipDir` only when intentionally pruning a directory.

## 4. Implementation, testing, and review order

Use this order unless an issue documents an exception:

1. Define or confirm the behavior and interfaces.
2. Implement the smallest production change.
3. Add focused unit tests for pure behavior and edge cases.
4. Add integration tests for filesystem/process boundaries.
5. Run formatting and static checks.
6. Run the full test suite.
7. Perform a manual Git Bash check for terminal and Neovim behavior when applicable.
8. Review the complete diff and verify no unrelated files changed.

For TUI changes, test the model without a real terminal where possible, then manually verify raw mode, alternate screen restoration, arrow keys, Enter, Esc, `q`, and Ctrl-C in Git Bash. Bubble Tea's terminal restoration and external-command behavior must be used as designed; do not bypass it with shell command strings.

## 5. Git, commit, and PR conventions

Use one branch per Issue or Task. Recommended names:

```text
feature/TASK-002-search
fix/TASK-006-editor-exit-status
docs/TASK-001-project-sop
```

Keep commits small and logically complete. Recommended format:

```text
feat(search): add case-insensitive filename traversal
test(search): cover ignored directories and spaced paths
docs: define development SOP
```

Each task should normally produce one focused PR. A PR must include:

- the linked Issue/Task ID;
- behavior changed and explicit non-goals;
- tests actually run and their results;
- platform/manual checks performed;
- known limitations or follow-up work.

Do not rewrite or revert unrelated user changes. Do not commit generated binaries, local logs, or editor state unless explicitly required.

## 6. Errors, logs, and cross-platform behavior

Errors must be actionable and include the relevant path or operation. User-facing messages go to stderr when they describe failure or warnings. Normal search results go to stdout or the TUI.

Do not add a logging framework for the MVP. Use concise `fmt.Fprintf`/`fmt.Errorf` messages. Do not print stack traces or internal implementation details to normal users.

Use `exec.Command("nvim", path)` with the path as a separate argument. Never concatenate a command string or invoke `bash -c`, `cmd /c`, or a shell alias. Check `exec.LookPath` at launch time and preserve Neovim's non-zero exit status when possible.

Use `filepath`, not hard-coded separators. Keep paths as Go strings and pass them directly to `exec.Command`; this preserves spaces and avoids shell quoting bugs. The Git Bash PATH must contain `nvim.exe`; Go cannot resolve a Bash-only alias or function.

The tool must not assume that PowerShell and Git Bash have identical PATH values. Windows + Git Bash is a required manual test environment.

## 7. Required checks before completion

Before marking a task complete:

- [ ] `gofmt` has been run on changed Go files.
- [ ] `go vet ./...` passes, or the documented exception is approved.
- [ ] `go test ./...` passes, including relevant integration tests.
- [ ] No test depends on a real user directory or installed Neovim.
- [ ] Changed behavior has an acceptance test or an explicit manual check.
- [ ] Error paths and non-zero process exits are covered.
- [ ] The diff contains no unrelated source, generated, or user files.
- [ ] Windows path and Git Bash terminal behavior were checked when relevant.
- [ ] Documentation and task checkboxes match the actual implementation.

For a release candidate, also run a Windows build for the supported architecture and verify that the resulting executable can be found through PATH.

## 8. Failure, change, and newly discovered issue handling

If implementation fails:

1. Preserve the failing evidence and exact command output.
2. Determine whether the cause is code, environment, dependency, or an incorrect assumption.
3. Do not weaken tests or silently broaden scope to make the task appear complete.
4. Create a follow-up Issue for unrelated work and return the current task to a clear state.
5. If the task cannot meet its acceptance criteria, mark it blocked and explain the required decision or environment change.

If requirements change, stop implementation, update the task acceptance criteria and dependencies, and obtain confirmation when the change affects scope, dependencies, or user-visible behavior.

If a new bug is found, record reproduction steps, expected behavior, actual behavior, severity, and affected platform. Fix it in the current PR only when it is directly caused by the change or is required for the stated acceptance criteria; otherwise create a separate task.

## 9. Definition of Done

A task is Done only when all applicable items are true:

- [ ] The stated behavior is implemented without unrelated changes.
- [ ] Acceptance criteria are demonstrably satisfied.
- [ ] Unit and integration tests cover the changed behavior and important failure paths.
- [ ] Formatting, vetting, and the full test suite pass.
- [ ] Windows path, process, and terminal behavior are verified when affected.
- [ ] User-facing errors are clear and actionable.
- [ ] No user files are modified by the tool itself.
- [ ] Documentation and task status are current.
- [ ] The self-review found no known unresolved correctness or compatibility issue.
- [ ] The focused PR has review evidence and no unaddressed blocking comments.
