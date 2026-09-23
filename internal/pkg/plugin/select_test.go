package plugin

import (
	"encoding/json"
	"testing"
)

func mustParse(t *testing.T, s string) interface{} {
	t.Helper()
	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	return v
}

func TestSelectItems_Allowlist(t *testing.T) {
	body := `{
		"articles": [
			{"id": 1, "make": "CARADO", "secretInternalField": "should-not-leak",
			 "images": {"outside": {"medium": "https://cdn.example.com/a.png"}}},
			{"id": 2, "make": "KNAUS", "secretInternalField": "also-hidden",
			 "images": {"outside": {"medium": "https://cdn.example.com/b.png"}}}
		]
	}`
	parsed := []byte(body)

	items, err := selectItems(parsed, Select{
		Root:   "articles",
		Fields: []string{"id", "make", "images.outside.medium"},
	})
	if err != nil {
		t.Fatalf("selectItems: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	for _, item := range items {
		if _, leaked := item["secretInternalField"]; leaked {
			t.Errorf("field not in allowlist leaked into item: %v", item)
		}
		if _, ok := item["images.outside.medium"]; !ok {
			t.Errorf("allowlisted nested field missing: %v", item)
		}
	}
	if items[0]["make"] != "CARADO" {
		t.Errorf("make = %v, want CARADO", items[0]["make"])
	}
}

func TestSelectItems_RootIsResponseItself(t *testing.T) {
	parsed := []byte(`[{"id": 1}, {"id": 2}]`)
	items, err := selectItems(parsed, Select{Fields: []string{"id"}})
	if err != nil {
		t.Fatalf("selectItems: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
}

func TestSelectItems_MissingRootErrors(t *testing.T) {
	parsed := []byte(`{"other": []}`)
	_, err := selectItems(parsed, Select{Root: "articles", Fields: []string{"id"}})
	if err == nil {
		t.Fatal("expected error for missing select.root")
	}
}

func TestSelectItems_MissingFieldOmittedNotError(t *testing.T) {
	parsed := []byte(`[{"id": 1}]`)
	items, err := selectItems(parsed, Select{Fields: []string{"id", "doesNotExist"}})
	if err != nil {
		t.Fatalf("selectItems: %v", err)
	}
	if _, ok := items[0]["doesNotExist"]; ok {
		t.Error("missing field should be omitted, not present as nil")
	}
	if items[0]["id"] != float64(1) {
		t.Errorf("id = %v, want 1", items[0]["id"])
	}
}

func TestSelectItems_KeyedCollectionKeepsResponseOrder(t *testing.T) {
	// Keys deliberately out of lexical and numeric order: the upstream's
	// order is the order to render, and a Go map would scramble it.
	body := `{"ads": {
		"30": {"id": 30, "title": "third-lexically", "internalNote": "hidden"},
		"4":  {"id": 4,  "title": "first-numerically"},
		"12": {"id": 12, "title": "middle"}
	}}`
	items, err := selectItems([]byte(body), Select{Root: "ads", Fields: []string{"id", "title"}})
	if err != nil {
		t.Fatalf("selectItems: %v", err)
	}
	var got []interface{}
	for _, it := range items {
		got = append(got, it["id"])
		if _, leaked := it["internalNote"]; leaked {
			t.Errorf("field not in allowlist leaked: %v", it)
		}
	}
	want := []interface{}{float64(30), float64(4), float64(12)}
	if len(got) != len(want) {
		t.Fatalf("ids = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ids = %v, want %v (response key order not kept)", got, want)
		}
	}
}

func TestSelectItems_KeyedCollectionAtResponseRoot(t *testing.T) {
	items, err := selectItems([]byte(`{"b": {"id": "b"}, "a": {"id": "a"}}`), Select{Fields: []string{"id"}})
	if err != nil {
		t.Fatalf("selectItems: %v", err)
	}
	if len(items) != 2 || items[0]["id"] != "b" || items[1]["id"] != "a" {
		t.Fatalf("items = %v, want [b a]", items)
	}
}

func TestSelectItems_WrapperObjectIsNotACollection(t *testing.T) {
	// A root configured one level too high: an object mixing a list with a
	// scalar must fail loudly, not render an empty fragment.
	_, err := selectItems([]byte(`{"items": [{"id": 1}], "total": 1}`), Select{Fields: []string{"id"}})
	if err == nil {
		t.Fatal("expected error for an object whose values are not all objects")
	}
}

func TestSelectItems_ScalarRootErrors(t *testing.T) {
	_, err := selectItems([]byte(`{"count": 3}`), Select{Root: "count", Fields: []string{"id"}})
	if err == nil {
		t.Fatal("expected error for a scalar select.root")
	}
}

func TestGetPath(t *testing.T) {
	v := mustParse(t, `{"a": {"b": {"c": "deep"}}}`)

	got, ok := getPath(v, "a.b.c")
	if !ok || got != "deep" {
		t.Errorf("getPath a.b.c = (%v, %v), want (deep, true)", got, ok)
	}

	_, ok = getPath(v, "a.b.missing")
	if ok {
		t.Error("getPath should not find a.b.missing")
	}

	_, ok = getPath(v, "a.b.c.d") // c is a string, cannot descend further
	if ok {
		t.Error("getPath should not descend into a scalar")
	}
}
