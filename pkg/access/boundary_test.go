package access_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// This module must stay reusable outside Sparrow: it may import only the
// standard library and itself. (The go.work workspace would otherwise let a
// Sparrow import compile.)
func TestModuleImportsOnlyStdlibAndItself(t *testing.T) {
	const self = "github.com/sarathsp06/sparrow/pkg/access"
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			first := strings.SplitN(p, "/", 2)[0]
			if p == self || strings.HasPrefix(p, self+"/") || !strings.Contains(first, ".") {
				continue
			}
			t.Errorf("%s imports %q: pkg/access may only depend on the standard library", path, p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
