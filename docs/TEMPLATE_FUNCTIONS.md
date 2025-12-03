# Template Functions Reference

This document describes all available template functions for webhook payload transformation.

# json

Converts any value to a JSON string.

## Usage
```
{{ .data | json }}
{{ json .payload }}
```

## Example
```
Input: map[string]any{"name": "John", "age": 30}
Output: {"name":"John","age":30}
```

---

# urlencode

URL encodes a string by escaping special characters for safe use in URLs.

## Usage
```
{{ .email | urlencode }}
{{ urlencode "hello world" }}
```

## Example
```
Input: "hello world@example.com"

Output: "hello+world%40example.com"
```

---

# base64

Base64 encodes a string.

## Usage
```
{{ .secret | base64 }}
{{ base64 "hello world" }}
```

## Example
```
Input: "hello world"
Output: "aGVsbG8gd29ybGQ="
```

---

# base64decode

Base64 decodes a string back to its original form.

## Usage
```
{{ .encodedSecret | base64decode }}
{{ base64decode "aGVsbG8gd29ybGQ=" }}
```

## Example
```
Input: "aGVsbG8gd29ybGQ="
Output: "hello world"
```

---

# now

Returns the current time as a time.Time object.

## Usage
```
{{ now }}
{{ now | formatTime "2006-01-02 15:04:05" }}
```

## Example
```
Output: 2023-11-21 14:30:45 +0000 UTC
```

---

# formatTime

Formats a time value using Go's time layout format.

## Usage
```
{{ formatTime "2006-01-02" .createdAt }}
{{ .timestamp | formatTime "15:04:05" }}
```

## Common Layouts
- RFC3339: `2006-01-02T15:04:05Z07:00`
- Date only: `2006-01-02`
- Time only: `15:04:05`
- Human readable: `January 2, 2006 at 3:04 PM`

## Example
```
Input: time.Now(), "2006-01-02 15:04:05"
Output: "2023-11-21 14:30:45"
```

---

# parseTime

Parses a time string using Go's time layout format.

## Usage
```
{{ parseTime "2006-01-02" "2023-11-21" }}
{{ parseTime "15:04:05" .timeString }}
```

## Example
```
Input: "2006-01-02", "2023-11-21"
Output: 2023-11-21 00:00:00 +0000 UTC
```

---

# upper

Converts string to uppercase.

## Usage
```
{{ .name | upper }}
{{ upper "hello world" }}
```

## Example
```
Input: "hello world"
Output: "HELLO WORLD"
```

---

# lower

Converts string to lowercase.

## Usage
```
{{ .name | lower }}
{{ lower "HELLO WORLD" }}
```

## Example
```
Input: "HELLO WORLD"
Output: "hello world"
```

---

# title

Converts string to title case (all letters uppercase).

## Usage
```
{{ .name | title }}
{{ title "hello world" }}
```

## Example
```
Input: "hello world"
Output: "HELLO WORLD"
```

Note: This converts ALL letters to uppercase. For proper title case (first letter of each word), use a custom function.

---

# trim

Trims specified characters from both ends of string.

## Usage
```
{{ trim " " .text }}
{{ trim "." "...hello..." }}
```

## Example
```
Input: " ", "  hello world  "
Output: "hello world"

Input: ".", "...hello world..."
Output: "hello world"
```

---

# split

Splits string by separator into a slice.

## Usage
```
{{ split "," "apple,banana,cherry" }}
{{ .tags | split "|" }}
```

## Example
```
Input: ",", "apple,banana,cherry"
Output: ["apple", "banana", "cherry"]
```

Use with range to iterate:
```
{{range split "," .tags}}
- {{.}}
{{end}}
```

---

# join

Joins string slice with separator.

## Usage
```
{{ join ", " .tags }}
{{ .items | join " | " }}
```

## Example
```
Input: ", ", ["apple", "banana", "cherry"]
Output: "apple, banana, cherry"
```

---

# default

Returns default value if the input value is nil or empty string.

## Usage
```
{{ .optionalField | default "N/A" }}
{{ default "Unknown" .name }}
```

## Example
```
Input: "N/A", ""
Output: "N/A"

Input: "N/A", "John"
Output: "John"
```

---

# slice

Returns substring from start to end index (end exclusive).

## Usage
```
{{ slice 0 5 "hello world" }}
{{ .text | slice 2 8 }}
```

## Example
```
Input: 0, 5, "hello world"
Output: "hello"

Input: 6, -1, "hello world"
Output: "world"
```

---

# len

Returns length of string, slice, or map.

## Usage
```
{{ len .name }}
{{ .items | len }}
```

## Example
```
Input: "hello"
Output: 5

Input: ["a", "b", "c"]
Output: 3
```

---

# truncate

Truncates string to maximum length (hard cut, no ellipsis).

## Usage
```
{{ truncate 10 .longText }}
{{ .description | truncate 50 }}
```

## Example
```
Input: 10, "this is a very long string"
Output: "this is a "
```

For truncation with ellipsis, use `ellipsis` function instead.

---

# ellipsis

Truncates string to maximum length and adds "..." if truncated.

## Usage
```
{{ ellipsis 20 .longText }}
{{ .description | ellipsis 100 }}
```

## Example
```
Input: 20, "this is a very long string that needs truncation"
Output: "this is a very lo..."

Input: 10, "short"
Output: "short"
```

If maxLen ≤ 3, no ellipsis is added (hard truncate).

---

# repeat

Repeats string specified number of times.

## Usage
```
{{ repeat 3 "*" }}
{{ .pattern | repeat 5 }}
```

## Example
```
Input: 3, "*"
Output: "***"

Input: 2, "hello"
Output: "hellohello"
```

---

# contains

Checks if string contains substring (case-sensitive).

## Usage
```
{{ if contains "error" .message }}
  Error found!
{{ end }}
```

## Example
```
Input: "world", "hello world"
Output: true

Input: "World", "hello world"
Output: false
```

---

# hasPrefix

Checks if string starts with the specified prefix (case-sensitive).

## Usage
```
{{ if hasPrefix "http" .url }}
  Valid URL
{{ end }}
```

## Example
```
Input: "hello", "hello world"
Output: true

Input: "Hello", "hello world"
Output: false
```

---

# hasSuffix

Checks if string ends with the specified suffix (case-sensitive).

## Usage
```
{{ if hasSuffix ".jpg" .filename }}
  Image file
{{ end }}
```

## Example
```
Input: "world", "hello world"
Output: true

Input: "World", "hello world"
Output: false
```

---

# trimSpace

Trims whitespace (spaces, tabs, newlines) from both ends of string.

## Usage
```
{{ .userInput | trimSpace }}
{{ trimSpace "  hello world  " }}
```

## Example
```
Input: "  hello world  \n"
Output: "hello world"
```

---

# replace

Replaces ALL occurrences of old substring with new substring.

## Usage
```
{{ replace " " "_" .name }}
{{ .text | replace "foo" "bar" }}
```

## Example
```
Input: " ", "_", "hello world test"
Output: "hello_world_test"

Input: "foo", "bar", "foo is foo"
Output: "bar is bar"
```

---

