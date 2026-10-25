// Command shell is the entry point for the standalone shell CLI.
package main

import (
	"fmt"
	"io"
	"os"

	"shell/internal/editor"
	"shell/internal/search"
	"shell/internal/tui"
)

const (
	exitSuccess = 0
	exitError   = 1
	exitUsage   = 2
)

type cliDeps struct {
	getwd        func() (string, error)
	search       func(root, query string) ([]string, []search.Warning, error)
	selectResult func(paths []string) (path string, selected bool, err error)
	open         func(path string) error
	stderr       io.Writer
}

func main() {
	//1. `os.Args[1:]`：去掉程序名本身，只拿到用户传入命令行参数
	//2. `defaultDeps()`：组装一套**真实环境依赖实现**
	os.Exit(run(os.Args[1:], defaultDeps()))
}

func defaultDeps() cliDeps {
	return cliDeps{
		getwd:        os.Getwd,
		search:       search.Search,
		selectResult: tui.Run,
		open:         editor.Open,
		stderr:       os.Stderr,
	}
}

func run(args []string, deps cliDeps) int {
	deps = withDefaults(deps)
	if len(args) != 1 || args[0] == "" {
		fmt.Fprintln(deps.stderr, "usage: shell <non-empty query>")
		return exitUsage
	}
	query := args[0]

	root, err := deps.getwd()
	if err != nil {
		fmt.Fprintf(deps.stderr, "search error: cannot determine current directory: %v\n", err)
		return exitError
	}
	//调用搜索
	paths, warnings, err := deps.search(root, query)
	for _, warning := range warnings {
		fmt.Fprintf(deps.stderr, "warning: %s\n", warning.Error())
	}
	if err != nil {
		fmt.Fprintf(deps.stderr, "search error: %v\n", err)
		return exitError
	}
	if len(paths) == 0 {
		fmt.Fprintf(deps.stderr, "no matches for %q\n", query)
		return exitError
	}

	selectedPath := paths[0]
	if len(paths) > 1 {
		var selected bool
		// 多条结果，唤起TUI选择器
		selectedPath, selected, err = deps.selectResult(paths)
		if err != nil {
			// TUI本身报错（终端不支持、IO异常）→ error
			fmt.Fprintf(deps.stderr, "selector error: %v\n", err)
			return exitError
		}
		if !selected {
			// TUI本身报错（终端不支持、IO异常）→ error
			return exitSuccess
		}
	}

	if err := deps.open(selectedPath); err != nil {
		fmt.Fprintf(deps.stderr, "editor error: %v\n", err)
		return exitError
	}
	return exitSuccess
}

func withDefaults(deps cliDeps) cliDeps {
	defaults := defaultDeps()
	if deps.getwd == nil {
		deps.getwd = defaults.getwd
	}
	if deps.search == nil {
		deps.search = defaults.search
	}
	if deps.selectResult == nil {
		deps.selectResult = defaults.selectResult
	}
	if deps.open == nil {
		deps.open = defaults.open
	}
	if deps.stderr == nil {
		deps.stderr = defaults.stderr
	}
	return deps
}
