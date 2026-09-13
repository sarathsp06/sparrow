// Command sparrow is the Sparrow CLI: push events, tail deliveries, listen
// for webhooks locally, apply recipes, and debug transform templates.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
)

// version is set via -ldflags "-X main.version=v1.2.3".
var version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		pal := newPalette(os.Stderr)
		fmt.Fprintln(os.Stderr, pal.red("sparrow:"), err)
		os.Exit(1)
	}
}

// run builds the command tree and executes args against it. It is the seam
// tests drive.
func run(ctx context.Context, args []string, out io.Writer) error {
	root := newRootCmd(out)
	root.SetArgs(args)
	return root.ExecuteContext(ctx)
}
