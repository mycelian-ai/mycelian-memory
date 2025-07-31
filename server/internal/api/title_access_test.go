package api

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"memory-backend/internal/storage"
)

// TestAPI_TitleBasedAccess verifies listing memories by vault title and fetching memory by title.
// This test is expected to fail until the corresponding endpoints are implemented.
func TestAPI_TitleBasedAccess(t *testing.T) {
	cleanupAPITables(t)

	// 1. Create user
	userReq := map[string]interface{}{"userId": "title_user", "email": "title@test.com", "timeZone": "UTC"}
	resp := makeRequest(t, "POST", "/api/users", userReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var user storage.User
	parseResponse(t, resp, &user)

	// Expect deterministic userID equals provided userId
	assert.Equal(t, "title_user", user.UserID)

	// 2. Create vault using title slug
	vaultTitle := "vault-one"
	createVaultReq := map[string]interface{}{"title": vaultTitle}
	vResp := makeRequest(t, "POST", "/api/users/"+user.UserID+"/vaults", createVaultReq)
	require.Equal(t, http.StatusCreated, vResp.StatusCode)
	var vault storage.Vault
	parseResponse(t, vResp, &vault)

	// 3. Create a memory inside the vault (using legacy UUID path)
	memTitle := "mem-one"
	memReq := map[string]interface{}{"memoryType": "PROJECT", "title": memTitle}
	mResp := makeRequest(t, "POST", "/api/users/"+user.UserID+"/vaults/"+vault.VaultID.String()+"/memories", memReq)
	require.Equal(t, http.StatusCreated, mResp.StatusCode)
	var memory storage.Memory
	parseResponse(t, mResp, &memory)

	// 4. List memories by vault title (new endpoint)
	listPath := "/api/users/" + user.UserID + "/vaults/" + vaultTitle + "/memories"
	listResp := makeRequest(t, "GET", listPath, nil)
	assert.Equal(t, http.StatusOK, listResp.StatusCode)

	var listResult map[string]interface{}
	parseResponse(t, listResp, &listResult)
	assert.Equal(t, float64(1), listResult["count"].(float64))

	// 5. Get memory by title (new endpoint)
	getPath := listPath + "/" + memTitle
	getResp := makeRequest(t, "GET", getPath, nil)
	assert.Equal(t, http.StatusOK, getResp.StatusCode)

	var memDetail storage.Memory
	parseResponse(t, getResp, &memDetail)
	assert.Equal(t, memory.MemoryID, memDetail.MemoryID)
	assert.Equal(t, memTitle, memDetail.Title)
}
