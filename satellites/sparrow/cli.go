package main

import (
	"io"

	"github.com/spf13/cobra"
)

// newRootCmd builds the full sparrow command tree, wiring every subcommand and
// writing all output to out (tests drive this seam via run).
func newRootCmd(out io.Writer) *cobra.Command {
	root := &cobra.Command{
		Use:   "sparrow",
		Short: "CLI for the Sparrow webhook delivery server",
		Long: `sparrow is the command-line client for the Sparrow webhook delivery server.

Push event occurrences, inspect event types and webhooks, tail deliveries as
they happen, receive webhooks on a local server, apply recipes, and debug
transform templates against a running Sparrow instance.

Connection settings resolve in order: environment (SPARROW_URL, SPARROW_API_KEY,
SPARROW_NAMESPACE) > --url/--api-key/--namespace flags > ~/.sparrow/config.yaml >
built-in defaults. Run 'sparrow init' to write a config file.`,
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetOut(out)
	root.SetErr(out)
	root.SetVersionTemplate("sparrow {{.Version}}\n")

	pf := root.PersistentFlags()
	pf.String("url", "", "Sparrow server URL (env SPARROW_URL, config server_url)")
	pf.String("api-key", "", "API key for the X-API-Key header (env SPARROW_API_KEY)")
	pf.String("namespace", "", "tenant namespace (env SPARROW_NAMESPACE, config namespace)")

	root.AddCommand(
		newInitCmd(),
		newPushCmd(),
		newEventsCmd(),
		newWebhooksCmd(),
		newTailCmd(),
		newStatsCmd(),
		newListenCmd(),
		newUseCmd(),
		newTemplateCmd(),
		newFunctionsCmd(),
		newVersionCmd(),
	)
	return root
}

// clientFromCmd resolves connection config from the shared flags (plus env and
// config file) and returns a ready API client.
func clientFromCmd(cmd *cobra.Command) (*apiClient, config, error) {
	url, _ := cmd.Flags().GetString("url")
	apiKey, _ := cmd.Flags().GetString("api-key")
	ns, _ := cmd.Flags().GetString("namespace")
	cfg, err := resolveConfig(url, apiKey, ns)
	if err != nil {
		return nil, cfg, err
	}
	return newAPIClient(cfg), cfg, nil
}

// addOutputFlag registers the shared -o/--output flag for machine-readable output.
func addOutputFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("output", "o", "", "output format: json or yaml (default is a human table)")
}

func outputFmt(cmd *cobra.Command) string {
	v, _ := cmd.Flags().GetString("output")
	return v
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the sparrow version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println("sparrow", version)
			return nil
		},
	}
}
