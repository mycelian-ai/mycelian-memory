package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/synapse/synapse-mcp-server/client"
	"github.com/synapse/synapse-mcp-server/client/schema"
	"github.com/synapse/synapse-mcp-server/internal/config"
)

var serviceURL string
var debug bool

const maxClientLimit = 50
const defaultTopK = 10

func dbg(v interface{}) {
	if !debug {
		return
	}
	log.Debug().Interface("data", v).Msg("debug output")
}

func main() {
	cmd := NewRootCmd()
	if err := cmd.Execute(); err != nil {
		log.Error().Err(err).Msg("command failed")
		os.Exit(1)
	}
}

// NewRootCmd constructs the root CLI command; exposed for unit testing.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "synapse",
		Short: "Synapse CLI for managing users and memories",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Initialize logger
			config.InitLogger()

			// Set log level based on debug flag
			if debug {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
				os.Setenv("SYNAPSE_DEBUG", "true")
				log.Debug().Msg("debug logging enabled")
			} else {
				zerolog.SetGlobalLevel(zerolog.InfoLevel)
			}
		},
	}

	defaultURL := getEnv("MEMORY_SERVICE_URL", "http://localhost:8080")
	rootCmd.PersistentFlags().StringVar(&serviceURL, "service-url", defaultURL, "Base URL of Synapse memory service")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable verbose debug output")

	// Sub-commands
	rootCmd.AddCommand(newCreateUserCmd())
	rootCmd.AddCommand(newCreateMemoryCmd())
	rootCmd.AddCommand(newCreateVaultCmd())
	rootCmd.AddCommand(newListVaultsCmd())
	rootCmd.AddCommand(newGetVaultCmd())
	rootCmd.AddCommand(newListMemoriesCmd())
	rootCmd.AddCommand(newDeleteVaultCmd())
	rootCmd.AddCommand(newGetUserCmd())
	rootCmd.AddCommand(newCreateEntryCmd())
	rootCmd.AddCommand(newListEntriesCmd())
	rootCmd.AddCommand(newGetPromptsCmd())
	rootCmd.AddCommand(newPutContextCmd())
	rootCmd.AddCommand(newGetContextCmd())
	rootCmd.AddCommand(newSearchCmd())
	rootCmd.AddCommand(newGetToolsSchemaCmd())
	rootCmd.AddCommand(newAwaitConsistencyCmd())
	rootCmd.AddCommand(newListAssetsCmd())
	rootCmd.AddCommand(newGetAssetCmd())

	return rootCmd
}

func newCreateUserCmd() *cobra.Command {
	var userID, email, displayName, timeZone string

	cmd := &cobra.Command{
		Use:   "create-user",
		Short: "Create a new user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if email == "" {
				return fmt.Errorf("--email is required")
			}
			if userID == "" {
				return fmt.Errorf("--user-id is required")
			}

			log.Debug().
				Str("email", email).
				Str("display_name", displayName).
				Str("time_zone", timeZone).
				Str("service_url", serviceURL).
				Msg("creating user")

			c := client.New(serviceURL, client.WithoutExecutor())
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()

			start := time.Now()
			user, err := c.CreateUser(ctx, client.CreateUserRequest{
				UserID:      userID,
				Email:       email,
				DisplayName: displayName,
				TimeZone:    timeZone,
			})
			elapsed := time.Since(start)

			if err != nil {
				log.Error().
					Err(err).
					Str("email", email).
					Dur("elapsed", elapsed).
					Msg("create user failed")
				return err
			}

			log.Debug().
				Str("user_id", user.ID).
				Str("email", user.Email).
				Str("display_name", user.DisplayName).
				Str("time_zone", user.TimeZone).
				Dur("elapsed", elapsed).
				Msg("create user completed")

			dbg(user)
			fmt.Printf("User created: %s (%s)\n", user.ID, user.Email)
			return nil
		},
	}

	cmd.Flags().StringVar(&userID, "user-id", "", "User ID (required)")
	cmd.Flags().StringVar(&email, "email", "", "User email (required)")
	cmd.Flags().StringVar(&displayName, "display-name", "", "Display name (optional)")
	cmd.Flags().StringVar(&timeZone, "time-zone", "", "Time zone (optional)")
	_ = cmd.MarkFlagRequired("email")
	_ = cmd.MarkFlagRequired("user-id")

	return cmd
}

func newCreateMemoryCmd() *cobra.Command {
	var userID, vaultID, title, memoryType, description string

	cmd := &cobra.Command{
		Use:   "create-memory",
		Short: "Create a new memory for a user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userID == "" || vaultID == "" || title == "" || memoryType == "" {
				return fmt.Errorf("--user-id, --vault-id, --title, and --memory-type are required")
			}

			log.Debug().
				Str("user_id", userID).
				Str("vault_id", vaultID).
				Str("title", title).
				Str("memory_type", memoryType).
				Str("description", description).
				Str("service_url", serviceURL).
				Msg("creating memory")

			c := client.New(serviceURL, client.WithoutExecutor())
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()

			start := time.Now()
			mem, err := c.CreateMemoryInVault(ctx, userID, vaultID, client.CreateMemoryRequest{
				Title:       title,
				MemoryType:  memoryType,
				Description: description,
			})
			elapsed := time.Since(start)

			if err != nil {
				log.Error().
					Err(err).
					Str("user_id", userID).
					Str("vault_id", vaultID).
					Str("title", title).
					Str("memory_type", memoryType).
					Dur("elapsed", elapsed).
					Msg("create memory failed")
				return err
			}

			log.Debug().
				Str("user_id", userID).
				Str("vault_id", vaultID).
				Str("memory_id", mem.ID).
				Str("title", mem.Title).
				Str("memory_type", mem.MemoryType).
				Dur("elapsed", elapsed).
				Msg("create memory completed")

			dbg(mem)
			fmt.Printf("Memory created: %s - %s\n", mem.ID, mem.Title)
			return nil
		},
	}

	cmd.Flags().StringVar(&userID, "user-id", "", "User ID (required)")
	cmd.Flags().StringVar(&vaultID, "vault-id", "", "Vault ID (required)")
	cmd.Flags().StringVar(&title, "title", "", "Memory title (required)")
	cmd.Flags().StringVar(&memoryType, "memory-type", "", "Memory type (required)")
	cmd.Flags().StringVar(&description, "description", "", "Description (optional)")

	_ = cmd.MarkFlagRequired("user-id")
	_ = cmd.MarkFlagRequired("vault-id")
	_ = cmd.MarkFlagRequired("title")
	_ = cmd.MarkFlagRequired("memory-type")

	return cmd
}

func newCreateEntryCmd() *cobra.Command {
	var userID, vaultID, memoryID, rawEntry, summary string

	cmd := &cobra.Command{
		Use:   "create-entry",
		Short: "Create a new entry for a memory",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userID == "" || vaultID == "" || memoryID == "" || rawEntry == "" {
				return fmt.Errorf("--user-id, --vault-id, --memory-id, and --raw-entry are required")
			}

			log.Debug().
				Str("user_id", userID).
				Str("vault_id", vaultID).
				Str("memory_id", memoryID).
				Int("raw_entry_len", len(rawEntry)).
				Str("summary", summary).
				Str("service_url", serviceURL).
				Msg("creating entry")

			c := client.New(serviceURL, client.WithoutExecutor())
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()
			defer c.Close() // Ensure queues are drained before context is cancelled

			start := time.Now()
			ack, err := c.AddEntryInVault(ctx, userID, vaultID, memoryID, client.AddEntryRequest{
				RawEntry: rawEntry,
				Summary:  summary,
			})
			elapsed := time.Since(start)

			if err != nil {
				log.Error().
					Err(err).
					Str("user_id", userID).
					Str("vault_id", vaultID).
					Str("memory_id", memoryID).
					Dur("elapsed", elapsed).
					Msg("add entry failed")
				return err
			}

			log.Debug().
				Str("user_id", userID).
				Str("vault_id", vaultID).
				Str("memory_id", memoryID).
				Dur("elapsed", elapsed).
				Str("status", ack.Status).
				Msg("add entry completed")

			dbg(ack)
			fmt.Println("Entry enqueued")

			return nil
		},
	}

	cmd.Flags().StringVar(&userID, "user-id", "", "User ID (required)")
	cmd.Flags().StringVar(&vaultID, "vault-id", "", "Vault ID (required)")
	cmd.Flags().StringVar(&memoryID, "memory-id", "", "Memory ID (required)")
	cmd.Flags().StringVar(&rawEntry, "raw-entry", "", "Raw entry text (required)")
	cmd.Flags().StringVar(&summary, "summary", "", "Summary (required)")

	_ = cmd.MarkFlagRequired("user-id")
	_ = cmd.MarkFlagRequired("vault-id")
	_ = cmd.MarkFlagRequired("memory-id")
	_ = cmd.MarkFlagRequired("raw-entry")
	_ = cmd.MarkFlagRequired("summary")

	return cmd
}

func newListEntriesCmd() *cobra.Command {
	var userID, vaultID, memoryID string
	var limit int

	cmd := &cobra.Command{
		Use:   "list-entries",
		Short: "List entries for a memory",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userID == "" || vaultID == "" || memoryID == "" {
				return fmt.Errorf("--user-id, --vault-id, and --memory-id are required")
			}

			log.Debug().
				Str("user_id", userID).
				Str("vault_id", vaultID).
				Str("memory_id", memoryID).
				Int("limit", limit).
				Str("service_url", serviceURL).
				Msg("listing entries")

			c := client.New(serviceURL, client.WithoutExecutor())
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()

			limit = applyUpperBoundToLimit(limit)

			start := time.Now()
			resp, err := c.ListEntriesInVault(ctx, userID, vaultID, memoryID, map[string]string{"limit": strconv.Itoa(limit)})
			elapsed := time.Since(start)

			if err != nil {
				log.Error().
					Err(err).
					Str("user_id", userID).
					Str("vault_id", vaultID).
					Str("memory_id", memoryID).
					Dur("elapsed", elapsed).
					Msg("list entries failed")
				return err
			}

			log.Debug().
				Str("user_id", userID).
				Str("vault_id", vaultID).
				Str("memory_id", memoryID).
				Dur("elapsed", elapsed).
				Int("count", resp.Count).
				Int("entries_returned", len(resp.Entries)).
				Msg("list entries completed")

			dbg(resp)

			// Output full JSON so automated callers (benchmark harness, CI scripts)
			// can parse the response without needing the Go client types.
			b, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Println(string(b))
			return nil
		},
	}

	cmd.Flags().StringVar(&userID, "user-id", "", "User ID (required)")
	cmd.Flags().StringVar(&vaultID, "vault-id", "", "Vault ID (required)")
	cmd.Flags().StringVar(&memoryID, "memory-id", "", "Memory ID (required)")
	cmd.Flags().IntVar(&limit, "limit", 25, "Number of entries to return (max 50)")

	_ = cmd.MarkFlagRequired("user-id")
	_ = cmd.MarkFlagRequired("vault-id")
	_ = cmd.MarkFlagRequired("memory-id")

	return cmd
}

func newGetPromptsCmd() *cobra.Command {
	var memoryType string

	cmd := &cobra.Command{
		Use:   "get-prompts",
		Short: "Print default prompt templates for a memory type in JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			if memoryType == "" {
				return fmt.Errorf("--memory-type is required")
			}

			c := client.New(serviceURL, client.WithoutExecutor())
			ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
			defer cancel()

			resp, err := c.GetDefaultPrompts(ctx, memoryType)
			if err != nil {
				return err
			}
			dbg(resp)
			b, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Println(string(b))
			return nil
		},
	}

	cmd.Flags().StringVar(&memoryType, "memory-type", "", "Memory type (chat, code, …)")
	_ = cmd.MarkFlagRequired("memory-type")

	return cmd
}

func newPutContextCmd() *cobra.Command {
	var userID, vaultID, memoryID, content string

	cmd := &cobra.Command{
		Use:   "put-context",
		Short: "Update context document for a memory (enqueue write)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userID == "" || vaultID == "" || memoryID == "" || content == "" {
				return fmt.Errorf("--user-id, --vault-id, --memory-id, and --content are required")
			}

			log.Debug().
				Str("user_id", userID).
				Str("vault_id", vaultID).
				Str("memory_id", memoryID).
				Int("content_len", len(content)).
				Str("service_url", serviceURL).
				Msg("putting context")

			c := client.New(serviceURL)
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()
			defer c.Close() // Ensure queues are drained before process exits

			start := time.Now()
			ack, err := c.PutMemoryContext(ctx, userID, vaultID, memoryID, content)
			elapsed := time.Since(start)

			if err != nil {
				log.Error().
					Err(err).
					Str("user_id", userID).
					Str("vault_id", vaultID).
					Str("memory_id", memoryID).
					Dur("elapsed", elapsed).
					Msg("put context failed")
				return err
			}

			log.Debug().
				Str("user_id", userID).
				Str("vault_id", vaultID).
				Str("memory_id", memoryID).
				Dur("elapsed", elapsed).
				Str("status", ack.Status).
				Msg("put context completed")

			dbg(ack)
			fmt.Println("Context enqueued")
			return nil
		},
	}

	cmd.Flags().StringVar(&userID, "user-id", "", "User ID (required)")
	cmd.Flags().StringVar(&vaultID, "vault-id", "", "Vault ID (required)")
	cmd.Flags().StringVar(&memoryID, "memory-id", "", "Memory ID (required)")
	cmd.Flags().StringVar(&content, "content", "", "Context content (required)")

	_ = cmd.MarkFlagRequired("user-id")
	_ = cmd.MarkFlagRequired("vault-id")
	_ = cmd.MarkFlagRequired("memory-id")
	_ = cmd.MarkFlagRequired("content")

	return cmd
}

func newGetContextCmd() *cobra.Command {
	var userID, vaultID, memoryID string

	cmd := &cobra.Command{
		Use:   "get-context",
		Short: "Fetch the latest context document for a memory",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userID == "" || vaultID == "" || memoryID == "" {
				return fmt.Errorf("--user-id, --vault-id, and --memory-id are required")
			}

			log.Debug().
				Str("user_id", userID).
				Str("vault_id", vaultID).
				Str("memory_id", memoryID).
				Str("service_url", serviceURL).
				Msg("getting context")

			c := client.New(serviceURL, client.WithoutExecutor())
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()

			start := time.Now()
			resp, err := c.GetLatestMemoryContext(ctx, userID, vaultID, memoryID)
			elapsed := time.Since(start)

			if err != nil {
				if err == client.ErrNotFound {
					log.Debug().
						Str("user_id", userID).
						Str("vault_id", vaultID).
						Str("memory_id", memoryID).
						Dur("elapsed", elapsed).
						Msg("get context: not found")
					fmt.Println("No context document found")
					return nil
				}
				log.Error().
					Err(err).
					Str("user_id", userID).
					Str("vault_id", vaultID).
					Str("memory_id", memoryID).
					Dur("elapsed", elapsed).
					Msg("get context failed")
				return err
			}

			if resp.Context != nil {
				switch v := resp.Context.(type) {
				case string:
					log.Debug().
						Str("user_id", userID).
						Str("vault_id", vaultID).
						Str("memory_id", memoryID).
						Dur("elapsed", elapsed).
						Int("content_len", len(v)).
						Str("type", "string").
						Msg("get context completed")
					fmt.Println(v)
				case map[string]interface{}:
					// If the backend wraps the snapshot under {"activeContext": "..."},
					// print the raw string value so downstream consumers receive exactly
					// the stored context without additional JSON decoration.
					if ac, ok := v["activeContext"].(string); ok {
						log.Debug().
							Str("user_id", userID).
							Str("vault_id", vaultID).
							Str("memory_id", memoryID).
							Dur("elapsed", elapsed).
							Int("content_len", len(ac)).
							Str("type", "activeContext").
							Msg("get context completed")
						fmt.Println(ac)
					} else {
						// Fallback: pretty-print the JSON map (same as default branch)
						b, _ := json.MarshalIndent(v, "", "  ")
						log.Debug().
							Str("user_id", userID).
							Str("vault_id", vaultID).
							Str("memory_id", memoryID).
							Dur("elapsed", elapsed).
							Int("content_len", len(b)).
							Str("type", "json").
							Msg("get context completed")
						fmt.Println(string(b))
					}
				default:
					b, _ := json.MarshalIndent(v, "", "  ")
					log.Debug().
						Str("user_id", userID).
						Str("vault_id", vaultID).
						Str("memory_id", memoryID).
						Dur("elapsed", elapsed).
						Int("content_len", len(b)).
						Str("type", "json").
						Msg("get context completed")
					fmt.Println(string(b))
				}
			} else {
				log.Debug().
					Str("user_id", userID).
					Str("vault_id", vaultID).
					Str("memory_id", memoryID).
					Dur("elapsed", elapsed).
					Str("type", "empty").
					Msg("get context completed")
				fmt.Println("(empty context)")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&userID, "user-id", "", "User ID (required)")
	cmd.Flags().StringVar(&vaultID, "vault-id", "", "Vault ID (required)")
	cmd.Flags().StringVar(&memoryID, "memory-id", "", "Memory ID (required)")
	_ = cmd.MarkFlagRequired("user-id")
	_ = cmd.MarkFlagRequired("vault-id")
	_ = cmd.MarkFlagRequired("memory-id")

	return cmd
}

func newSearchCmd() *cobra.Command {
	var userID, memoryID, query string
	var topK int

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search within a memory (hybrid semantic/keyword)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userID == "" || memoryID == "" || query == "" {
				return fmt.Errorf("--user-id, --memory-id and --query are required")
			}

			if topK <= 0 || topK > 100 {
				return fmt.Errorf("--top-k must be between 1 and 100")
			}

			log.Debug().
				Str("user_id", userID).
				Str("memory_id", memoryID).
				Str("query", query).
				Int("top_k", topK).
				Str("service_url", serviceURL).
				Msg("searching memories")

			c := client.New(serviceURL, client.WithoutExecutor())
			ctx, cancel := context.WithTimeout(cmd.Context(), 20*time.Second)
			defer cancel()

			start := time.Now()
			resp, err := c.Search(ctx, client.SearchRequest{
				UserID:   userID,
				MemoryID: memoryID,
				Query:    query,
				TopK:     topK,
			})
			elapsed := time.Since(start)

			if err != nil {
				log.Error().Err(err).
					Str("user_id", userID).
					Str("memory_id", memoryID).
					Dur("elapsed", elapsed).
					Msg("search failed")
				return err
			}

			log.Debug().
				Str("user_id", userID).
				Str("memory_id", memoryID).
				Dur("elapsed", elapsed).
				Int("count", resp.Count).
				Msg("search completed")

			dbg(resp)
			b, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Println(string(b))
			return nil
		},
	}

	cmd.Flags().StringVar(&userID, "user-id", "", "User ID (required)")
	cmd.Flags().StringVar(&memoryID, "memory-id", "", "Memory ID (required)")
	cmd.Flags().StringVar(&query, "query", "", "Search query (required)")
	cmd.Flags().IntVar(&topK, "top-k", defaultTopK, "Number of results to return (1-100)")
	_ = cmd.MarkFlagRequired("user-id")
	_ = cmd.MarkFlagRequired("memory-id")
	_ = cmd.MarkFlagRequired("query")

	return cmd
}

func newGetUserCmd() *cobra.Command {
	var userID string

	cmd := &cobra.Command{
		Use:   "get-user",
		Short: "Get detailed information about a user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userID == "" {
				return fmt.Errorf("--user-id is required")
			}

			log.Debug().
				Str("user_id", userID).
				Str("service_url", serviceURL).
				Msg("getting user")

			c := client.New(serviceURL, client.WithoutExecutor())
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()

			start := time.Now()
			user, err := c.GetUser(ctx, userID)
			elapsed := time.Since(start)
			if err != nil {
				log.Error().
					Err(err).
					Str("user_id", userID).
					Dur("elapsed", elapsed).
					Msg("get user failed")
				return err
			}

			log.Debug().
				Str("user_id", userID).
				Dur("elapsed", elapsed).
				Msg("get user completed")

			dbg(user)
			b, _ := json.MarshalIndent(user, "", "  ")
			fmt.Println(string(b))
			return nil
		},
	}

	cmd.Flags().StringVar(&userID, "user-id", "", "User ID (required)")
	_ = cmd.MarkFlagRequired("user-id")

	return cmd
}

func newGetToolsSchemaCmd() *cobra.Command {
	var pretty bool

	cmd := &cobra.Command{
		Use:   "get-tools-schema",
		Short: "Print the canonical JSON-Schema array describing all MCP tools",
		RunE: func(cmd *cobra.Command, _ []string) error {
			raw := schema.ToolsSchema()
			if pretty {
				var buf bytes.Buffer
				if err := json.Indent(&buf, raw, "", "  "); err == nil {
					raw = buf.Bytes()
				}
			}
			_, err := cmd.OutOrStdout().Write(raw)
			return err
		},
	}

	cmd.Flags().BoolVar(&pretty, "pretty", false, "Pretty-print JSON output")
	return cmd
}

func newAwaitConsistencyCmd() *cobra.Command {
	var userID, memoryID string

	cmd := &cobra.Command{
		Use:   "await-consistency",
		Short: "Block until previous writes for the memory are durably visible",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userID == "" || memoryID == "" {
				return fmt.Errorf("--user-id and --memory-id are required")
			}

			log.Debug().
				Str("user_id", userID).
				Str("memory_id", memoryID).
				Str("service_url", serviceURL).
				Msg("awaiting consistency")

			c := client.New(serviceURL)
			ctx, cancel := context.WithTimeout(cmd.Context(), 20*time.Second)
			defer cancel()

			start := time.Now()
			if err := c.AwaitConsistency(ctx, memoryID); err != nil {
				log.Error().Err(err).
					Str("user_id", userID).
					Str("memory_id", memoryID).
					Msg("await-consistency failed")
				return err
			}

			log.Debug().
				Str("user_id", userID).
				Str("memory_id", memoryID).
				Dur("elapsed", time.Since(start)).
				Msg("await-consistency completed")
			fmt.Println("OK")
			return nil
		},
	}

	cmd.Flags().StringVar(&userID, "user-id", "", "User ID (required)")
	cmd.Flags().StringVar(&memoryID, "memory-id", "", "Memory ID (required)")
	_ = cmd.MarkFlagRequired("user-id")
	_ = cmd.MarkFlagRequired("memory-id")

	return cmd
}

// ----------------------------- Assets ------------------------------

func assetIDMap() map[string]string {
	return map[string]string{
		"ctx_rules":           "prompts/system/context_summary_rules.md",
		"ctx_prompt_chat":     "prompts/default/chat/context_prompt.md",
		"entry_prompt_chat":   "prompts/default/chat/entry_capture_prompt.md",
		"summary_prompt_chat": "prompts/default/chat/summary_prompt.md",
	}
}

func newListAssetsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-assets",
		Short: "List logical IDs of static assets available via get-asset",
		RunE: func(cmd *cobra.Command, args []string) error {
			ids := make([]string, 0, len(assetIDMap()))
			for id := range assetIDMap() {
				ids = append(ids, id)
			}
			sort.Strings(ids)
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(map[string]any{"assets": ids})
		},
	}
	return cmd
}

func newGetAssetCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "get-asset",
		Short: "Print the raw text content of a static prompt or rule asset",
		RunE: func(cmd *cobra.Command, args []string) error {
			mp := assetIDMap()
			path, ok := mp[id]
			if !ok {
				return fmt.Errorf("unknown asset id: %s", id)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			cmd.Println(string(data))
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Logical asset ID (required)")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

// ------------------ Vault Commands -------------------

func newCreateVaultCmd() *cobra.Command {
	var userID, title, description string

	cmd := &cobra.Command{
		Use:   "create-vault",
		Short: "Create a new vault for a user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userID == "" || title == "" {
				return fmt.Errorf("--user-id and --title are required")
			}

			c := client.New(serviceURL, client.WithoutExecutor())
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()

			v, err := c.CreateVault(ctx, userID, client.CreateVaultRequest{Title: title, Description: description})
			if err != nil {
				return err
			}
			fmt.Printf("Vault created: %s (%s)\n", v.VaultID, v.Title)
			return nil
		},
	}

	cmd.Flags().StringVar(&userID, "user-id", "", "User ID (required)")
	cmd.Flags().StringVar(&title, "title", "", "Vault title (required)")
	cmd.Flags().StringVar(&description, "description", "", "Description (optional)")
	_ = cmd.MarkFlagRequired("user-id")
	_ = cmd.MarkFlagRequired("title")

	return cmd
}

func newListVaultsCmd() *cobra.Command {
	var userID string
	cmd := &cobra.Command{
		Use:   "list-vaults",
		Short: "List all vaults for a user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userID == "" {
				return fmt.Errorf("--user-id required")
			}

			c := client.New(serviceURL, client.WithoutExecutor())
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()

			vaults, err := c.ListVaults(ctx, userID)
			if err != nil {
				return err
			}
			for _, v := range vaults {
				fmt.Printf("%s\t%s\n", v.VaultID, v.Title)
			}
			fmt.Printf("Total: %d\n", len(vaults))
			return nil
		},
	}
	cmd.Flags().StringVar(&userID, "user-id", "", "User ID (required)")
	_ = cmd.MarkFlagRequired("user-id")
	return cmd
}

func newGetVaultCmd() *cobra.Command {
	var userID, title string
	cmd := &cobra.Command{
		Use:   "get-vault",
		Short: "Retrieve a vault by title",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userID == "" || title == "" {
				return fmt.Errorf("--user-id and --title required")
			}

			c := client.New(serviceURL, client.WithoutExecutor())
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()

			v, err := c.GetVaultByTitle(ctx, userID, title)
			if err != nil {
				return err
			}
			b, _ := json.MarshalIndent(v, "", "  ")
			fmt.Println(string(b))
			return nil
		},
	}
	cmd.Flags().StringVar(&userID, "user-id", "", "User ID (required)")
	cmd.Flags().StringVar(&title, "title", "", "Vault title (required)")
	_ = cmd.MarkFlagRequired("user-id")
	_ = cmd.MarkFlagRequired("title")
	return cmd
}

func newDeleteVaultCmd() *cobra.Command {
	var userID, vaultID string

	cmd := &cobra.Command{
		Use:   "delete-vault",
		Short: "Delete an empty vault (must contain no memories)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userID == "" || vaultID == "" {
				return fmt.Errorf("--user-id and --vault-id are required")
			}

			c := client.New(serviceURL, client.WithoutExecutor())
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()

			if err := c.DeleteVault(ctx, userID, vaultID); err != nil {
				return err
			}
			fmt.Println("Vault deleted")
			return nil
		},
	}

	cmd.Flags().StringVar(&userID, "user-id", "", "User ID (required)")
	cmd.Flags().StringVar(&vaultID, "vault-id", "", "Vault ID (required)")
	_ = cmd.MarkFlagRequired("user-id")
	_ = cmd.MarkFlagRequired("vault-id")
	return cmd
}

// ------------------ Memory Listing Command -------------------

func newListMemoriesCmd() *cobra.Command {
	var userID, vaultID, vaultTitle string

	cmd := &cobra.Command{
		Use:   "list-memories",
		Short: "List memories within a vault (by ID or title)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userID == "" {
				return fmt.Errorf("--user-id required")
			}

			if vaultID == "" && vaultTitle == "" {
				return fmt.Errorf("either --vault-id or --vault-title must be provided")
			}
			if vaultID != "" && vaultTitle != "" {
				return fmt.Errorf("provide only one of --vault-id or --vault-title, not both")
			}

			c := client.New(serviceURL, client.WithoutExecutor())
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()

			var mems []client.Memory
			var err error
			if vaultID != "" {
				mems, err = c.ListMemories(ctx, userID, vaultID)
			} else {
				mems, err = c.ListMemoriesByVaultTitle(ctx, userID, vaultTitle)
			}
			if err != nil {
				return err
			}

			for _, m := range mems {
				fmt.Printf("%s\t%s\n", m.ID, m.Title)
			}
			fmt.Printf("Total: %d\n", len(mems))
			return nil
		},
	}

	cmd.Flags().StringVar(&userID, "user-id", "", "User ID (required)")
	cmd.Flags().StringVar(&vaultID, "vault-id", "", "Vault ID (mutually exclusive with --vault-title)")
	cmd.Flags().StringVar(&vaultTitle, "vault-title", "", "Vault title (mutually exclusive with --vault-id)")
	_ = cmd.MarkFlagRequired("user-id")
	return cmd
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func applyUpperBoundToLimit(l int) int {
	if l <= 0 {
		return 25
	}
	if l > maxClientLimit {
		if debug {
			log.Warn().Msgf("limit capped at %d (requested %d)", maxClientLimit, l)
		}
		return maxClientLimit
	}
	return l
}
