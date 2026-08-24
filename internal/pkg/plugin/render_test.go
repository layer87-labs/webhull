package plugin

import (
	"html"
	"html/template"
	"strings"
	"testing"
)

func TestRenderHTML_ToJSON_EscapesForAttributeContext(t *testing.T) {
	tmpl, err := template.New("t").Funcs(funcMap).Parse(
		`{{range .Items}}<button data-vehicle='{{toJSON .}}'></button>{{end}}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	items := []Item{{
		"make":  `KNAUS "Special" <script>alert(1)</script>`,
		"model": "T135",
	}}

	out, err := renderHTML(tmpl, items)
	if err != nil {
		t.Fatalf("renderHTML: %v", err)
	}

	if strings.Contains(out, "<script>alert(1)</script>") {
		t.Fatalf("raw <script> leaked into output unescaped: %s", out)
	}
	if strings.Contains(out, `data-vehicle='{"make":"KNAUS \"Special\"`) {
		t.Fatalf("unescaped single-quote-breaking content in attribute: %s", out)
	}
	// html/template must have re-escaped the JSON string for the
	// single-quoted attribute context (e.g. ' -> &#39;), not left it as
	// literal JSON syntax that could break out of the attribute.
	if !strings.Contains(out, "data-vehicle=") {
		t.Fatalf("expected data-vehicle attribute in output: %s", out)
	}
}

func TestRenderHTML_ToJSON_RoundTripsInAttribute(t *testing.T) {
	// html/template HTML-escapes every text node, including plain "quote"
	// characters outside an attribute — that is correct, expected behavior,
	// not a bug in toJSON. The real usage is inside an HTML attribute, where
	// the browser un-escapes entities before JS ever sees the string, so
	// JSON.parse(el.getAttribute(...)) gets valid JSON back. Assert that
	// round trip via html.UnescapeString instead of expecting raw JSON in
	// the rendered byte stream.
	tmpl, err := template.New("t").Funcs(funcMap).Parse(
		`{{range .Items}}<button data-vehicle='{{toJSON .}}'></button>{{end}}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	items := []Item{{"id": float64(23672), "make": "CARADO"}}

	out, err := renderHTML(tmpl, items)
	if err != nil {
		t.Fatalf("renderHTML: %v", err)
	}

	start := strings.Index(out, "data-vehicle='") + len("data-vehicle='")
	end := strings.Index(out[start:], "'")
	attr := html.UnescapeString(out[start : start+end])

	if attr != `{"id":23672,"make":"CARADO"}` {
		t.Fatalf("attribute did not round-trip to valid JSON: %q (from %q)", attr, out)
	}
}
