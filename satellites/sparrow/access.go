package main

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// --- API shapes (see internal/rest/access.go on the server) ---

type tokenOut struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Consumer   *string    `json:"consumer"`
	Status     string     `json:"status"`
	CreatedBy  string     `json:"created_by"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
}

type inviteOut struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Consumer  *string   `json:"consumer"`
	Status    string    `json:"status"`
	CreatedBy string    `json:"created_by"`
	ExpiresAt time.Time `json:"expires_at"`
}

type createdToken struct {
	Token  tokenOut `json:"token"`
	Secret string   `json:"secret"`
}

type createdInvite struct {
	Invite inviteOut `json:"invite"`
	Secret string    `json:"secret"`
	Path   string    `json:"path"`
}

func (c *apiClient) createToken(ctx context.Context, name, consumer string, ttl lifetimeFlag) (createdToken, error) {
	var out createdToken
	body := map[string]any{"name": name}
	if consumer != "" {
		body["consumer"] = consumer
	}
	if ttl.d > 0 {
		body["ttl_seconds"] = int64(ttl.d / time.Second)
	}
	if ttl.never {
		body["never_expires"] = true
	}
	err := c.do(ctx, "POST", "/v1/tokens", body, &out)
	return out, err
}

func (c *apiClient) listTokens(ctx context.Context, all bool) ([]tokenOut, error) {
	var out struct {
		Items []tokenOut `json:"items"`
	}
	err := c.do(ctx, "GET", "/v1/tokens?"+url.Values{"include_inactive": {strconv.FormatBool(all)}}.Encode(), nil, &out)
	return out.Items, err
}

func (c *apiClient) revokeToken(ctx context.Context, id string) error {
	return c.do(ctx, "DELETE", "/v1/tokens/"+url.PathEscape(id), nil, nil)
}

func (c *apiClient) createInvite(ctx context.Context, name, consumer string, ttl time.Duration, tokenTTL lifetimeFlag) (createdInvite, error) {
	var out createdInvite
	body := map[string]any{"name": name}
	if consumer != "" {
		body["consumer"] = consumer
	}
	if ttl > 0 {
		body["ttl_seconds"] = int64(ttl / time.Second)
	}
	if tokenTTL.d > 0 {
		body["token_ttl_seconds"] = int64(tokenTTL.d / time.Second)
	}
	if tokenTTL.never {
		body["token_never_expires"] = true
	}
	err := c.do(ctx, "POST", "/v1/invites", body, &out)
	return out, err
}

func (c *apiClient) listInvites(ctx context.Context, all bool) ([]inviteOut, error) {
	var out struct {
		Items []inviteOut `json:"items"`
	}
	err := c.do(ctx, "GET", "/v1/invites?"+url.Values{"include_inactive": {strconv.FormatBool(all)}}.Encode(), nil, &out)
	return out.Items, err
}

func (c *apiClient) cancelInvite(ctx context.Context, id string) error {
	return c.do(ctx, "DELETE", "/v1/invites/"+url.PathEscape(id), nil, nil)
}

// --- durations: Go syntax plus a "d" (days) suffix, e.g. 90d, 36h, 15m ---

type durationFlag struct{ d time.Duration }

func (f *durationFlag) String() string {
	if f.d == 0 {
		return ""
	}
	return f.d.String()
}

func (f *durationFlag) Set(s string) error {
	d, err := parseDuration(s)
	if err != nil {
		return err
	}
	f.d = d
	return nil
}

func (f *durationFlag) Type() string { return "duration" }

// lifetimeFlag is a token lifetime: a duration, or "never" for a tenant-wide
// token that does not expire.
type lifetimeFlag struct {
	d     time.Duration
	never bool
}

func (f *lifetimeFlag) String() string {
	switch {
	case f.never:
		return "never"
	case f.d == 0:
		return ""
	}
	return f.d.String()
}

func (f *lifetimeFlag) Set(s string) error {
	if strings.EqualFold(s, "never") {
		f.d, f.never = 0, true
		return nil
	}
	d, err := parseDuration(s)
	if err != nil {
		return fmt.Errorf("%w, or \"never\"", err)
	}
	f.d, f.never = d, false
	return nil
}

func (f *lifetimeFlag) Type() string { return "duration|never" }

func parseDuration(s string) (time.Duration, error) {
	if days, ok := strings.CutSuffix(s, "d"); ok {
		n, err := strconv.Atoi(days)
		if err != nil || n < 0 {
			return 0, fmt.Errorf("invalid duration %q (use e.g. 90d, 12h, 15m)", s)
		}
		return time.Duration(n) * 24 * time.Hour, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil || d < 0 {
		return 0, fmt.Errorf("invalid duration %q (use e.g. 90d, 12h, 15m)", s)
	}
	return d, nil
}

func accessLabel(consumer *string) string {
	if consumer == nil {
		return "full"
	}
	return "portal:" + *consumer
}

func timeOr(t *time.Time, fallback string) string {
	if t == nil {
		return fallback
	}
	return t.Local().Format("2006-01-02 15:04")
}

// --- sparrow tokens ---

func newTokensCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tokens",
		Short: "Manage access tokens (for scripts, CI, and services)",
		Long: `Access tokens are named, revocable credentials. A tenant-wide token works
exactly like SPARROW_API_KEY and expires after the server's default (90 days
unless SPARROW_TOKEN_DEFAULT_TTL changes it; --ttl never for none); a
consumer token only works through the portal API, limited to that consumer.

To give a person access to the web UI, prefer 'sparrow invite'.`,
	}
	cmd.AddCommand(newTokensCreateCmd(), newTokensListCmd(), newTokensRevokeCmd())
	return cmd
}

func newTokensCreateCmd() *cobra.Command {
	var name, consumer string
	var ttl lifetimeFlag
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a token; the secret is printed once",
		Example: `  sparrow tokens create --name ci-deploy
  sparrow tokens create --name build-bot --ttl never
  sparrow tokens create --name acme-sync --consumer acme --ttl 30d`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, _, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			return runTokensCreate(cmd.Context(), cmd.OutOrStdout(), client, name, consumer, ttl, outputFmt(cmd))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "who or what the token is for (required)")
	cmd.Flags().StringVar(&consumer, "consumer", "", "limit the token to one consumer's portal API")
	cmd.Flags().Var(&ttl, "ttl", "lifetime, e.g. 90d or 12h, or \"never\" (default: the server's default, 90d unless changed, for tenant-wide; 7d for consumer tokens)")
	_ = cmd.MarkFlagRequired("name")
	addOutputFlag(cmd)
	return cmd
}

func runTokensCreate(ctx context.Context, out io.Writer, client *apiClient, name, consumer string, ttl lifetimeFlag, format string) error {
	res, err := client.createToken(ctx, name, consumer, ttl)
	if err != nil {
		return err
	}
	if done, err := renderStructured(out, format, res); done {
		return err
	}
	_, _ = fmt.Fprintln(out, res.Secret)
	pal := newPalette(out)
	_, _ = fmt.Fprintf(out, "%s token %s (%s), expires: %s\n", accessLabel(res.Token.Consumer), res.Token.ID, res.Token.Name, timeOr(res.Token.ExpiresAt, "never"))
	_, _ = fmt.Fprintln(out, pal.dim("Store it now: it is not shown again. Send it as X-API-Key or Authorization: Bearer."))
	return nil
}

func newTokensListCmd() *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List access tokens",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, _, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			tokens, err := client.listTokens(cmd.Context(), all)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if done, err := renderStructured(out, outputFmt(cmd), tokens); done {
				return err
			}
			if len(tokens) == 0 {
				_, _ = fmt.Fprintln(out, "no tokens")
				return nil
			}
			pal := newPalette(out)
			row := "%-28s %-24s %-16s %-16s %-16s %-16s %s\n"
			_, _ = fmt.Fprint(out, pal.bold(fmt.Sprintf(row, "ID", "NAME", "ACCESS", "CREATED BY", "LAST USED", "EXPIRES", "STATUS")))
			for _, t := range tokens {
				_, _ = fmt.Fprintf(out, row, t.ID, t.Name, accessLabel(t.Consumer), t.CreatedBy, timeOr(t.LastUsedAt, "never"), timeOr(t.ExpiresAt, "never"), t.Status)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "include revoked and expired tokens")
	addOutputFlag(cmd)
	return cmd
}

func newTokensRevokeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "revoke <token-id>",
		Short: "Revoke a token (it stops working within 30 seconds everywhere)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := client.revokeToken(cmd.Context(), args[0]); err != nil {
				return err
			}
			cmd.Println("revoked", args[0])
			return nil
		},
	}
}

// --- sparrow invite / sparrow invites ---

func newInviteCmd() *cobra.Command {
	var consumer, uiURL string
	var ttl durationFlag
	var tokenTTL lifetimeFlag
	cmd := &cobra.Command{
		Use:   "invite <name>",
		Short: "Invite someone to the web UI with a one-time link",
		Long: `Create a one-time link that signs a browser into the Sparrow web UI.

Opening it creates an access token named <name> for that browser, so nobody
pastes a key into chat. The link works once, expires after --ttl, and can be
cancelled with 'sparrow invites cancel'. With --consumer, the link opens that
consumer's portal instead of the operator console.

The link points at --ui-url, which defaults to the server URL (right when the
server serves the UI itself). Pass the UI's address if it is hosted separately.`,
		Example: `  sparrow invite alice
  sparrow invite "acme support" --consumer acme --ttl 7d
  sparrow invite bob --ttl 15m --ui-url https://sparrow.example.com`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, cfg, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if uiURL == "" {
				uiURL = cfg.ServerURL
			}
			return runInvite(cmd.Context(), cmd.OutOrStdout(), client, args[0], consumer, uiURL, ttl.d, tokenTTL, outputFmt(cmd))
		},
	}
	cmd.Flags().StringVar(&consumer, "consumer", "", "invite into this consumer's portal instead of the console")
	cmd.Flags().Var(&ttl, "ttl", "how long the link can be used, e.g. 15m, 24h, 7d (default 24h, max 7d)")
	cmd.Flags().Var(&tokenTTL, "token-ttl", "lifetime of the access it grants, or \"never\" (default: the server's default, 90d unless changed; 7d for a consumer)")
	cmd.Flags().StringVar(&uiURL, "ui-url", "", "base URL of the web UI (default: the server URL)")
	addOutputFlag(cmd)
	return cmd
}

type inviteResult struct {
	URL    string    `json:"url"`
	Invite inviteOut `json:"invite"`
}

func runInvite(ctx context.Context, out io.Writer, client *apiClient, name, consumer, uiURL string, ttl time.Duration, tokenTTL lifetimeFlag, format string) error {
	res, err := client.createInvite(ctx, name, consumer, ttl, tokenTTL)
	if err != nil {
		return err
	}
	link := inviteResult{URL: strings.TrimRight(uiURL, "/") + res.Path, Invite: res.Invite}
	if done, err := renderStructured(out, format, link); done {
		return err
	}
	_, _ = fmt.Fprintln(out, link.URL)
	_, _ = fmt.Fprintf(out, "single use, for %s (%s access), expires %s\n", res.Invite.Name, accessLabel(res.Invite.Consumer), res.Invite.ExpiresAt.Local().Format(time.RFC1123))
	return nil
}

func newInvitesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invites",
		Short: "List or cancel invites",
	}
	var all bool
	list := &cobra.Command{
		Use:   "list",
		Short: "List pending invites",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, _, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			invites, err := client.listInvites(cmd.Context(), all)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if done, err := renderStructured(out, outputFmt(cmd), invites); done {
				return err
			}
			if len(invites) == 0 {
				_, _ = fmt.Fprintln(out, "no pending invites")
				return nil
			}
			pal := newPalette(out)
			row := "%-28s %-24s %-16s %-16s %-16s %s\n"
			_, _ = fmt.Fprint(out, pal.bold(fmt.Sprintf(row, "ID", "FOR", "ACCESS", "INVITED BY", "EXPIRES", "STATUS")))
			for _, inv := range invites {
				_, _ = fmt.Fprintf(out, row, inv.ID, inv.Name, accessLabel(inv.Consumer), inv.CreatedBy, timeOr(&inv.ExpiresAt, ""), inv.Status)
			}
			return nil
		},
	}
	list.Flags().BoolVar(&all, "all", false, "include redeemed, cancelled, and expired invites")
	addOutputFlag(list)
	cancel := &cobra.Command{
		Use:   "cancel <invite-id>",
		Short: "Cancel a pending invite",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := client.cancelInvite(cmd.Context(), args[0]); err != nil {
				return err
			}
			cmd.Println("cancelled", args[0])
			return nil
		},
	}
	cmd.AddCommand(list, cancel)
	return cmd
}
