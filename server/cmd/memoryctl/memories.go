package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	memCmd := &cobra.Command{Use: "memories", Short: "Memory operations"}

	var userID, mType, title, desc string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create memory for user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userID == "" {
				return fmt.Errorf("--user required")
			}
			if vaultFlag == "" {
				return fmt.Errorf("--vault required (set with -v)")
			}
			if mType == "" || title == "" {
				return fmt.Errorf("--type and --title required")
			}
			if len(title) > 256 {
				return fmt.Errorf("title exceeds 256 characters (got %d)", len(title))
			}
			if len(desc) > 2048 {
				return fmt.Errorf("description exceeds 2048 characters (got %d)", len(desc))
			}
			payload := map[string]interface{}{"memoryType": mType, "title": title}
			if desc != "" {
				payload["description"] = desc
			}
			url := fmt.Sprintf("%s/api/users/%s/vaults/%s/memories", apiFlag, userID, vaultFlag)
			data, err := doPostJSON(url, payload)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(os.Stdout, string(data))
			return nil
		},
	}
	createCmd.Flags().StringVarP(&userID, "user", "u", "", "User ID (required)")
	createCmd.Flags().StringVarP(&mType, "type", "t", "CONVERSATION", "Memory type")
	createCmd.Flags().StringVarP(&title, "title", "l", "", "Title (required)")
	createCmd.Flags().StringVarP(&desc, "desc", "d", "", "Description")
	memCmd.AddCommand(createCmd)

	// list
	listCmd := &cobra.Command{
		Use:   "list USER_ID",
		Short: "List memories for user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if vaultFlag == "" {
				return fmt.Errorf("--vault required (set with -v)")
			}
			url := fmt.Sprintf("%s/api/users/%s/vaults/%s/memories", apiFlag, args[0], vaultFlag)
			data, err := doGet(url)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(os.Stdout, string(data))
			return nil
		},
	}
	memCmd.AddCommand(listCmd)

	// get
	getCmd := &cobra.Command{
		Use:   "get USER_ID MEMORY_ID",
		Short: "Get memory",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if vaultFlag == "" {
				return fmt.Errorf("--vault required (set with -v)")
			}
			url := fmt.Sprintf("%s/api/users/%s/vaults/%s/memories/%s", apiFlag, args[0], vaultFlag, args[1])
			data, err := doGet(url)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(os.Stdout, string(data))
			return nil
		},
	}
	memCmd.AddCommand(getCmd)

	rootCmd.AddCommand(memCmd)
}
