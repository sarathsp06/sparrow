package receivercode

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"mime"
	"sort"
	"strings"
)

// Payload formats of a delivery body.
const (
	FormatJSON = "json"
	FormatXML  = "xml"
	FormatText = "text"
)

// formatFromContentType maps a Content-Type header value to a format, or ""
// when the value is empty or unparsable.
func formatFromContentType(ct string) string {
	media, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return ""
	}
	switch {
	case media == "application/json" || strings.HasSuffix(media, "+json"):
		return FormatJSON
	case media == "application/xml" || media == "text/xml" || strings.HasSuffix(media, "+xml"):
		return FormatXML
	default:
		return FormatText
	}
}

// DetectFormat classifies a rendered body as JSON, XML or text.
func DetectFormat(body []byte) string {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return FormatText
	}
	switch trimmed[0] {
	case '{', '[':
		if json.Valid(trimmed) {
			return FormatJSON
		}
	case '<':
		if wellFormedXML(trimmed) {
			return FormatXML
		}
	}
	return FormatText
}

// wellFormedXML reports whether b is one well-formed XML document with a
// root element.
func wellFormedXML(b []byte) bool {
	dec := xml.NewDecoder(bytes.NewReader(b))
	dec.Strict = true
	depth, roots := 0, 0
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return roots == 1 && depth == 0
		}
		if err != nil {
			return false
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if depth == 0 {
				roots++
			}
			depth++
		case xml.EndElement:
			depth--
		case xml.CharData:
			if depth == 0 && len(bytes.TrimSpace(t)) > 0 {
				return false
			}
		}
	}
}

// headerValue looks a header up case-insensitively. If several spellings
// are set, the first in sorted order wins, so the result is stable.
func headerValue(h map[string]string, name string) string {
	keys := make([]string, 0, len(h))
	for k := range h {
		if strings.EqualFold(k, name) {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return ""
	}
	sort.Strings(keys)
	return h[keys[0]]
}
