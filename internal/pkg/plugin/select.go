package plugin

import (
	"fmt"
	"strings"
)

// Item is one allowlisted record ready for rendering. Keys are the exact
// dot paths from the manifest's select.fields, e.g. "images.outside.medium".
// Values keep their original JSON type (string, float64, bool, []interface{},
// map[string]interface{}) — the render template decides how to use them.
type Item map[string]interface{}

// selectItems extracts the array at select.root from the parsed response,
// then applies the field allowlist to every element. Elements that are not
// JSON objects are skipped. Never returns an error for a missing/absent
// field — a manifest referencing a field the upstream doesn't always send
// should still render the fields that are present.
func selectItems(parsed interface{}, sel Select) ([]Item, error) {
	root := parsed
	if sel.Root != "" {
		v, ok := getPath(parsed, sel.Root)
		if !ok {
			return nil, fmt.Errorf("select.root %q not found in response", sel.Root)
		}
		root = v
	}

	list, ok := root.([]interface{})
	if !ok {
		return nil, fmt.Errorf("select.root %q does not resolve to an array (got %T)", sel.Root, root)
	}

	items := make([]Item, 0, len(list))
	for _, raw := range list {
		obj, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		items = append(items, selectFields(obj, sel.Fields))
	}
	return items, nil
}

// selectFields extracts an allowlisted set of top-level (dot-path) fields
// from a single parsed JSON object — used for enrich responses, which are
// one object per item rather than a list. Missing fields are silently
// omitted, matching selectItems' behavior for the base list.
func selectFields(parsed interface{}, fields []string) Item {
	item := make(Item, len(fields))
	for _, field := range fields {
		if v, ok := getPath(parsed, field); ok {
			item[field] = v
		}
	}
	return item
}

// getPath navigates a dot path through nested map[string]interface{} values
// (as produced by encoding/json). It does not descend into arrays — a
// manifest field path that would cross an array boundary is not supported
// in v1 and simply returns not-found.
func getPath(v interface{}, path string) (interface{}, bool) {
	cur := v
	for _, seg := range strings.Split(path, ".") {
		m, ok := cur.(map[string]interface{})
		if !ok {
			return nil, false
		}
		next, ok := m[seg]
		if !ok {
			return nil, false
		}
		cur = next
	}
	return cur, true
}
