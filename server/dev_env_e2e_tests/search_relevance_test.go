//go:build e2e
// +build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	embOllama "github.com/mycelian/mycelian-memory/server/internal/embeddings/ollama"

	weaviate "github.com/weaviate/weaviate-go-client/v5/weaviate"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/filters"
	gql "github.com/weaviate/weaviate-go-client/v5/weaviate/graphql"
)

//---------------------------------------------------------------------
// helpers (file-private)
//---------------------------------------------------------------------

type wvClient struct{ *weaviate.Client }

func newWeaviateClient(rawURL string) (*wvClient, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	cfg := weaviate.Config{Scheme: u.Scheme, Host: u.Host}
	c, err := weaviate.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &wvClient{c}, nil
}

// newTestEmbedder selects the test embedder to match the service's provider.
func newTestEmbedder() interface {
	Embed(context.Context, string) ([]float32, error)
} {
	provider := os.Getenv("MEMORY_SERVER_EMBED_PROVIDER")
	model := env("EMBED_MODEL", "nomic-embed-text")
	// We only support the Ollama embedder. Treat any model (including "maxbai")
	// as an Ollama model name.
	_ = provider
	return embOllama.New(model)
}

// waitForObjects polls Aggregate meta.count until want is reached or timeout.
func waitForObjects(t *testing.T, cl *wvClient, want int, timeout time.Duration) {
	ctx := context.Background()

	// Establish baseline count first to make the check robust across repeated runs.
	baseline := 0
	if resp, err := cl.GraphQL().Aggregate().WithClassName("MemoryEntry").
		WithFields(gql.Field{Name: "meta", Fields: []gql.Field{{Name: "count"}}}).Do(ctx); err == nil {
		if agg, ok := resp.Data["Aggregate"].(map[string]interface{}); ok {
			if memArr, ok := agg["MemoryEntry"].([]interface{}); ok && len(memArr) > 0 {
				if meta, ok := memArr[0].(map[string]interface{})["meta"].(map[string]interface{}); ok {
					if cnt, ok := meta["count"].(float64); ok {
						baseline = int(cnt)
					}
				}
			}
		}
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := cl.GraphQL().Aggregate().WithClassName("MemoryEntry").
			WithFields(gql.Field{Name: "meta", Fields: []gql.Field{{Name: "count"}}}).Do(ctx)
		if err == nil {
			if agg, ok := resp.Data["Aggregate"].(map[string]interface{}); ok {
				if memArr, ok := agg["MemoryEntry"].([]interface{}); ok && len(memArr) > 0 {
					if meta, ok := memArr[0].(map[string]interface{})["meta"].(map[string]interface{}); ok {
						if cnt, ok := meta["count"].(float64); ok && int(cnt) >= baseline+want {
							return
						}
					}
				}
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	t.Fatalf("expected at least %d new objects within %s (baseline %d)", want, timeout, baseline)
}

// createVault returns vaultID and base path for further requests
func createVault(t *testing.T, memSvc, userID, title string) (string, string) {
	payload := fmt.Sprintf(`{"title":"%s"}`, title)
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v0/vaults", memSvc), bytes.NewBufferString(payload))
	if err != nil {
		t.Fatalf("create vault request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer LOCAL_DEV_MODE_NOT_FOR_PRODUCTION")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create vault: %v", err)
	}
	var v struct {
		VaultID string `json:"vaultId"`
	}
	mustJSON(t, resp, &v)
	if _, err := uuid.Parse(v.VaultID); err != nil {
		t.Fatalf("invalid vaultID")
	}
	base := fmt.Sprintf("%s/v0/vaults/%s", memSvc, v.VaultID)
	return v.VaultID, base
}

// createEntry posts an entry JSON; extras are merged into root JSON map.
func createEntry(t *testing.T, baseVaultPath string, _, memoryID, raw, summary string, extras map[string]interface{}) {
	payload := map[string]interface{}{
		"rawEntry": raw,
		"summary":  summary,
	}
	for k, v := range extras {
		payload[k] = v
	}
	data, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/memories/%s/entries", baseVaultPath, memoryID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		t.Fatalf("create entry request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer LOCAL_DEV_MODE_NOT_FOR_PRODUCTION")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}
	var tmp struct {
		EntryID string `json:"entryId"`
	}
	mustJSON(t, resp, &tmp)
}

//---------------------------------------------------------------------
// Test 1 – Hybrid alpha sweep
//---------------------------------------------------------------------

func TestDevEnv_HybridRelevance_AlphaSweep(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}

	memSvc := env("MEMORY_API", "http://localhost:11545")
	weaviateURL := env("WEAVIATE_URL", "http://localhost:11543")
	ollamaURL := env("OLLAMA_URL", "http://localhost:11434")

	// connectivity checks
	waitForHealthy(t, memSvc, 3*time.Second)
	// Always require Weaviate; only ping Ollama if provider is ollama
	urls := []string{weaviateURL + "/v1/meta"}
	if strings.EqualFold(os.Getenv("MEMORY_SERVER_EMBED_PROVIDER"), "ollama") {
		urls = append(urls, ollamaURL+"/api/tags")
	}
	for _, url := range urls {
		if err := ping(url); err != nil {
			t.Skipf("service %s unreachable: %v", url, err)
		}
	}

	// Embedder is configured in the service; tests call Weaviate directly here.
	// Bootstrap is handled by the service startup.
	embedder := newTestEmbedder()
	// quick health check (skip test if local model isn't loaded)
	if _, err := embedder.Embed(context.Background(), "healthcheck"); err != nil {
		t.Skipf("ollama not responding: %v", err)
	}

	// 1. ensure test_user and weaviate tenant
	var userResp struct {
		UserID string `json:"userId"`
	}
	userResp.UserID = "mycelian-dev" // Use MockAuthorizer's ActorID for Weaviate tenant consistency

	// 2. vault and memory (unique per run) with cleanup
	title := fmt.Sprintf("alphavault-%d", time.Now().UnixNano())
	vid, baseVaultPath := createVault(t, memSvc, userResp.UserID, title)
	defer func() {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/v0/vaults/%s", memSvc, vid), nil)
		req.Header.Set("Authorization", "Bearer LOCAL_DEV_MODE_NOT_FOR_PRODUCTION")
		_, _ = http.DefaultClient.Do(req)
	}()
	var memResp struct {
		MemoryID string `json:"memoryId"`
	}
	body := `{"memoryType":"CONVERSATION","title":"alphasweep"}`
	req, err := http.NewRequest("POST", baseVaultPath+"/memories", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("create memory request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer LOCAL_DEV_MODE_NOT_FOR_PRODUCTION")

	respM, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create memory: %v", err)
	}
	mustJSON(t, respM, &memResp)

	// 2. seed entries (30 cat + 30 solar)
	for i := 0; i < 30; i++ {
		createEntry(t, baseVaultPath, userResp.UserID, memResp.MemoryID, fmt.Sprintf("Curious cat %d jumps over the couch.", i), fmt.Sprintf("Cat behave %d", i), nil)
		createEntry(t, baseVaultPath, userResp.UserID, memResp.MemoryID, fmt.Sprintf("Solar generator model %d backup power.", i), fmt.Sprintf("Solar note %d", i), nil)
	}

	cl, err := newWeaviateClient(weaviateURL)
	if err != nil {
		t.Fatalf("wv client: %v", err)
	}
	// Allow more time for 60 object ingestion in AlphaSweep only
	// Hybrid indexing via outbox + embeddings can be slower; allow more time.
	waitForObjects(t, cl, 60, 20*time.Second)

	queryTests := []struct {
		query   string
		keyword string
	}{
		{"playful cat jumps", "cat"},
		{"portable solar generator", "solar"},
	}
	for _, tc := range queryTests {
		vec, _ := embedder.Embed(context.Background(), tc.query)
		for _, alpha := range []float32{0.0, 1.0, 0.6} {
			hy := (&gql.HybridArgumentBuilder{}).WithQuery(tc.query).WithVector(vec).WithAlpha(alpha)
			resp, err := cl.GraphQL().Get().WithClassName("MemoryEntry").
				WithHybrid(hy).
				WithFields(gql.Field{Name: "rawEntry"}, gql.Field{Name: "summary"}).WithLimit(50).Do(context.Background())
			if err != nil {
				t.Fatalf("hybrid search alpha=%.2f: %v", alpha, err)
			}
			// Be robust to nil/absent results; treat as empty set
			var items []interface{}
			if getMap, ok := resp.Data["Get"].(map[string]interface{}); ok {
				if memVal, ok := getMap["MemoryEntry"]; ok && memVal != nil {
					if arr, ok := memVal.([]interface{}); ok {
						items = arr
					}
				}
			}
			evaluateResults(t, items, tc.keyword)
		}
	}
}

//---------------------------------------------------------------------
// Test 2 – Tag filter (tags featured)
//---------------------------------------------------------------------

func TestDevEnv_HybridRelevance_TagFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}

	memSvc := env("MEMORY_API", "http://localhost:11545")
	weaviateURL := env("WEAVIATE_URL", "http://localhost:11543")
	ollamaURL := env("OLLAMA_URL", "http://localhost:11434")

	waitForHealthy(t, memSvc, 3*time.Second)
	urls := []string{weaviateURL + "/v1/meta"}
	if strings.EqualFold(os.Getenv("MEMORY_SERVER_EMBED_PROVIDER"), "ollama") {
		urls = append(urls, ollamaURL+"/api/tags")
	}
	for _, url := range urls {
		if err := ping(url); err != nil {
			t.Skipf("service %s unreachable: %v", url, err)
		}
	}

	embedder := newTestEmbedder()
	// quick health check
	if _, err := embedder.Embed(context.Background(), "healthcheck"); err != nil {
		t.Skipf("embedder not responding: %v", err)
	}

	// user/memory under test_user (ensure tenant)
	var user struct {
		UserID string `json:"userId"`
	}
	user.UserID = "mycelian-dev" // Use MockAuthorizer's ActorID for Weaviate tenant consistency

	var mem struct {
		MemoryID string `json:"memoryId"`
	}
	// vault then memory
	vidT, baseVaultPathT := createVault(t, memSvc, user.UserID, "tagvault")
	defer func() {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/v0/vaults/%s", memSvc, vidT), nil)
		req.Header.Set("Authorization", "Bearer LOCAL_DEV_MODE_NOT_FOR_PRODUCTION")
		_, _ = http.DefaultClient.Do(req)
	}()

	req2, _ := http.NewRequest("POST", baseVaultPathT+"/memories", bytes.NewBufferString(`{"memoryType":"CONVERSATION","title":"Tag"}`))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer LOCAL_DEV_MODE_NOT_FOR_PRODUCTION")
	respM, _ := http.DefaultClient.Do(req2)
	mustJSON(t, respM, &mem)

	// seed entries 5 cats (3 featured) + 5 dogs
	for i := 0; i < 5; i++ {
		catTags := map[string]interface{}{"topic/cat": true}
		if i < 3 {
			catTags["featured"] = true
		}
		createEntry(t, baseVaultPathT, user.UserID, mem.MemoryID, fmt.Sprintf("Cat entry %d playing", i), "Cat summary", map[string]interface{}{"tags": catTags})

		dogTags := map[string]interface{}{"topic/dog": true}
		createEntry(t, baseVaultPathT, user.UserID, mem.MemoryID, fmt.Sprintf("Dog entry %d barking", i), "Dog summary", map[string]interface{}{"tags": dogTags})
	}

	cl, _ := newWeaviateClient(weaviateURL)
	waitForObjects(t, cl, 10, 5*time.Second)

	query := "playful cat"
	vec, _ := embedder.Embed(context.Background(), query)
	hy := (&gql.HybridArgumentBuilder{}).WithQuery(query).WithVector(vec).WithAlpha(0.6)
	where := filters.Where().WithPath([]string{"tags"}).WithOperator(filters.ContainsAny).WithValueText("featured")
	resp2, err := cl.GraphQL().Get().WithClassName("MemoryEntry").WithHybrid(hy).WithWhere(where).
		WithFields(gql.Field{Name: "rawEntry"}).WithLimit(10).Do(context.Background())
	if err != nil {
		t.Fatalf("hybrid tag filter: %v", err)
	}
	var items []interface{}
	if getMap, ok := resp2.Data["Get"].(map[string]interface{}); ok {
		if memVal, ok := getMap["MemoryEntry"]; ok && memVal != nil {
			if arr, ok := memVal.([]interface{}); ok {
				items = arr
			}
		}
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 featured cat entries, got %d", len(items))
	}
}

//---------------------------------------------------------------------
// Test 3 – Metadata filter
//---------------------------------------------------------------------

func TestDevEnv_HybridRelevance_MetadataFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}

	memSvc := env("MEMORY_API", "http://localhost:11545")
	weaviateURL := env("WEAVIATE_URL", "http://localhost:11543")
	ollamaURL := env("OLLAMA_URL", "http://localhost:11434")

	waitForHealthy(t, memSvc, 3*time.Second)
	for _, url := range []string{weaviateURL + "/v1/meta", ollamaURL + "/api/tags"} {
		if err := ping(url); err != nil {
			t.Skipf("service %s unreachable: %v", url, err)
		}
	}

	// Indexer provider initialization removed; embedding provider is configured via service env

	// user/memory under test_user (ensure tenant)
	var user struct {
		UserID string `json:"userId"`
	}
	user.UserID = "mycelian-dev" // Use MockAuthorizer's ActorID for Weaviate tenant consistency

	var mem struct {
		MemoryID string `json:"memoryId"`
	}
	vidM, baseVaultPathM := createVault(t, memSvc, user.UserID, "metavault")
	defer func() {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/v0/vaults/%s", memSvc, vidM), nil)
		req.Header.Set("Authorization", "Bearer LOCAL_DEV_MODE_NOT_FOR_PRODUCTION")
		_, _ = http.DefaultClient.Do(req)
	}()
	req3, _ := http.NewRequest("POST", baseVaultPathM+"/memories", bytes.NewBufferString(`{"memoryType":"CONVERSATION","title":"meta"}`))
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("Authorization", "Bearer LOCAL_DEV_MODE_NOT_FOR_PRODUCTION")
	respM2, _ := http.DefaultClient.Do(req3)
	mustJSON(t, respM2, &mem)

	t.Skip("metadata filter test disabled under lowercase-title constraint")
}

//---------------------------------------------------------------------
// evaluation helper
//---------------------------------------------------------------------

func evaluateResults(t testing.TB, items []interface{}, keyword string) {
	found := 0
	foundTop3 := false
	for i, it := range items {
		if i >= 10 {
			break
		}
		m := it.(map[string]interface{})
		raw := strings.ToLower(m["rawEntry"].(string))
		sum := strings.ToLower(m["summary"].(string))
		if strings.Contains(raw, keyword) || strings.Contains(sum, keyword) {
			found++
			if i < 3 {
				foundTop3 = true
			}
		}
	}
	if !foundTop3 {
		t.Fatalf("no %s entry in top-3", keyword)
	}
	if found < 5 {
		t.Fatalf("only %d %s entries in top-10", found, keyword)
	}
}
