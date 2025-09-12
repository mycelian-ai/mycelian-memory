package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mycelian/mycelian-memory/client"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var serviceURL string
var debug bool

const maxClientLimit = 50
const defaultKE = 5
const defaultKC = 2

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
		Use:   "mycelianCli",
		Short: "MycelianCli for managing users and memories",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Initialize logger
			zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
			log.Logger = log.Output(zerolog.ConsoleWriter{
				Out:        os.Stderr,
				TimeFormat: "2006-01-02 15:04:05",
				NoColor:    true,
			})

			// Set log level based on debug flag
			if debug {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
				_ = os.Setenv("MYCELIAN_DEBUG", "true")
				log.Debug().Msg("debug logging enabled")
			} else {
				zerolog.SetGlobalLevel(zerolog.InfoLevel)
			}
		},
	}

	defaultURL := getEnv("MEMORY_SERVICE_URL", "http://localhost:11545")
	rootCmd.PersistentFlags().StringVar(&serviceURL, "service-url", defaultURL, "Base URL of Mycelian memory service")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable verbose debug output")

	// Sub-commands
	rootCmd.AddCommand(newCreateMemoryCmd())
	rootCmd.AddCommand(newCreateVaultCmd())
	rootCmd.AddCommand(newListVaultsCmd())
	rootCmd.AddCommand(newGetVaultCmd())
	rootCmd.AddCommand(newListMemoriesCmd())
	rootCmd.AddCommand(newDeleteVaultCmd())
	rootCmd.AddCommand(newCreateEntryCmd())
	rootCmd.AddCommand(newListEntriesCmd())
	rootCmd.AddCommand(newGetPromptsCmd())
	rootCmd.AddCommand(newPutContextCmd())
	rootCmd.AddCommand(newGetContextCmd())
	rootCmd.AddCommand(newSearchCmd())
	rootCmd.AddCommand(newGetToolsSchemaCmd())
	rootCmd.AddCommand(newAwaitConsistencyCmd())

	return rootCmd
}

func newCreateMemoryCmd() *cobra.Command {
	var vaultID, title, memoryType, description string

	cmd := &cobra.Command{
		Use:   "create-memory",
		Short: "Create a new memory",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Client-side validation removed; rely on server-side validation

			log.Debug().
				Str("vault_id", vaultID).
				Str("title", title).
				Str("memory_type", memoryType).
				Str("description", description).
				Str("service_url", serviceURL).
				Msg("creating memory")

			c, err := client.NewWithDevMode(serviceURL)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()

			start := time.Now()
			mem, err := c.CreateMemory(ctx, vaultID, client.CreateMemoryRequest{
				Title:       title,
				MemoryType:  memoryType,
				Description: description,
			})
			elapsed := time.Since(start)

			if err != nil {
				log.Error().
					Err(err).
					Str("vault_id", vaultID).
					Str("title", title).
					Str("memory_type", memoryType).
					Dur("elapsed", elapsed).
					Msg("create memory failed")
				return err
			}

			log.Debug().
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

	cmd.Flags().StringVar(&vaultID, "vault-id", "", "Vault ID (required)")
	cmd.Flags().StringVar(&title, "title", "", "Memory title (required)")
	cmd.Flags().StringVar(&memoryType, "memory-type", "", "Memory type (required)")
	cmd.Flags().StringVar(&description, "description", "", "Description (optional)")

	_ = cmd.MarkFlagRequired("vault-id")
	_ = cmd.MarkFlagRequired("title")
	_ = cmd.MarkFlagRequired("memory-type")

	return cmd
}

func newCreateEntryCmd() *cobra.Command {
	var vaultID, memoryID, rawEntry, summary string

	cmd := &cobra.Command{
		Use:   "create-entry",
		Short: "Create a new entry for a memory",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Client-side validation removed; rely on server-side validation

			log.Debug().
				Str("vault_id", vaultID).
				Str("memory_id", memoryID).
				Int("raw_entry_len", len(rawEntry)).
				Str("summary", summary).
				Str("service_url", serviceURL).
				Msg("creating entry")

			c, err := client.NewWithDevMode(serviceURL)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()
			defer func() { _ = c.Close() }() // Ensure queues are drained before context is cancelled

			start := time.Now()
			ack, err := c.AddEntry(ctx, vaultID, memoryID, client.AddEntryRequest{
				RawEntry: rawEntry,
				Summary:  summary,
			})
			elapsed := time.Since(start)

			if err != nil {
				log.Error().
					Err(err).
					Str("vault_id", vaultID).
					Str("memory_id", memoryID).
					Dur("elapsed", elapsed).
					Msg("add entry failed")
				return err
			}

			log.Debug().
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

	cmd.Flags().StringVar(&vaultID, "vault-id", "", "Vault ID (required)")
	cmd.Flags().StringVar(&memoryID, "memory-id", "", "Memory ID (required)")
	cmd.Flags().StringVar(&rawEntry, "raw-entry", "", "Raw entry text (required)")
	cmd.Flags().StringVar(&summary, "summary", "", "Summary (required)")

	_ = cmd.MarkFlagRequired("vault-id")
	_ = cmd.MarkFlagRequired("memory-id")
	_ = cmd.MarkFlagRequired("raw-entry")
	_ = cmd.MarkFlagRequired("summary")

	return cmd
}

func newListEntriesCmd() *cobra.Command {
	var vaultID, memoryID string
	var limit int

	cmd := &cobra.Command{
		Use:   "list-entries",
		Short: "List entries for a memory",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Client-side validation removed; rely on server-side validation

			log.Debug().
				Str("vault_id", vaultID).
				Str("memory_id", memoryID).
				Int("limit", limit).
				Str("service_url", serviceURL).
				Msg("listing entries")

			c, err := client.NewWithDevMode(serviceURL)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()

			limit = applyUpperBoundToLimit(limit)

			start := time.Now()
			resp, err := c.ListEntries(ctx, vaultID, memoryID, map[string]string{"limit": strconv.Itoa(limit)})
			elapsed := time.Since(start)

			if err != nil {
				log.Error().
					Err(err).
					Str("vault_id", vaultID).
					Str("memory_id", memoryID).
					Dur("elapsed", elapsed).
					Msg("list entries failed")
				return err
			}

			log.Debug().
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

	cmd.Flags().StringVar(&vaultID, "vault-id", "", "Vault ID (required)")
	cmd.Flags().StringVar(&memoryID, "memory-id", "", "Memory ID (required)")
	cmd.Flags().IntVar(&limit, "limit", 25, "Number of entries to return (max 50)")

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
			// Client-side validation removed; rely on server-side validation

			c, err := client.NewWithDevMode(serviceURL)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
			defer cancel()

			resp, err := c.LoadDefaultPrompts(ctx, memoryType)
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
	var vaultID, memoryID, content string

	cmd := &cobra.Command{
		Use:   "put-context",
		Short: "Update context document for a memory (enqueue write)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Client-side validation removed; rely on server-side validation

			log.Debug().
				Str("vault_id", vaultID).
				Str("memory_id", memoryID).
				Int("content_len", len(content)).
				Str("service_url", serviceURL).
				Msg("putting context")

			c, err := client.NewWithDevMode(serviceURL)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()
			defer func() { _ = c.Close() }() // Ensure queues are drained before process exits

			start := time.Now()
			ack, err := c.PutContext(ctx, vaultID, memoryID, content)
			elapsed := time.Since(start)

			if err != nil {
				log.Error().
					Err(err).
					Str("vault_id", vaultID).
					Str("memory_id", memoryID).
					Dur("elapsed", elapsed).
					Msg("put context failed")
				return err
			}

			log.Debug().
				Str("vault_id", vaultID).
				Str("memory_id", memoryID).
				Str("status", ack.Status).
				Dur("elapsed", elapsed).
				Msg("put context completed")

			dbg(ack)
			fmt.Println("Context enqueued")
			return nil
		},
	}

	cmd.Flags().StringVar(&vaultID, "vault-id", "", "Vault ID (required)")
	cmd.Flags().StringVar(&memoryID, "memory-id", "", "Memory ID (required)")
	cmd.Flags().StringVar(&content, "content", "", "Context content (required)")

	_ = cmd.MarkFlagRequired("vault-id")
	_ = cmd.MarkFlagRequired("memory-id")
	_ = cmd.MarkFlagRequired("content")

	return cmd
}

func newGetContextCmd() *cobra.Command {
	var vaultID, memoryID string

	cmd := &cobra.Command{
		Use:   "get-context",
		Short: "Fetch the latest context document for a memory",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Client-side validation removed; rely on server-side validation

			log.Debug().
				Str("vault_id", vaultID).
				Str("memory_id", memoryID).
				Str("service_url", serviceURL).
				Msg("getting context")

			c, err := client.NewWithDevMode(serviceURL)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()

			start := time.Now()
			text, err := c.GetLatestContext(ctx, vaultID, memoryID)
			elapsed := time.Since(start)

			if err != nil {
				if err == client.ErrNotFound {
					log.Debug().
						Str("vault_id", vaultID).
						Str("memory_id", memoryID).
						Dur("elapsed", elapsed).
						Msg("get context: not found")
					fmt.Println("No context document found")
					return nil
				}
				log.Error().
					Err(err).
					Str("vault_id", vaultID).
					Str("memory_id", memoryID).
					Dur("elapsed", elapsed).
					Msg("get context failed")
				return err
			}

			log.Debug().
				Str("vault_id", vaultID).
				Str("memory_id", memoryID).
				Dur("elapsed", elapsed).
				Int("content_len", len(text)).
				Msg("get context completed")
			fmt.Println(text)
			return nil
		},
	}

	cmd.Flags().StringVar(&vaultID, "vault-id", "", "Vault ID (required)")
	cmd.Flags().StringVar(&memoryID, "memory-id", "", "Memory ID (required)")

	_ = cmd.MarkFlagRequired("vault-id")
	_ = cmd.MarkFlagRequired("memory-id")

	return cmd
}

func newSearchCmd() *cobra.Command {
	var memoryID, query string
	var ke, kc int
	var includeRawEntries bool

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search within a memory (hybrid semantic/keyword)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Client-side validation removed; rely on server-side validation

			// Validate query is provided
			if query == "" {
				return fmt.Errorf("--query is required")
			}

			if ke < 0 || ke > 25 {
				return fmt.Errorf("--ke must be between 0 and 25")
			}
			if kc < 0 || kc > 10 {
				return fmt.Errorf("--kc must be between 0 and 10")
			}

			log.Debug().
				Str("memory_id", memoryID).
				Str("query", query).
				Int("ke", ke).
				Int("kc", kc).
				Bool("include_raw_entries", includeRawEntries).
				Str("service_url", serviceURL).
				Msg("searching memories")

			c, err := client.NewWithDevMode(serviceURL)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 20*time.Second)
			defer cancel()

			start := time.Now()
			// Convert int to pointer values for the request
			topKEPtr := ke
			if ke == 0 {
				topKEPtr = defaultKE
			}
			topKCPtr := kc
			if kc == 0 {
				topKCPtr = defaultKC
			}

			resp, err := c.Search(ctx, client.SearchRequest{
				MemoryID:          memoryID,
				Query:             query,
				TopKE:             &topKEPtr,
				TopKC:             &topKCPtr,
				IncludeRawEntries: includeRawEntries,
			})
			elapsed := time.Since(start)

			if err != nil {
				log.Error().Err(err).
					Str("memory_id", memoryID).
					Dur("elapsed", elapsed).
					Msg("search failed")
				return err
			}

			log.Debug().
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

	cmd.Flags().StringVar(&memoryID, "memory-id", "", "Memory ID (required)")
	cmd.Flags().StringVar(&query, "query", "", "Search query (required)")
	cmd.Flags().IntVar(&ke, "ke", 0, "Number of entries to return (0-25, default 5)")
	cmd.Flags().IntVar(&kc, "kc", 0, "Number of context shards to return (0-10, default 2)")
	cmd.Flags().BoolVar(&includeRawEntries, "include-raw-entries", false, "Include raw entry content in results (default: false)")

	_ = cmd.MarkFlagRequired("memory-id")
	_ = cmd.MarkFlagRequired("query")

	return cmd
}

// findMCPServerBinary locates the mycelian-mcp-server binary.
func findMCPServerBinary() (string, error) {
	// Get the directory where mycelianCli binary is running from
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}
	cliDir := filepath.Dir(executable)

	// Try common locations relative to mycelianCli binary
	candidates := []string{
		filepath.Join(cliDir, "mycelian-mcp-server"),
		filepath.Join(cliDir, "..", "mycelian-mcp-server"),
		filepath.Join(cliDir, "..", "bin", "mycelian-mcp-server"),
		"mycelian-mcp-server", // In PATH
	}

	for _, candidate := range candidates {
		if _, err := exec.LookPath(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("mycelian-mcp-server binary not found. Build it with: cd clients/go && go build -o bin/mycelian-mcp-server ./cmd/mycelian-mcp-server")
}

// loadToolsFromMCPServer calls the live MCP server to get current tools.
func loadToolsFromMCPServer() ([]byte, error) {
	mcpBinary, err := findMCPServerBinary()
	if err != nil {
		return nil, err
	}

	// Prepare MCP tools/list request
	mcpRequest := `{"jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": {}}`

	// Execute MCP server with stdio transport
	cmd := exec.Command("bash", "-c", fmt.Sprintf("echo '%s' | MCP_STDIO=true %s", mcpRequest, mcpBinary))
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("MCP server execution failed: %w", err)
	}

	// Parse response - filter out log lines, find JSON response
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, `{"jsonrpc"`) {
			var response struct {
				Result struct {
					Tools []json.RawMessage `json:"tools"`
				} `json:"result"`
			}
			if err := json.Unmarshal([]byte(line), &response); err == nil {
				return json.Marshal(response.Result.Tools)
			}
		}
	}

	return nil, fmt.Errorf("no valid JSON-RPC response found in MCP server output")
}

func newGetToolsSchemaCmd() *cobra.Command {
	var pretty bool

	cmd := &cobra.Command{
		Use:   "get-tools-schema",
		Short: "Print MCP tools schema from live mycelian-mcp-server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if debug {
				log.Debug().Msg("Loading tools schema from live MCP server")
			}

			raw, err := loadToolsFromMCPServer()
			if err != nil {
				return fmt.Errorf("failed to load tools schema from MCP server: %w", err)
			}

			if pretty {
				var buf bytes.Buffer
				if err := json.Indent(&buf, raw, "", "  "); err == nil {
					raw = buf.Bytes()
				}
			}
			_, err = cmd.OutOrStdout().Write(raw)
			return err
		},
	}

	cmd.Flags().BoolVar(&pretty, "pretty", false, "Pretty-print JSON output")
	return cmd
}

func newAwaitConsistencyCmd() *cobra.Command {
	var memoryID string

	cmd := &cobra.Command{
		Use:   "await-consistency",
		Short: "Block until previous writes for the memory are durably visible",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Client-side validation removed; rely on server-side validation

			log.Debug().
				Str("memory_id", memoryID).
				Str("service_url", serviceURL).
				Msg("awaiting consistency")

			c, err := client.NewWithDevMode(serviceURL)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 20*time.Second)
			defer cancel()

			start := time.Now()
			if err := c.AwaitConsistency(ctx, memoryID); err != nil {
				log.Error().Err(err).
					Str("memory_id", memoryID).
					Msg("await-consistency failed")
				return err
			}

			log.Debug().
				Str("memory_id", memoryID).
				Dur("elapsed", time.Since(start)).
				Msg("await-consistency completed")
			fmt.Println("OK")
			return nil
		},
	}

	cmd.Flags().StringVar(&memoryID, "memory-id", "", "Memory ID (required)")

	_ = cmd.MarkFlagRequired("memory-id")

	return cmd
}

// ------------------ Vault Commands -------------------

func newCreateVaultCmd() *cobra.Command {
	var title, description string

	cmd := &cobra.Command{
		Use:   "create-vault",
		Short: "Create a new vault for a user",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Client-side validation removed; rely on server-side validation

			c, err := client.NewWithDevMode(serviceURL)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()

			v, err := c.CreateVault(ctx, client.CreateVaultRequest{Title: title, Description: description})
			if err != nil {
				return err
			}
			fmt.Printf("Vault created: %s (%s)\n", v.VaultID, v.Title)
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Vault title (required)")
	cmd.Flags().StringVar(&description, "description", "", "Description (optional)")

	_ = cmd.MarkFlagRequired("title")

	return cmd
}

func newListVaultsCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "list-vaults",
		Short: "List all vaults for a user",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Client-side validation removed; rely on server-side validation

			c, err := client.NewWithDevMode(serviceURL)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()

			vaults, err := c.ListVaults(ctx)
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

	return cmd
}

func newGetVaultCmd() *cobra.Command {
	var title string
	cmd := &cobra.Command{
		Use:   "get-vault",
		Short: "Retrieve a vault by title",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Client-side validation removed; rely on server-side validation

			c, err := client.NewWithDevMode(serviceURL)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()

			v, err := c.GetVaultByTitle(ctx, title)
			if err != nil {
				return err
			}
			b, _ := json.MarshalIndent(v, "", "  ")
			fmt.Println(string(b))
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Vault title (required)")

	_ = cmd.MarkFlagRequired("title")
	return cmd
}

func newDeleteVaultCmd() *cobra.Command {
	var vaultID string

	cmd := &cobra.Command{
		Use:   "delete-vault",
		Short: "Delete an empty vault (must contain no memories)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Client-side validation removed; rely on server-side validation

			c, err := client.NewWithDevMode(serviceURL)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()

			if err := c.DeleteVault(ctx, vaultID); err != nil {
				return err
			}
			fmt.Println("Vault deleted")
			return nil
		},
	}

	cmd.Flags().StringVar(&vaultID, "vault-id", "", "Vault ID (required)")

	_ = cmd.MarkFlagRequired("vault-id")
	return cmd
}

// ------------------ Memory Listing Command -------------------

func newListMemoriesCmd() *cobra.Command {
	var vaultID, vaultTitle string

	cmd := &cobra.Command{
		Use:   "list-memories",
		Short: "List memories within a vault (by ID or title)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Client-side validation removed; rely on server-side validation

			if vaultID == "" && vaultTitle == "" {
				return fmt.Errorf("either --vault-id or --vault-title must be provided")
			}
			if vaultID != "" && vaultTitle != "" {
				return fmt.Errorf("provide only one of --vault-id or --vault-title, not both")
			}

			// Client-side validation removed; rely on server-side validation

			c, err := client.NewWithDevMode(serviceURL)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()

			var mems []client.Memory
			if vaultID != "" {
				var err error
				mems, err = c.ListMemories(ctx, vaultID)
				if err != nil {
					return err
				}
			} else {
				// Get vault by title to obtain vault ID
				vault, err := c.GetVaultByTitle(ctx, vaultTitle)
				if err != nil {
					return fmt.Errorf("failed to get vault by title '%s': %w", vaultTitle, err)
				}
				mems, err = c.ListMemories(ctx, vault.VaultID)
				if err != nil {
					return err
				}
			}

			for _, m := range mems {
				fmt.Printf("%s\t%s\n", m.ID, m.Title)
			}
			fmt.Printf("Total: %d\n", len(mems))
			return nil
		},
	}

	cmd.Flags().StringVar(&vaultID, "vault-id", "", "Vault ID (mutually exclusive with --vault-title)")
	cmd.Flags().StringVar(&vaultTitle, "vault-title", "", "Vault title (mutually exclusive with --vault-id)")

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
