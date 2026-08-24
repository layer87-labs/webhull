package plugin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestEnrichItems_MergesPerItemData(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"minDate":"2026-08-26","availableDays":["2026-08-26","2026-08-27"],"echoId":"` + id + `"}`))
	}))
	defer upstream.Close()

	e := &Enrich{
		Source: EnrichSource{
			IDField:        "id",
			URL:            upstream.URL + "/avail",
			Query:          map[string]string{"id": "{id}"},
			Timeout:        2 * time.Second,
			MaxConcurrency: 5,
		},
		Select: EnrichSelect{Fields: []string{"minDate", "availableDays", "echoId"}},
	}

	items := []Item{
		{"id": float64(1)},
		{"id": float64(2)},
	}

	enrichItems(t.Context(), &http.Client{}, e, items, zap.NewNop())

	for _, item := range items {
		if item["minDate"] != "2026-08-26" {
			t.Errorf("minDate not merged: %v", item)
		}
		wantEcho := enrichIDStringMust(t, item["id"])
		if item["echoId"] != wantEcho {
			t.Errorf("echoId = %v, want %v (per-item id substitution failed)", item["echoId"], wantEcho)
		}
		days, ok := item["availableDays"].([]interface{})
		if !ok || len(days) != 2 {
			t.Errorf("availableDays not merged correctly: %v", item["availableDays"])
		}
	}
}

func enrichIDStringMust(t *testing.T, v interface{}) string {
	t.Helper()
	s, err := enrichIDString(v)
	if err != nil {
		t.Fatalf("enrichIDString: %v", err)
	}
	return s
}

func TestEnrichItems_OneFailureDoesNotBlockOthers(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("id") == "1" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"minDate":"2026-08-26"}`))
	}))
	defer upstream.Close()

	e := &Enrich{
		Source: EnrichSource{
			IDField:        "id",
			URL:            upstream.URL + "/avail",
			Query:          map[string]string{"id": "{id}"},
			Timeout:        2 * time.Second,
			MaxConcurrency: 5,
		},
		Select: EnrichSelect{Fields: []string{"minDate"}},
	}

	items := []Item{{"id": float64(1)}, {"id": float64(2)}}
	enrichItems(t.Context(), &http.Client{}, e, items, zap.NewNop())

	if _, ok := items[0]["minDate"]; ok {
		t.Error("item 1's fetch failed — it should not have minDate merged")
	}
	if items[1]["minDate"] != "2026-08-26" {
		t.Error("item 2's fetch succeeded independently — expected minDate merged")
	}
}

func TestEnrichItems_RespectsMaxConcurrency(t *testing.T) {
	var inFlight int32
	var maxObserved int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cur := atomic.AddInt32(&inFlight, 1)
		defer atomic.AddInt32(&inFlight, -1)
		for {
			old := atomic.LoadInt32(&maxObserved)
			if cur <= old || atomic.CompareAndSwapInt32(&maxObserved, old, cur) {
				break
			}
		}
		time.Sleep(30 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"minDate":"2026-08-26"}`))
	}))
	defer upstream.Close()

	e := &Enrich{
		Source: EnrichSource{
			IDField:        "id",
			URL:            upstream.URL + "/avail",
			Query:          map[string]string{"id": "{id}"},
			Timeout:        2 * time.Second,
			MaxConcurrency: 2,
		},
		Select: EnrichSelect{Fields: []string{"minDate"}},
	}

	items := make([]Item, 8)
	for i := range items {
		items[i] = Item{"id": float64(i)}
	}

	enrichItems(t.Context(), &http.Client{}, e, items, zap.NewNop())

	if got := atomic.LoadInt32(&maxObserved); got > 2 {
		t.Errorf("observed %d concurrent enrich requests, want <= 2 (MaxConcurrency)", got)
	}
}

func TestValidateEnrich_RequiresPlaceholder(t *testing.T) {
	dir := t.TempDir()
	bad := `
apiVersion: webhull.layer87.de/v1
kind: HTTPDataSource
name: noplaceholder
source: { url: https://api.example.com/v1/items }
select: { fields: [id] }
render:
  template: fragment.tmpl.html
  into: { page: vermietung, contentKey: fleet }
enrich:
  source:
    url: https://api.example.com/v1/availability
  select:
    fields: [minDate]
`
	writeManifest(t, dir, "noplaceholder", bad)

	_, err := loadManifest(dir + "/noplaceholder/plugin.yaml")
	if err == nil || !strings.Contains(err.Error(), "placeholder") {
		t.Fatalf("expected placeholder validation error, got: %v", err)
	}
}

func TestValidateEnrich_LiteralSecretRejected(t *testing.T) {
	dir := t.TempDir()
	bad := `
apiVersion: webhull.layer87.de/v1
kind: HTTPDataSource
name: enrichleak
source: { url: https://api.example.com/v1/items }
select: { fields: [id] }
render:
  template: fragment.tmpl.html
  into: { page: vermietung, contentKey: fleet }
enrich:
  source:
    url: https://api.example.com/v1/items/{id}/availability
    headers:
      Authorization: "Bearer literal-secret"
  select:
    fields: [minDate]
`
	writeManifest(t, dir, "enrichleak", bad)

	_, err := loadManifest(dir + "/enrichleak/plugin.yaml")
	if err == nil {
		t.Fatal("expected error for literal secret in enrich.source.headers")
	}
}
