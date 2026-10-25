# 学习指南：shell 命令行工具

> 这份文档带你从零读懂并掌握这个项目。它不是 README 的复述，而是按「先整体、后局部、再动手」的顺序，把每个包的设计意图、关键代码、以及背后的 Go 工程思想讲清楚。
>
> 建议阅读顺序：跟着章节编号走，遇到「动手练习」就停下来敲一遍，遇到「代码定位」就在编辑器里打开对应文件对照着看。

---

## 目录

1. [这个项目是做什么的](#1-这个项目是做什么的)
2. [一句话架构](#2-一句话架构)
3. [项目结构总览](#3-项目结构总览)
4. [数据是怎么流动的（端到端）](#4-数据是怎么流动的端到端)
5. [逐包精读](#5-逐包精读)
   - 5.1 [入口：`cmd/shell/main.go`](#51-入口-cmdshellmaingo)
   - 5.2 [搜索：`internal/search`](#52-搜索-internalsearch)
   - 5.3 [编辑器适配器：`internal/editor`](#53-编辑器适配器-internaleditor)
   - 5.4 [选择器：`internal/tui`](#54-选择器-internaltui)
6. [贯穿全项目的 6 个设计思想](#6-贯穿全项目的-6-个设计思想)
7. [测试是怎么写的](#7-测试是怎么写的)
8. [构建、运行、验证](#8-构建运行验证)
9. [项目约定：SOP 与 AGENTS](#9-项目约定sop-与-agents)
10. [动手练习（由浅入深）](#10-动手练习由浅入深)
11. [延伸阅读](#11-延伸阅读)

---

## 1. 这个项目是做什么的

`shell` 是一个在 Windows + Git Bash 环境下运行的命令行工具。它的全部工作就是：

1. 接收一个**文件名关键字**作为参数；
2. 从**当前目录**开始递归搜索，找出文件名包含该关键字的文件（大小写不敏感、子串匹配）；
3. 如果只找到 1 个，直接用 Neovim 打开它；
4. 如果找到多个，弹出一个**可上下选择**的列表（终端交互界面），你选一个用 Neovim 打开。

它**刻意不做什么**：不读文件内容、不解析 `.gitignore`、不跟随符号链接、不修改/移动/删除任何用户文件。工具本身只负责「找文件」和「把文件交给 Neovim」，真正的编辑动作发生在用户显式打开之后。

**为什么值得学它**：这是一个教科书级的「小而完整」的 Go CLI 项目——只有一个外部依赖（Bubble Tea），其余全靠标准库，代码量小，但把依赖注入、接口抽象、错误分层、跨平台处理、可测试性这几个 Go 工程的核心概念都覆盖到了。

---

## 2. 一句话架构

```
命令行参数 → main.go（编排层）→ search（搜索）→ tui（多选）→ editor（打开 Neovim）
```

四个包各司其职，通过**接口**和**函数类型**解耦。`main.go` 不关心搜索怎么实现，只关心「给我一个能搜的函数」；`editor` 不关心谁调用了它，只关心「给我一个能跑进程的 Runner」。

整个工具唯一的第三方依赖是 [Bubble Tea](https://github.com/charmbracelet/bubbletea)（`go.mod` 第 5 行），一个基于 Elm 架构的终端 UI 库，只用在 `internal/tui` 里。

---

## 3. 项目结构总览

```text
go-shell-tool/
├── cmd/
│   └── shell/                 # 可执行入口（main 包）
│       ├── main.go            #   编排：参数解析 + 调用 search/tui/editor
│       ├── main_test.go       #   单元测试（注入假依赖）
│       └── integration_test.go#   集成测试（真 search + 假 nvim）
├── internal/                  # 内部包，只能被本项目 import
│   ├── search/                # 文件搜索
│   │   ├── search.go
│   │   ├── doc.go             #   包级注释
│   │   └── search_test.go
│   ├── editor/                # Neovim 进程适配器
│   │   ├── editor.go
│   │   ├── doc.go
│   │   └── editor_test.go
│   └── tui/                   # 多选列表（Bubble Tea）
│       ├── model.go
│       ├── doc.go
│       └── model_test.go
├── docs/
│   └── TASK-006-VERIFICATION.md  # 手动验证记录（真实命令 + 结果）
├── AGENTS.md                  # 给 AI/协作者的项目约定
├── SOP.md                     # 开发规范（Standard Operating Procedure）
├── task.md                    # 任务拆解与验收标准
├── go.mod / go.sum
└── README.md
```

**两个约定值得注意**：

- `cmd/` + `internal/` 是 Go 社区的标准布局：`cmd/` 放可执行程序入口，`internal/` 放不希望被外部项目 import 的内部代码。
- 每个包都有一个 `doc.go`，只写包级注释。这是为了满足 Go 的文档约定（`go doc` 会读取它），也表明这个项目很在意「包的责任边界清晰」。

---

## 4. 数据是怎么流动的（端到端）

这是理解全项目最关键的一张图。先看整体流程，再逐包看实现，你会豁然开朗。

```
用户输入: shell NOTES
                    │
                    ▼
┌─────────────────────────────────────────────────────────────┐
│ main.go 的 run(args, deps)                                  │
│                                                             │
│  1. 校验参数：必须恰好 1 个非空参数，否则 exit 2             │
│  2. os.Getwd() 得到当前目录 root                            │
│  3. search.Search(root, query)                              │
│       ├─ 返回 paths（匹配文件，已排序）                      │
│       ├─ 返回 warnings（遍历中的非致命错误，如无权访问目录）  │
│       └─ 返回 err（root 本身打不开，致命）                   │
│  4. 打印 warnings 到 stderr（不影响继续）                    │
│  5. 分支：                                                  │
│       ├─ 0 个匹配 → 打印 "no matches" → exit 1             │
│       ├─ 1 个匹配 → 直接打开                                │
│       └─ N 个匹配 → tui.Run(paths) 弹出选择器               │
│            ├─ 用户 Esc/q/Ctrl-C → 取消 → exit 0            │
│            └─ 用户 Enter → 得到选中的 path                  │
│  6. editor.Open(path) → 启动 nvim 打开该文件                 │
│       ├─ 成功 → exit 0                                      │
│       └─ 失败（找不到 nvim / nvim 非零退出）→ exit 1         │
└─────────────────────────────────────────────────────────────┘
```

**退出码约定**（定义在 `main.go:14-18`，README 里也有表）：

| 退出码 | 含义 |
| ---: | --- |
| `0` | 成功打开，或用户主动取消 |
| `1` | 无匹配 / 搜索失败 / 选择器失败 / Neovim 失败 |
| `2` | 用法错误（缺参数、空参数、多参数） |

一个关键点：**用户按 Esc 取消是「正常结束」（exit 0），不是错误**。这一点在 `main.go:77-79` 和测试 `TestRunCancellationDoesNotOpenEditor` 里都有体现。

---

## 5. 逐包精读

### 5.1 入口：`cmd/shell/main.go`

这是编排层，也是最容易读懂的地方。核心是 `run` 函数（`main.go:42`）。真正值得学的是它上面的**依赖注入**设计：

```go
type cliDeps struct {
    getwd        func() (string, error)
    search       func(root, query string) ([]string, []search.Warning, error)
    selectResult func(paths []string) (path string, selected bool, err error)
    open         func(path string) error
    stderr       io.Writer
}
```

**为什么这么设计？** `run` 里要用到的四个「外部能力」（获取当前目录、搜索、弹选择器、打开编辑器）全部被抽象成了**函数字段**。生产环境用真实的实现（`defaultDeps`，`main.go:32`），测试环境换成假实现（见 `main_test.go` 的 `testDeps`）。

这样 `run` 就是一个**纯函数**——给它参数和依赖，返回退出码，不依赖真实文件系统、真实终端、真实 Neovim。这是整个项目「可测试性」的基石。

**`withDefaults`（`main.go:89`）**：这是一个小细节，它保证即使调用者只传了部分依赖，剩下的也会被填上默认实现。这是 Go 里常见的「可选依赖」模式——用结构体 + nil 字段，而不是构造器里的 5 个参数。

**错误信息的层次**：注意 `run` 里所有错误输出都带前缀，让人一眼看出错误发生在哪一层：

```go
fmt.Fprintf(deps.stderr, "search error: ...")   // main.go:61
fmt.Fprintf(deps.stderr, "selector error: ...") // main.go:74
fmt.Fprintf(deps.stderr, "editor error: ...")   // main.go:83
```

而**警告**（非致命）单独用 `warning:` 前缀（`main.go:58`），且先于结果打印。这就是 SOP 里要求的「错误要可操作、分层清晰」。

> 📌 **代码定位练习**：打开 `main.go`，只看 `run` 函数，用一句话向别人解释每一步在干什么，再对照上面那张流程图。

---

### 5.2 搜索：`internal/search`

**职责**：遍历文件系统，找文件名匹配的文件。是纯逻辑包，没有任何外部依赖。

**核心 API**（`search.go:47`）：

```go
func Search(root, query string) (paths []string, warnings []Warning, err error)
```

注意它返回**三个值**，这是这个项目的关键设计：

- `paths`：匹配的文件路径，**已排序**（保证多次运行结果稳定，对应任务里的「deterministic」要求）。
- `warnings`：遍历中遇到的非致命错误（比如某个子目录没权限读），单独收集，不打断搜索。
- `err`：只有 **root 本身打不开**才返回的致命错误。

**遍历用的是 `filepath.WalkDir`**（`search.go:50`），这是 Go 1.16+ 推荐的做法（比老的 `filepath.Walk` 更高效，因为它只传 `DirEntry` 而不读完整 `FileInfo`）。

**回调函数里的三个关键分支**（`search.go:50-70`）：

1. **`walkErr != nil`**：说明访问某个路径出错。这时要判断是不是 root 本身——是则作为致命错误返回，否则作为 warning 收集后 `return nil` 继续。
   ```go
   if filepath.Clean(path) == filepath.Clean(root) {
       return fmt.Errorf("walk root %q: %w", root, walkErr)
   }
   warnings = append(warnings, Warning{Path: path, Err: walkErr})
   ```
   用 `filepath.Clean` 比较是为了规范化路径（去掉多余的分隔符等），避免因为路径写法不同导致误判。

2. **`entry.IsDir()`**：是目录。如果是被忽略的目录（`.git` / `node_modules` / `vendor`），返回 `filepath.SkipDir` 剪掉整棵子树。
   ```go
   if isIgnoredDirectory(entry.Name()) {
       return filepath.SkipDir
   }
   ```

3. **是文件**：判断文件名（`entry.Name()`，只取文件名不含目录部分）是否包含 query。
   ```go
   if strings.Contains(strings.ToLower(entry.Name()), needle) {
       paths = append(paths, path)
   }
   ```

**几个值得学的细节**：

- **大小写不敏感**的实现：`strings.ToLower` 在**两边都做**（`search.go:48` 的 `needle` 和 `search.go:66` 的 `entry.Name()`）。为什么不直接 `strings.Contains(strings.ToLower(name), strings.ToLower(query))` 每次都转小写 query？——因为 query 不变，提前转一次（`needle`）省得每个文件都重复转换，这是一个微小的性能习惯。
- **忽略目录判断用 `strings.EqualFold`**（`search.go:81`）：也是大小写不敏感，所以 `NODE_MODULES` 也会被忽略（测试里专门验证了这一点）。
- **`Warning` 是一个实现了 `error` 接口的结构体**（`search.go:15`），还实现了 `Unwrap()`（`search.go:29`）。这意味着调用方可以既把 warning 当字符串打印，又能用 `errors.Is` / `errors.As` 拿到底层的原始错误。这是一个「既满足 error 接口，又保留结构化信息」的漂亮做法。
- **符号链接不跟随**：这是 `WalkDir` 的默认行为，代码里没写任何特殊处理，靠的就是标准库的默认语义（`search.go:44` 的注释明确说明了这一点）。SOP 里也强调「`filepath.WalkDir` does not follow symbolic links」。

> 📌 **代码定位练习**：在 `search.go` 里找到 `sortWarnings`，想想为什么 warning 也要排序（提示：答案在「确定性」这个目标里）。

---

### 5.3 编辑器适配器：`internal/editor`

**职责**：安全地用 Neovim 打开一个路径，并报告它的退出状态。这是「把进程边界抽象掉」的典范。

**核心抽象：`Runner` 接口**（`editor.go:15`）：

```go
type Runner interface {
    LookPath(file string) (string, error)
    Run(executable string, args []string, stdin io.Reader, stdout, stderr io.Writer) error
}
```

这个接口把「查 PATH」和「跑进程」两件事抽象出来。真实实现 `systemRunner`（`editor.go:89`）底层就是 `exec.LookPath` 和 `exec.Command`。测试时用一个 `fakeRunner`（`editor_test.go:12`）记录调用参数、返回假错误——**不需要真的装 Neovim**。

**打开文件的流程（`editor.go:50` 的 `Open`）**：

1. 防御性检查：`e == nil`、`path == ""`。
2. 兜底：如果 runner 或 IO 流是 nil，补上默认值（`editor.go:58-71`）。
3. `LookPath("nvim")` → 找不到就返回带 PATH 提示的错误（`editor.go:75`）。
4. `Run(executable, []string{path}, ...)` → 注意 `path` 是**作为一个单独的参数**传进去的（`editor.go:78`）。

**为什么第 4 步如此重要**：如果写成字符串拼接（`"nvim " + path`）再用 shell 执行，路径里的空格或特殊字符就会出问题（`C:\work dir\notes.md` 会被 shell 拆成两个参数）。用 `exec.Command` 传参数切片，Go 会直接把这些参数原样传给进程，**不经过 shell**，空格就安全了。这是 SOP 里反复强调的「Never concatenate a command string」。

**退出码的保留**（`editor.go:79-84`）——这段很值得细品：

```go
if err := runner.Run(...); err != nil {
    var exitCoder interface{ ExitCode() int }
    if errors.As(err, &exitCoder) {
        return fmt.Errorf("editor: nvim exited with status %d while opening %q: %w",
            exitCoder.ExitCode(), path, err)
    }
    return fmt.Errorf("editor: failed to open %q with nvim: %w", path, err)
}
```

它用 `errors.As` 判断底层错误是否暴露了 `ExitCode()` 方法（`*exec.ExitError` 就有）。如果有，就把**具体的退出码**塞进错误消息；如果没有，就退回到一个通用消息。这样「nvim 启动失败」和「nvim 启动成功但退出码 23」能被区分开来，用户能看懂发生了什么。

> 📌 **代码定位练习**：对比 `editor_test.go` 里的 `fakeExitError`（第 38 行）和真实场景，理解 `errors.As` 是怎么「探测」接口的。

---

### 5.4 选择器：`internal/tui`

**职责**：多个结果时，提供一个用 ↑/↓ 选择、Enter 确认、Esc/q/Ctrl-C 取消的交互列表。这是唯一用 Bubble Tea 的地方。

**Bubble Tea 的 Elm 架构**：Bubble Tea 要求每个界面是一个 `Model`，实现三个方法：

- `Init() tea.Cmd`：启动时做什么（这里什么都不做，返回 nil，`model.go:66`）。
- `Update(msg) (tea.Model, tea.Cmd)`：收到事件（按键、窗口尺寸变化）后如何更新状态。
- `View() string`：如何把当前状态渲染成字符串。

这个项目的 `Model`（`model.go:22`）只管理几样状态：

```go
type Model struct {
    items    []string  // 待选的路径列表
    cursor   int       // 当前选中项的索引
    offset   int       // 滚动偏移（第一可见项的索引）
    width    int       // 终端宽度
    height   int       // 终端高度
    done     bool      // 是否已结束（选中或取消）
    selected bool      // 是否「选中」了某项（而非取消）
}
```

**一个关键的架构原则**：`Model` **只管理选择状态，绝不启动外部进程**（`model.go:18-21` 的注释明确写了这一点）。它把「用户选了哪个」通过 `Selection()` 方法暴露出去，至于选完之后要不要开 Neovim，是 `main.go` 的事。

**为什么这样拆？** 因为 Bubble Tea 的 `Run` 会接管终端、进入 alternate screen。如果在这里面直接启动 Neovim，Neovim 会和 Bubble Tea 抢终端控制权，导致乱屏。所以流程是：

```
tui.Run(paths)  →  用户选择  →  Bubble Tea 退出、终端恢复  →  main.go 再调 editor.Open
```

「选择」和「打开」严格分离，SOP 里有一条专门讲这个（「No Neovim process is started from inside the TUI model」）。

**滚动逻辑**（`ensureVisible`，`model.go:216`）是这段代码里最容易出 bug 的部分，值得单独看：

- `visibleRows()`（`model.go:208`）算出可见行数 = `height - 2`（减掉标题和底栏两行）。
- `ensureVisible` 保证 `cursor` 始终在 `[offset, offset+rows)` 这个可见窗口内，同时让 `offset` 不超过 `len(items) - rows`（这样不会滚过头）。

测试 `TestNavigationClampsAndScrolls`（`model_test.go:19`）就是专门验证这个边界行为的：把窗口设成只有 3 行，然后狂按 ↓ 键按到列表底部，断言 cursor 停在最后一项、offset 停在正确位置。

**按键处理**（`model.go:72-109`）里有个容易忽略的细节：`q` 不是用 `KeyMsg.Type` 判断的（没有 `KeyQ` 这种类型），而是通过 `tea.KeyRunes` 拿到实际字符再判断是不是 `'q'`：

```go
case tea.KeyRunes:
    if len(msg.Runes) == 1 && msg.Runes[0] == 'q' {
        m.finish(false)
        return m, tea.Quit
    }
```

而 Esc 和 Ctrl-C 是 Bubble Tea 定义的专用类型（`tea.KeyEsc`、`tea.KeyCtrlC`）。Enter 也同理（`tea.KeyEnter`）。

**返回 `tea.Quit`** 这个命令是通知 Bubble Tea 结束程序。结束后 `Run` 从 `final.(*Model)` 取出最终状态，用 `Selection()` 读结果（`model.go:57-62`）。

> 📌 **代码定位练习**：看 `View()`（`model.go:113`），理解 `start`/`end` 两个变量是如何用 `offset` 和 `visibleRows()` 决定「渲染列表的哪一段」的。

---

## 6. 贯穿全项目的 6 个设计思想

把这 6 点吃透，你就掌握了这个项目（以及大多数优秀 Go 项目）的精髓：

### 6.1 依赖注入，让代码可测试

`cliDeps` 和 `Runner` 接口是同一个思想的两种形式：**把「会触碰外部世界的操作」抽象成可替换的接口或函数**。生产代码注入真实实现，测试代码注入假实现。这样核心逻辑（`run`、`Open`）就变成了不依赖真实环境的纯逻辑。

### 6.2 小接口优先

`Runner` 只有两个方法，`cliDeps` 只是一堆函数字段。Go 的惯例是「接口越小越好」，调用方需要什么就定义什么，而不是让实现方去实现一个巨大的接口。`editor` 不 import `search`，`search` 也不知道 `editor` 的存在——它们只通过 `main.go` 里的函数签名「间接」协作。

### 6.3 错误分层

- **致命错误**（root 打不开、找不到 nvim）→ 返回 error，最终 exit 1。
- **非致命警告**（子目录无权限）→ 收集成 `Warning` 切片，打印后继续。
- **用法错误**（参数不对）→ 单独的 exit 2。

区分「严重程度」并分通道处理，是这个项目错误设计的核心。同时所有错误都用 `%w` 包装（保留错误链），让上层能用 `errors.Is`/`errors.As` 探查。

### 6.4 确定性

「同样的输入产生同样的输出」贯穿始终：`search.go:75` 对结果 `sort.Strings`，`sortWarnings` 对警告排序。这让测试可以断言精确结果，也让用户看到稳定有序的列表。任务文档里反复出现「deterministic」「stable across repeated runs」。

### 6.5 跨平台意识（Windows + Git Bash）

这个项目跑在 Windows 的 Git Bash 上，于是有几个刻意为之的细节：

- 用 `filepath`（而不是手动拼接 `/` 或 `\`）处理路径。
- 用 `exec.LookPath` 找 `nvim.exe`——**Bash 的 alias 和函数对 Go 进程是不可见的**，只有真正在 PATH 里的可执行文件才行（README 和 SOP 都强调了这一点）。
- 路径作为独立参数传给 `exec.Command`，避免 shell 引号问题。
- PowerShell 和 Git Bash 的 PATH 可能不同，所以「在哪里运行 `shell`，就要在哪里确认 `nvim` 可用」。

### 6.6 最小依赖

整个项目只引了 Bubble Tea 一个库（外加它的传递依赖）。SOP 明确禁止「加框架、命令框架、配置系统、文件内容索引、并发层」——除非有经过批准的需求。这是一种「克制」的工程态度：MVP 阶段，标准库能做的就用标准库。

---

## 7. 测试是怎么写的

这个项目的测试是「如何在不依赖真实环境的情况下测试 CLI」的绝佳教材。打开测试文件对照看：

### 7.1 表格驱动测试（table-driven tests）

`main_test.go:26` 的 `TestRunRejectsInvalidArguments` 把多个用例放在一个 `tests` 切片里循环跑，每个用例用 `t.Run` 起子测试。这是 Go 社区最主流的测试风格。

### 7.2 注入假依赖

`main_test.go:12` 的 `testDeps` 返回一个全假实现的 `cliDeps`，`getwd` 返回固定路径 `C:\work dir`（故意带空格，顺便测路径处理）。然后每个测试按需覆盖其中的函数，用闭包捕获「是否被调用」「收到了什么参数」。

例如 `TestRunOpensSingleMatchWithSpaces`（`main_test.go:74`）里：

```go
var openedPath string
deps.open = func(path string) error {
    openedPath = path
    return nil
}
```

用闭包变量 `openedPath` 记录「编辑器打开的是哪个路径」，测试结束时断言它等于期望的带空格路径。

### 7.3 用 `t.TempDir()` 造真实但隔离的文件树

搜索测试（`search_test.go`）需要真实的文件系统行为（`WalkDir` 真的去遍历），但又不能碰用户真实目录。解法是 `t.TempDir()`——Go 测试框架创建一个临时目录，测试结束自动清理。

`writeTestFile`（`search_test.go:172`）是个 `t.Helper()` 辅助函数，负责 `MkdirAll` + `WriteFile` 造文件。

### 7.4 平台相关的跳过（skip）

`search_test.go:138` 的 `TestSearchContinuesAfterInaccessibleChild` 在 Windows 上直接 `t.Skip`，因为「改目录权限」在 Windows 上不可移植。符号链接测试（`search_test.go:116`）则是在 `os.Symlink` 失败时 `t.Skipf`。这是诚实的做法：**测不了就明确跳过，而不是假装测了**（SOP 也要求这一点）。

### 7.5 集成测试：真 search + 假 nvim

`integration_test.go` 比单元测试更「真」：它用**真实的** `search.Search` 遍历真实的临时目录树，但 Neovim 仍然是假的（`integrationRunner`，`integration_test.go:17`）。它验证的是「搜索 → 选择 → 打开」这条完整链路，以及带空格路径从搜索到编辑器参数是否一路无损。

`integrationExitError`（`integration_test.go:46`）实现了一个带 `ExitCode()` 的假错误，用来模拟「nvim 退出码 23」的场景。

> 📌 **动手练习**：跑一遍 `go test ./... -v`，看每个测试的名字和输出，对照上面的讲解找到对应文件。

---

## 8. 构建、运行、验证

### 8.1 构建

```sh
go build -o shell.exe ./cmd/shell
```

### 8.2 放到 PATH（Git Bash）

```sh
export PATH="/e/Desktop/go-shell-tool:$PATH"
command -v shell.exe   # 应该能看到路径
```

要持久化就把它加到 `~/.bashrc`。

### 8.3 确认 nvim 可用

```sh
command -v nvim
nvim --version
```

记住：`shell` 用的是 `exec.LookPath`，所以 `nvim.exe` 必须真的在 PATH 里，Bash 的 alias 不行。

### 8.4 使用

```sh
shell <关键字>
# 例如
shell notes
```

- 0 个结果 → 打印 `no matches for "notes"`，exit 1
- 1 个结果 → 直接开 Neovim
- 多个结果 → 弹列表，↑/↓ 移动，Enter 打开，Esc/q/Ctrl-C 取消

### 8.5 项目自带的验证命令（来自 AGENTS.md，提交前必跑）

```sh
gofmt -l . && echo "gofmt 通过"
go vet ./... && echo "go vet 通过"
go test ./... && echo "go test 通过"
```

`docs/TASK-006-VERIFICATION.md` 里还记录了更完整的验证：`go test -race`、`go mod verify`、真实 Git Bash + PTY 的手动测试（包括真正的 nvim v0.12.3）。这份文档是「如何诚实记录验证结果」的范例——每条都有真实命令和真实输出，连测试用的假 nvim 包装脚本都讲清楚了。

---

## 9. 项目约定：SOP 与 AGENTS

这个项目除了代码，还有一套清晰的工作流约定，值得一读：

- **`SOP.md`**：开发规范。核心是「标准任务工作流」（Issue → 分析 → 最小实现 → 测试 → 自审 → PR → 评审 → 合并），以及「Definition of Done」清单（第 9 节）。它还规定了一些技术红线：不跟随符号链接、不解析 `.gitignore`、不构建命令字符串、不碰用户文件等。
- **`task.md`**：任务拆解。TASK-001 到 TASK-007 是 MVP，TASK-008/009 是可选的后续增强。每个任务都有「目标 / 具体工作 / 交付物 / 验收标准 / 测试要求」。这张「依赖图」（`task.md:8`）展示了任务是如何被拆分和排定顺序的——搜索、编辑器、TUI 三者可并行，集成在它们之后。
- **`AGENTS.md`**：给 AI/协作者的约定。核心有两条：① 不要添加任何 AI/Claude 署名（提交里不加 Co-Authored-By）；② 提交前必须跑通 gofmt/vet/test 三件套。

**为什么要读这些**：它们展示了「一个规范的 Go 项目不只靠代码，还靠清晰的流程和边界」。你学到的不只是「怎么写这个工具」，还有「怎么有条理地推进一个小项目」。

---

## 10. 动手练习（由浅入深）

按顺序做，做完一个再下一个：

1. **读懂流程**：不看代码，凭第 4 节的流程图，向自己复述一遍「输入 `shell notes` 后发生了什么」。

2. **跑起来**：构建、放到 PATH、在一个有 `.md` 文件的目录里运行，体验 0/1/N 个结果三种情况（N 个结果时试试 ↑↓ Enter Esc q Ctrl-C）。

3. **改一个字符串**：把 `main.go:45` 的 usage 提示改成你自己的话，重新构建，验证 `shell`（无参数）时输出变了。感受「改 → 构建 → 验证」的循环。

4. **加一个忽略目录**：在 `search.go:33` 的 `ignoredDirectories` 里加一个 `"target"`，然后造一个 `target/` 目录放个文件，确认搜索会跳过它。再想想：为什么用 `map[string]struct{}` 而不是 `[]string`？（提示：查 `strings.EqualFold` 的循环成本，或者想想「集合」语义。）

5. **写一个测试**：在 `search_test.go` 里仿照现有测试，写一个「搜索空 query」或「搜索带中文名字的文件」的测试，跑 `go test ./internal/search/ -v`。

6. **改退出码逻辑**：现在「用户取消」返回 exit 0。如果业务上要求「取消返回一个特殊码」，改 `main.go` 和对应测试，跑 `go test ./...` 确认全绿。（这题练的是「改了行为，测试也要跟着改」的闭环。）

7. **读边界代码**：给 `model.go` 的 `ensureVisible` 写注释，逐行解释它如何保证 `cursor` 始终可见。能讲清楚，说明滚动逻辑你真的懂了。

8. **进阶（可选）**：实现 `task.md` 里的 TASK-008「可配置忽略目录」（加一个 `--ignore-dir` 参数）。这会逼你同时动 `main.go`（参数解析）、`search.go`（接受忽略列表）、测试。做完你会对整个项目的扩展点有完整认识。

---

## 11. 延伸阅读

学完这个项目，如果想再深入，往这几个方向看：

- **[Go 标准库 `filepath.WalkDir`](https://pkg.go.dev/path/filepath#WalkDir)**：理解为什么它比 `filepath.Walk` 好，以及 `SkipDir` 的语义。
- **[Go `os/exec`](https://pkg.go.dev/os/exec)**：重点看「不用 shell」执行命令、`*exec.ExitError` 的 `ExitCode()`，以及 `LookPath` 的局限。
- **[Bubble Tea 教程](https://github.com/charmbracelet/bubbletea/tree/master/tutorials)**：从「hello world」到更复杂的多组件 UI，理解 Elm 架构在 Go 里的落地。
- **《Effective Go》** 和 **《Practical Go》** 中关于接口设计、错误处理、测试的章节——这个项目的很多做法都能在里面找到理论依据。
- **`errors.Is` / `errors.As` 与 `%w` 包装**：这是现代 Go 错误处理的核心，`editor.go` 和 `search.go` 都在用。

---

## 附：关键文件速查

| 想了解什么 | 看哪里 |
| --- | --- |
| 整体流程 & 退出码 | `cmd/shell/main.go` |
| 依赖注入怎么做的 | `cmd/shell/main.go:20-40`（`cliDeps`、`defaultDeps`） |
| 文件搜索 & 忽略目录 | `internal/search/search.go` |
| 错误/警告怎么分层 | `internal/search/search.go:50-77`（`Search`） |
| 安全地启动 nvim | `internal/editor/editor.go:50-87`（`Open`） |
| 进程抽象（Runner） | `internal/editor/editor.go:15-26` |
| 选择器状态 & 按键 | `internal/tui/model.go` |
| 滚动逻辑（易错点） | `internal/tui/model.go:208-239` |
| 单元测试风格 | `cmd/shell/main_test.go` |
| 集成测试（真搜索 + 假 nvim） | `cmd/shell/integration_test.go` |
| 手动验证记录 | `docs/TASK-006-VERIFICATION.md` |
| 开发规范 | `SOP.md` |
| 任务拆解 | `task.md` |

---

*这份文档的目的是让你「带着问题读代码」，而不是「读完所有代码再找问题」。建议先按第 10 节的练习做前 4 个，再回来看第 5 节的精读，会有事半功倍的效果。*
