// Command gen writes one generated receiver project to disk, for the
// behavioural contract tests.
//
//	go run ./cmd/gen -in testdata/hmac.json -target go-nethttp -o /tmp/receiver -env /tmp/receiver.env
//
// -in is a receivercode.Input as JSON (its Target is replaced by -target).
// The output directory must be empty or absent. -env writes shell exports of
// the spec's throwaway credentials, for starting the receiver under test.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/sarathsp06/sparrow/pkg/receivercode"
)

func main() {
	in := flag.String("in", "", "receivercode.Input as JSON")
	target := flag.String("target", "", "target id, e.g. go-nethttp")
	out := flag.String("o", "", "output directory (empty or absent)")
	env := flag.String("env", "", "file to write the test credentials to, as shell exports")
	flag.Parse()
	if err := run(*in, *target, *out, *env); err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
}

func run(inPath, targetID, outDir, envPath string) error {
	if inPath == "" || targetID == "" || outDir == "" {
		return errors.New("-in, -target and -o are required")
	}
	raw, err := os.ReadFile(inPath) // #nosec G304 -- a test tool reading the file it is given
	if err != nil {
		return err
	}
	var input receivercode.Input
	if err := json.Unmarshal(raw, &input); err != nil {
		return fmt.Errorf("decode %s: %w", inPath, err)
	}
	found := false
	for _, t := range receivercode.Targets() {
		if t.ID == targetID {
			input.Target, found = t.Target, true
		}
	}
	if !found {
		return fmt.Errorf("unknown target %q", targetID)
	}
	gen, err := receivercode.Generate(input)
	if err != nil {
		return err
	}
	if entries, err := os.ReadDir(outDir); err == nil && len(entries) > 0 {
		return fmt.Errorf("%s is not empty", outDir)
	}
	for _, f := range gen.Files {
		p := filepath.Join(outDir, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			return err
		}
		if err := os.WriteFile(p, []byte(f.Content), fs.FileMode(f.Mode)); err != nil {
			return err
		}
	}
	if envPath != "" {
		// The receiver under test verifies with the throwaway credentials
		// the vectors are signed with, never a real secret.
		env := "export " + receivercode.EnvSecret + "=" + gen.Spec.Test.Secret + "\n"
		if gen.Spec.Webhook.SigningScheme == receivercode.SchemeEd25519 {
			env = "export " + receivercode.EnvPublicKey + "=" + gen.Spec.Test.PublicKey + "\n"
		}
		if err := os.WriteFile(envPath, []byte(env), 0o600); err != nil {
			return err
		}
	}
	fmt.Printf("wrote %d files to %s\nrun:  %s\ntest: %s\n", len(gen.Files), outDir, gen.Run, gen.Test)
	return nil
}
