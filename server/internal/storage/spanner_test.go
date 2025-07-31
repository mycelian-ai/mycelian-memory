package storage

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	"cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/api/option"
)

const (
	PROJECT_ID  = "local-project"
	INSTANCE_ID = "local-instance"
	DATABASE_ID = "local-database"
)

// Test constants - following your Kotlin safety checks
var (
	spannerEmulator testcontainers.Container
	emulatorHost    string
	spannerClient   *spanner.Client
)

// TestMain sets up the Spanner emulator for all tests
func TestMain(m *testing.M) {
	ctx := context.Background()

	// Setup emulator
	if err := setupSpannerEmulator(ctx); err != nil {
		fmt.Printf("Failed to setup Spanner emulator: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	if err := cleanupSpannerEmulator(ctx); err != nil {
		fmt.Printf("Failed to cleanup Spanner emulator: %v\n", err)
	}

	os.Exit(code)
}

func setupSpannerEmulator(ctx context.Context) error {
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

	spannerEmulator = container

	// Get emulator endpoint
	host, err := container.Host(ctx)
	if err != nil {
		return fmt.Errorf("failed to get container host: %w", err)
	}

	port, err := container.MappedPort(ctx, "9010")
	if err != nil {
		return fmt.Errorf("failed to get container port: %w", err)
	}

	emulatorHost = fmt.Sprintf("%s:%s", host, port.Port())

	// Set emulator environment variable - critical for auth bypass
	os.Setenv("SPANNER_EMULATOR_HOST", emulatorHost)

	// Safety check - ensure we're pointing to emulator
	if !isEmulatorHost(emulatorHost) {
		return fmt.Errorf("CRITICAL: emulator host %s does not look like a local emulator", emulatorHost)
	}

	// Create instance and database
	if err := createInstance(ctx); err != nil {
		return fmt.Errorf("failed to create instance: %w", err)
	}

	if err := createDatabase(ctx); err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	// Create Spanner client
	if err := createSpannerClient(ctx); err != nil {
		return fmt.Errorf("failed to create spanner client: %w", err)
	}

	fmt.Printf("Spanner emulator setup complete: %s\n", emulatorHost)
	return nil
}

func createInstance(ctx context.Context) error {
	// Create instance admin client with no credentials (emulator mode)
	client, err := instance.NewInstanceAdminClient(ctx, option.WithoutAuthentication())
	if err != nil {
		return fmt.Errorf("failed to create instance admin client: %w", err)
	}
	defer client.Close()

	// Create instance request
	req := &instancepb.CreateInstanceRequest{
		Parent:     fmt.Sprintf("projects/%s", PROJECT_ID),
		InstanceId: INSTANCE_ID,
		Instance: &instancepb.Instance{
			Name:        fmt.Sprintf("projects/%s/instances/%s", PROJECT_ID, INSTANCE_ID),
			Config:      fmt.Sprintf("projects/%s/instanceConfigs/emulator-config", PROJECT_ID),
			DisplayName: "Local Test Instance",
			NodeCount:   1,
		},
	}

	// Create instance (handle already exists)
	op, err := client.CreateInstance(ctx, req)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "ALREADY_EXISTS") {
			fmt.Printf("Instance %s already exists\n", INSTANCE_ID)
			return nil
		}
		return fmt.Errorf("failed to create instance: %w", err)
	}

	_, err = op.Wait(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait for instance creation: %w", err)
	}

	fmt.Printf("Instance %s created successfully\n", INSTANCE_ID)
	return nil
}

func createDatabase(ctx context.Context) error {
	// Create database admin client with no credentials (emulator mode)
	client, err := database.NewDatabaseAdminClient(ctx, option.WithoutAuthentication())
	if err != nil {
		return fmt.Errorf("failed to create database admin client: %w", err)
	}
	defer client.Close()

	// Read schema from embedded file
	ddlStatements := getDDLStatements()

	// Create database request
	req := &databasepb.CreateDatabaseRequest{
		Parent:          fmt.Sprintf("projects/%s/instances/%s", PROJECT_ID, INSTANCE_ID),
		CreateStatement: fmt.Sprintf("CREATE DATABASE `%s`", DATABASE_ID),
		ExtraStatements: ddlStatements,
	}

	// Drop database if it exists to ensure fresh schema
	dropErr := client.DropDatabase(ctx, &databasepb.DropDatabaseRequest{
		Database: fmt.Sprintf("projects/%s/instances/%s/databases/%s", PROJECT_ID, INSTANCE_ID, DATABASE_ID),
	})
	if dropErr != nil && !strings.Contains(dropErr.Error(), "not found") && !strings.Contains(dropErr.Error(), "NOT_FOUND") {
		fmt.Printf("Warning: failed to drop database: %v\n", dropErr)
	} else if dropErr == nil {
		fmt.Printf("Dropped existing database %s\n", DATABASE_ID)
	}

	// Create database with fresh schema
	op, err := client.CreateDatabase(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	_, err = op.Wait(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait for database creation: %w", err)
	}

	fmt.Printf("Database %s created successfully\n", DATABASE_ID)
	return nil
}

func createSpannerClient(ctx context.Context) error {
	databasePath := fmt.Sprintf("projects/%s/instances/%s/databases/%s", PROJECT_ID, INSTANCE_ID, DATABASE_ID)

	client, err := spanner.NewClient(ctx, databasePath, option.WithoutAuthentication())
	if err != nil {
		return fmt.Errorf("failed to create spanner client: %w", err)
	}

	spannerClient = client
	return nil
}

func cleanupTables(ctx context.Context) error {
	// Safety check - ensure we're only cleaning emulator
	currentHost := os.Getenv("SPANNER_EMULATOR_HOST")
	if !isEmulatorHost(currentHost) {
		return fmt.Errorf("CRITICAL: refusing to cleanup - not pointing to emulator host: %s", currentHost)
	}

	if PROJECT_ID != "local-project" || INSTANCE_ID != "local-instance" || DATABASE_ID != "local-database" {
		return fmt.Errorf("CRITICAL: refusing to cleanup - not using test database identifiers")
	}

	// Clean tables in reverse dependency order
	_, err := spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
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
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to cleanup tables: %w", err)
	}

	fmt.Println("Test tables cleaned successfully")
	return nil
}

func cleanupSpannerEmulator(ctx context.Context) error {
	if spannerClient != nil {
		spannerClient.Close()
	}

	if spannerEmulator != nil {
		return spannerEmulator.Terminate(ctx)
	}

	return nil
}

// Safety check following your Kotlin pattern
func isEmulatorHost(host string) bool {
	return strings.Contains(host, "localhost") || strings.Contains(host, "127.0.0.1") || strings.Contains(host, "emulator")
}

// getDDLStatements parses the embedded schema file
func getDDLStatements() []string {
	return DefaultDDLStatements()
}

// Basic test to verify emulator setup
func TestSpannerEmulatorSetup(t *testing.T) {
	ctx := context.Background()

	// Verify emulator is running
	if spannerClient == nil {
		t.Fatal("Spanner client is nil - emulator setup failed")
	}

	// Clean up any data from previous tests
	err := cleanupTables(ctx)
	if err != nil {
		t.Fatalf("Failed to cleanup tables: %v", err)
	}

	// Try a simple query to verify tables exist
	stmt := spanner.Statement{SQL: "SELECT COUNT(*) FROM Users"}
	iter := spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		t.Fatalf("Failed to query Users table: %v", err)
	}

	var count int64
	if err := row.Columns(&count); err != nil {
		t.Fatalf("Failed to read count: %v", err)
	}

	// Should be 0 (empty table after cleanup)
	if count != 0 {
		t.Errorf("Expected 0 users after cleanup, got %d", count)
	}

	t.Logf("Spanner emulator setup verified - Users table exists and is clean")
}

// TestCreateMemory_DefaultContext verifies that CreateMemory automatically inserts a default context snapshot.
func TestCreateMemory_DefaultContext(t *testing.T) {
	ctx := context.Background()

	storage := NewSpannerStorage(spannerClient)

	// Create user first – Memories table has FK to Users
	userReq := CreateUserRequest{Email: "default@example.com", TimeZone: "UTC"}
	user, err := storage.CreateUser(ctx, userReq)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	userID := user.UserID

	vaultID := uuid.New()

	// Create vault prerequisite
	_, err = storage.CreateVault(ctx, CreateVaultRequest{
		UserID:  userID,
		VaultID: vaultID,
		Title:   "Default Context Vault",
	})
	if err != nil {
		t.Fatalf("create vault: %v", err)
	}

	memReq := CreateMemoryRequest{
		VaultID:    vaultID,
		UserID:     userID,
		MemoryType: "PROJECT",
		Title:      "Default context check",
	}

	mem, err := storage.CreateMemory(ctx, memReq)
	if err != nil {
		t.Fatalf("create memory: %v", err)
	}

	// Immediately fetch latest context – should exist and match default payload
	ctxSnap, err := storage.GetLatestMemoryContext(ctx, userID, vaultID, mem.MemoryID)
	if err != nil {
		t.Fatalf("get latest context: %v", err)
	}

	if ctxSnap == nil || len(ctxSnap.Context) == 0 {
		t.Fatalf("expected non-empty default context")
	}

	s := string(ctxSnap.Context)
	if !strings.Contains(s, "default context that's created with the memory") {
		t.Fatalf("unexpected default context payload: %s", s)
	}
}
