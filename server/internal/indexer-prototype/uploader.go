package indexer

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/go-openapi/strfmt"
	"github.com/rs/zerolog"
	weaviate "github.com/weaviate/weaviate-go-client/v5/weaviate"
	"github.com/weaviate/weaviate/entities/models"
)

// IndexedEntry bundles a Memory entry with its dense vector.
// Vector length must match the class dimension configured in Waviate.

type IndexedEntry struct {
	Entry         Entry
	SummaryVector []float32
}

// Uploader handles class creation and batch upserts into Waviate.

type Uploader struct {
	client    *weaviate.Client
	className string
	log       zerolog.Logger
}

// NewUploader initialises a Waviate client and ensures the MemoryEntry class exists.
func NewUploader(baseURL string, log zerolog.Logger) (*Uploader, error) {
	cfg := weaviate.Config{Scheme: "http", Host: baseURL}
	clnt, err := weaviate.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	up := &Uploader{client: clnt, className: "MemoryEntry", log: log.With().Str("component", "uploader").Logger()}

	if err := up.ensureClass(context.Background()); err != nil {
		return nil, err
	}

	return up, nil
}

func (u *Uploader) ensureClass(ctx context.Context) error {
	schema, err := u.client.Schema().Getter().Do(ctx)
	if err != nil {
		return err
	}

	hasEntry := false
	hasCtx := false

	for _, cls := range schema.Classes {
		switch cls.Class {
		case u.className:
			hasEntry = true
		case "MemoryContext":
			hasCtx = true
		}
	}

	if hasEntry {
		// validate tags tokenization
		for _, cls := range schema.Classes {
			if cls.Class == u.className {
				for _, prop := range cls.Properties {
					if prop.Name == "tags" {
						if prop.Tokenization != "whitespace" {
							u.log.Warn().Str("current", prop.Tokenization).Msg("recreating MemoryEntry class with correct tags tokenization (whitespace)")
							// delete class (ignore error)
							_ = u.client.Schema().ClassDeleter().WithClassName(u.className).Do(ctx)
							hasEntry = false // forces creation below
						}
					}
				}
			}
		}
	}

	if !hasEntry {
		model := &models.Class{
			Class:              u.className,
			Vectorizer:         "none",
			MultiTenancyConfig: &models.MultiTenancyConfig{Enabled: true},
			Properties: []*models.Property{
				{Name: "entryId", DataType: []string{"string"}},
				{Name: "userId", DataType: []string{"string"}},
				{Name: "memoryId", DataType: []string{"string"}},
				{Name: "creationTime", DataType: []string{"date"}},
				{Name: "summary", DataType: []string{"text"}},
				{Name: "rawEntry", DataType: []string{"text"}},
				{Name: "tags", DataType: []string{"text"}, Tokenization: "whitespace"},
				{Name: "metadata", DataType: []string{"text"}},
			},
		}
		if err := u.client.Schema().ClassCreator().WithClass(model).Do(ctx); err != nil {
			return err
		}
	}

	if !hasCtx {
		ctxModel := &models.Class{
			Class:              "MemoryContext",
			Vectorizer:         "none",
			MultiTenancyConfig: &models.MultiTenancyConfig{Enabled: true},
			Properties: []*models.Property{
				{Name: "contextId", DataType: []string{"string"}},
				{Name: "memoryId", DataType: []string{"string"}},
				{Name: "creationTime", DataType: []string{"date"}},
				{Name: "context", DataType: []string{"text"}},
			},
		}
		if err := u.client.Schema().ClassCreator().WithClass(ctxModel).Do(ctx); err != nil {
			return err
		}
	}

	// NOTE: If the class already exists we assume tokenization is correct.
	// Developers can drop the class or update schema manually if needed.

	return nil
}

// ensureTenant creates tenant for the given class and userId if not present (idempotent)
func (u *Uploader) ensureTenant(ctx context.Context, className, userID string) {
	tenant := models.Tenant{Name: userID}
	_ = u.client.Schema().TenantsCreator().WithClassName(className).WithTenants(tenant).Do(ctx)
}

// Upsert uploads entries in batches of 10 for efficiency.
func (u *Uploader) Upsert(ctx context.Context, entries []IndexedEntry) error {
	const batchSize = 10
	for offset := 0; offset < len(entries); offset += batchSize {
		end := offset + batchSize
		if end > len(entries) {
			end = len(entries)
		}

		// ensure tenants for users in this batch
		users := make(map[string]struct{})
		for _, it := range entries[offset:end] {
			users[it.Entry.UserID] = struct{}{}
		}
		for uid := range users {
			u.ensureTenant(ctx, u.className, uid)
		}

		b := u.client.Batch().ObjectsBatcher()
		var objs []*models.Object
		for _, it := range entries[offset:end] {
			props := map[string]interface{}{
				"entryId":      it.Entry.EntryID,
				"userId":       it.Entry.UserID,
				"memoryId":     it.Entry.MemoryID,
				"creationTime": it.Entry.CreationTime.UTC().Format("2006-01-02T15:04:05.000Z"),
				"summary":      it.Entry.Summary,
				"rawEntry":     it.Entry.RawEntry,
			}

			if len(it.Entry.Tags) > 0 {
				keys := make([]string, 0, len(it.Entry.Tags))
				for k := range it.Entry.Tags {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				props["tags"] = strings.Join(keys, " ") // space-delimited string for field tokenization
			}

			if len(it.Entry.Metadata) > 0 {
				if data, err := json.Marshal(it.Entry.Metadata); err == nil {
					props["metadata"] = string(data)
				} else {
					u.log.Warn().Err(err).Msg("metadata marshal failed; skipping")
				}
			}

			vec := toFloat32Slice(it.SummaryVector)

			obj := &models.Object{
				Class:      u.className,
				ID:         strfmt.UUID(it.Entry.EntryID),
				Properties: props,
				Vector:     vec,
				Tenant:     it.Entry.UserID,
			}
			objs = append(objs, obj)
		}
		if len(objs) == 0 {
			continue
		}
		b = b.WithObjects(objs...)
		if _, err := b.Do(ctx); err != nil {
			if len(objs) > 0 {
				firstID := objs[0].ID.String()
				u.log.Error().Err(err).Int("batch_size", end-offset).Str("first_object_id", firstID).Msg("batch upload failed")
			} else {
				u.log.Error().Err(err).Int("batch_size", end-offset).Msg("batch upload failed")
			}
			return err
		}
		u.log.Info().Int("batch_size", end-offset).Msg("batch uploaded")
	}
	return nil
}

// Delete removes objects from Waviate by entry IDs. Best-effort; ignores 404s.
func (u *Uploader) Delete(ctx context.Context, userID string, entryIDs []string) error {
	if u == nil || u.client == nil || len(entryIDs) == 0 {
		return nil
	}
	for _, id := range entryIDs {
		// ignore errors for now; best-effort cleanup
		_ = u.client.Data().Deleter().WithClassName(u.className).WithID(id).WithTenant(userID).Do(ctx)
	}
	return nil
}

func toFloat32Slice(in []float32) []float32 { // ensure nil safe
	if len(in) == 0 {
		return nil
	}
	out := make([]float32, len(in))
	copy(out, in)
	return out
}

// UpsertContexts uploads context snapshots
func (u *Uploader) UpsertContexts(ctx context.Context, snaps []ContextSnapshot, embedder Embedder) error {
	if len(snaps) == 0 {
		return nil
	}
	const batchSize = 1
	for offset := 0; offset < len(snaps); offset += batchSize {
		end := offset + batchSize
		if end > len(snaps) {
			end = len(snaps)
		}

		// ensure tenants
		users := make(map[string]struct{})
		for _, s := range snaps[offset:end] {
			users[s.UserID] = struct{}{}
		}
		for uid := range users {
			u.ensureTenant(ctx, "MemoryContext", uid)
			// also ensure tenant exists for MemoryEntry so search works before first entry
			u.ensureTenant(ctx, u.className, uid)
		}

		b := u.client.Batch().ObjectsBatcher()
		var objs []*models.Object
		for _, snap := range snaps[offset:end] {
			vec := []float32{}
			if embedder != nil {
				v, err := embedder.Embed(ctx, snap.Text)
				if err != nil {
					u.log.Warn().Err(err).Msg("embedding failed; skipping context")
					continue
				}
				vec = toFloat32Slice(v)
			}

			props := map[string]interface{}{
				"contextId":    snap.ContextID,
				"memoryId":     snap.MemoryID,
				"creationTime": snap.CreationTime.UTC().Format("2006-01-02T15:04:05.000Z"),
				"context":      snap.Text,
			}

			obj := &models.Object{
				Class:      "MemoryContext",
				ID:         strfmt.UUID(snap.ContextID),
				Properties: props,
				Vector:     vec,
				Tenant:     snap.UserID,
			}
			objs = append(objs, obj)
		}
		if len(objs) == 0 {
			continue
		}
		b = b.WithObjects(objs...)
		if _, err := b.Do(ctx); err != nil {
			if len(objs) > 0 {
				firstID := objs[0].ID.String()
				u.log.Error().Err(err).Int("context_batch", len(objs)).Str("first_context_id", firstID).Msg("context batch upload failed")
			}
			return err
		}
		u.log.Info().Int("context_batch", len(objs)).Msg("context batch uploaded")
	}
	return nil
}
