// Package tui provides the interactive result selector for the shell CLI.
package tui

import (
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	defaultWidth  = 80
	defaultHeight = 12
)

// Model is the Bubble Tea model for selecting one path from a list.
//
// Model only manages selection state and rendering. It never starts an
// external process; callers can use Selection after Run returns to decide
// what to do with the selected path.
type Model struct {
	items    []string
	cursor   int
	offset   int
	width    int
	height   int
	done     bool
	selected bool
}

// New creates a selector model. The input slice is copied so later caller
// mutations cannot change the list while the model is running.
func New(paths []string) Model {
	items := append([]string(nil), paths...)
	return Model{
		items:  items,
		width:  defaultWidth,
		height: defaultHeight,
	}
}

// Run starts the selector in the alternate screen and returns the selected
// path, whether the user selected a path, and any Bubble Tea error. An empty
// list is treated as cancellation and does not start a terminal program.
func Run(paths []string) (string, bool, error) {
	if len(paths) == 0 {
		return "", false, nil
	}

	model := New(paths)
	final, err := tea.NewProgram(&model, tea.WithAltScreen()).Run()
	if err != nil {
		return "", false, fmt.Errorf("run result selector: %w", err)
	}

	finished, ok := final.(*Model)
	if !ok || finished == nil {
		return "", false, errors.New("result selector returned an unexpected model")
	}
	path, selected := finished.Selection()
	return path, selected, nil
}

// Init implements tea.Model. The selector has no startup command.
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model and handles navigation, selection, and
// cancellation keys.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.done {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			m.move(-1)
		case tea.KeyDown:
			m.move(1)
		case tea.KeyEnter:
			m.finish(true)
			return m, tea.Quit
		case tea.KeyEsc, tea.KeyCtrlC:
			m.finish(false)
			return m, tea.Quit
		case tea.KeyRunes:
			if len(msg.Runes) == 1 {
				if msg.Runes[0] == 'q' {
					m.finish(false)
					return m, tea.Quit
				}
			}
		}
	case tea.WindowSizeMsg:
		if msg.Width > 0 {
			m.width = msg.Width
		}
		if msg.Height > 0 {
			m.height = msg.Height
		}
		m.ensureVisible()
	}

	return m, nil
}

// View implements tea.Model. Long paths are deliberately rendered in full;
// terminals wrap them naturally, preserving the exact path shown to the user.
func (m *Model) View() string {
	if len(m.items) == 0 {
		return "No results.\nEsc/q to cancel"
	}

	start := m.offset
	if start < 0 {
		start = 0
	}
	if start >= len(m.items) {
		start = len(m.items) - 1
	}

	end := start + m.visibleRows()
	if end > len(m.items) {
		end = len(m.items)
	}

	var view strings.Builder
	view.WriteString("Select a file:\n")
	for i := start; i < end; i++ {
		if i == m.cursor {
			view.WriteString("> ")
		} else {
			view.WriteString("  ")
		}
		view.WriteString(m.items[i])
		view.WriteByte('\n')
	}

	view.WriteString("\n↑/↓ move · Enter select · Esc/q cancel")
	if start > 0 {
		view.WriteString(" · ↑ more")
	}
	if end < len(m.items) {
		view.WriteString(" · ↓ more")
	}
	return view.String()
}

// Selection reports the selected path and whether Enter selected it. A
// cancelled or unfinished model returns an empty path and false.
func (m Model) Selection() (string, bool) {
	if !m.selected || m.cursor < 0 || m.cursor >= len(m.items) {
		return "", false
	}
	return m.items[m.cursor], true
}

// Selected is an alias for Selection for callers that prefer a verb-style
// name.
func (m Model) Selected() (string, bool) {
	return m.Selection()
}

// Cursor returns the zero-based index of the current selection.
func (m Model) Cursor() int {
	return m.cursor
}

// Offset returns the zero-based index of the first visible item.
func (m Model) Offset() int {
	return m.offset
}

// Done reports whether the model has selected or cancelled.
func (m Model) Done() bool {
	return m.done
}

// Items returns a copy of the paths displayed by the model.
func (m Model) Items() []string {
	return append([]string(nil), m.items...)
}

func (m *Model) finish(selected bool) {
	m.done = true
	m.selected = selected && len(m.items) > 0
}

func (m *Model) move(delta int) {
	if len(m.items) == 0 {
		return
	}

	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.items) {
		m.cursor = len(m.items) - 1
	}
	m.ensureVisible()
}

func (m *Model) visibleRows() int {
	rows := m.height - 2 // title and footer
	if rows < 1 {
		return 1
	}
	return rows
}

func (m *Model) ensureVisible() {
	if len(m.items) == 0 {
		m.offset = 0
		return
	}

	rows := m.visibleRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+rows {
		m.offset = m.cursor - rows + 1
	}
	maxOffset := len(m.items) - rows
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.offset > maxOffset {
		m.offset = maxOffset
	}
	if m.offset < 0 {
		m.offset = 0
	}
}
