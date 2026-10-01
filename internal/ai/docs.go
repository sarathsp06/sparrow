package ai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"golang.org/x/net/html"

	"github.com/sarathsp06/sparrow/internal/webhooks/client"
)

// Limits for documentation fetches. The byte cap bounds what a receiver can
// make Sparrow download; the text cap bounds what is sent to the model.
const (
	docMaxBytes = 2 << 20 // 2 MiB
	docMaxChars = 20000
	docTimeout  = 15 * time.Second
)

// NewDocFetcher returns a DocFetcher that downloads an http(s) page and
// returns its visible text, trimmed to docMaxChars. The URL host is checked
// against policy before connecting and the dialer re-checks every resolved
// address and redirect, so the same rules that govern webhook deliveries
// govern what a docs_url can reach (no loopback, private ranges, or cloud
// metadata unless the operator allowed them).
func NewDocFetcher(policy client.NetworkPolicy) DocFetcher {
	httpClient := policy.NewHTTPClient(docTimeout)
	return func(ctx context.Context, rawURL string) (string, error) {
		u, err := url.Parse(strings.TrimSpace(rawURL))
		if err != nil {
			return "", fmt.Errorf("invalid URL: %w", err)
		}
		scheme := strings.ToLower(u.Scheme)
		if scheme != "http" && scheme != "https" {
			return "", errors.New("only http and https URLs are supported")
		}
		if u.Hostname() == "" {
			return "", errors.New("URL has no host")
		}
		if err := policy.CheckHost(u.Hostname()); err != nil {
			return "", err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("User-Agent", "Sparrow-AI-Drafter/1.0 (+https://github.com/sarathsp06/sparrow)")
		req.Header.Set("Accept", "text/html, text/plain, text/markdown, application/json;q=0.9, */*;q=0.1")
		resp, err := httpClient.Do(req)
		if err != nil {
			return "", err
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return "", fmt.Errorf("%s returned HTTP %d", client.RedactURL(u.String()), resp.StatusCode)
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, docMaxBytes+1))
		if err != nil {
			return "", err
		}
		if len(body) > docMaxBytes {
			body = body[:docMaxBytes]
		}
		text := string(body)
		if ct := resp.Header.Get("Content-Type"); strings.Contains(ct, "html") || looksLikeHTML(body) {
			text = htmlToText(text)
		}
		text = collapseSpace(text)
		if text == "" {
			return "", errors.New("page contained no readable text")
		}
		if len(text) > docMaxChars {
			text = text[:docMaxChars] + "\n[truncated]"
		}
		return text, nil
	}
}

func looksLikeHTML(b []byte) bool {
	head := strings.ToLower(string(b[:min(len(b), 512)]))
	return strings.Contains(head, "<html") || strings.Contains(head, "<!doctype html")
}

// htmlToText keeps visible text, drops script/style/nav/header/footer, and
// puts block boundaries on their own lines so headings and code stay apart.
func htmlToText(src string) string {
	doc, err := html.Parse(strings.NewReader(src))
	if err != nil {
		return src
	}
	var b strings.Builder
	skip := map[string]bool{"script": true, "style": true, "noscript": true, "svg": true, "nav": true, "header": true, "footer": true, "template": true}
	block := map[string]bool{"p": true, "div": true, "pre": true, "li": true, "tr": true, "br": true, "h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true, "section": true, "article": true, "table": true, "code": false}
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && skip[n.Data] {
			return
		}
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
		if n.Type == html.ElementNode && block[n.Data] {
			b.WriteString("\n")
		}
	}
	walk(doc)
	return b.String()
}

// collapseSpace trims lines, drops runs of blank lines, and squeezes
// horizontal whitespace, without touching content inside the text.
func collapseSpace(s string) string {
	var out []string
	blank := 0
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(strings.Map(func(r rune) rune {
			if r == '\t' || (unicode.IsSpace(r) && r != '\n') {
				return ' '
			}
			return r
		}, line))
		for strings.Contains(line, "  ") {
			line = strings.ReplaceAll(line, "  ", " ")
		}
		if line == "" {
			blank++
			if blank > 1 {
				continue
			}
		} else {
			blank = 0
		}
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
