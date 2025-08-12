package searchindex

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	weaviate "github.com/weaviate/weaviate-go-client/v5/weaviate"
	filters "github.com/weaviate/weaviate-go-client/v5/weaviate/filters"
	gql "github.com/weaviate/weaviate-go-client/v5/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"

	"github.com/mycelian/mycelian-memory/server/internal/model"
)

// wavNative is a native implementation of Index using the Weaviate Go client.
type wavNative struct{ client *weaviate.Client }

// NewWaviateNativeIndex constructs an Index backed by Weaviate at baseURL.
// baseURL should be host:port (without scheme), e.g., "localhost:8081".
func NewWaviateNativeIndex(baseURL string) (Index, error) {
	cfg := weaviate.Config{Scheme: "http", Host: baseURL}
	cl, err := weaviate.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &wavNative{client: cl}, nil
}

func (w *wavNative) Search(ctx context.Context, userID, memoryID, query string, vec []float32, topK int, alpha float32) ([]model.SearchHit, error) {
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
		WithLimit(topK).
		WithFields(
			gql.Field{Name: "entryId"},
			gql.Field{Name: "userId"},
			gql.Field{Name: "memoryId"},
			gql.Field{Name: "summary"},
			gql.Field{Name: "rawEntry"},
			gql.Field{Name: "_additional", Fields: []gql.Field{{Name: "score"}}},
		)
	if userID != "" {
		req = req.WithTenant(userID)
	}

	resp, err := req.Do(ctx)
	if err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("waviate graphql: %s", formatGraphQLErrors(resp.Errors))
	}

	getData, ok := resp.Data["Get"].(map[string]interface{})
	if !ok {
		return nil, nil
	}
	memVal := getData["MemoryEntry"]
	if memVal == nil {
		return []model.SearchHit{}, nil
	}
	raw, ok := memVal.([]interface{})
	if !ok {
		return nil, nil
	}
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
		out = append(out, model.SearchHit{
			EntryID: m["entryId"].(string),
			Summary: m["summary"].(string),
			Score:   score,
		})
	}
	return out, nil
}

func (w *wavNative) LatestContext(ctx context.Context, userID, memoryID string) (string, time.Time, error) {
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
	if userID != "" {
		req = req.WithTenant(userID)
	}
	resp, err := req.Do(ctx)
	if err != nil {
		return "", time.Time{}, err
	}
	if len(resp.Errors) > 0 {
		return "", time.Time{}, fmt.Errorf("waviate graphql: %s", formatGraphQLErrors(resp.Errors))
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

func (w *wavNative) BestContext(ctx context.Context, userID, memoryID, query string, vec []float32, alpha float32) (string, time.Time, float64, error) {
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
		WithLimit(1).
		WithFields(
			gql.Field{Name: "context"},
			gql.Field{Name: "creationTime"},
			gql.Field{Name: "_additional", Fields: []gql.Field{{Name: "score"}}},
		)
	if userID != "" {
		req = req.WithTenant(userID)
	}
	resp, err := req.Do(ctx)
	if err != nil {
		return "", time.Time{}, 0, err
	}
	if len(resp.Errors) > 0 {
		return "", time.Time{}, 0, fmt.Errorf("waviate graphql: %s", formatGraphQLErrors(resp.Errors))
	}
	getData, ok := resp.Data["Get"].(map[string]interface{})
	if !ok {
		return "", time.Time{}, 0, nil
	}
	val := getData["MemoryContext"]
	if val == nil {
		return "", time.Time{}, 0, nil
	}
	arr, ok := val.([]interface{})
	if !ok || len(arr) == 0 {
		return "", time.Time{}, 0, nil
	}
	item := arr[0].(map[string]interface{})
	ctxText, _ := item["context"].(string)
	tsStr, _ := item["creationTime"].(string)
	ts, _ := time.Parse(time.RFC3339, tsStr)
	var score float64
	if add, ok := item["_additional"].(map[string]interface{}); ok {
		switch v := add["score"].(type) {
		case float64:
			score = v
		case string:
			f, _ := strconv.ParseFloat(v, 64)
			score = f
		}
	}
	return ctxText, ts, score, nil
}

func (w *wavNative) DeleteEntry(ctx context.Context, userID, entryID string) error {
	if w == nil || w.client == nil || userID == "" || entryID == "" {
		return nil
	}
	// Best-effort ensure tenant exists before deletion
	_ = ensureTenant(ctx, w.client, "MemoryEntry", userID)
	_ = w.client.Data().Deleter().WithClassName("MemoryEntry").WithTenant(userID).WithID(entryID).Do(ctx)
	return nil
}

func (w *wavNative) DeleteContext(ctx context.Context, userID, contextID string) error {
	if w == nil || w.client == nil || userID == "" || contextID == "" {
		return nil
	}
	_ = ensureTenant(ctx, w.client, "MemoryContext", userID)
	_ = w.client.Data().Deleter().WithClassName("MemoryContext").WithTenant(userID).WithID(contextID).Do(ctx)
	return nil
}

func (w *wavNative) DeleteMemory(ctx context.Context, userID, memoryID string) error {
	if w == nil || w.client == nil || userID == "" || memoryID == "" {
		return nil
	}
	_ = ensureTenant(ctx, w.client, "MemoryEntry", userID)
	_ = ensureTenant(ctx, w.client, "MemoryContext", userID)
	// List entries for memory and delete by id
	where := filters.Where().WithPath([]string{"memoryId"}).WithOperator(filters.Equal).WithValueText(memoryID)
	req := w.client.GraphQL().Get().
		WithClassName("MemoryEntry").
		WithWhere(where).
		WithFields(gql.Field{Name: "entryId"})
	req = req.WithTenant(userID)
	if resp, err := req.Do(ctx); err == nil && len(resp.Errors) == 0 {
		if getData, ok := resp.Data["Get"].(map[string]interface{}); ok {
			if arr, ok := getData["MemoryEntry"].([]interface{}); ok {
				for _, item := range arr {
					id, _ := item.(map[string]interface{})["entryId"].(string)
					if id != "" {
						_ = w.client.Data().Deleter().WithClassName("MemoryEntry").WithTenant(userID).WithID(id).Do(ctx)
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
	req2 = req2.WithTenant(userID)
	if resp, err := req2.Do(ctx); err == nil && len(resp.Errors) == 0 {
		if getData, ok := resp.Data["Get"].(map[string]interface{}); ok {
			if arr, ok := getData["MemoryContext"].([]interface{}); ok {
				for _, item := range arr {
					id, _ := item.(map[string]interface{})["contextId"].(string)
					if id != "" {
						_ = w.client.Data().Deleter().WithClassName("MemoryContext").WithTenant(userID).WithID(id).Do(ctx)
					}
				}
			}
		}
	}
	return nil
}

// DeleteVault cannot be efficiently implemented without vaultId stored in the index.
// Rely on service-level enumeration + per-object deletes. No-op here for forward compatibility.
func (w *wavNative) DeleteVault(ctx context.Context, userID, vaultID string) error { return nil }

// UpsertEntry implements a best-effort upsert using Waviate Data Creator.
func (w *wavNative) UpsertEntry(ctx context.Context, entryID string, vec []float32, payload map[string]interface{}) error {
	if w == nil || w.client == nil {
		return nil
	}
	tenant, _ := payload["userId"].(string)
	if tenant == "" {
		return nil
	}
	if err := ensureTenant(ctx, w.client, "MemoryEntry", tenant); err != nil {
		return err
	}
	_, err := w.client.Data().Creator().WithClassName("MemoryEntry").WithTenant(tenant).WithID(entryID).WithProperties(payload).WithVector(vec).Do(ctx)
	return err
}

// UpsertContext implements a best-effort upsert for MemoryContext class.
func (w *wavNative) UpsertContext(ctx context.Context, contextID string, vec []float32, payload map[string]interface{}) error {
	if w == nil || w.client == nil {
		return nil
	}
	tenant, _ := payload["userId"].(string)
	if tenant == "" {
		return nil
	}
	if err := ensureTenant(ctx, w.client, "MemoryContext", tenant); err != nil {
		return err
	}
	_, err := w.client.Data().Creator().WithClassName("MemoryContext").WithTenant(tenant).WithID(contextID).WithProperties(payload).WithVector(vec).Do(ctx)
	return err
}

// formatGraphQLErrors returns compact string with messages extracted for logging.
func formatGraphQLErrors(errs interface{}) string {
	if b, err := json.Marshal(errs); err == nil {
		return string(b)
	}
	return fmt.Sprintf("%v", errs)
}

// ensureTenant creates the tenant for the given class if it does not already exist.
func ensureTenant(ctx context.Context, cl *weaviate.Client, className, tenant string) error {
	if tenant == "" {
		return nil
	}
	// Check existing tenants first to avoid 409 errors
	ex, err := cl.Schema().TenantsGetter().WithClassName(className).Do(ctx)
	if err == nil && ex != nil {
		for _, t := range ex {
			if t.Name == tenant {
				return nil
			}
		}
	}
	// Create tenant
	ts := []models.Tenant{{Name: tenant}}
	return cl.Schema().TenantsCreator().WithClassName(className).WithTenants(ts...).Do(ctx)
}
