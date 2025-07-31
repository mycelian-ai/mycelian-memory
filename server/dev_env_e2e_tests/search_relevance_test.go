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
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	indexer "memory-backend/internal/indexer-prototype"

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

// waitForObjects polls Aggregate meta.count until want is reached or timeout.
func waitForObjects(t *testing.T, cl *wvClient, tenant string, want int, timeout time.Duration) {
	ctx := context.Background()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := cl.GraphQL().Aggregate().WithClassName("MemoryEntry").WithTenant(tenant).
			WithFields(gql.Field{Name: "meta", Fields: []gql.Field{{Name: "count"}}}).Do(ctx)
		if err == nil {
			agg, ok := resp.Data["Aggregate"].(map[string]interface{})
			if ok {
				memVal := agg["MemoryEntry"]
				if memArr, ok := memVal.([]interface{}); ok && len(memArr) > 0 {
					meta := memArr[0].(map[string]interface{})["meta"].(map[string]interface{})
					if cnt, ok := meta["count"].(float64); ok && int(cnt) == want {
						return
					}
				}
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	t.Fatalf("expected %d objects within %s", want, timeout)
}

// createVault returns vaultID and base path for further requests
func createVault(t *testing.T, memSvc, userID, title string) (string, string) {
	payload := fmt.Sprintf(`{"title":"%s"}`, title)
	resp, err := http.Post(fmt.Sprintf("%s/api/users/%s/vaults", memSvc, userID), "application/json", bytes.NewBufferString(payload))
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
	base := fmt.Sprintf("%s/api/users/%s/vaults/%s", memSvc, userID, v.VaultID)
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
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
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

	memSvc := env("MEMORY_API", "http://localhost:8080")
	waviateURL := env("WAVIATE_URL", "http://localhost:8082")
	ollamaURL := env("OLLAMA_URL", "http://localhost:11434")

	// connectivity checks
	waitForHealthy(t, memSvc, 3*time.Second)
	for _, url := range []string{waviateURL + "/v1/meta", ollamaURL + "/api/tags"} {
		if err := ping(url); err != nil {
			t.Skipf("service %s unreachable: %v", url, err)
		}
	}

	embedder, err := indexer.NewProvider("ollama", env("EMBED_MODEL", "mxbai-embed-large"))
	if err != nil {
		t.Fatalf("embed provider: %v", err)
	}
	// quick health check
	if _, err := embedder.Embed(context.Background(), "healthcheck"); err != nil {
		t.Skipf("ollama not responding: %v", err)
	}

	// 1. user
	email := fmt.Sprintf("alpha-%d@example.com", time.Now().UnixNano())
	var userResp struct {
		UserID string `json:"userId"`
	}
	resp, err := http.Post(memSvc+"/api/users", "application/json", bytes.NewBufferString(fmt.Sprintf(`{"email":"%s"}`, email)))
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	mustJSON(t, resp, &userResp)

	// 2. vault and memory
	_, baseVaultPath := createVault(t, memSvc, userResp.UserID, "alphavault")
	var memResp struct {
		MemoryID string `json:"memoryId"`
	}
	body := `{"memoryType":"CONVERSATION","title":"alphasweep"}`
	resp, err = http.Post(baseVaultPath+"/memories", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("create memory: %v", err)
	}
	mustJSON(t, resp, &memResp)

	// 2. seed entries (30 cat + 30 solar)
	for i := 0; i < 30; i++ {
		createEntry(t, baseVaultPath, userResp.UserID, memResp.MemoryID, fmt.Sprintf("Curious cat %d jumps over the couch.", i), fmt.Sprintf("Cat behave %d", i), nil)
		createEntry(t, baseVaultPath, userResp.UserID, memResp.MemoryID, fmt.Sprintf("Solar generator model %d backup power.", i), fmt.Sprintf("Solar note %d", i), nil)
	}

	cl, err := newWeaviateClient(waviateURL)
	if err != nil {
		t.Fatalf("wv client: %v", err)
	}
	waitForObjects(t, cl, userResp.UserID, 60, 5*time.Second)

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
			resp, err := cl.GraphQL().Get().WithClassName("MemoryEntry").WithTenant(userResp.UserID).
				WithHybrid(hy).
				WithFields(gql.Field{Name: "rawEntry"}, gql.Field{Name: "summary"}).WithLimit(10).Do(context.Background())
			if err != nil {
				t.Fatalf("hybrid search alpha=%.2f: %v", alpha, err)
			}
			items := resp.Data["Get"].(map[string]interface{})["MemoryEntry"].([]interface{})
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

	memSvc := env("MEMORY_API", "http://localhost:8080")
	waviateURL := env("WAVIATE_URL", "http://localhost:8082")
	ollamaURL := env("OLLAMA_URL", "http://localhost:11434")

	waitForHealthy(t, memSvc, 3*time.Second)
	for _, url := range []string{waviateURL + "/v1/meta", ollamaURL + "/api/tags"} {
		if err := ping(url); err != nil {
			t.Skipf("service %s unreachable: %v", url, err)
		}
	}

	embedder, err := indexer.NewProvider("ollama", env("EMBED_MODEL", "mxbai-embed-large"))
	if err != nil {
		t.Fatalf("embed provider: %v", err)
	}
	// quick health check
	if _, err := embedder.Embed(context.Background(), "healthcheck"); err != nil {
		t.Skipf("ollama not responding: %v", err)
	}

	// user/memory
	email := fmt.Sprintf("tag-%d@example.com", time.Now().UnixNano())
	var user struct {
		UserID string `json:"userId"`
	}
	resp, _ := http.Post(memSvc+"/api/users", "application/json", bytes.NewBufferString(fmt.Sprintf(`{"email":"%s"}`, email)))
	mustJSON(t, resp, &user)
	var mem struct {
		MemoryID string `json:"memoryId"`
	}
	// vault then memory
	_, baseVaultPathT := createVault(t, memSvc, user.UserID, "tagvault")

	resp, _ = http.Post(baseVaultPathT+"/memories", "application/json", bytes.NewBufferString(`{"memoryType":"CONVERSATION","title":"Tag"}`))
	mustJSON(t, resp, &mem)

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

	cl, _ := newWeaviateClient(waviateURL)
	waitForObjects(t, cl, user.UserID, 10, 5*time.Second)

	query := "playful cat"
	vec, _ := embedder.Embed(context.Background(), query)
	hy := (&gql.HybridArgumentBuilder{}).WithQuery(query).WithVector(vec).WithAlpha(0.6)
	where := filters.Where().WithPath([]string{"tags"}).WithOperator(filters.ContainsAny).WithValueText("featured")
	resp2, err := cl.GraphQL().Get().WithClassName("MemoryEntry").WithTenant(user.UserID).WithHybrid(hy).WithWhere(where).
		WithFields(gql.Field{Name: "rawEntry"}).WithLimit(10).Do(context.Background())
	if err != nil {
		t.Fatalf("hybrid tag filter: %v", err)
	}
	items := resp2.Data["Get"].(map[string]interface{})["MemoryEntry"].([]interface{})
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

	memSvc := env("MEMORY_API", "http://localhost:8080")
	waviateURL := env("WAVIATE_URL", "http://localhost:8082")
	ollamaURL := env("OLLAMA_URL", "http://localhost:11434")

	waitForHealthy(t, memSvc, 3*time.Second)
	for _, url := range []string{waviateURL + "/v1/meta", ollamaURL + "/api/tags"} {
		if err := ping(url); err != nil {
			t.Skipf("service %s unreachable: %v", url, err)
		}
	}

	_, _ = indexer.NewProvider("ollama", env("EMBED_MODEL", "mxbai-embed-large"))

	// user/memory
	email := fmt.Sprintf("meta-%d@example.com", time.Now().UnixNano())
	var user struct {
		UserID string `json:"userId"`
	}
	resp, _ := http.Post(memSvc+"/api/users", "application/json", bytes.NewBufferString(fmt.Sprintf(`{"email":"%s"}`, email)))
	mustJSON(t, resp, &user)
	var mem struct {
		MemoryID string `json:"memoryId"`
	}
	_, baseVaultPathM := createVault(t, memSvc, user.UserID, "metavault")
	resp, _ = http.Post(baseVaultPathM+"/memories", "application/json", bytes.NewBufferString(`{"memoryType":"CONVERSATION","title":"meta"}`))
	mustJSON(t, resp, &mem)

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
