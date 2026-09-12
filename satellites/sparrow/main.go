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

const usage = `sparrow — CLI for the Sparrow webhook delivery server

Usage:
  sparrow init            Configure server URL, API key, and namespace
  sparrow push <event>    Push an event occurrence
  sparrow tail deliveries Stream delivery rows as they happen
  sparrow listen          Receive deliveries on a local HTTP server
  sparrow use <recipe>    Apply a recipe (register webhook + transform)
  sparrow template test <file>  Render a transform template locally
  sparrow version         Print version

Run 'sparrow <command> -h' for command flags.
`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "sparrow:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		_, _ = fmt.Fprint(out, usage)
		return nil
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "init":
		return runInit(ctx, rest, out)
	case "push":
		return runPush(ctx, rest, out)
	case "tail":
		return runTail(ctx, rest, out)
	case "listen":
		return runListen(ctx, rest, out)
	case "use":
		return runUse(ctx, rest, out)
	case "template":
		if len(rest) == 0 || rest[0] != "test" {
			return fmt.Errorf("usage: sparrow template test <file>")
		}
		return runTemplateTest(rest[1:], out)
	case "version":
		_, _ = fmt.Fprintln(out, "sparrow", version)
		return nil
	case "help", "-h", "--help":
		_, _ = fmt.Fprint(out, usage)
		return nil
	default:
		return fmt.Errorf("unknown command %q\n\n%s", cmd, usage)
	}
}
