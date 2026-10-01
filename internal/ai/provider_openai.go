package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	svcerrors "github.com/sarathsp06/sparrow/pkg/errors"
)

// openAIClient talks to any OpenAI-compatible chat completions endpoint over
// plain HTTP: no SDK, because the point is to work with Ollama, vLLM, LM
// Studio, llama.cpp, OpenRouter, Azure-style gateways and OpenAI itself with
// one small client. It asks for a JSON schema response; servers that only
// support json_object still return parseable JSON, and parseDraft tolerates
// a fenced object from servers that support neither.
type openAIClient struct {
	http    *http.Client
	baseURL string
	apiKey  string
	model   string
}

func newOpenAIClient(cfg Config) *openAIClient {
	return &openAIClient{
		http:    &http.Client{Timeout: 120 * time.Second},
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
	}
}

type oaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type oaRequest struct {
	Model          string      `json:"model"`
	Messages       []oaMessage `json:"messages"`
	Temperature    float64     `json:"temperature"`
	ResponseFormat any         `json:"response_format,omitempty"`
}

type oaResponse struct {
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Content string `json:"content"`
			Refusal string `json:"refusal"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *openAIClient) complete(ctx context.Context, system string, turns []turn, schema map[string]any) (completion, error) {
	msgs := make([]oaMessage, 0, len(turns)+1)
	msgs = append(msgs, oaMessage{Role: "system", Content: system})
	for _, t := range turns {
		msgs = append(msgs, oaMessage{Role: t.role, Content: t.text})
	}
	body, err := json.Marshal(oaRequest{
		Model:       c.model,
		Messages:    msgs,
		Temperature: 0.2,
		ResponseFormat: map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   "template_draft",
				"schema": schema,
				"strict": true,
			},
		},
	})
	if err != nil {
		return completion{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return completion{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return completion{}, mapTransportError(err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return completion{}, mapTransportError(err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		if mapped, ok := mapStatus(resp.StatusCode); ok {
			return completion{}, mapped
		}
		return completion{}, svcerrors.Wrapf(fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(raw), 300)), svcerrors.Internal, "ai: provider error")
	}
	var out oaResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return completion{}, svcerrors.Wrapf(err, svcerrors.Internal, "ai: provider returned a non-JSON body")
	}
	if out.Error != nil {
		return completion{}, svcerrors.Wrapf(fmt.Errorf("%s", out.Error.Message), svcerrors.Internal, "ai: provider error")
	}
	if len(out.Choices) == 0 {
		return completion{}, svcerrors.Error(svcerrors.Internal, "ai: provider returned no choices")
	}
	ch := out.Choices[0]
	if ch.FinishReason == "content_filter" || ch.Message.Refusal != "" {
		return completion{refused: true}, nil
	}
	return completion{text: ch.Message.Content}, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
