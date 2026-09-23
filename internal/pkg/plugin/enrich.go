package plugin

import (
	"context"
	"encoding/json"
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
		idStr, err := enrichID(items[i], e.Source.IDField)
		if err != nil {
			logger.Warn("enrich skipped", zap.String("idField", e.Source.IDField), zap.Error(err))
			continue
		}
		if idStr == "" {
			continue // an empty list: nothing to look up
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

	raw, err := fetchURL(ctx, client, request{URL: rawURL, Query: query, Headers: e.Source.Headers, Auth: e.Source.Auth})
	if err != nil {
		return nil, err
	}

	if e.Select.As == "" {
		var parsed interface{}
		if err = json.Unmarshal(raw, &parsed); err != nil {
			return nil, fmt.Errorf("parse enrich response: %w", err)
		}
		return selectFields(parsed, e.Select.Fields), nil
	}

	list, err := collectionAt(raw, e.Select.Root)
	if err != nil {
		return nil, fmt.Errorf("enrich: %w", err)
	}
	selected := make([]Item, 0, len(list))
	for _, el := range list {
		if obj, ok := el.(map[string]interface{}); ok {
			selected = append(selected, selectFields(obj, e.Select.Fields))
		}
	}
	return Item{e.Select.As: selected}, nil
}

// enrichID resolves an item's idField to the string substituted into the
// enrich placeholder. "media[].id" joins a sub path of every element of the
// list field "media" with commas, "tags[]" joins the elements themselves;
// an empty list yields "". Any other idField is a plain scalar field.
func enrichID(item Item, idField string) (string, error) {
	base, sub, isList := strings.Cut(idField, "[]")
	if !isList {
		v, ok := item[idField]
		if !ok {
			return "", fmt.Errorf("item missing id field")
		}
		return enrichIDString(v)
	}

	v, ok := item[base]
	if !ok {
		return "", fmt.Errorf("item missing list field %q", base)
	}
	list, ok := v.([]interface{})
	if !ok {
		return "", fmt.Errorf("field %q is not a list", base)
	}
	sub = strings.TrimPrefix(sub, ".")
	ids := make([]string, 0, len(list))
	for _, el := range list {
		if sub != "" {
			var found bool
			if el, found = getPath(el, sub); !found {
				continue
			}
		}
		s, err := enrichIDString(el)
		if err != nil {
			return "", err
		}
		ids = append(ids, s)
	}
	return strings.Join(ids, ","), nil
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
