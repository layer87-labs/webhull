package plugin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Item is one allowlisted record ready for rendering. Keys are the exact
// dot paths from the manifest's select.fields, e.g. "images.outside.medium".
// Values keep their original JSON type (string, float64, bool, []interface{},
// map[string]interface{}) — the render template decides how to use them.
type Item map[string]interface{}

// selectItems extracts the collection at select.root from the raw response,
// then applies the field allowlist to every element. Elements that are not
// JSON objects are skipped. Never returns an error for a missing/absent
// field — a manifest referencing a field the upstream doesn't always send
// should still render the fields that are present.
func selectItems(raw []byte, sel Select) ([]Item, error) {
	list, err := collectionAt(raw, sel.Root)
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(list))
	for _, el := range list {
		obj, ok := el.(map[string]interface{})
		if !ok {
			continue
		}
		items = append(items, selectFields(obj, sel.Fields))
	}
	return items, nil
}

// collectionAt resolves the dot path root in a raw JSON document and returns
// the collection found there as a list. A collection is an array, or an
// object whose values are all objects (records keyed by id); the latter is
// returned in the key order of the document — the order the upstream chose,
// which decoding into a Go map would lose.
func collectionAt(raw []byte, root string) ([]interface{}, error) {
	node := json.RawMessage(raw)
	if root != "" {
		for _, seg := range strings.Split(root, ".") {
			var obj map[string]json.RawMessage
			if err := json.Unmarshal(node, &obj); err != nil {
				return nil, fmt.Errorf("select.root %q not found in response", root)
			}
			next, ok := obj[seg]
			if !ok {
				return nil, fmt.Errorf("select.root %q not found in response", root)
			}
			node = next
		}
	}

	switch firstByte(node) {
	case '[':
		var list []interface{}
		if err := json.Unmarshal(node, &list); err != nil {
			return nil, fmt.Errorf("select.root %q: %w", root, err)
		}
		return list, nil
	case '{':
		list, err := orderedObjectValues(node)
		if err != nil {
			return nil, fmt.Errorf("select.root %q: %w", root, err)
		}
		return list, nil
	default:
		return nil, fmt.Errorf("select.root %q does not resolve to a collection (array or object of objects)", root)
	}
}

// orderedObjectValues returns an object's values in document order. It
// fails unless every value is itself an object: an object mixing records
// with scalars or arrays is a wrapper ({"items": [...], "total": 3}) whose
// root was configured one level too high, not a keyed collection, and
// silently rendering nothing would hide that.
func orderedObjectValues(raw []byte) ([]interface{}, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if _, err := dec.Token(); err != nil { // opening '{'
		return nil, err
	}
	var values []interface{}
	for dec.More() {
		key, err := dec.Token()
		if err != nil {
			return nil, err
		}
		var v interface{}
		if err := dec.Decode(&v); err != nil {
			return nil, err
		}
		if _, ok := v.(map[string]interface{}); !ok {
			return nil, fmt.Errorf("does not resolve to a collection: value under key %q is not an object", key)
		}
		values = append(values, v)
	}
	return values, nil
}

func firstByte(raw []byte) byte {
	trimmed := bytes.TrimLeft(raw, " \t\r\n")
	if len(trimmed) == 0 {
		return 0
	}
	return trimmed[0]
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
