//go:build e2e
// +build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// -----------------------------------------------------------------------------
//
//	Test 1: Ingestion + direct BM25 search (fast path)
//
// -----------------------------------------------------------------------------
// Creates a user → memory → entry via public REST API and verifies that the
// entry is visible via a raw BM25 GraphQL query to Weaviate.  This bypasses the
// /api/search endpoint to give a quick signal that the indexer pipeline is
// healthy.
func TestDevEnv_Ingestion_BM25_Direct(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	memSvc := env("MEMORY_API", "http://localhost:8080")
	waviate := env("WAVIATE_URL", "http://localhost:8082")

	// quick connectivity checks – skip if the stack isn't up
	for _, url := range []string{memSvc + "/api/health", waviate + "/v1/meta"} {
		if err := ping(url); err != nil {
			t.Skipf("service %s unreachable: %v", url, err)
		}
	}

	// 1. Create user
	email := fmt.Sprintf("ingest-%d@example.com", time.Now().UnixNano())
	var userResp struct {
		UserID string `json:"userId"`
	}
	resp, err := http.Post(memSvc+"/api/users", "application/json", bytes.NewBufferString(fmt.Sprintf(`{"email":"%s"}`, email)))
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	mustJSON(t, resp, &userResp)

	// 2. Create vault then memory
	var vaultResp struct {
		VaultID string `json:"vaultId"`
	}
	vPayload := `{"title":"BmVault"}`
	vResp, err := http.Post(fmt.Sprintf("%s/api/users/%s/vaults", memSvc, userResp.UserID), "application/json", bytes.NewBufferString(vPayload))
	if err != nil {
		t.Fatalf("create vault: %v", err)
	}
	mustJSON(t, vResp, &vaultResp)

	baseVaultPath := fmt.Sprintf("%s/api/users/%s/vaults/%s", memSvc, userResp.UserID, vaultResp.VaultID)

	// 2. Create memory
	var memResp struct {
		MemoryID string `json:"memoryId"`
	}
	body := `{"memoryType":"CONVERSATION","title":"BmSmoke"}`
	resp, err = http.Post(baseVaultPath+"/memories", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("create memory: %v", err)
	}
	mustJSON(t, resp, &memResp)

	// 2b. Verify default context snapshot exists immediately after memory creation
	ctxURL := fmt.Sprintf("%s/memories/%s/contexts", baseVaultPath, memResp.MemoryID)
	ctxResp, err := http.Get(ctxURL)
	if err != nil {
		t.Fatalf("get default context: %v", err)
	}
	var ctxBody struct {
		Context map[string]interface{} `json:"context"`
	}
	mustJSON(t, ctxResp, &ctxBody)
	active, _ := ctxBody.Context["activeContext"].(string)
	expectedDefault := "This is default context that's created with the memory. Instructions for AI Agent: Provide relevant context as soon as it's available."
	if active != expectedDefault {
		t.Fatalf("default context mismatch. want %q, got %q", expectedDefault, active)
	}

	// 3. Create entry
	entryText := fmt.Sprintf("BM25 smoke test %d", time.Now().UnixNano())
	entryBody := fmt.Sprintf(`{"rawEntry":"%s","summary":"smoke summary"}`, entryText)
	resp, err = http.Post(fmt.Sprintf("%s/memories/%s/entries", baseVaultPath, memResp.MemoryID), "application/json", bytes.NewBufferString(entryBody))
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}
	var entryResp struct {
		EntryID string `json:"entryId"`
	}
	mustJSON(t, resp, &entryResp)

	// DEBUG: print identifiers so we can reproduce with curl
	t.Logf("BM25 smoke: userID=%s memoryID=%s entryID=%s", userResp.UserID, memResp.MemoryID, entryResp.EntryID)

	// 4. Poll Weaviate BM25 until entry appears.
	// NOTE: 5-second window is tuned for one indexer cycle; do NOT extend.
	deadline := time.Now().Add(5 * time.Second)
	query := fmt.Sprintf(`{ Get { MemoryEntry(tenant:"%s", where:{operator:And, operands:[{path:["memoryId"], operator:Equal, valueString:"%s"},{path:["entryId"], operator:Equal, valueString:"%s"}]}, limit:1){ entryId } } }`, userResp.UserID, memResp.MemoryID, entryResp.EntryID)
	t.Logf("BM25 smoke GraphQL query: %s", query)
	payload := map[string]string{"query": query}
	buf, _ := json.Marshal(payload)
	for {
		if time.Now().After(deadline) {
			t.Fatalf("entry %s not visible in Weaviate within timeout", entryResp.EntryID)
		}
		r, err := http.Post(waviate+"/v1/graphql", "application/json", bytes.NewBuffer(buf))
		if err == nil && r.StatusCode == http.StatusOK {
			var out struct {
				Data struct {
					Get struct {
						MemoryEntry []struct {
							EntryID string `json:"entryId"`
						} `json:"MemoryEntry"`
					} `json:"Get"`
				} `json:"data"`
			}
			if err := json.NewDecoder(r.Body).Decode(&out); err == nil && len(out.Data.Get.MemoryEntry) == 1 {
				r.Body.Close()
				break // success
			}
			r.Body.Close()
		} else if r != nil {
			r.Body.Close()
		}
		time.Sleep(300 * time.Millisecond)
	}
}

// -----------------------------------------------------------------------------
//
//	Test 2: /api/search hybrid round-trip (embedding + context enrichment)
//
// -----------------------------------------------------------------------------
func TestDevEnv_SearchAPI_Hybrid(t *testing.T) {
	if testing.Short() {
		t.Skip("skip in short mode")
	}

	memSvc := env("MEMORY_API", "http://localhost:8080")
	waviate := env("WAVIATE_URL", "http://localhost:8082")
	ollama := env("OLLAMA_URL", "http://localhost:11434")
	embedMod := env("EMBED_MODEL", "mxbai-embed-large")

	// Ensure services are reachable
	waitForHealthy(t, memSvc, 3*time.Second)
	for _, url := range []string{memSvc + "/api/health", waviate + "/v1/meta", ollama + "/api/tags"} {
		if err := ping(url); err != nil {
			t.Skipf("service %s unreachable: %v", url, err)
		}
	}

	if !modelExists(t, ollama, embedMod) {
		t.Fatalf("ollama model %s not available", embedMod)
	}

	// 1. user
	email := fmt.Sprintf("search-e2e-%d@example.com", time.Now().UnixNano())
	var userResp struct {
		UserID string `json:"userId"`
	}
	resp, err := http.Post(memSvc+"/api/users", "application/json", bytes.NewBufferString(fmt.Sprintf(`{"email":"%s"}`, email)))
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	mustJSON(t, resp, &userResp)

	// 2. vault then memory
	var vaultResp struct {
		VaultID string `json:"vaultId"`
	}
	vPayload := `{"title":"SearchVault"}`
	vResp, err := http.Post(fmt.Sprintf("%s/api/users/%s/vaults", memSvc, userResp.UserID), "application/json", bytes.NewBufferString(vPayload))
	if err != nil {
		t.Fatalf("create vault: %v", err)
	}
	mustJSON(t, vResp, &vaultResp)

	baseVaultPath := fmt.Sprintf("%s/api/users/%s/vaults/%s", memSvc, userResp.UserID, vaultResp.VaultID)

	// 2. memory
	var memResp struct {
		MemoryID string `json:"memoryId"`
	}
	body := `{"memoryType":"CONVERSATION","title":"SearchSmoke"}`
	resp, err = http.Post(baseVaultPath+"/memories", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("create memory: %v", err)
	}
	mustJSON(t, resp, &memResp)

	// 3. entry
	entryText := fmt.Sprintf("Search API smoke %d", time.Now().UnixNano())
	entryBody := fmt.Sprintf(`{"rawEntry":"%s","summary":"search summary"}`, entryText)
	resp, err = http.Post(fmt.Sprintf("%s/memories/%s/entries", baseVaultPath, memResp.MemoryID), "application/json", bytes.NewBufferString(entryBody))
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}
	var entryResp struct {
		EntryID string `json:"entryId"`
	}
	mustJSON(t, resp, &entryResp)

	// 4. Wait until the object is ingested into Weaviate
	objectURL := fmt.Sprintf("%s/v1/objects/MemoryEntry/%s?tenant=%s", waviate, entryResp.EntryID, userResp.UserID)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		r, err := http.Get(objectURL)
		if err == nil && r.StatusCode == http.StatusOK {
			r.Body.Close()
			break
		}
		if r != nil {
			r.Body.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}

	// 5. Call search API until it returns the entry
	searchBody := fmt.Sprintf(`{"userId":"%s","memoryId":"%s","query":"%s"}`, userResp.UserID, memResp.MemoryID, entryText)
	deadline = time.Now().Add(5 * time.Second)
	for {
		if time.Now().After(deadline) {
			t.Fatalf("search API did not return expected entry within timeout")
		}
		resp, err = http.Post(memSvc+"/api/search", "application/json", bytes.NewBufferString(searchBody))
		if err != nil {
			t.Fatalf("search request: %v", err)
		}
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK && bytes.Contains(data, []byte(entryResp.EntryID)) {
			// quick sanity: verify latestContext keys present (optional)
			if !bytes.Contains(data, []byte("latestContext")) || !bytes.Contains(data, []byte("entries")) {
				t.Fatalf("search API response missing expected fields: %s", string(data))
			}
			break // success
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// -----------------------------------------------------------------------------
//
// Test 3: Context API round-trip (PUT + GET)
//
// -----------------------------------------------------------------------------
func TestDevEnv_ContextAPI_PutGet(t *testing.T) {
	if testing.Short() {
		t.Skip("skip in short mode")
	}

	memSvc := env("MEMORY_API", "http://localhost:8080")

	// Ensure service reachable
	waitForHealthy(t, memSvc, 3*time.Second)

	// 1. user
	email := fmt.Sprintf("ctx-%d@example.com", time.Now().UnixNano())
	var userResp struct {
		UserID string `json:"userId"`
	}
	resp, err := http.Post(memSvc+"/api/users", "application/json", bytes.NewBufferString(fmt.Sprintf(`{"email":"%s"}`, email)))
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	mustJSON(t, resp, &userResp)

	// 2. vault
	vResp2, err := http.Post(fmt.Sprintf("%s/api/users/%s/vaults", memSvc, userResp.UserID), "application/json", bytes.NewBufferString(`{"title":"CtxVault"}`))
	if err != nil {
		t.Fatalf("create vault2: %v", err)
	}
	var v2 struct {
		VaultID string `json:"vaultId"`
	}
	mustJSON(t, vResp2, &v2)
	baseVaultPath2 := fmt.Sprintf("%s/api/users/%s/vaults/%s", memSvc, userResp.UserID, v2.VaultID)

	// memory under new vault
	var memResp struct {
		MemoryID string `json:"memoryId"`
	}
	body := `{"memoryType":"CONVERSATION","title":"ContextSmoke"}`
	respMem, err := http.Post(baseVaultPath2+"/memories", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("create memory2: %v", err)
	}
	mustJSON(t, respMem, &memResp)

	putURL := fmt.Sprintf("%s/memories/%s/contexts", baseVaultPath2, memResp.MemoryID)

	// 3. PUT context
	ctxPayload := `{"context":{"note":"smoke-test"}}`
	req, _ := http.NewRequest(http.MethodPut, putURL, bytes.NewBufferString(ctxPayload))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("put context: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(body))
	}
	resp.Body.Close()

	// 4. GET latest context and verify
	getURL := putURL
	resp, err = http.Get(getURL)
	if err != nil {
		t.Fatalf("get context: %v", err)
	}
	var ctxResp struct {
		Context map[string]interface{} `json:"context"`
	}
	mustJSON(t, resp, &ctxResp)
	note, ok := ctxResp.Context["note"].(string)
	if !ok || note != "smoke-test" {
		t.Fatalf("unexpected context note: %+v", ctxResp.Context)
	}
}

// -----------------------------------------------------------------------------
// local helpers (file-private)
// -----------------------------------------------------------------------------

// modelExists queries /api/tags from Ollama and checks whether the given model
// name (before ':') is present.
func modelExists(t *testing.T, baseURL, name string) bool {
	type tagResp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	r, err := http.Get(baseURL + "/api/tags")
	if err != nil {
		t.Logf("ollama tags error: %v", err)
		return false
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		t.Logf("ollama tags status %d", r.StatusCode)
		return false
	}
	var tr tagResp
	if err := json.NewDecoder(r.Body).Decode(&tr); err != nil {
		t.Logf("decode tags: %v", err)
		return false
	}
	want := strings.Split(name, ":")[0]
	for _, m := range tr.Models {
		got := strings.Split(m.Name, ":")[0]
		if got == want {
			return true
		}
	}
	t.Logf("model %s not found in tag list", name)
	return false
}
