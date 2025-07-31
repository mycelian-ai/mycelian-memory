package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"google.golang.org/api/option"
)

type Config struct {
	ProjectID       string
	InstanceID      string
	DatabaseID      string
	CredentialsFile string
	EmulatorHost    string
	SchemaFile      string
}

func main() {
	var config Config
	var operation string

	flag.StringVar(&config.ProjectID, "project", "", "Google Cloud Project ID")
	flag.StringVar(&config.InstanceID, "instance", "", "Spanner Instance ID")
	flag.StringVar(&config.DatabaseID, "database", "", "Spanner Database ID")
	flag.StringVar(&config.CredentialsFile, "credentials", "", "Path to service account credentials file (optional for emulator)")
	flag.StringVar(&config.EmulatorHost, "emulator", "", "Spanner emulator host (e.g., localhost:9010)")
	flag.StringVar(&config.SchemaFile, "schema", "internal/storage/schema.sql", "Path to schema file")
	flag.StringVar(&operation, "operation", "", "Operation: create-tables, drop-tables, validate-schema")
	flag.Parse()

	if operation == "" {
		fmt.Println("Usage: schema-manager [flags] -operation <operation>")
		fmt.Println("\nOperations:")
		fmt.Println("  create-tables    Create tables from schema file")
		fmt.Println("  drop-tables      Drop all tables (MANUAL CONFIRMATION REQUIRED)")
		fmt.Println("  validate-schema  Validate current schema against file")
		fmt.Println("\nFlags:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if config.ProjectID == "" || config.InstanceID == "" || config.DatabaseID == "" {
		log.Fatal("❌ Project ID, Instance ID, and Database ID are required")
	}

	ctx := context.Background()

	// Setup client options
	var opts []option.ClientOption
	if config.EmulatorHost != "" {
		opts = append(opts, option.WithEndpoint(config.EmulatorHost))
		opts = append(opts, option.WithoutAuthentication())
		fmt.Printf("🔧 Using Spanner emulator: %s\n", config.EmulatorHost)
	} else if config.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(config.CredentialsFile))
	}

	manager := &SchemaManager{
		config: config,
		opts:   opts,
	}

	switch operation {
	case "create-tables":
		if err := manager.CreateTables(ctx); err != nil {
			log.Fatalf("❌ Failed to create tables: %v", err)
		}
	case "drop-tables":
		if err := manager.DropTables(ctx); err != nil {
			log.Fatalf("❌ Failed to drop tables: %v", err)
		}
	case "validate-schema":
		if err := manager.ValidateSchema(ctx); err != nil {
			log.Fatalf("❌ Schema validation failed: %v", err)
		}
	default:
		log.Fatalf("❌ Unknown operation: %s", operation)
	}
}

type SchemaManager struct {
	config Config
	opts   []option.ClientOption
}

func (sm *SchemaManager) CreateTables(ctx context.Context) error {
	fmt.Println("🏗️  Creating tables from schema file...")

	// Read schema file
	schemaPath, err := filepath.Abs(sm.config.SchemaFile)
	if err != nil {
		return fmt.Errorf("failed to resolve schema path: %w", err)
	}

	schemaContent, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file %s: %w", schemaPath, err)
	}

	fmt.Printf("📄 Reading schema from: %s\n", schemaPath)

	// Parse DDL statements
	statements := sm.parseDDLStatements(string(schemaContent))
	if len(statements) == 0 {
		return fmt.Errorf("no DDL statements found in schema file")
	}

	fmt.Printf("📋 Found %d DDL statements\n", len(statements))

	// Create database admin client
	adminClient, err := database.NewDatabaseAdminClient(ctx, sm.opts...)
	if err != nil {
		return fmt.Errorf("failed to create admin client: %w", err)
	}
	defer adminClient.Close()

	databasePath := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		sm.config.ProjectID, sm.config.InstanceID, sm.config.DatabaseID)

	// Apply DDL statements
	fmt.Println("⚡ Applying DDL statements...")
	for i, stmt := range statements {
		fmt.Printf("  %d. %s\n", i+1, sm.truncateStatement(stmt))
	}

	op, err := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
		Database:   databasePath,
		Statements: statements,
	})
	if err != nil {
		return fmt.Errorf("failed to update database DDL: %w", err)
	}

	fmt.Println("⏳ Waiting for operation to complete...")
	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("DDL operation failed: %w", err)
	}

	fmt.Println("✅ Tables created successfully!")
	return nil
}

func (sm *SchemaManager) DropTables(ctx context.Context) error {
	fmt.Println("🚨 WARNING: This will DROP ALL TABLES in the database!")
	fmt.Println("🚨 This operation is IRREVERSIBLE and will DELETE ALL DATA!")
	fmt.Print("Type 'DELETE ALL TABLES' to confirm: ")

	var confirmation string
	fmt.Scanln(&confirmation)

	if confirmation != "DELETE ALL TABLES" {
		fmt.Println("❌ Operation cancelled")
		return nil
	}

	fmt.Println("🗑️  Dropping all tables...")

	// Create Spanner client to list tables
	client, err := spanner.NewClient(ctx, fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		sm.config.ProjectID, sm.config.InstanceID, sm.config.DatabaseID), sm.opts...)
	if err != nil {
		return fmt.Errorf("failed to create spanner client: %w", err)
	}
	defer client.Close()

	// Query for all tables
	stmt := spanner.Statement{
		SQL: `SELECT table_name FROM information_schema.tables 
			  WHERE table_catalog = '' AND table_schema = '' 
			  ORDER BY table_name`,
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var tables []string
	for {
		row, err := iter.Next()
		if err != nil {
			break
		}
		var tableName string
		if err := row.Column(0, &tableName); err != nil {
			return fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, tableName)
	}

	if len(tables) == 0 {
		fmt.Println("📭 No tables found to drop")
		return nil
	}

	// Create drop statements (reverse order due to foreign keys)
	var dropStatements []string
	for i := len(tables) - 1; i >= 0; i-- {
		dropStatements = append(dropStatements, fmt.Sprintf("DROP TABLE %s", tables[i]))
	}

	// Create database admin client
	adminClient, err := database.NewDatabaseAdminClient(ctx, sm.opts...)
	if err != nil {
		return fmt.Errorf("failed to create admin client: %w", err)
	}
	defer adminClient.Close()

	databasePath := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		sm.config.ProjectID, sm.config.InstanceID, sm.config.DatabaseID)

	fmt.Printf("🗑️  Dropping %d tables...\n", len(tables))
	for _, table := range tables {
		fmt.Printf("  - %s\n", table)
	}

	op, err := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
		Database:   databasePath,
		Statements: dropStatements,
	})
	if err != nil {
		return fmt.Errorf("failed to drop tables: %w", err)
	}

	fmt.Println("⏳ Waiting for operation to complete...")
	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("drop operation failed: %w", err)
	}

	fmt.Println("✅ All tables dropped successfully!")
	return nil
}

func (sm *SchemaManager) ValidateSchema(ctx context.Context) error {
	fmt.Println("🔍 Validating current schema against file...")

	// Read schema file
	schemaContent, err := os.ReadFile(sm.config.SchemaFile)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	expectedStatements := sm.parseDDLStatements(string(schemaContent))
	fmt.Printf("📄 Expected %d DDL statements from schema file\n", len(expectedStatements))

	// Create Spanner client
	client, err := spanner.NewClient(ctx, fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		sm.config.ProjectID, sm.config.InstanceID, sm.config.DatabaseID), sm.opts...)
	if err != nil {
		return fmt.Errorf("failed to create spanner client: %w", err)
	}
	defer client.Close()

	// Query current tables
	stmt := spanner.Statement{
		SQL: `SELECT table_name FROM information_schema.tables 
			  WHERE table_catalog = '' AND table_schema = '' 
			  ORDER BY table_name`,
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var currentTables []string
	for {
		row, err := iter.Next()
		if err != nil {
			break
		}
		var tableName string
		if err := row.Column(0, &tableName); err != nil {
			return fmt.Errorf("failed to scan table name: %w", err)
		}
		currentTables = append(currentTables, tableName)
	}

	fmt.Printf("🏗️  Current database has %d tables\n", len(currentTables))
	for _, table := range currentTables {
		fmt.Printf("  - %s\n", table)
	}

	// Simple validation - check if we have the expected tables
	// Keep this list in sync with internal/storage/schema.sql top-level tables
	expectedTables := []string{"Users", "Vaults", "Memories", "MemoryEntries", "MemoryContexts"}
	var missingTables []string

	for _, expected := range expectedTables {
		found := false
		for _, current := range currentTables {
			if current == expected {
				found = true
				break
			}
		}
		if !found {
			missingTables = append(missingTables, expected)
		}
	}

	if len(missingTables) > 0 {
		fmt.Printf("❌ Missing tables: %v\n", missingTables)
		fmt.Println("💡 Run with -operation create-tables to create missing tables")
		return fmt.Errorf("schema validation failed")
	}

	fmt.Println("✅ Schema validation passed!")
	return nil
}

func (sm *SchemaManager) parseDDLStatements(schema string) []string {
	// Remove comments but preserve statement structure
	lines := strings.Split(schema, "\n")
	var cleanLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip empty lines and comment-only lines
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		// Remove inline comments but preserve the line structure
		if commentPos := strings.Index(line, "--"); commentPos != -1 {
			line = strings.TrimSpace(line[:commentPos])
			if line == "" {
				continue
			}
		}
		cleanLines = append(cleanLines, line)
	}

	// Join lines with proper spacing and split by semicolon
	fullSchema := strings.Join(cleanLines, "\n")
	statements := strings.Split(fullSchema, ";")

	var ddlStatements []string
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt != "" {
			ddlStatements = append(ddlStatements, stmt)
		}
	}

	return ddlStatements
}

func (sm *SchemaManager) truncateStatement(stmt string) string {
	if len(stmt) > 80 {
		return stmt[:77] + "..."
	}
	return stmt
}
