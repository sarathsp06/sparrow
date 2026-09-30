package webhooks

import (
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"
)

// SchemaCompatibility is the result of comparing two versions of an event
// type's JSON Schema from a subscriber's point of view.
type SchemaCompatibility struct {
	// Breaking is true when the change could break a subscription's payload
	// transformation.
	Breaking bool
	// Reasons lists each breaking change with the path it affects, e.g.
	// "order.total: removed required property".
	Reasons []string
}

// Result is "breaking" or "compatible".
func (c SchemaCompatibility) Result() string {
	if c.Breaking {
		return "breaking"
	}
	return "compatible"
}

// compositionKeywords are JSON Schema keywords whose effect on the payload
// shape the classifier cannot reason about. A change under a schema that uses
// them cannot be proven safe, so it counts as breaking.
var compositionKeywords = []string{"oneOf", "anyOf", "allOf", "not", "$ref", "patternProperties", "if", "then", "else", "dependentSchemas"}

// ClassifySchemaChange decides whether moving an event type from oldSchema to
// newSchema could break a subscription's payload transformation. The rule is
// strict: anything the classifier cannot prove safe is breaking.
//
// Compatible:
//   - adding a property, optional or required (a subscriber only reads what
//     it already knew about)
//   - removing an optional property (the schema never promised it)
//   - narrowing a type (number to integer, or adding a type where there was none)
//   - changing value constraints: enum, minimum, maxLength, pattern, format, ...
//
// Breaking:
//   - removing a required property, or making it optional
//   - a type change that allows values of a type the old schema did not
//     (string to integer, object to array, integer to number)
//   - removing the schema entirely
//   - any change under oneOf, anyOf, allOf, not, $ref, patternProperties,
//     if/then/else or dependentSchemas
//
// Adding a first schema to a type that had none is not a change to a
// contract, so it is never breaking.
func ClassifySchemaChange(oldSchema, newSchema map[string]any) SchemaCompatibility {
	var c SchemaCompatibility
	switch {
	case len(oldSchema) == 0:
		return c
	case len(newSchema) == 0:
		c.add("", "schema removed; every required property is gone")
		return c
	}
	c.compare("", oldSchema, newSchema)
	sort.Strings(c.Reasons)
	return c
}

func (c *SchemaCompatibility) add(path, reason string) {
	c.Breaking = true
	if path == "" {
		path = "(root)"
	}
	c.Reasons = append(c.Reasons, path+": "+reason)
}

// compare checks one schema node and recurses into properties and items.
func (c *SchemaCompatibility) compare(path string, oldNode, newNode map[string]any) {
	if reflect.DeepEqual(oldNode, newNode) {
		return
	}
	if kw := firstComposition(oldNode, newNode); kw != "" {
		c.add(path, fmt.Sprintf("changed under %s, which cannot be checked for compatibility", kw))
		return
	}

	if reason := typeChange(oldNode, newNode); reason != "" {
		c.add(path, reason)
	}

	oldProps, _ := oldNode["properties"].(map[string]any)
	newProps, _ := newNode["properties"].(map[string]any)
	oldReq, newReq := requiredSet(oldNode), requiredSet(newNode)
	for _, name := range sortedKeys(oldProps) {
		propPath := joinPath(path, name)
		oldProp, _ := oldProps[name].(map[string]any)
		newPropRaw, stillThere := newProps[name]
		wasRequired, isRequired := oldReq[name], newReq[name]

		switch {
		case !stillThere && !wasRequired:
			// Removed optional property: the schema never promised it.
			continue
		case !stillThere && !isRequired:
			c.add(propPath, "removed required property")
			continue
		case !stillThere:
			c.add(propPath, "property definition removed, so its shape is no longer described")
			continue
		case wasRequired && !isRequired:
			c.add(propPath, "no longer required, so it may be absent")
		}
		newProp, _ := newPropRaw.(map[string]any)
		c.compare(propPath, oldProp, newProp)
	}

	if oldItems, ok := oldNode["items"].(map[string]any); ok {
		if newItems, ok := newNode["items"].(map[string]any); ok {
			c.compare(path+"[]", oldItems, newItems)
		} else {
			c.add(path+"[]", "item schema removed, so items may have any shape")
		}
	}
}

// typeChange reports a type change that lets through values the old schema
// did not, or "" if the new types are a subset of the old ones.
func typeChange(oldNode, newNode map[string]any) string {
	oldTypes, newTypes := typesOf(oldNode), typesOf(newNode)
	if len(oldTypes) == 0 {
		// Old schema allowed any type: whatever the new one says is narrower.
		return ""
	}
	if len(newTypes) == 0 {
		return fmt.Sprintf("type %s removed, so any type is now allowed", strings.Join(oldTypes, "|"))
	}
	var widened []string
	for _, t := range newTypes {
		if !typeAllows(oldTypes, t) {
			widened = append(widened, t)
		}
	}
	if len(widened) == 0 {
		return ""
	}
	return fmt.Sprintf("type changed from %s to %s", strings.Join(oldTypes, "|"), strings.Join(newTypes, "|"))
}

// typeAllows reports whether a node typed as `types` accepts values of type t.
// An integer is a number, so "number" accepts "integer" but not the reverse.
func typeAllows(types []string, t string) bool {
	if len(types) == 0 {
		return true
	}
	if slices.Contains(types, t) {
		return true
	}
	return t == "integer" && slices.Contains(types, "number")
}

func typesOf(node map[string]any) []string {
	switch t := node["type"].(type) {
	case string:
		return []string{t}
	case []any:
		out := make([]string, 0, len(t))
		for _, v := range t {
			if s, ok := v.(string); ok {
				out = append(out, s)
			}
		}
		sort.Strings(out)
		return out
	}
	return nil
}

func requiredSet(node map[string]any) map[string]bool {
	set := map[string]bool{}
	switch list := node["required"].(type) {
	case []any:
		for _, v := range list {
			if s, ok := v.(string); ok {
				set[s] = true
			}
		}
	case []string:
		for _, s := range list {
			set[s] = true
		}
	}
	return set
}

// firstComposition returns the first composition keyword present in either
// node, or "".
func firstComposition(a, b map[string]any) string {
	for _, kw := range compositionKeywords {
		if _, ok := a[kw]; ok {
			return kw
		}
		if _, ok := b[kw]; ok {
			return kw
		}
	}
	return ""
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func joinPath(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
}
