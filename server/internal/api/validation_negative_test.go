package api

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPI_NegativeValidation(t *testing.T) {
	cleanupAPITables(t)

	//---------------- CreateUser ----------------
	t.Run("CreateUser invalid email", func(t *testing.T) {
		bad := map[string]interface{}{"userId": "neg_user1", "email": "bad", "timeZone": "UTC"}
		resp := makeRequest(t, "POST", "/api/users", bad)
		assert.Equal(t, 400, resp.StatusCode)
	})

	//---------------- Prepare valid user & memory for further tests -------------
	createUser := map[string]interface{}{"userId": "neg_valid_user", "email": "valid@example.com", "timeZone": "UTC"}
	r := makeRequest(t, "POST", "/api/users", createUser)
	require.Equal(t, 201, r.StatusCode)
	var userID string
	parseResponse(t, r, &struct {
		UserID *string `json:"userId"`
	}{&userID})
	if userID == "" {
		t.Fatalf("failed to get userID")
	}

	// create vault for further tests
	createVaultReq := map[string]interface{}{"title": "validation-vault"}
	vResp := makeRequest(t, "POST", "/api/users/"+userID+"/vaults", createVaultReq)
	require.Equal(t, 201, vResp.StatusCode)
	var vaultID uuid.UUID
	parseResponse(t, vResp, &struct {
		VaultID *uuid.UUID `json:"vaultId"`
	}{&vaultID})

	baseVaultPath := "/api/users/" + userID + "/vaults/" + vaultID.String()

	longTitle := strings.Repeat("a", 257)
	badMem := map[string]interface{}{"memoryType": "PROJECT", "title": longTitle}
	resp := makeRequest(t, "POST", baseVaultPath+"/memories", badMem)
	assert.Equal(t, 400, resp.StatusCode)

	// create good memory for entry tests
	goodMem := map[string]interface{}{"memoryType": "PROJECT", "title": "ok"}
	r = makeRequest(t, "POST", baseVaultPath+"/memories", goodMem)
	require.Equal(t, 201, r.StatusCode)
	var memID string
	parseResponse(t, r, &struct {
		MemoryID *string `json:"memoryId"`
	}{&memID})

	//---------------- CreateMemoryEntry with bad metadata ----------------
	badEntry := map[string]interface{}{"rawEntry": "hi", "metadata": "notobject"}
	resp = makeRequest(t, "POST", baseVaultPath+"/memories/"+memID+"/entries", badEntry)
	assert.Equal(t, 400, resp.StatusCode)

	//---------------- PutMemoryContext with non-string fragment ----------
	badCtx := `{"agenda": 123}`
	resp = makeRequest(t, "PUT", baseVaultPath+"/memories/"+memID+"/contexts", map[string]interface{}{"context": jsonRaw(badCtx)})
	assert.Equal(t, 400, resp.StatusCode)

	//---------------- Search empty query --------------------------------
	searchReq := map[string]interface{}{"userId": userID, "vaultId": vaultID.String(), "memoryId": memID, "query": " "}
	resp = makeRequest(t, "POST", "/api/search", searchReq)
	assert.Equal(t, 400, resp.StatusCode)
}

// jsonRaw is helper to embed raw json string in map for request payload.
func jsonRaw(s string) interface{} { return jsonRawMessage(s) }

type jsonRawMessage string

func (j jsonRawMessage) MarshalJSON() ([]byte, error) { return []byte(j), nil }
