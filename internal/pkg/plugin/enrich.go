package plugin

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"go.uber.org/zap"
)

// enrichItems fetches the manifest's enrich resource for every item,
// bounded by Enrich.Source.MaxConcurrency, and merges the selected fields
// into each item in place. A failure fetching one item's enrich data is
// logged and that item simply keeps its base fields — it does not fail the
// whole refresh, since a single per-vehicle hiccup (e.g. a transient 500
// on one availability lookup) shouldn't blank out the rest of the fleet.
func enrichItems(ctx context.Context, client *http.Client, e *Enrich, items []Item, logger *zap.Logger) {
	sem := make(chan struct{}, e.Source.MaxConcurrency)
	var wg sync.WaitGroup

	for i := range items {
		idVal, ok := items[i][e.Source.IDField]
		if !ok {
			logger.Warn("enrich skipped: item missing id field",
				zap.String("idField", e.Source.IDField))
			continue
		}
		idStr, err := enrichIDString(idVal)
		if err != nil {
			logger.Warn("enrich skipped: id field is not a scalar",
				zap.String("idField", e.Source.IDField), zap.Error(err))
			continue
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(item Item, id string) {
			defer wg.Done()
			defer func() { <-sem }()

			fetchCtx, cancel := context.WithTimeout(ctx, e.Source.Timeout)
			defer cancel()

			enriched, err := fetchEnrichOne(fetchCtx, client, e, id)
			if err != nil {
				logger.Warn("enrich fetch failed for item", zap.String("id", id), zap.Error(err))
				return
			}
			for k, v := range enriched {
				item[k] = v
			}
		}(items[i], idStr)
	}

	wg.Wait()
}

func fetchEnrichOne(ctx context.Context, client *http.Client, e *Enrich, id string) (Item, error) {
	placeholder := "{" + e.Source.IDField + "}"

	rawURL := strings.ReplaceAll(e.Source.URL, placeholder, url.QueryEscape(id))
	query := make(map[string]string, len(e.Source.Query))
	for k, v := range e.Source.Query {
		query[k] = strings.ReplaceAll(v, placeholder, id)
	}

	parsed, err := fetchURL(ctx, client, rawURL, query, e.Source.Headers)
	if err != nil {
		return nil, err
	}
	return selectFields(parsed, e.Select.Fields), nil
}

// enrichIDString converts a JSON-decoded id value (float64 or string) to
// the string form used in URL/query substitution. JSON numbers decode as
// float64; formatted via strconv to avoid Go's default float formatting
// (e.g. "23672" not "23672.0" or scientific notation for large ids).
func enrichIDString(v interface{}) (string, error) {
	switch t := v.(type) {
	case string:
		return t, nil
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64), nil
	default:
		return "", fmt.Errorf("unsupported id type %T", v)
	}
}
