package database

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
)

// SpannerConfig holds generic Spanner connection configuration
type SpannerConfig struct {
	ProjectID  string
	InstanceID string
	DatabaseID string
}

// NewSpannerClient creates a new Spanner client with the given configuration
func NewSpannerClient(ctx context.Context, config SpannerConfig) (*spanner.Client, error) {
	if config.ProjectID == "" || config.InstanceID == "" || config.DatabaseID == "" {
		return nil, fmt.Errorf("all Spanner config fields are required")
	}

	database := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		config.ProjectID, config.InstanceID, config.DatabaseID)

	client, err := spanner.NewClient(ctx, database)
	if err != nil {
		return nil, fmt.Errorf("failed to create Spanner client: %w", err)
	}

	return client, nil
}

// WithTransaction executes a function within a Spanner read-write transaction
func WithTransaction(ctx context.Context, client *spanner.Client, fn func(*spanner.ReadWriteTransaction) error) error {
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return fn(txn)
	})
	return err
}

// HealthCheck performs a basic health check on the Spanner client
func HealthCheck(ctx context.Context, client *spanner.Client) error {
	// Simple query to test connectivity
	stmt := spanner.Statement{SQL: "SELECT 1"}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	_, err := iter.Next()
	if err != nil {
		return fmt.Errorf("spanner health check failed: %w", err)
	}

	return nil
}
