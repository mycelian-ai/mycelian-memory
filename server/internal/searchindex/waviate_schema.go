package searchindex

import (
	"context"
	"fmt"
	"time"

	weaviate "github.com/weaviate/weaviate-go-client/v5/weaviate"
	"github.com/weaviate/weaviate/entities/models"
)

// BootstrapWeaviate ensures required classes exist with multi-tenancy enabled.
// In dev/e2e, if classes exist without MT enabled, they are dropped and recreated.
func BootstrapWeaviate(ctx context.Context, baseURL string) error {
	cfg := weaviate.Config{Scheme: "http", Host: baseURL}
	cl, err := weaviate.NewClient(cfg)
	if err != nil {
		return err
	}

	cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	entry := &models.Class{
		Class:      "MemoryEntry",
		Vectorizer: "none",
		Properties: []*models.Property{
			{Name: "entryId", DataType: []string{"uuid"}},
			{Name: "userId", DataType: []string{"text"}},
			{Name: "memoryId", DataType: []string{"uuid"}},
			{Name: "rawEntry", DataType: []string{"text"}},
			{Name: "summary", DataType: []string{"text"}},
			{Name: "tags", DataType: []string{"text[]"}},
			{Name: "creationTime", DataType: []string{"date"}},
		},
		MultiTenancyConfig: &models.MultiTenancyConfig{Enabled: true},
	}

	ctxCls := &models.Class{
		Class:      "MemoryContext",
		Vectorizer: "none",
		Properties: []*models.Property{
			{Name: "contextId", DataType: []string{"uuid"}},
			{Name: "userId", DataType: []string{"text"}},
			{Name: "memoryId", DataType: []string{"uuid"}},
			{Name: "context", DataType: []string{"text"}},
			{Name: "creationTime", DataType: []string{"date"}},
		},
		MultiTenancyConfig: &models.MultiTenancyConfig{Enabled: true},
	}

	if err := ensureMTClass(cctx, cl, entry); err != nil {
		return fmt.Errorf("bootstrap MemoryEntry: %w", err)
	}
	if err := ensureEntryTagsProperty(cctx, cl); err != nil {
		return fmt.Errorf("ensure tags property: %w", err)
	}
	if err := ensureMTClass(cctx, cl, ctxCls); err != nil {
		return fmt.Errorf("bootstrap MemoryContext: %w", err)
	}
	return nil
}

func ensureMTClass(ctx context.Context, cl *weaviate.Client, desired *models.Class) error {
	ex, err := cl.Schema().ClassGetter().WithClassName(desired.Class).Do(ctx)
	if err == nil && ex != nil {
		if ex.MultiTenancyConfig != nil && ex.MultiTenancyConfig.Enabled {
			return nil
		}
		if err := cl.Schema().ClassDeleter().WithClassName(desired.Class).Do(ctx); err != nil {
			return fmt.Errorf("delete class %s: %w", desired.Class, err)
		}
	}
	if err := cl.Schema().ClassCreator().WithClass(desired).Do(ctx); err != nil {
		return fmt.Errorf("create class %s: %w", desired.Class, err)
	}
	return nil
}

func ensureEntryTagsProperty(ctx context.Context, cl *weaviate.Client) error {
	ex, err := cl.Schema().ClassGetter().WithClassName("MemoryEntry").Do(ctx)
	if err != nil || ex == nil {
		return err
	}
	for _, p := range ex.Properties {
		if p.Name == "tags" {
			return nil
		}
	}
	prop := &models.Property{Name: "tags", DataType: []string{"text[]"}}
	return cl.Schema().PropertyCreator().WithClassName("MemoryEntry").WithProperty(prop).Do(ctx)
}
