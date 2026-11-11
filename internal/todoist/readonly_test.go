package todoist

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// writeMethods matches any way Go code names an HTTP method that changes
// something.
var writeMethods = regexp.MustCompile(`http\.Method(Post|Put|Patch|Delete)|"(POST|PUT|PATCH|DELETE)"|\.Post(Form)?\(`)

// Harold writes nothing to Todoist until a spec says it may (AGENTS.md). No
// code outside internal/auth, whose OAuth exchanges are POSTs by definition,
// names a method that writes. When triage actions arrive, the module that
// makes them becomes the one other exception here.
func TestNothingButAuthWrites(t *testing.T) {
	root := filepath.Join("..", "..")
	allowed := filepath.Join(root, "internal", "auth")
	checked := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path == allowed || strings.HasPrefix(d.Name(), ".") && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		checked++
		if match := writeMethods.Find(source); match != nil {
			t.Errorf("%s names %s", path, match)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked == 0 {
		t.Fatal("no Go files checked")
	}
}
