package search

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	weaviate "github.com/weaviate/weaviate-go-client/v5/weaviate"
	filters "github.com/weaviate/weaviate-go-client/v5/weaviate/filters"
	gql "github.com/weaviate/weaviate-go-client/v5/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
)

// Result represents minimal fields returned by search
type Result struct {
	EntryID  string  `json:"entryId"`
	UserID   string  `json:"userId"`
	MemoryID string  `json:"memoryId"`
	Summary  string  `json:"summary"`
	RawEntry string  `json:"rawEntry"`
	Score    float64 `json:"score"`
}

// Searcher abstracts Waviate interactions for search and indexing.
//
//go:generate mockery --name=Searcher
type Searcher interface {
	// Hybrid search over MemoryEntry
	Search(ctx context.Context, userID, memoryID, query string, vec []float32, topK int, alpha float32) ([]Result, error)
	// Context helpers
	LatestContext(ctx context.Context, userID, memoryID string) (string, time.Time, error)
	BestContext(ctx context.Context, userID, memoryID, query string, vec []float32, alpha float32) (text string, ts time.Time, score float64, err error)

	// Index upsert operations (no-ops for implementations that don’t support it)
	UpsertEntry(ctx context.Context, entryID string, vec []float32, payload map[string]interface{}) error
	UpsertContext(ctx context.Context, contextID string, vec []float32, payload map[string]interface{}) error

	// Deletion helpers (best-effort; ignore not found)
	DeleteEntry(ctx context.Context, userID, entryID string) error
	DeleteContext(ctx context.Context, userID, contextID string) error
}

// waviateSearcher implements Searcher using weaviate-go-client.
type waviateSearcher struct {
	client *weaviate.Client
}

// NewWaviateSearcher constructs a Searcher for baseURL host.
func NewWaviateSearcher(baseURL string) (Searcher, error) {
	cfg := weaviate.Config{Scheme: "http", Host: baseURL}
	cl, err := weaviate.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &waviateSearcher{client: cl}, nil
}

// ensureTenant creates the tenant for the given class if it does not already exist.
func (w *waviateSearcher) ensureTenant(ctx context.Context, className, tenant string) {
	if tenant == "" {
		return
	}
	// Attempt creation; ignore errors (already exists, multi-tenant disabled, etc.)
	t := models.Tenant{Name: tenant}
	_ = w.client.Schema().TenantsCreator().WithClassName(className).WithTenants(t).Do(ctx)
}

// UpsertEntry inserts or updates a single MemoryEntry object.
func (w *waviateSearcher) UpsertEntry(ctx context.Context, entryID string, vec []float32, payload map[string]interface{}) error {
	if w == nil || w.client == nil {
		return fmt.Errorf("waviate client not initialised")
	}

	tenant, _ := payload["userId"].(string)
	w.ensureTenant(ctx, "MemoryEntry", tenant)

	_, err := w.client.Data().Creator().WithClassName("MemoryEntry").WithTenant(tenant).WithID(entryID).WithProperties(payload).WithVector(vec).Do(ctx)
	return err
}

// UpsertContext inserts/updates a MemoryContext snapshot.
func (w *waviateSearcher) UpsertContext(ctx context.Context, contextID string, vec []float32, payload map[string]interface{}) error {
	if w == nil || w.client == nil {
		return fmt.Errorf("waviate client not initialised")
	}
	tenant, _ := payload["userId"].(string)
	w.ensureTenant(ctx, "MemoryContext", tenant)

	_, err := w.client.Data().Creator().WithClassName("MemoryContext").WithTenant(tenant).WithID(contextID).WithProperties(payload).WithVector(vec).Do(ctx)
	return err
}

// DeleteEntry removes a MemoryEntry object from Waviate for the given tenant.
func (w *waviateSearcher) DeleteEntry(ctx context.Context, userID, entryID string) error {
	if w == nil || w.client == nil || userID == "" || entryID == "" {
		return nil
	}
	// Best-effort; ignore errors to avoid coupling API latency to index cleanup
	_ = w.client.Data().Deleter().WithClassName("MemoryEntry").WithTenant(userID).WithID(entryID).Do(ctx)
	return nil
}

// DeleteContext removes a MemoryContext object from Waviate for the given tenant.
func (w *waviateSearcher) DeleteContext(ctx context.Context, userID, contextID string) error {
	if w == nil || w.client == nil || userID == "" || contextID == "" {
		return nil
	}
	_ = w.client.Data().Deleter().WithClassName("MemoryContext").WithTenant(userID).WithID(contextID).Do(ctx)
	return nil
}

// helper to detect tenant not found error
func isTenantNotFound(errs interface{}) bool {
	switch v := errs.(type) {
	case []interface{}:
		for _, e := range v {
			if tenantMsg(e) {
				return true
			}
		}
	case []error:
		for _, e := range v {
			if strings.Contains(e.Error(), "tenant not found") {
				return true
			}
		}
	}
	// generic string check fallback
	if strings.Contains(strings.ToLower(fmt.Sprintf("%v", errs)), "tenant not found") {
		return true
	}
	return false
}

func tenantMsg(e interface{}) bool {
	switch m := e.(type) {
	case map[string]interface{}:
		if msg, ok := m["message"].(string); ok {
			return strings.Contains(strings.ToLower(msg), "tenant not found")
		}
	case string:
		return strings.Contains(strings.ToLower(m), "tenant not found")
	}
	return false
}

func (w *waviateSearcher) Search(ctx context.Context, userID, memoryID, query string, vec []float32, topK int, alpha float32) ([]Result, error) {
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
	// Scope the query to the tenant (user) when multi-tenancy is enabled.
	if userID != "" {
		req = req.WithTenant(userID)
	}

	resp, err := req.Do(ctx)
	if err != nil {
		return nil, err
	}

	// check GraphQL errors for tenant issues
	if len(resp.Errors) > 0 {
		if isTenantNotFound(resp.Errors) {
			return nil, ErrTenantNotFound
		}
		return nil, fmt.Errorf("waviate graphql: %s", formatGraphQLErrors(resp.Errors))
	}

	// HANDLE null or missing MemoryEntry to avoid panic (#search-api-null-guard)
	getData, ok := resp.Data["Get"].(map[string]interface{})
	if !ok {
		return nil, nil // unexpected shape, treat as no results
	}
	memVal := getData["MemoryEntry"]
	if memVal == nil {
		return []Result{}, nil // no hits
	}

	raw, ok := memVal.([]interface{})
	if !ok {
		return nil, nil // unexpected type
	}

	out := make([]Result, 0, len(raw))
	for _, item := range raw {
		m := item.(map[string]interface{})
		add := m["_additional"].(map[string]interface{})
		var score float64
		switch v := add["score"].(type) {
		case float64:
			score = v
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				score = f
			}
		default:
			// leave zero
		}
		out = append(out, Result{
			EntryID:  m["entryId"].(string),
			UserID:   m["userId"].(string),
			MemoryID: m["memoryId"].(string),
			Summary:  m["summary"].(string),
			RawEntry: m["rawEntry"].(string),
			Score:    score,
		})
	}
	return out, nil
}

// LatestContext returns the most recent context string and its timestamp for a memory.
func (w *waviateSearcher) LatestContext(ctx context.Context, userID, memoryID string) (string, time.Time, error) {
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
		if isTenantNotFound(resp.Errors) {
			return "", time.Time{}, ErrTenantNotFound
		}
		return "", time.Time{}, fmt.Errorf("waviate graphql: %s", formatGraphQLErrors(resp.Errors))
	}

	// SAFE nil handling for MemoryEntry field (#latest-context-null-guard)
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

// BestContext performs a hybrid search over MemoryContext class and returns top-1 match.
func (w *waviateSearcher) BestContext(ctx context.Context, userID, memoryID, query string, vec []float32, alpha float32) (string, time.Time, float64, error) {
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
		if isTenantNotFound(resp.Errors) {
			return "", time.Time{}, 0, ErrTenantNotFound
		}
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
	ts, _ := time.Parse(time.RFC3339, tsStr)
	return ctxText, ts, score, nil
}

// formatGraphQLErrors returns compact string with messages extracted for logging.
func formatGraphQLErrors(errs interface{}) string {
	// try marshal to JSON for readability; fall back to fmt.Sprintf
	if b, err := json.Marshal(errs); err == nil {
		return string(b)
	}
	return fmt.Sprintf("%v", errs)
}
