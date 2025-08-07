package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mycelian/mycelian-memory/server/internal/storage"

	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	"cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/api/option"
)

const (
	API_PROJECT_ID  = "api-test-project"
	API_INSTANCE_ID = "api-test-instance"
	API_DATABASE_ID = "api-test-database"
)

var (
	apiSpannerEmulator testcontainers.Container
	apiEmulatorHost    string
	apiSpannerClient   *spanner.Client
	apiStorage         storage.Storage
	apiServer          *httptest.Server
)

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	ctx := context.Background()

	// Setup Spanner emulator for API tests
	if err := setupAPISpannerEmulator(ctx); err != nil {
		fmt.Printf("Failed to setup API Spanner emulator: %v\n", err)
		os.Exit(1)
	}

	// Create storage and test server
	apiStorage = storage.NewSpannerStorage(apiSpannerClient)
	router := NewRouter(apiStorage)
	apiServer = httptest.NewServer(router)

	// Run tests
	code := m.Run()

	// Cleanup
	apiServer.Close()
	if err := cleanupAPISpannerEmulator(ctx); err != nil {
		fmt.Printf("Failed to cleanup API Spanner emulator: %v\n", err)
	}

	os.Exit(code)
}

func setupAPISpannerEmulator(ctx context.Context) error {
	// Start Spanner emulator container
	req := testcontainers.ContainerRequest{
		Image:        "gcr.io/cloud-spanner-emulator/emulator:latest",
		ExposedPorts: []string{"9010/tcp", "9020/tcp"},
		WaitingFor:   wait.ForLog("Cloud Spanner emulator running"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	apiSpannerEmulator = container

	// Get emulator endpoint
	host, err := container.Host(ctx)
	if err != nil {
		return fmt.Errorf("failed to get container host: %w", err)
	}

	port, err := container.MappedPort(ctx, "9010")
	if err != nil {
		return fmt.Errorf("failed to get container port: %w", err)
	}

	apiEmulatorHost = fmt.Sprintf("%s:%s", host, port.Port())
	_ = os.Setenv("SPANNER_EMULATOR_HOST", apiEmulatorHost) //nolint:errcheck

	// Safety check
	if !isAPIEmulatorHost(apiEmulatorHost) {
		return fmt.Errorf("CRITICAL: emulator host %s does not look like a local emulator", apiEmulatorHost)
	}

	// Create instance and database
	if err := createAPIInstance(ctx); err != nil {
		return fmt.Errorf("failed to create instance: %w", err)
	}

	if err := createAPIDatabase(ctx); err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	// Create Spanner client
	if err := createAPISpannerClient(ctx); err != nil {
		return fmt.Errorf("failed to create spanner client: %w", err)
	}

	fmt.Printf("API Spanner emulator setup complete: %s\n", apiEmulatorHost)
	return nil
}

func createAPIInstance(ctx context.Context) error {
	client, err := instance.NewInstanceAdminClient(ctx, option.WithoutAuthentication())
	if err != nil {
		return fmt.Errorf("failed to create instance admin client: %w", err)
	}
	defer func() { _ = client.Close() }()

	req := &instancepb.CreateInstanceRequest{
		Parent:     fmt.Sprintf("projects/%s", API_PROJECT_ID),
		InstanceId: API_INSTANCE_ID,
		Instance: &instancepb.Instance{
			Name:        fmt.Sprintf("projects/%s/instances/%s", API_PROJECT_ID, API_INSTANCE_ID),
			Config:      fmt.Sprintf("projects/%s/instanceConfigs/emulator-config", API_PROJECT_ID),
			DisplayName: "API Test Instance",
			NodeCount:   1,
		},
	}

	op, err := client.CreateInstance(ctx, req)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "ALREADY_EXISTS") {
			fmt.Printf("API Instance %s already exists\n", API_INSTANCE_ID)
			return nil
		}
		return fmt.Errorf("failed to create instance: %w", err)
	}

	_, err = op.Wait(ctx)
	return err
}

func createAPIDatabase(ctx context.Context) error {
	client, err := database.NewDatabaseAdminClient(ctx, option.WithoutAuthentication())
	if err != nil {
		return fmt.Errorf("failed to create database admin client: %w", err)
	}
	defer func() { _ = client.Close() }()

	ddlStatements := getAPIDDLStatements()

	req := &databasepb.CreateDatabaseRequest{
		Parent:          fmt.Sprintf("projects/%s/instances/%s", API_PROJECT_ID, API_INSTANCE_ID),
		CreateStatement: fmt.Sprintf("CREATE DATABASE `%s`", API_DATABASE_ID),
		ExtraStatements: ddlStatements,
	}

	// Drop database if it exists to ensure fresh schema
	dropErr := client.DropDatabase(ctx, &databasepb.DropDatabaseRequest{
		Database: fmt.Sprintf("projects/%s/instances/%s/databases/%s", API_PROJECT_ID, API_INSTANCE_ID, API_DATABASE_ID),
	})
	if dropErr != nil && !strings.Contains(dropErr.Error(), "not found") && !strings.Contains(dropErr.Error(), "NOT_FOUND") {
		fmt.Printf("Warning: failed to drop API database: %v\n", dropErr)
	}

	op, err := client.CreateDatabase(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	_, err = op.Wait(ctx)
	return err
}

func createAPISpannerClient(ctx context.Context) error {
	databasePath := fmt.Sprintf("projects/%s/instances/%s/databases/%s", API_PROJECT_ID, API_INSTANCE_ID, API_DATABASE_ID)

	client, err := spanner.NewClient(ctx, databasePath, option.WithoutAuthentication())
	if err != nil {
		return fmt.Errorf("failed to create spanner client: %w", err)
	}

	apiSpannerClient = client
	return nil
}

func cleanupAPISpannerEmulator(ctx context.Context) error {
	if apiSpannerClient != nil {
		apiSpannerClient.Close()
	}

	if apiSpannerEmulator != nil {
		return apiSpannerEmulator.Terminate(ctx)
	}

	return nil
}

func isAPIEmulatorHost(host string) bool {
	return strings.Contains(host, "localhost") || strings.Contains(host, "127.0.0.1") || strings.Contains(host, "emulator")
}

func getAPIDDLStatements() []string {
	return storage.DefaultDDLStatements()
}

// Helper function to clean tables between tests
func cleanupAPITables(t *testing.T) {
	ctx := context.Background()
	_, err := apiSpannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Clean child tables first
		_, err := txn.Update(ctx, spanner.Statement{SQL: "DELETE FROM MemoryEntries WHERE true"})
		if err != nil {
			return err
		}

		_, err = txn.Update(ctx, spanner.Statement{SQL: "DELETE FROM Memories WHERE true"})
		if err != nil {
			return err
		}

		_, err = txn.Update(ctx, spanner.Statement{SQL: "DELETE FROM Users WHERE true"})
		return err
	})
	require.NoError(t, err)
}

// Test helper functions
func makeRequest(t *testing.T, method, path string, body interface{}) *http.Response {
	var bodyReader *bytes.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		require.NoError(t, err)
		bodyReader = bytes.NewReader(bodyBytes)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	req, err := http.NewRequest(method, apiServer.URL+path, bodyReader)
	require.NoError(t, err)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	return resp
}

func parseResponse(t *testing.T, resp *http.Response, v interface{}) {
	defer resp.Body.Close()
	err := json.NewDecoder(resp.Body).Decode(v)
	require.NoError(t, err)
}

// API Integration Tests

func TestAPI_HealthEndpoints(t *testing.T) {
	t.Run("Health Check", func(t *testing.T) {
		resp := makeRequest(t, "GET", "/api/health", nil)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		parseResponse(t, resp, &result)
		assert.Equal(t, "UP", result["status"])
		assert.Equal(t, "Service is healthy", result["message"])
		assert.NotNil(t, result["timestamp"])
	})

	t.Run("Spanner Health Check", func(t *testing.T) {
		resp := makeRequest(t, "GET", "/api/health/db", nil)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		parseResponse(t, resp, &result)
		assert.Equal(t, "UP", result["status"])
		assert.Contains(t, result["message"], "database")
		assert.NotNil(t, result["timestamp"])
	})
}

func TestAPI_UserOperations(t *testing.T) {
	cleanupAPITables(t)

	var createdUser storage.User

	t.Run("Create User", func(t *testing.T) {
		createReq := map[string]interface{}{
			"userId":      "api_test_user",
			"email":       "test@example.com",
			"displayName": "Test User",
			"timeZone":    "UTC",
		}

		resp := makeRequest(t, "POST", "/api/users", createReq)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		parseResponse(t, resp, &createdUser)
		assert.Equal(t, "api_test_user", createdUser.UserID)
		assert.Equal(t, "test@example.com", createdUser.Email)
		assert.Equal(t, "Test User", *createdUser.DisplayName)
		assert.Equal(t, "UTC", createdUser.TimeZone)
		assert.Equal(t, "ACTIVE", createdUser.Status)
	})

	t.Run("Get User", func(t *testing.T) {
		resp := makeRequest(t, "GET", "/api/users/"+createdUser.UserID, nil)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var user storage.User
		parseResponse(t, resp, &user)
		assert.Equal(t, createdUser.UserID, user.UserID)
		assert.Equal(t, createdUser.Email, user.Email)
	})

	t.Run("Create User - Invalid JSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", apiServer.URL+"/api/users", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Create User - Missing Email", func(t *testing.T) {
		createReq := map[string]interface{}{
			"userId":   "missing_email_user",
			"timeZone": "UTC",
		}

		resp := makeRequest(t, "POST", "/api/users", createReq)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Get User - Not Found", func(t *testing.T) {
		resp := makeRequest(t, "GET", "/api/users/nonexistent", nil)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestAPI_MemoryOperations(t *testing.T) {
	cleanupAPITables(t)

	// Create a user first
	createUserReq := map[string]interface{}{
		"userId":   "mem_test_user",
		"email":    "memory@example.com",
		"timeZone": "UTC",
	}

	resp := makeRequest(t, "POST", "/api/users", createUserReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var user storage.User
	parseResponse(t, resp, &user)

	// Create a vault for the user
	createVaultReq := map[string]interface{}{"title": "test-vault"}
	vResp := makeRequest(t, "POST", "/api/users/"+user.UserID+"/vaults", createVaultReq)
	require.Equal(t, http.StatusCreated, vResp.StatusCode)
	var vault storage.Vault
	parseResponse(t, vResp, &vault)

	baseVaultPath := "/api/users/" + user.UserID + "/vaults/" + vault.VaultID.String()

	var createdMemory storage.Memory

	t.Run("Create Memory", func(t *testing.T) {
		createReq := map[string]interface{}{
			"memoryType":  "PROJECT",
			"title":       "test-memory",
			"description": "Test memory description",
		}

		resp := makeRequest(t, "POST", baseVaultPath+"/memories", createReq)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		parseResponse(t, resp, &createdMemory)
		assert.NotEmpty(t, createdMemory.MemoryID)
		assert.Equal(t, user.UserID, createdMemory.UserID)
		assert.Equal(t, "PROJECT", createdMemory.MemoryType)
		assert.Equal(t, "test-memory", createdMemory.Title)
		assert.Equal(t, "Test memory description", *createdMemory.Description)
	})

	t.Run("Get Memory", func(t *testing.T) {
		resp := makeRequest(t, "GET", baseVaultPath+"/memories/"+createdMemory.MemoryID, nil)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var memory storage.Memory
		parseResponse(t, resp, &memory)
		assert.Equal(t, createdMemory.MemoryID, memory.MemoryID)
		assert.Equal(t, createdMemory.Title, memory.Title)
	})

	t.Run("List Memories", func(t *testing.T) {
		// Create another memory
		createReq := map[string]interface{}{
			"memoryType": "CONVERSATION",
			"title":      "second-memory",
		}
		makeRequest(t, "POST", baseVaultPath+"/memories", createReq)

		resp := makeRequest(t, "GET", baseVaultPath+"/memories", nil)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		parseResponse(t, resp, &result)

		memories := result["memories"].([]interface{})
		count := result["count"].(float64)

		assert.Equal(t, float64(2), count)
		assert.Len(t, memories, 2)
	})

	t.Run("Delete Memory", func(t *testing.T) {
		resp := makeRequest(t, "DELETE", baseVaultPath+"/memories/"+createdMemory.MemoryID, nil)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		// Verify it's deleted
		resp = makeRequest(t, "GET", baseVaultPath+"/memories/"+createdMemory.MemoryID, nil)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Create Memory - Invalid Input", func(t *testing.T) {
		createReq := map[string]interface{}{
			"title": "Missing memory type",
		}

		resp := makeRequest(t, "POST", baseVaultPath+"/memories", createReq)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestAPI_MemoryEntryOperations(t *testing.T) {
	cleanupAPITables(t)

	// Create a user first
	createUserReq := map[string]interface{}{
		"userId":   "entry_test_user",
		"email":    "entry@example.com",
		"timeZone": "UTC",
	}

	resp := makeRequest(t, "POST", "/api/users", createUserReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var user storage.User
	parseResponse(t, resp, &user)

	// Create a vault for the user
	createVaultReq := map[string]interface{}{"title": "entry-test-vault"}
	vResp := makeRequest(t, "POST", "/api/users/"+user.UserID+"/vaults", createVaultReq)
	require.Equal(t, http.StatusCreated, vResp.StatusCode)
	var vault storage.Vault
	parseResponse(t, vResp, &vault)

	baseVaultPath := "/api/users/" + user.UserID + "/vaults/" + vault.VaultID.String()

	// Create a memory for the user
	createMemoryReq := map[string]interface{}{
		"memoryType": "PROJECT",
		"title":      "entry-test-memory",
	}

	resp = makeRequest(t, "POST", baseVaultPath+"/memories", createMemoryReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var memory storage.Memory
	parseResponse(t, resp, &memory)

	var createdEntry storage.MemoryEntry

	t.Run("Create Memory Entry", func(t *testing.T) {
		createReq := map[string]interface{}{
			"rawEntry": "This is a test memory entry",
			"summary":  "Test entry summary",
			"metadata": map[string]interface{}{
				"key":  "value",
				"type": "test",
			},
		}

		resp := makeRequest(t, "POST", baseVaultPath+"/memories/"+memory.MemoryID+"/entries", createReq)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		parseResponse(t, resp, &createdEntry)
		assert.NotEmpty(t, createdEntry.EntryID)
		assert.Equal(t, user.UserID, createdEntry.UserID)
		assert.Equal(t, memory.MemoryID, createdEntry.MemoryID)
		assert.Equal(t, "This is a test memory entry", createdEntry.RawEntry)
		assert.Equal(t, "Test entry summary", *createdEntry.Summary)
		assert.NotNil(t, createdEntry.Metadata)
	})

	t.Run("List Memory Entries", func(t *testing.T) {
		// Create another entry
		createReq := map[string]interface{}{
			"rawEntry": "Second test entry",
			"summary":  "Second summary",
		}
		makeRequest(t, "POST", baseVaultPath+"/memories/"+memory.MemoryID+"/entries", createReq)

		resp := makeRequest(t, "GET", baseVaultPath+"/memories/"+memory.MemoryID+"/entries", nil)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		parseResponse(t, resp, &result)

		entries := result["entries"].([]interface{})
		count := result["count"].(float64)

		assert.Equal(t, float64(2), count)
		assert.Len(t, entries, 2)
	})

	t.Run("List Memory Entries with Limit", func(t *testing.T) {
		resp := makeRequest(t, "GET", baseVaultPath+"/memories/"+memory.MemoryID+"/entries?limit=1", nil)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		parseResponse(t, resp, &result)

		entries := result["entries"].([]interface{})
		count := result["count"].(float64)

		assert.Equal(t, float64(1), count)
		assert.Len(t, entries, 1)
	})

	t.Run("Create Memory Entry - Invalid Input", func(t *testing.T) {
		createReq := map[string]interface{}{
			"summary": "Missing raw entry",
		}

		resp := makeRequest(t, "POST", baseVaultPath+"/memories/"+memory.MemoryID+"/entries", createReq)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestAPI_ErrorCases(t *testing.T) {
	cleanupAPITables(t)

	t.Run("Invalid User ID in Path", func(t *testing.T) {
		resp := makeRequest(t, "GET", "/api/users/", nil)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Invalid Memory ID in Path", func(t *testing.T) {
		resp := makeRequest(t, "GET", "/api/users/user123/memories/", nil)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Nonexistent Endpoint", func(t *testing.T) {
		resp := makeRequest(t, "GET", "/api/nonexistent", nil)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestAPI_TagsOperations(t *testing.T) {
	cleanupAPITables(t)

	// Create a user first
	createUserReq := map[string]interface{}{
		"userId":   "tags_test_user",
		"email":    "tags@example.com",
		"timeZone": "UTC",
	}

	resp := makeRequest(t, "POST", "/api/users", createUserReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var user storage.User
	parseResponse(t, resp, &user)

	// Create a vault for the user
	createVaultReq := map[string]interface{}{"title": "tags-test-vault"}
	vResp := makeRequest(t, "POST", "/api/users/"+user.UserID+"/vaults", createVaultReq)
	require.Equal(t, http.StatusCreated, vResp.StatusCode)
	var vault storage.Vault
	parseResponse(t, vResp, &vault)

	baseVaultPath := "/api/users/" + user.UserID + "/vaults/" + vault.VaultID.String()

	// Create a memory to attach entries/tags
	createMemoryReq := map[string]interface{}{
		"memoryType": "PROJECT",
		"title":      "tags-test-memory",
	}

	resp = makeRequest(t, "POST", baseVaultPath+"/memories", createMemoryReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var memory storage.Memory
	parseResponse(t, resp, &memory)

	t.Run("Create Memory Entry with Tags", func(t *testing.T) {
		createEntryReq := map[string]interface{}{
			"rawEntry": "Test entry with tags",
			"summary":  "Test summary",
			"metadata": map[string]interface{}{"type": "test"},
			"tags":     map[string]interface{}{"status": "draft", "priority": "high"},
		}

		resp := makeRequest(t, "POST", baseVaultPath+"/memories/"+memory.MemoryID+"/entries", createEntryReq)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var entry storage.MemoryEntry
		parseResponse(t, resp, &entry)

		// Verify tags were set correctly
		assert.Equal(t, "Test entry with tags", entry.RawEntry)
		assert.Equal(t, map[string]interface{}{"type": "test"}, entry.Metadata)
		assert.Equal(t, map[string]interface{}{"status": "draft", "priority": "high"}, entry.Tags)

		// Test updating tags
		updateTagsReq := map[string]interface{}{
			"tags": map[string]interface{}{
				"status":   "in_progress",
				"priority": "urgent",
				"assignee": "user123",
			},
		}

		creationTime := entry.CreationTime.Format(time.RFC3339Nano)
		updateURL := baseVaultPath + "/memories/" + memory.MemoryID + "/entries/" + creationTime + "/tags"

		resp = makeRequest(t, "PATCH", updateURL, updateTagsReq)

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var updatedEntry storage.MemoryEntry
		parseResponse(t, resp, &updatedEntry)

		// Verify tags were updated but other fields remain unchanged
		assert.Equal(t, "Test entry with tags", updatedEntry.RawEntry)
		assert.Equal(t, map[string]interface{}{"type": "test"}, updatedEntry.Metadata)
		assert.Equal(t, map[string]interface{}{
			"status":   "in_progress",
			"priority": "urgent",
			"assignee": "user123",
		}, updatedEntry.Tags)
		assert.NotNil(t, updatedEntry.LastUpdateTime)
	})

	t.Run("Update Tags - Invalid Creation Time", func(t *testing.T) {
		updateTagsReq := map[string]interface{}{
			"tags": map[string]interface{}{"status": "test"},
		}

		updateURL := baseVaultPath + "/memories/" + memory.MemoryID + "/entries/invalid-time/tags"

		resp := makeRequest(t, "PATCH", updateURL, updateTagsReq)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Update Tags - Nonexistent Entry", func(t *testing.T) {
		updateTagsReq := map[string]interface{}{
			"tags": map[string]interface{}{"status": "test"},
		}

		// Use a valid RFC3339 timestamp that doesn't exist
		nonexistentTime := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
		updateURL := baseVaultPath + "/memories/" + memory.MemoryID + "/entries/" + nonexistentTime + "/tags"

		resp := makeRequest(t, "PATCH", updateURL, updateTagsReq)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
