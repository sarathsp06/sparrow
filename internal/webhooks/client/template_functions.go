package client

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"text/template"
	"time"
)

// TemplateFunc represents a template utility function with enhanced documentation
type TemplateFunc struct {
	Name        string
	Func        any
	Description string
}

// GetTemplateFunctions returns all available template utility functions with comprehensive documentation
func GetTemplateFunctions() []TemplateFunc {
	return []TemplateFunc{
		{
			Name: "json",
			Func: func(v any) (string, error) {
				b, err := json.Marshal(v)
				return string(b), err
			},
			Description: "# json\n\nConverts any value to a JSON string.\n\n## Usage\n```\n{{ .data | json }}\n{{ json .payload }}\n```\n\n## Example\n```\nInput: map[string]any{\"name\": \"John\", \"age\": 30}\nOutput: {\"name\":\"John\",\"age\":30}\n```",
		},
		{
			Name: "urlencode",
			Func: func(s string) string {
				return url.QueryEscape(s)
			},
			Description: "# urlencode\n\nURL encodes a string by escaping special characters for safe use in URLs.\n\n## Usage\n```\n{{ .email | urlencode }}\n{{ urlencode \"hello world\" }}\n```\n\n## Example\n```\nInput: \"hello world@example.com\"\nOutput: \"hello+world%40example.com\"\n```",
		},
		{
			Name: "base64",
			Func: func(s string) string {
				return base64.StdEncoding.EncodeToString([]byte(s))
			},
			Description: "# base64\n\nBase64 encodes a string.\n\n## Usage\n```\n{{ .secret | base64 }}\n{{ base64 \"hello world\" }}\n```\n\n## Example\n```\nInput: \"hello world\"\nOutput: \"aGVsbG8gd29ybGQ=\"\n```",
		},
		{
			Name: "base64decode",
			Func: func(s string) (string, error) {
				b, err := base64.StdEncoding.DecodeString(s)
				if err != nil {
					return "", err
				}
				return string(b), nil
			},
			Description: "# base64decode\n\nBase64 decodes a string back to its original form.\n\n## Usage\n```\n{{ .encodedSecret | base64decode }}\n{{ base64decode \"aGVsbG8gd29ybGQ=\" }}\n```\n\n## Example\n```\nInput: \"aGVsbG8gd29ybGQ=\"\nOutput: \"hello world\"\n```",
		},
		{
			Name: "now",
			Func: func() time.Time {
				return time.Now()
			},
			Description: "# now\n\nReturns the current time as a time.Time object.\n\n## Usage\n```\n{{ now }}\n{{ now | formatTime \"2006-01-02 15:04:05\" }}\n```\n\n## Example\n```\nOutput: 2023-11-21 14:30:45 +0000 UTC\n```",
		},
		{
			Name: "formatTime",
			Func: func(layout string, t time.Time) string {
				return t.Format(layout)
			},
			Description: "# formatTime\n\nFormats a time value using Go's time layout format.\n\n## Usage\n```\n{{ formatTime \"2006-01-02\" .createdAt }}\n{{ .timestamp | formatTime \"15:04:05\" }}\n```\n\n## Common Layouts\n- RFC3339: `2006-01-02T15:04:05Z07:00`\n- Date only: `2006-01-02`\n- Time only: `15:04:05`\n- Human readable: `January 2, 2006 at 3:04 PM`\n\n## Example\n```\nInput: time.Now(), \"2006-01-02 15:04:05\"\nOutput: \"2023-11-21 14:30:45\"\n```",
		},
		{
			Name: "parseTime",
			Func: func(layout, value string) (time.Time, error) {
				return time.Parse(layout, value)
			},
			Description: "# parseTime\n\nParses a time string using Go's time layout format.\n\n## Usage\n```\n{{ parseTime \"2006-01-02\" \"2023-11-21\" }}\n{{ parseTime \"15:04:05\" .timeString }}\n```\n\n## Example\n```\nInput: \"2006-01-02\", \"2023-11-21\"\nOutput: 2023-11-21 00:00:00 +0000 UTC\n```",
		},
		{
			Name: "upper",
			Func: func(s string) string {
				return strings.ToUpper(s)
			},
			Description: "# upper\n\nConverts string to uppercase.\n\n## Usage\n```\n{{ .name | upper }}\n{{ upper \"hello world\" }}\n```\n\n## Example\n```\nInput: \"hello world\"\nOutput: \"HELLO WORLD\"\n```",
		},
		{
			Name: "lower",
			Func: func(s string) string {
				return strings.ToLower(s)
			},
			Description: "# lower\n\nConverts string to lowercase.\n\n## Usage\n```\n{{ .name | lower }}\n{{ lower \"HELLO WORLD\" }}\n```\n\n## Example\n```\nInput: \"HELLO WORLD\"\nOutput: \"hello world\"\n```",
		},
		{
			Name: "title",
			Func: func(s string) string {
				return strings.ToTitle(s)
			},
			Description: "# title\n\nConverts string to title case (all letters uppercase).\n\n## Usage\n```\n{{ .name | title }}\n{{ title \"hello world\" }}\n```\n\n## Example\n```\nInput: \"hello world\"\nOutput: \"HELLO WORLD\"\n```\n\nNote: This converts ALL letters to uppercase. For proper title case (first letter of each word), use a custom function.",
		},
		{
			Name: "trim",
			Func: func(cutset, s string) string {
				return strings.Trim(s, cutset)
			},
			Description: "# trim\n\nTrims specified characters from both ends of string.\n\n## Usage\n```\n{{ trim \" \" .text }}\n{{ trim \".\" \"...hello...\" }}\n```\n\n## Example\n```\nInput: \" \", \"  hello world  \"\nOutput: \"hello world\"\n\nInput: \".\", \"...hello world...\"\nOutput: \"hello world\"\n```",
		},
		{
			Name: "split",
			Func: func(sep, s string) []string {
				return strings.Split(s, sep)
			},
			Description: "# split\n\nSplits string by separator into a slice.\n\n## Usage\n```\n{{ split \",\" \"apple,banana,cherry\" }}\n{{ .tags | split \"|\" }}\n```\n\n## Example\n```\nInput: \",\", \"apple,banana,cherry\"\nOutput: [\"apple\", \"banana\", \"cherry\"]\n```\n\nUse with range to iterate:\n```\n{{range split \",\" .tags}}\n- {{.}}\n{{end}}\n```",
		},
		{
			Name: "join",
			Func: func(sep string, elems []string) string {
				return strings.Join(elems, sep)
			},
			Description: "# join\n\nJoins string slice with separator.\n\n## Usage\n```\n{{ join \", \" .tags }}\n{{ .items | join \" | \" }}\n```\n\n## Example\n```\nInput: \", \", [\"apple\", \"banana\", \"cherry\"]\nOutput: \"apple, banana, cherry\"\n```",
		},
		{
			Name: "default",
			Func: func(def, val any) any {
				if val == nil || val == "" {
					return def
				}
				return val
			},
			Description: "# default\n\nReturns default value if the input value is nil or empty string.\n\n## Usage\n```\n{{ .optionalField | default \"N/A\" }}\n{{ default \"Unknown\" .name }}\n```\n\n## Example\n```\nInput: \"N/A\", \"\"\nOutput: \"N/A\"\n\nInput: \"N/A\", \"John\"\nOutput: \"John\"\n```",
		},
		{
			Name: "slice",
			Func: func(start, end int, s string) string {
				if start < 0 || start >= len(s) {
					return ""
				}
				if end < 0 || end > len(s) {
					end = len(s)
				}
				if end <= start {
					return ""
				}
				return s[start:end]
			},
			Description: "# slice\n\nReturns substring from start to end index (end exclusive).\n\n## Usage\n```\n{{ slice 0 5 \"hello world\" }}\n{{ .text | slice 2 8 }}\n```\n\n## Example\n```\nInput: 0, 5, \"hello world\"\nOutput: \"hello\"\n\nInput: 6, -1, \"hello world\"\nOutput: \"world\"\n```",
		},
		{
			Name: "len",
			Func: func(v any) int {
				switch val := v.(type) {
				case string:
					return len(val)
				case []any:
					return len(val)
				case map[string]any:
					return len(val)
				default:
					return 0
				}
			},
			Description: "# len\n\nReturns length of string, slice, or map.\n\n## Usage\n```\n{{ len .name }}\n{{ .items | len }}\n```\n\n## Example\n```\nInput: \"hello\"\nOutput: 5\n\nInput: [\"a\", \"b\", \"c\"]\nOutput: 3\n```",
		},
		{
			Name: "truncate",
			Func: func(maxLen int, s string) string {
				if len(s) <= maxLen {
					return s
				}
				return s[:maxLen]
			},
			Description: "# truncate\n\nTruncates string to maximum length (hard cut, no ellipsis).\n\n## Usage\n```\n{{ truncate 10 .longText }}\n{{ .description | truncate 50 }}\n```\n\n## Example\n```\nInput: 10, \"this is a very long string\"\nOutput: \"this is a \"\n```\n\nFor truncation with ellipsis, use `ellipsis` function instead.",
		},
		{
			Name: "ellipsis",
			Func: func(maxLen int, s string) string {
				if len(s) <= maxLen {
					return s
				}
				if maxLen <= 3 {
					return s[:maxLen]
				}
				return s[:maxLen-3] + "..."
			},
			Description: "# ellipsis\n\nTruncates string to maximum length and adds \"...\" if truncated.\n\n## Usage\n```\n{{ ellipsis 20 .longText }}\n{{ .description | ellipsis 100 }}\n```\n\n## Example\n```\nInput: 20, \"this is a very long string that needs truncation\"\nOutput: \"this is a very lo...\"\n\nInput: 10, \"short\"\nOutput: \"short\"\n```\n\nIf maxLen ≤ 3, no ellipsis is added (hard truncate).",
		},
		{
			Name: "repeat",
			Func: func(count int, s string) (string, error) {
				const maxRepeat = 1000
				if count < 0 {
					count = 0
				}
				if count > maxRepeat {
					return "", fmt.Errorf("repeat count %d exceeds maximum of %d", count, maxRepeat)
				}
				return strings.Repeat(s, count), nil
			},
			Description: "# repeat\n\nRepeats string specified number of times.\n\n## Usage\n```\n{{ repeat 3 \"*\" }}\n{{ .pattern | repeat 5 }}\n```\n\n## Example\n```\nInput: 3, \"*\"\nOutput: \"***\"\n\nInput: 2, \"hello\"\nOutput: \"hellohello\"\n```",
		},
		{
			Name: "contains",
			Func: func(substr, s string) bool {
				return strings.Contains(s, substr)
			},
			Description: "# contains\n\nChecks if string contains substring (case-sensitive).\n\n## Usage\n```\n{{ if contains \"error\" .message }}\n  Error found!\n{{ end }}\n```\n\n## Example\n```\nInput: \"world\", \"hello world\"\nOutput: true\n\nInput: \"World\", \"hello world\"\nOutput: false\n```",
		},
		{
			Name: "hasPrefix",
			Func: func(prefix, s string) bool {
				return strings.HasPrefix(s, prefix)
			},
			Description: "# hasPrefix\n\nChecks if string starts with the specified prefix (case-sensitive).\n\n## Usage\n```\n{{ if hasPrefix \"http\" .url }}\n  Valid URL\n{{ end }}\n```\n\n## Example\n```\nInput: \"hello\", \"hello world\"\nOutput: true\n\nInput: \"Hello\", \"hello world\"\nOutput: false\n```",
		},
		{
			Name: "hasSuffix",
			Func: func(suffix, s string) bool {
				return strings.HasSuffix(s, suffix)
			},
			Description: "# hasSuffix\n\nChecks if string ends with the specified suffix (case-sensitive).\n\n## Usage\n```\n{{ if hasSuffix \".jpg\" .filename }}\n  Image file\n{{ end }}\n```\n\n## Example\n```\nInput: \"world\", \"hello world\"\nOutput: true\n\nInput: \"World\", \"hello world\"\nOutput: false\n```",
		},
		{
			Name: "trimSpace",
			Func: func(s string) string {
				return strings.TrimSpace(s)
			},
			Description: "# trimSpace\n\nTrims whitespace (spaces, tabs, newlines) from both ends of string.\n\n## Usage\n```\n{{ .userInput | trimSpace }}\n{{ trimSpace \"  hello world  \" }}\n```\n\n## Example\n```\nInput: \"  hello world  \\n\"\nOutput: \"hello world\"\n```",
		},
		{
			Name: "replace",
			Func: func(old, new, s string) string {
				return strings.ReplaceAll(s, old, new)
			},
			Description: "# replace\n\nReplaces ALL occurrences of old substring with new substring.\n\n## Usage\n```\n{{ replace \" \" \"_\" .name }}\n{{ .text | replace \"foo\" \"bar\" }}\n```\n\n## Example\n```\nInput: \" \", \"_\", \"hello world test\"\nOutput: \"hello_world_test\"\n\nInput: \"foo\", \"bar\", \"foo is foo\"\nOutput: \"bar is bar\"\n```",
		},
	}
}

// GetFunctionMap returns a template.FuncMap with all utility functions
func GetFunctionMap() template.FuncMap {
	funcMap := make(template.FuncMap)
	for _, tf := range GetTemplateFunctions() {
		funcMap[tf.Name] = tf.Func
	}
	return funcMap
}

// GetFunctionDocumentation returns markdown documentation for all template functions
func GetFunctionDocumentation() string {
	var docs strings.Builder

	docs.WriteString("# Template Functions Reference\n\n")
	docs.WriteString("This document describes all available template functions for webhook payload transformation.\n\n")

	for _, tf := range GetTemplateFunctions() {
		docs.WriteString(tf.Description)
		docs.WriteString("\n\n---\n\n")
	}

	return docs.String()
}
