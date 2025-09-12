package searchindex

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	weaviate "github.com/weaviate/weaviate-go-client/v5/weaviate"
	filters "github.com/weaviate/weaviate-go-client/v5/weaviate/filters"
	gql "github.com/weaviate/weaviate-go-client/v5/weaviate/graphql"

	"github.com/mycelian/mycelian-memory/server/internal/model"
)

// weavNative is a native implementation of Index using the Weaviate Go client.
type weavNative struct {
	client  *weaviate.Client
	baseURL string // host:port without scheme
}

// NewWeaviateNativeIndex constructs an Index backed by Weaviate at baseURL.
// baseURL should be host:port (without scheme), e.g., "localhost:8081".
func NewWeaviateNativeIndex(baseURL string) (Index, error) {
	cfg := weaviate.Config{Scheme: "http", Host: baseURL}
	cl, err := weaviate.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &weavNative{client: cl, baseURL: baseURL}, nil
}

func (w *weavNative) Search(ctx context.Context, actorID string, memoryID, query string, vec []float32, topKE int, alpha float32, includeRawEntries bool) ([]model.SearchHit, error) {
	log.Info().Str("memoryId", memoryID).Str("query", query).Str("actorID", actorID).Int("topKE", topKE).Float32("alpha", alpha).Int("vectorLength", len(vec)).Msg("weaviate search starting")

	// helper to safely extract strings
	safeString := func(v interface{}) string {
		s, _ := v.(string)
		return s
	}

	hy := (&gql.HybridArgumentBuilder{}).
		WithQuery(query).
		WithVector(vec).
		WithAlpha(alpha).
		WithProperties([]string{"summary", "rawEntry"})

	where := filters.Where().WithPath([]string{"memoryId"}).WithOperator(filters.Equal).WithValueText(memoryID)

	req := w.client.GraphQL().Get().
		WithClassName("MemoryEntry").
		WithWhere(where).
		WithHybrid(hy).
		WithLimit(topKE).
		WithFields(
			gql.Field{Name: "entryId"},
			gql.Field{Name: "actorId"},
			gql.Field{Name: "memoryId"},
			gql.Field{Name: "summary"},
			gql.Field{Name: "rawEntry"},
			gql.Field{Name: "creationTime"},
			gql.Field{Name: "_additional", Fields: []gql.Field{{Name: "score"}}},
		)

	log.Debug().Msg("executing weaviate graphql query")
	resp, err := req.Do(ctx)
	if err != nil {
		log.Error().Err(err).Str("memoryId", memoryID).Msg("weaviate graphql query failed")
		return nil, err
	}
	if len(resp.Errors) > 0 {
		log.Error().Interface("errors", resp.Errors).Str("memoryId", memoryID).Msg("weaviate graphql errors")
		return nil, fmt.Errorf("weaviate graphql: %s", formatGraphQLErrors(resp.Errors))
	}

	getData, ok := resp.Data["Get"].(map[string]interface{})
	if !ok {
		log.Warn().Str("memoryId", memoryID).Msg("weaviate response has no Get data")
		return nil, nil
	}
	memVal := getData["MemoryEntry"]
	if memVal == nil {
		log.Info().Str("memoryId", memoryID).Msg("weaviate returned no MemoryEntry results")
		return []model.SearchHit{}, nil
	}
	raw, ok := memVal.([]interface{})
	if !ok {
		log.Warn().Str("memoryId", memoryID).Interface("memVal", memVal).Msg("MemoryEntry is not an array")
		return nil, nil
	}

	log.Info().Int("rawResultCount", len(raw)).Str("memoryId", memoryID).Msg("weaviate returned results")

	out := make([]model.SearchHit, 0, len(raw))
	for _, item := range raw {
		m := item.(map[string]interface{})
		var score float64
		if add, ok := m["_additional"].(map[string]interface{}); ok {
			switch v := add["score"].(type) {
			case float64:
				score = v
			case string:
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					score = f
				}
			}
		}
		// Parse creation time
		var creationTime time.Time
		if ctStr := safeString(m["creationTime"]); ctStr != "" {
			creationTime, _ = time.Parse(time.RFC3339, ctStr)
		}

		rawEntry := safeString(m["rawEntry"])
		// Clear raw entry if not requested to save tokens
		if !includeRawEntries {
			rawEntry = ""
		}

		hit := model.SearchHit{
			EntryID:      safeString(m["entryId"]),
			ActorID:      safeString(m["actorId"]),
			MemoryID:     safeString(m["memoryId"]),
			Summary:      safeString(m["summary"]),
			RawEntry:     rawEntry,
			Score:        score,
			CreationTime: creationTime,
		}
		log.Debug().Str("entryId", hit.EntryID).Str("summary", hit.Summary).Float64("score", score).Msg("search hit")
		out = append(out, hit)
	}
	log.Info().Int("finalResultCount", len(out)).Str("memoryId", memoryID).Msg("weaviate search completed")
	return out, nil
}

func (w *weavNative) LatestContext(ctx context.Context, actorID string, memoryID string) (string, time.Time, error) {
	where := filters.Where().WithPath([]string{"memoryId"}).WithOperator(filters.Equal).WithValueText(memoryID)
	req := w.client.GraphQL().Get().
		WithClassName("MemoryContext").
		WithWhere(where).
		WithSort(gql.Sort{Path: []string{"creationTime"}, Order: gql.Desc}).
		WithLimit(1).
		WithFields(
			gql.Field{Name: "context"},
			gql.Field{Name: "creationTime"},
		)
	resp, err := req.Do(ctx)
	if err != nil {
		return "", time.Time{}, err
	}
	if len(resp.Errors) > 0 {
		return "", time.Time{}, fmt.Errorf("weaviate graphql: %s", formatGraphQLErrors(resp.Errors))
	}
	getData, ok := resp.Data["Get"].(map[string]interface{})
	if !ok {
		return "", time.Time{}, nil
	}
	memVal := getData["MemoryContext"]
	if memVal == nil {
		return "", time.Time{}, nil
	}
	arr, ok := memVal.([]interface{})
	if !ok || len(arr) == 0 {
		return "", time.Time{}, nil
	}
	item := arr[0].(map[string]interface{})
	tsStr, _ := item["creationTime"].(string)
	ts, _ := time.Parse(time.RFC3339, tsStr)
	var ctxStr string
	switch v := item["context"].(type) {
	case string:
		ctxStr = v
	case map[string]interface{}, []interface{}:
		if b, err := json.Marshal(v); err == nil {
			ctxStr = string(b)
		}
	}
	return ctxStr, ts, nil
}

// SearchContexts returns top-K matching context shards for a query.
func (w *weavNative) SearchContexts(ctx context.Context, actorID, memoryID, query string, vec []float32, topKC int, alpha float32) ([]model.ContextHit, error) {
	hy := (&gql.HybridArgumentBuilder{}).
		WithQuery(query).
		WithVector(vec).
		WithAlpha(alpha).
		WithProperties([]string{"context"})

	where := filters.Where().WithPath([]string{"memoryId"}).WithOperator(filters.Equal).WithValueText(memoryID)
	req := w.client.GraphQL().Get().
		WithClassName("MemoryContext").
		WithWhere(where).
		WithHybrid(hy).
		WithLimit(topKC).
		WithFields(
			gql.Field{Name: "context"},
			gql.Field{Name: "creationTime"},
			gql.Field{Name: "_additional", Fields: []gql.Field{{Name: "score"}}},
		)
	resp, err := req.Do(ctx)
	if err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("weaviate graphql: %s", formatGraphQLErrors(resp.Errors))
	}
	getData, ok := resp.Data["Get"].(map[string]interface{})
	if !ok {
		return nil, nil
	}
	val := getData["MemoryContext"]
	if val == nil {
		return []model.ContextHit{}, nil
	}
	arr, ok := val.([]interface{})
	if !ok {
		return nil, nil
	}
	out := make([]model.ContextHit, 0, len(arr))
	for _, it := range arr {
		m := it.(map[string]interface{})
		ctxText, _ := m["context"].(string)
		tsStr, _ := m["creationTime"].(string)
		ts, _ := time.Parse(time.RFC3339, tsStr)
		var score float64
		if add, ok := m["_additional"].(map[string]interface{}); ok {
			switch v := add["score"].(type) {
			case float64:
				score = v
			case string:
				f, _ := strconv.ParseFloat(v, 64)
				score = f
			}
		}
		out = append(out, model.ContextHit{Context: ctxText, Timestamp: ts, Score: score})
	}
	return out, nil
}

func (w *weavNative) DeleteEntry(ctx context.Context, actorID string, entryID string) error {
	if w == nil || w.client == nil || entryID == "" {
		return nil
	}
	_ = w.client.Data().Deleter().WithClassName("MemoryEntry").WithID(entryID).Do(ctx)
	return nil
}

func (w *weavNative) DeleteContext(ctx context.Context, actorID string, contextID string) error {
	if w == nil || w.client == nil || contextID == "" {
		return nil
	}
	_ = w.client.Data().Deleter().WithClassName("MemoryContext").WithID(contextID).Do(ctx)
	return nil
}

func (w *weavNative) DeleteMemory(ctx context.Context, actorID string, memoryID string) error {
	if w == nil || w.client == nil || memoryID == "" {
		return nil
	}
	// List entries for memory and delete by id
	where := filters.Where().WithPath([]string{"memoryId"}).WithOperator(filters.Equal).WithValueText(memoryID)
	req := w.client.GraphQL().Get().
		WithClassName("MemoryEntry").
		WithWhere(where).
		WithFields(gql.Field{Name: "entryId"})
	if resp, err := req.Do(ctx); err == nil && len(resp.Errors) == 0 {
		if getData, ok := resp.Data["Get"].(map[string]interface{}); ok {
			if arr, ok := getData["MemoryEntry"].([]interface{}); ok {
				for _, item := range arr {
					id, _ := item.(map[string]interface{})["entryId"].(string)
					if id != "" {
						_ = w.client.Data().Deleter().WithClassName("MemoryEntry").WithID(id).Do(ctx)
					}
				}
			}
		}
	}
	// List contexts for memory and delete by id
	req2 := w.client.GraphQL().Get().
		WithClassName("MemoryContext").
		WithWhere(where).
		WithFields(gql.Field{Name: "contextId"})
	if resp, err := req2.Do(ctx); err == nil && len(resp.Errors) == 0 {
		if getData, ok := resp.Data["Get"].(map[string]interface{}); ok {
			if arr, ok := getData["MemoryContext"].([]interface{}); ok {
				for _, item := range arr {
					id, _ := item.(map[string]interface{})["contextId"].(string)
					if id != "" {
						_ = w.client.Data().Deleter().WithClassName("MemoryContext").WithID(id).Do(ctx)
					}
				}
			}
		}
	}
	return nil
}

// DeleteVault cannot be efficiently implemented without vaultId stored in the index.
// Rely on service-level enumeration + per-object deletes. No-op here for forward compatibility.
func (w *weavNative) DeleteVault(ctx context.Context, actorID string, vaultID string) error {
	return nil
}

// UpsertEntry implements a best-effort upsert using Weaviate Data Creator.
func (w *weavNative) UpsertEntry(ctx context.Context, entryID string, vec []float32, payload map[string]interface{}) error {
	if w == nil || w.client == nil {
		return nil
	}
	_, err := w.client.Data().Creator().WithClassName("MemoryEntry").WithID(entryID).WithProperties(payload).WithVector(vec).Do(ctx)
	return err
}

// UpsertContext implements a best-effort upsert for MemoryContext class.
func (w *weavNative) UpsertContext(ctx context.Context, contextID string, vec []float32, payload map[string]interface{}) error {
	if w == nil || w.client == nil {
		return nil
	}
	_, err := w.client.Data().Creator().WithClassName("MemoryContext").WithID(contextID).WithProperties(payload).WithVector(vec).Do(ctx)
	return err
}

// HealthPing implements health.HealthPinger for weaviate-based index.
// It calls GET http://<baseURL>/v1/meta and expects 200 OK.
func (w *weavNative) HealthPing(ctx context.Context) error {
	if w == nil || w.baseURL == "" {
		return fmt.Errorf("weaviate baseURL missing")
	}
	url := w.baseURL
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "http://" + url
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url+"/v1/meta", nil)
	if err != nil {
		return err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("weaviate status %d", resp.StatusCode)
	}
	return nil
}

// formatGraphQLErrors returns compact string with messages extracted for logging.
func formatGraphQLErrors(errs interface{}) string {
	if b, err := json.Marshal(errs); err == nil {
		return string(b)
	}
	return fmt.Sprintf("%v", errs)
}
