package plugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every shipped connector must load as a valid manifest and parse its
// template, with the env vars it references unset.
func TestConnectors_LoadAndParse(t *testing.T) {
	paths, err := discover(filepath.Join("..", "..", "..", "connectors"))
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("no connectors found")
	}
	for _, p := range paths {
		m, err := loadManifest(p)
		if err != nil {
			t.Errorf("%s: %v", p, err)
			continue
		}
		if _, err := loadTemplate(m); err != nil {
			t.Errorf("%s: %v", p, err)
		}
	}
}

func TestConnectors_SyscaraRendersSyntheticListing(t *testing.T) {
	m, err := loadManifest(filepath.Join("..", "..", "..", "connectors", "syscara", "plugin.yaml"))
	if err != nil {
		t.Fatalf("loadManifest: %v", err)
	}
	tmpl, err := loadTemplate(m)
	if err != nil {
		t.Fatalf("loadTemplate: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join("testdata", "syscara-ads.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	items, err := selectItems(raw, m.Select)
	if err != nil {
		t.Fatalf("selectItems: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	for _, it := range items {
		for _, secret := range []string{"prices.purchase", "prices.b2b", "accounting", "user.seller", "store"} {
			if _, leaked := it[secret]; leaked {
				t.Errorf("%s leaked into item", secret)
			}
		}
	}
	items[0]["images"] = []Item{{"id": float64(11), "file": "https://cdn.example.com/aa/1500.jpg"}}

	html, err := renderHTML(tmpl, items)
	if err != nil {
		t.Fatalf("renderHTML: %v", err)
	}
	for _, want := range []string{
		"Example Maker Series 600",           // title
		"45.990 €",                           // formatNumber on prices.offer
		"EZ 03/2022",                         // date.registration
		"32.500 km",                          // mileage
		"https://cdn.example.com/aa/500.jpg", // card image resized
		"Neufahrzeug",                        // second listing is NEW
		"Preis auf Anfrage",                  // second listing has no price
	} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered fragment missing %q", want)
		}
	}
	if strings.Contains(html, "31000") || strings.Contains(html, "31.000") {
		t.Error("purchase price reached the rendered fragment")
	}
}
