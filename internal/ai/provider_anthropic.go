package ai

import (
	"context"
	"errors"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	svcerrors "github.com/sarathsp06/sparrow/pkg/errors"
)

// anthropicClient talks to the Anthropic Messages API with structured output.
type anthropicClient struct {
	client anthropic.Client
	model  string
}

func newAnthropicClient(cfg Config, opts ...option.RequestOption) *anthropicClient {
	ro := []option.RequestOption{option.WithAPIKey(cfg.APIKey)}
	if cfg.BaseURL != "" {
		ro = append(ro, option.WithBaseURL(cfg.BaseURL))
	}
	ro = append(ro, opts...)
	return &anthropicClient{client: anthropic.NewClient(ro...), model: cfg.Model}
}

func (c *anthropicClient) complete(ctx context.Context, system string, turns []turn, schema map[string]any) (completion, error) {
	messages := make([]anthropic.MessageParam, 0, len(turns))
	for _, t := range turns {
		if t.role == "assistant" {
			messages = append(messages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(t.text)))
		} else {
			messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(t.text)))
		}
	}
	resp, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     c.model,
		MaxTokens: 8000,
		System:    []anthropic.TextBlockParam{{Text: system}},
		Messages:  messages,
		OutputConfig: anthropic.OutputConfigParam{
			Format: anthropic.JSONOutputFormatParam{Schema: schema},
		},
	})
	if err != nil {
		var apiErr *anthropic.Error
		if errors.As(err, &apiErr) {
			if mapped, ok := mapStatus(apiErr.StatusCode); ok {
				return completion{}, mapped
			}
			return completion{}, svcerrors.Wrapf(err, svcerrors.Internal, "ai: provider error")
		}
		return completion{}, mapTransportError(err)
	}
	if resp.StopReason == anthropic.StopReasonRefusal {
		return completion{refused: true}, nil
	}
	var text strings.Builder
	for _, block := range resp.Content {
		if tb, ok := block.AsAny().(anthropic.TextBlock); ok {
			text.WriteString(tb.Text)
		}
	}
	return completion{text: text.String()}, nil
}
