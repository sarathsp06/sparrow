---
title: Template Functions
description: Complete reference for Go template functions available in payload transformation
---

Sparrow provides 37 built-in template functions for transforming webhook payloads. These are available in subscription `transform_template` fields. They cover string manipulation, encoding, time formatting, arithmetic, and structural map/list building — enough to reshape a payload entirely without an embedded scripting runtime.

Templates use Go's `text/template` syntax. The event payload is available as the template data context (e.g., `{{ .payload.field_name }}`).

## Quick Reference

| Function | Description |
|----------|-------------|
| `json` | Convert value to JSON string |
| `urlencode` | URL-encode a string |
| `base64` | Base64-encode a string |
| `base64decode` | Base64-decode a string |
| `now` | Current time |
| `formatTime` | Format a time value |
| `parseTime` | Parse a time string |
| `upper` | Uppercase string |
| `lower` | Lowercase string |
| `title` | Title case (capitalize first letter of each word) |
| `trim` | Trim characters from both ends |
| `trimSpace` | Trim whitespace |
| `split` | Split string into slice |
| `join` | Join slice with separator |
| `default` | Default value for nil/empty |
| `slice` | Substring by index |
| `len` | Length of string/slice/map |
| `truncate` | Truncate string (hard cut) |
| `ellipsis` | Truncate with "..." suffix |
| `repeat` | Repeat string N times |
| `contains` | Check if string contains substring |
| `hasPrefix` | Check string prefix |
| `hasSuffix` | Check string suffix |
| `replace` | Replace all occurrences |
| `dict` | Build a map from key/value pairs |
| `list` | Build a slice from arguments |
| `append` | Append items to a slice |
| `merge` | Merge maps (later keys win) |
| `add` | Add two numbers |
| `sub` | Subtract two numbers |
| `mul` | Multiply two numbers |
| `div` | Divide two numbers |
| `mod` | Integer remainder |
| `dig` | Safe nested map lookup with default |
| `toString` | Convert any value to string |
| `toInt` | Convert to integer |
| `toFloat` | Convert to float |

---

## json

Converts any value to a JSON string.

```go
{{ .data | json }}
{{ json .payload }}
```

**Example:**
```
Input:  map[string]any{"name": "John", "age": 30}
Output: {"name":"John","age":30}
```

---

## urlencode

URL-encodes a string by escaping special characters.

```go
{{ .email | urlencode }}
{{ urlencode "hello world" }}
```

**Example:**
```
Input:  "hello world@example.com"
Output: "hello+world%40example.com"
```

---

## base64

Base64-encodes a string.

```go
{{ .secret | base64 }}
{{ base64 "hello world" }}
```

**Example:**
```
Input:  "hello world"
Output: "aGVsbG8gd29ybGQ="
```

---

## base64decode

Base64-decodes a string back to its original form.

```go
{{ .encodedSecret | base64decode }}
{{ base64decode "aGVsbG8gd29ybGQ=" }}
```

**Example:**
```
Input:  "aGVsbG8gd29ybGQ="
Output: "hello world"
```

---

## now

Returns the current time as a `time.Time` object.

```go
{{ now }}
{{ now | formatTime "2006-01-02 15:04:05" }}
```

**Example:**
```
Output: 2023-11-21 14:30:45 +0000 UTC
```

---

## formatTime

Formats a time value using Go's time layout format.

```go
{{ formatTime "2006-01-02" .createdAt }}
{{ .timestamp | formatTime "15:04:05" }}
```

**Common layouts:**
- RFC3339: `2006-01-02T15:04:05Z07:00`
- Date only: `2006-01-02`
- Time only: `15:04:05`
- Human readable: `January 2, 2006 at 3:04 PM`

---

## parseTime

Parses a time string using Go's time layout format.

```go
{{ parseTime "2006-01-02" "2023-11-21" }}
{{ parseTime "15:04:05" .timeString }}
```

**Example:**
```
Input:  "2006-01-02", "2023-11-21"
Output: 2023-11-21 00:00:00 +0000 UTC
```

---

## upper

Converts string to uppercase.

```go
{{ .name | upper }}
{{ upper "hello world" }}
```

**Example:** `"hello world"` -> `"HELLO WORLD"`

---

## lower

Converts string to lowercase.

```go
{{ .name | lower }}
{{ lower "HELLO WORLD" }}
```

**Example:** `"HELLO WORLD"` -> `"hello world"`

---

## title

Capitalizes the first letter of each word; other characters are left unchanged.

```go
{{ .name | title }}
{{ title "hello world" }}
```

**Example:** `"hello world"` -> `"Hello World"`

---

## trim

Trims specified characters from both ends of a string.

```go
{{ trim " " .text }}
{{ trim "." "...hello..." }}
```

**Example:**
```
Input:  " ", "  hello world  "
Output: "hello world"
```

---

## trimSpace

Trims whitespace (spaces, tabs, newlines) from both ends.

```go
{{ .userInput | trimSpace }}
{{ trimSpace "  hello world  " }}
```

**Example:** `"  hello world  \n"` -> `"hello world"`

---

## split

Splits a string by separator into a slice.

```go
{{ split "," "apple,banana,cherry" }}
{{ .tags | split "|" }}
```

Use with `range` to iterate:

```go
{{range split "," .tags}}
- {{.}}
{{end}}
```

---

## join

Joins a string slice with a separator.

```go
{{ join ", " .tags }}
{{ .items | join " | " }}
```

**Example:** `["apple", "banana", "cherry"]` -> `"apple, banana, cherry"`

---

## default

Returns a default value if the input is nil or empty string.

```go
{{ .optionalField | default "N/A" }}
{{ default "Unknown" .name }}
```

**Example:**
```
Input:  "N/A", ""      -> "N/A"
Input:  "N/A", "John"  -> "John"
```

---

## slice

Returns a substring from start to end index (end exclusive). Use `-1` for end of string.

```go
{{ slice 0 5 "hello world" }}
{{ .text | slice 2 8 }}
```

**Example:**
```
Input:  0, 5, "hello world"  -> "hello"
Input:  6, -1, "hello world" -> "world"
```

---

## len

Returns the length of a string, slice, or map.

```go
{{ len .name }}
{{ .items | len }}
```

---

## truncate

Truncates a string to a maximum length (hard cut, no ellipsis).

```go
{{ truncate 10 .longText }}
{{ .description | truncate 50 }}
```

**Example:** `truncate 10 "this is a very long string"` -> `"this is a "`

---

## ellipsis

Truncates a string to a maximum length and adds `...` if truncated.

```go
{{ ellipsis 20 .longText }}
{{ .description | ellipsis 100 }}
```

**Example:**
```
ellipsis 20 "this is a very long string that needs truncation"
-> "this is a very lo..."

ellipsis 10 "short"
-> "short"
```

If maxLen <= 3, no ellipsis is added.

---

## repeat

Repeats a string a specified number of times.

```go
{{ repeat 3 "*" }}
{{ .pattern | repeat 5 }}
```

**Example:** `repeat 3 "*"` -> `"***"`

---

## contains

Checks if a string contains a substring (case-sensitive).

```go
{{ if contains "error" .message }}
  Error found!
{{ end }}
```

---

## hasPrefix

Checks if a string starts with a prefix (case-sensitive).

```go
{{ if hasPrefix "http" .url }}
  Valid URL
{{ end }}
```

---

## hasSuffix

Checks if a string ends with a suffix (case-sensitive).

```go
{{ if hasSuffix ".jpg" .filename }}
  Image file
{{ end }}
```

---

## replace

Replaces all occurrences of a substring with another.

```go
{{ replace " " "_" .name }}
{{ .text | replace "foo" "bar" }}
```

**Example:** `replace " " "_" "hello world test"` -> `"hello_world_test"`

---

## dict

Builds a map from alternating key/value pairs. Keys must be strings. Pipe through `json` to emit a structured object without hand-writing braces (which avoids quoting/escaping bugs).

```go
{{ dict "user" .payload.id "amount" .payload.amount | json }}
```

**Example:** `dict "a" 1 "b" 2 | json` -> `{"a":1,"b":2}`

---

## list

Builds a slice from its arguments. Pipe through `json` to emit an array.

```go
{{ list .payload.a .payload.b | json }}
```

**Example:** `list 1 2 3 | json` -> `[1,2,3]`

---

## append

Appends items to a slice, returning the new slice. Combine with `=` reassignment inside `range` to accumulate an array.

```go
{{ $out := list }}
{{ range .payload.items }}{{ $out = append $out .id }}{{ end }}
{{ $out | json }}
```

---

## merge

Merges maps into a new map. Later keys overwrite earlier ones; inputs are not modified.

```go
{{ merge .payload (dict "source" "sparrow") | json }}
```

---

## add / sub / mul / div / mod

Arithmetic on numbers. Values are coerced from JSON numbers or numeric strings. `add`, `sub`, `mul`, and `div` return floats; `mod` returns an integer. `div` and `mod` error on a zero divisor.

```go
{{ add .payload.subtotal .payload.tax }}
{{ mul .payload.amount 100 }}
{{ div .payload.amount_cents 100 }}
{{ mod .payload.sequence 10 }}
```

**Example:** `mul 5.5 100` -> `550`

---

## dig

Safely reads a nested value from a map by a path of keys, returning the default if any key along the path is missing. Arguments are one or more keys, then a default value, then the map (last). Avoids template errors on optional fields.

```go
{{ dig "customer" "address" "city" "unknown" .payload }}
```

**Example:** on `{"customer":{"id":"c1"}}`, `dig "customer" "email" "none" .payload` -> `"none"`

---

## toString / toInt / toFloat

Type conversion. `toString` renders any value as a string; `toInt` and `toFloat` coerce JSON numbers or numeric strings (`toInt` truncates toward zero).

```go
{{ toString .payload.count }}
{{ dict "count" (toInt .payload.count) | json }}
{{ toFloat .payload.price }}
```
