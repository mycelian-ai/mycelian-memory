package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	apiFlag    string
	userFlag   string
	vaultFlag  string
	memoryFlag string
	rootCmd    = &cobra.Command{
		Use:   "memoryctl",
		Short: "CLI client for Memory backend REST API",
	}
)

func main() {
	rootCmd.PersistentFlags().StringVarP(&apiFlag, "api", "a", "http://localhost:8080", "Memory service base URL")
	rootCmd.PersistentFlags().StringVarP(&vaultFlag, "vault", "v", "", "Vault ID (required for memory operations)")

	// search subcommand
	searchCmd := &cobra.Command{
		Use:   "search",
		Short: "Search entries in a memory",
		RunE: func(cmd *cobra.Command, args []string) error {
			query, _ := cmd.Flags().GetString("query")
			topk, _ := cmd.Flags().GetInt("topk")
			if userFlag == "" || memoryFlag == "" {
				return fmt.Errorf("--user and --memory required")
			}
			return runSearch(apiFlag, userFlag, memoryFlag, query, topk, os.Stdout)
		},
	}
	searchCmd.Flags().StringVarP(&userFlag, "user", "u", "", "User ID (required)")
	searchCmd.Flags().StringVarP(&memoryFlag, "memory", "m", "", "Memory ID (required)")
	searchCmd.Flags().StringP("query", "q", "", "Search query text (required)")
	searchCmd.Flags().IntP("topk", "k", 5, "Number of top results to return")
	_ = searchCmd.MarkFlagRequired("query")
	rootCmd.AddCommand(searchCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
