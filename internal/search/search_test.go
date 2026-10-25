package search

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestSearchMatchesNamesCaseInsensitivelyAndSorts(t *testing.T) {
	root := t.TempDir()
	files := []string{
		filepath.Join(root, "zeta", "Report FINAL.txt"),
		filepath.Join(root, "alpha", "report-notes.TXT"),
		filepath.Join(root, "space dir", "Résumé 文件.TXT"),
		filepath.Join(root, "plain.md"),
		filepath.Join(root, "match-directory", "nested.txt"),
	}
	for _, path := range files {
		writeTestFile(t, path)
	}

	want := []string{
		filepath.Join(root, "alpha", "report-notes.TXT"),
		filepath.Join(root, "zeta", "Report FINAL.txt"),
	}

	got, warnings, err := Search(root, "REPORT")
	if err != nil {
		t.Fatalf("Search returned fatal error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("Search returned unexpected warnings: %#v", warnings)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Search results = %#v, want %#v", got, want)
	}

	repeated, repeatedWarnings, err := Search(root, "REPORT")
	if err != nil {
		t.Fatalf("repeated Search returned fatal error: %v", err)
	}
	if !reflect.DeepEqual(repeated, got) || !reflect.DeepEqual(repeatedWarnings, warnings) {
		t.Fatalf("repeated Search changed results: first (%#v, %#v), second (%#v, %#v)", got, warnings, repeated, repeatedWarnings)
	}

	unicodeMatches, warnings, err := Search(root, "RÉSUMÉ")
	if err != nil {
		t.Fatalf("Unicode Search returned fatal error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("Unicode Search returned unexpected warnings: %#v", warnings)
	}
	wantUnicode := []string{filepath.Join(root, "space dir", "Résumé 文件.TXT")}
	if !reflect.DeepEqual(unicodeMatches, wantUnicode) {
		t.Fatalf("Unicode Search results = %#v, want %#v", unicodeMatches, wantUnicode)
	}

	zero, warnings, err := Search(root, "does-not-exist")
	if err != nil {
		t.Fatalf("zero-match Search returned fatal error: %v", err)
	}
	if len(zero) != 0 || len(warnings) != 0 {
		t.Fatalf("zero-match Search = (%#v, %#v), want no results or warnings", zero, warnings)
	}

	directoryOnly, warnings, err := Search(root, "match-directory")
	if err != nil {
		t.Fatalf("directory-only Search returned fatal error: %v", err)
	}
	if len(directoryOnly) != 0 || len(warnings) != 0 {
		t.Fatalf("directory-only Search = (%#v, %#v), want no results or warnings", directoryOnly, warnings)
	}
}

func TestSearchSkipsIgnoredDirectories(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "visible", "keep.txt"))
	for _, directory := range []string{".git", "node_modules", "vendor", "NODE_MODULES"} {
		writeTestFile(t, filepath.Join(root, directory, "hidden.txt"))
	}

	got, warnings, err := Search(root, ".txt")
	if err != nil {
		t.Fatalf("Search returned fatal error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("Search returned unexpected warnings: %#v", warnings)
	}
	want := []string{filepath.Join(root, "visible", "keep.txt")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Search results = %#v, want %#v", got, want)
	}
}

func TestSearchRootErrorIsFatal(t *testing.T) {
	root := filepath.Join(t.TempDir(), "does-not-exist")
	paths, warnings, err := Search(root, "anything")
	if err == nil {
		t.Fatal("Search returned nil error for an inaccessible root")
	}
	if len(paths) != 0 {
		t.Fatalf("Search returned paths with fatal root error: %#v", paths)
	}
	if len(warnings) != 0 {
		t.Fatalf("Search returned warnings for a fatal root error: %#v", warnings)
	}
	if !strings.Contains(err.Error(), root) {
		t.Fatalf("fatal error %q does not mention root %q", err, root)
	}
}

func TestSearchDoesNotFollowDirectorySymlinks(t *testing.T) {
	root := t.TempDir()
	target := t.TempDir()
	writeTestFile(t, filepath.Join(target, "hidden.txt"))

	link := filepath.Join(root, "linked-directory")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("directory symlinks are unavailable: %v", err)
	}

	paths, warnings, err := Search(root, "hidden")
	if err != nil {
		t.Fatalf("Search returned fatal error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("Search returned unexpected warnings: %#v", warnings)
	}
	if len(paths) != 0 {
		t.Fatalf("Search followed a directory symlink and returned %#v", paths)
	}
}

func TestSearchContinuesAfterInaccessibleChild(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("portable child-directory permission changes are not available on Windows")
	}

	root := t.TempDir()
	accessible := filepath.Join(root, "accessible.txt")
	restrictedDir := filepath.Join(root, "restricted")
	restrictedFile := filepath.Join(restrictedDir, "hidden.txt")
	writeTestFile(t, accessible)
	writeTestFile(t, restrictedFile)
	if err := os.Chmod(restrictedDir, 0); err != nil {
		t.Skipf("cannot restrict child directory: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(restrictedDir, 0o755) })

	paths, warnings, err := Search(root, ".txt")
	if err != nil {
		t.Fatalf("Search returned fatal error: %v", err)
	}
	if containsPath(paths, restrictedFile) {
		t.Skip("test process can read the restricted directory")
	}
	if !containsPath(paths, accessible) {
		t.Fatalf("Search omitted accessible file: %#v", paths)
	}
	if len(warnings) == 0 {
		t.Fatal("Search did not report a warning for the inaccessible child")
	}
	if !containsWarningPath(warnings, restrictedDir) {
		t.Fatalf("warnings = %#v, want a warning for %q", warnings, restrictedDir)
	}
}

func writeTestFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

func containsPath(paths []string, want string) bool {
	for _, path := range paths {
		if path == want {
			return true
		}
	}
	return false
}

func containsWarningPath(warnings []Warning, want string) bool {
	for _, warning := range warnings {
		if warning.Path == want {
			return true
		}
	}
	return false
}

func TestSearchResultsAreSorted(t *testing.T) {
	root := t.TempDir()
	paths := []string{
		filepath.Join(root, "z.txt"),
		filepath.Join(root, "a.txt"),
		filepath.Join(root, "m.txt"),
	}
	for _, path := range paths {
		writeTestFile(t, path)
	}

	got, warnings, err := Search(root, ".txt")
	if err != nil {
		t.Fatalf("Search returned fatal error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("Search returned unexpected warnings: %#v", warnings)
	}
	want := append([]string(nil), paths...)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Search results = %#v, want sorted %#v", got, want)
	}
}
