package receivercode

import (
	"bytes"
	"flag"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden files in testdata/")

// TestGolden generates every target for every case and compares the files
// with testdata/<target>/<case>/. Run with -update to rewrite them, then
// review the diff.
func TestGolden(t *testing.T) {
	for _, target := range Targets() {
		for _, tc := range testCases {
			t.Run(target.ID+"/"+tc.name, func(t *testing.T) {
				in := tc.in
				in.Target = target.Target
				in.SparrowVersion = "v1.2.3"
				out, err := Generate(in)
				if err != nil {
					t.Fatal(err)
				}
				dir := filepath.Join("testdata", target.ID, tc.name)
				if *update {
					writeTree(t, dir, out.Files)
					return
				}
				compareTree(t, dir, out.Files)
			})
		}
	}
}

func writeTree(t *testing.T, dir string, files []File) {
	t.Helper()
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		p := filepath.Join(dir, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(f.Content), fs.FileMode(f.Mode)); err != nil {
			t.Fatal(err)
		}
	}
}

func compareTree(t *testing.T, dir string, files []File) {
	t.Helper()
	want := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		rel, _ := filepath.Rel(dir, p)
		want[filepath.ToSlash(rel)] = string(b)
		return err
	})
	if err != nil {
		t.Fatalf("read golden tree (run go test -update): %v", err)
	}
	for _, f := range files {
		golden, ok := want[f.Path]
		if !ok {
			t.Errorf("%s: not in the golden tree (run go test -update)", f.Path)
			continue
		}
		delete(want, f.Path)
		if golden != f.Content {
			t.Errorf("%s differs from the golden file (run go test -update and review the diff)", f.Path)
		}
	}
	extra := make([]string, 0, len(want))
	for p := range want {
		extra = append(extra, p)
	}
	sort.Strings(extra)
	for _, p := range extra {
		t.Errorf("%s: golden file no longer generated", p)
	}
}

// TestDeterministic generates each case twice and requires identical
// output.
func TestDeterministic(t *testing.T) {
	for _, target := range Targets() {
		for _, tc := range testCases {
			in := tc.in
			in.Target = target.Target
			a, err := Generate(in)
			if err != nil {
				t.Fatal(err)
			}
			b, _ := Generate(in)
			if len(a.Files) != len(b.Files) {
				t.Fatalf("%s/%s: file count differs", target.ID, tc.name)
			}
			for i := range a.Files {
				if a.Files[i] != b.Files[i] {
					t.Errorf("%s/%s: %s differs between runs", target.ID, tc.name, a.Files[i].Path)
				}
			}
		}
	}
}

// TestGoTargetBuildsAndPasses writes the Go target of every case to a temp
// dir and runs its own tests: it must build, vet and pass.
func TestGoTargetBuildsAndPasses(t *testing.T) {
	if testing.Short() {
		t.Skip("builds generated projects")
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go not in PATH")
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			in := tc.in
			in.Target = Target{Language: "go", Framework: "nethttp"}
			out, err := Generate(in)
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			writeTree(t, dir, out.Files)
			for _, args := range [][]string{{"vet", "./..."}, {"test", "./..."}} {
				cmd := exec.Command(goBin, args...)
				cmd.Dir = dir
				cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod")
				var stderr bytes.Buffer
				cmd.Stdout, cmd.Stderr = &stderr, &stderr
				if err := cmd.Run(); err != nil {
					t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, stderr.String())
				}
			}
		})
	}
}
