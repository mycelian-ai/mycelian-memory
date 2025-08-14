//go:build e2e
// +build e2e

package e2e

import (
	"bytes"
	"encoding/base64"
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
// /v0/search endpoint to give a quick signal that the indexer pipeline is
// healthy.
func TestDevEnv_Ingestion_BM25_Direct(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	memSvc := env("MEMORY_API", "http://localhost:11545")
	weaviate := env("WEAVIATE_URL", "http://localhost:8082")

	// quick connectivity checks – skip if the stack isn't up
	for _, url := range []string{memSvc + "/v0/health", weaviate + "/v1/meta"} {
		if err := ping(url); err != nil {
			t.Skipf("service %s unreachable: %v", url, err)
		}
	}

	// 1. Ensure dedicated test user and create weaviate tenant
	userResp := struct {
		UserID string `json:"userId"`
	}{UserID: ensureUser(t, memSvc, env("E2E_USER", "test_user"), "test.user@example.com")}
	ensureWeaviateTenants(t, weaviate, userResp.UserID)

	// 2. Create vault then memory (unique per run) and ensure cleanup
	var vaultResp struct {
		VaultID string `json:"vaultId"`
	}
	vPayload := fmt.Sprintf(`{"title":"BmVault-%d"}`, time.Now().UnixNano())
	vResp, err := http.Post(fmt.Sprintf("%s/v0/users/%s/vaults", memSvc, userResp.UserID), "application/json", bytes.NewBufferString(vPayload))
	if err != nil {
		t.Fatalf("create vault: %v", err)
	}
	mustJSON(t, vResp, &vaultResp)
	// Cleanup vault at end
	defer func() {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/v0/users/%s/vaults/%s", memSvc, userResp.UserID, vaultResp.VaultID), nil)
		_, _ = http.DefaultClient.Do(req)
	}()

	baseVaultPath := fmt.Sprintf("%s/v0/users/%s/vaults/%s", memSvc, userResp.UserID, vaultResp.VaultID)

	// 2. Create memory
	var memResp struct {
		MemoryID string `json:"memoryId"`
	}
	body := fmt.Sprintf(`{"memoryType":"CONVERSATION","title":"BmSmoke-%d"}`, time.Now().UnixNano())
	respM, err := http.Post(baseVaultPath+"/memories", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("create memory: %v", err)
	}
	mustJSON(t, respM, &memResp)

	// 2b. Verify default context snapshot exists immediately after memory creation
	ctxURL := fmt.Sprintf("%s/memories/%s/contexts", baseVaultPath, memResp.MemoryID)
	ctxResp, err := http.Get(ctxURL)
	if err != nil {
		t.Fatalf("get default context: %v", err)
	}
	// Accept either string or object for the `context` field
	var raw map[string]interface{}
	mustJSON(t, ctxResp, &raw)
	expectedDefault := "This is default context that's created with the memory. Instructions for AI Agent: Provide relevant context as soon as it's available."
	ctxVal, ok := raw["context"]
	if !ok {
		t.Fatalf("missing context field in response: %+v", raw)
	}
	switch v := ctxVal.(type) {
	case string:
		// Attempt base64 decode; if JSON with activeContext, extract and compare
		if dec, err := base64.StdEncoding.DecodeString(v); err == nil {
			var m map[string]interface{}
			if json.Unmarshal(dec, &m) == nil {
				if s, _ := m["activeContext"].(string); s == expectedDefault {
					break
				}
				t.Fatalf("default context mismatch. want %q, got decoded %+v", expectedDefault, m)
			}
		}
		if v != expectedDefault {
			t.Fatalf("default context mismatch. want %q, got %q", expectedDefault, v)
		}
	case map[string]interface{}:
		active, _ := v["activeContext"].(string)
		if active != expectedDefault {
			t.Fatalf("default context mismatch. want %q, got %q", expectedDefault, active)
		}
	default:
		t.Fatalf("unexpected context type %T", v)
	}

	// 3. Create entry
	entryText := fmt.Sprintf("BM25 smoke test %d", time.Now().UnixNano())
	entryBody := fmt.Sprintf(`{"rawEntry":"%s","summary":"smoke summary"}`, entryText)
	respE, err := http.Post(fmt.Sprintf("%s/memories/%s/entries", baseVaultPath, memResp.MemoryID), "application/json", bytes.NewBufferString(entryBody))
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}
	var entryResp struct {
		EntryID string `json:"entryId"`
	}
	mustJSON(t, respE, &entryResp)

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
		r, err := http.Post(weaviate+"/v1/graphql", "application/json", bytes.NewBuffer(buf))
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
//	Test 2: /v0/search hybrid round-trip (embedding + context enrichment)
//
// -----------------------------------------------------------------------------
func TestDevEnv_SearchAPI_Hybrid(t *testing.T) {
	if testing.Short() {
		t.Skip("skip in short mode")
	}

	memSvc := env("MEMORY_API", "http://localhost:11545")
	weaviate := env("WEAVIATE_URL", "http://localhost:8082")
	ollama := env("OLLAMA_URL", "http://localhost:11434")
	embedMod := env("EMBED_MODEL", "nomic-embed-text")

	// Ensure services are reachable
	waitForHealthy(t, memSvc, 3*time.Second)
	for _, url := range []string{memSvc + "/v0/health", weaviate + "/v1/meta", ollama + "/api/tags"} {
		if err := ping(url); err != nil {
			t.Skipf("service %s unreachable: %v", url, err)
		}
	}

	if !modelExists(t, ollama, embedMod) {
		t.Fatalf("ollama model %s not available", embedMod)
	}

	// 1. ensure test_user and create weaviate tenant
	var userResp struct {
		UserID string `json:"userId"`
	}
	userResp.UserID = ensureUser(t, memSvc, env("E2E_USER", "test_user"), "test.user@example.com")
	ensureWeaviateTenants(t, weaviate, userResp.UserID)

	// 2. vault then memory
	var vaultResp struct {
		VaultID string `json:"vaultId"`
	}
	vPayload := fmt.Sprintf(`{"title":"SearchVault-%d"}`, time.Now().UnixNano())
	vResp, err := http.Post(fmt.Sprintf("%s/v0/users/%s/vaults", memSvc, userResp.UserID), "application/json", bytes.NewBufferString(vPayload))
	if err != nil {
		t.Fatalf("create vault: %v", err)
	}
	mustJSON(t, vResp, &vaultResp)
	// Cleanup vault at end
	defer func() {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/v0/users/%s/vaults/%s", memSvc, userResp.UserID, vaultResp.VaultID), nil)
		_, _ = http.DefaultClient.Do(req)
	}()

	baseVaultPath := fmt.Sprintf("%s/v0/users/%s/vaults/%s", memSvc, userResp.UserID, vaultResp.VaultID)

	// 2. memory
	var memResp struct {
		MemoryID string `json:"memoryId"`
	}
	body := fmt.Sprintf(`{"memoryType":"CONVERSATION","title":"SearchSmoke-%d"}`, time.Now().UnixNano())
	resp, err := http.Post(baseVaultPath+"/memories", "application/json", bytes.NewBufferString(body))
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
	objectURL := fmt.Sprintf("%s/v1/objects/MemoryEntry/%s?tenant=%s", weaviate, entryResp.EntryID, userResp.UserID)
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
		rs, err := http.Post(memSvc+"/v0/search", "application/json", bytes.NewBufferString(searchBody))
		if err != nil {
			t.Fatalf("search request: %v", err)
		}
		data, _ := io.ReadAll(rs.Body)
		_ = rs.Body.Close()
		if rs.StatusCode == http.StatusOK && bytes.Contains(data, []byte(entryResp.EntryID)) {
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

	memSvc := env("MEMORY_API", "http://localhost:11545")

	// Ensure service reachable
	waitForHealthy(t, memSvc, 3*time.Second)

	// 1. ensure test_user
	var userResp struct {
		UserID string `json:"userId"`
	}
	userResp.UserID = ensureUser(t, memSvc, env("E2E_USER", "test_user"), "test.user@example.com")

	// 2. vault
	vResp2, err := http.Post(fmt.Sprintf("%s/v0/users/%s/vaults", memSvc, userResp.UserID), "application/json", bytes.NewBufferString(`{"title":"CtxVault"}`))
	if err != nil {
		t.Fatalf("create vault2: %v", err)
	}
	var v2 struct {
		VaultID string `json:"vaultId"`
	}
	mustJSON(t, vResp2, &v2)
	baseVaultPath2 := fmt.Sprintf("%s/v0/users/%s/vaults/%s", memSvc, userResp.UserID, v2.VaultID)
	// Cleanup vault at end
	defer func() {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/v0/users/%s/vaults/%s", memSvc, userResp.UserID, v2.VaultID), nil)
		_, _ = http.DefaultClient.Do(req)
	}()

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
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("put context: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(body))
	}
	_ = resp.Body.Close()

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
