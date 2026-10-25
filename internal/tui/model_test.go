package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func applyKey(t *testing.T, model *Model, msg tea.KeyMsg) tea.Cmd {
	t.Helper()
	updated, cmd := model.Update(msg)
	if updated != model {
		t.Fatalf("Update returned a different model: %T", updated)
	}
	return cmd
}

func TestNavigationClampsAndScrolls(t *testing.T) {
	paths := make([]string, 10)
	for i := range paths {
		paths[i] = string(rune('a' + i))
	}

	model := New(paths)
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 5}) // three rows

	applyKey(t, &model, tea.KeyMsg{Type: tea.KeyUp})
	if got := model.Cursor(); got != 0 {
		t.Fatalf("cursor moved above first item: %d", got)
	}

	for i := 0; i < len(paths)+3; i++ {
		applyKey(t, &model, tea.KeyMsg{Type: tea.KeyDown})
	}
	if got := model.Cursor(); got != len(paths)-1 {
		t.Fatalf("cursor = %d, want %d", got, len(paths)-1)
	}
	if got, want := model.Offset(), len(paths)-3; got != want {
		t.Fatalf("offset = %d, want %d", got, want)
	}

	for i := 0; i < len(paths)+3; i++ {
		applyKey(t, &model, tea.KeyMsg{Type: tea.KeyUp})
	}
	if got := model.Cursor(); got != 0 {
		t.Fatalf("cursor = %d after moving up, want 0", got)
	}
	if got := model.Offset(); got != 0 {
		t.Fatalf("offset = %d after moving up, want 0", got)
	}
}

func TestEnterReturnsSelectedPath(t *testing.T) {
	model := New([]string{"first", "path with spaces", "last"})
	applyKey(t, &model, tea.KeyMsg{Type: tea.KeyDown})
	cmd := applyKey(t, &model, tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Fatal("Enter did not request program quit")
	}
	if !model.Done() {
		t.Fatal("model is not done after Enter")
	}
	if got, ok := model.Selection(); !ok || got != "path with spaces" {
		t.Fatalf("Selection() = %q, %v; want %q, true", got, ok, "path with spaces")
	}
}

func TestCancellationKeysReturnNoSelection(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyMsg
	}{
		{name: "escape", key: tea.KeyMsg{Type: tea.KeyEsc}},
		{name: "q", key: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}},
		{name: "ctrl-c", key: tea.KeyMsg{Type: tea.KeyCtrlC}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := New([]string{"one", "two"})
			if cmd := applyKey(t, &model, test.key); cmd == nil {
				t.Fatal("cancellation did not request program quit")
			}
			if !model.Done() {
				t.Fatal("model is not done after cancellation")
			}
			if got, ok := model.Selection(); ok || got != "" {
				t.Fatalf("Selection() = %q, %v; want empty, false", got, ok)
			}
		})
	}
}

func TestViewRendersLongPathsAndScrollHints(t *testing.T) {
	longPath := strings.Repeat("directory/", 20) + "file with spaces.txt"
	model := New([]string{"first", longPath, "third", "fourth"})
	model.Update(tea.WindowSizeMsg{Width: 20, Height: 5})

	view := model.View()
	if !strings.Contains(view, "first") {
		t.Fatalf("view does not contain first item:\n%s", view)
	}
	if !strings.Contains(view, longPath) {
		t.Fatalf("view does not preserve long path:\n%s", view)
	}
	if !strings.Contains(view, "↓ more") {
		t.Fatalf("view does not show a downward scroll hint:\n%s", view)
	}

	applyKey(t, &model, tea.KeyMsg{Type: tea.KeyDown})
	if !strings.Contains(model.View(), "> "+longPath) {
		t.Fatalf("selected long path is not marked:\n%s", model.View())
	}
}

func TestNewCopiesItems(t *testing.T) {
	paths := []string{"original"}
	model := New(paths)
	paths[0] = "changed"
	if got := model.Items()[0]; got != "original" {
		t.Fatalf("model retained caller mutation: %q", got)
	}

	items := model.Items()
	items[0] = "changed again"
	if got := model.Items()[0]; got != "original" {
		t.Fatalf("Items returned model-owned slice: %q", got)
	}
}

func TestEmptyModelCancelsWithoutStartingProgram(t *testing.T) {
	model := New(nil)
	applyKey(t, &model, tea.KeyMsg{Type: tea.KeyEnter})
	if got, ok := model.Selection(); ok || got != "" {
		t.Fatalf("empty model Selection() = %q, %v; want empty, false", got, ok)
	}
	if got := model.View(); !strings.Contains(got, "No results") {
		t.Fatalf("empty model view = %q", got)
	}

	if path, ok, err := Run(nil); err != nil || ok || path != "" {
		t.Fatalf("Run(nil) = %q, %v, %v; want empty, false, nil", path, ok, err)
	}
}
