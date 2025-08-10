package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	vaultCmd := &cobra.Command{
		Use:   "vaults",
		Short: "Vault operations",
	}

	var userID, title, desc string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a vault for a user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userID == "" {
				return fmt.Errorf("--user required")
			}
			if title == "" {
				return fmt.Errorf("--title required")
			}
			payload := map[string]interface{}{"title": title}
			if desc != "" {
				payload["description"] = desc
			}
			url := fmt.Sprintf("%s/api/users/%s/vaults", apiFlag, userID)
			data, err := doPostJSON(url, payload)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(os.Stdout, string(data))
			return nil
		},
	}
	createCmd.Flags().StringVarP(&userID, "user", "u", "", "User ID (required)")
	createCmd.Flags().StringVarP(&title, "title", "l", "", "Vault title (required)")
	createCmd.Flags().StringVarP(&desc, "desc", "d", "", "Description")
	vaultCmd.AddCommand(createCmd)

	// list vaults
	listCmd := &cobra.Command{
		Use:   "list USER_ID",
		Short: "List vaults for a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := fmt.Sprintf("%s/api/users/%s/vaults", apiFlag, args[0])
			data, err := doGet(url)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(os.Stdout, string(data))
			return nil
		},
	}
	vaultCmd.AddCommand(listCmd)

	// get vault
	getCmd := &cobra.Command{
		Use:   "get USER_ID VAULT_ID",
		Short: "Get a vault by ID",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := fmt.Sprintf("%s/api/users/%s/vaults/%s", apiFlag, args[0], args[1])
			data, err := doGet(url)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(os.Stdout, string(data))
			return nil
		},
	}
	vaultCmd.AddCommand(getCmd)

	// delete vault
	delCmd := &cobra.Command{
		Use:   "delete USER_ID VAULT_ID",
		Short: "Delete a vault",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := fmt.Sprintf("%s/api/users/%s/vaults/%s", apiFlag, args[0], args[1])
			return doDelete(url)
		},
	}
	vaultCmd.AddCommand(delCmd)

	rootCmd.AddCommand(vaultCmd)
}
