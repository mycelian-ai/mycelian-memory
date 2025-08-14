package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	entCmd := &cobra.Command{Use: "entries", Short: "Memory entry operations"}

	// add entry
	var userID, memoryID, raw, summary string
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add entry to memory",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userID == "" || memoryID == "" {
				return fmt.Errorf("--user and --memory required")
			}
			if raw == "" {
				return fmt.Errorf("--raw required")
			}
			payload := map[string]interface{}{"rawEntry": raw}
			if summary != "" {
				payload["summary"] = summary
			}
			url := fmt.Sprintf("%s/api/users/%s/vaults/%s/memories/%s/entries", apiFlag, userID, vaultFlag, memoryID)
			data, err := doPostJSON(url, payload)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(os.Stdout, string(data))
			return nil
		},
	}
	addCmd.Flags().StringVarP(&userID, "user", "u", "", "User ID (required)")
	addCmd.Flags().StringVarP(&memoryID, "memory", "m", "", "Memory ID (required)")
	addCmd.Flags().StringVarP(&raw, "raw", "r", "", "Raw entry text (required)")
	addCmd.Flags().StringVarP(&summary, "summary", "s", "", "Summary")
	entCmd.AddCommand(addCmd)

	// list entries
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List entries in memory",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userID == "" || memoryID == "" {
				return fmt.Errorf("--user and --memory required")
			}
			if vaultFlag == "" {
				return fmt.Errorf("--vault required (set with -v)")
			}
			url := fmt.Sprintf("%s/api/users/%s/vaults/%s/memories/%s/entries", apiFlag, userID, vaultFlag, memoryID)
			data, err := doGet(url)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(os.Stdout, string(data))
			return nil
		},
	}
	listCmd.Flags().StringVarP(&userID, "user", "u", "", "User ID (required)")
	listCmd.Flags().StringVarP(&memoryID, "memory", "m", "", "Memory ID (required)")
	entCmd.AddCommand(listCmd)

	rootCmd.AddCommand(entCmd)
}
