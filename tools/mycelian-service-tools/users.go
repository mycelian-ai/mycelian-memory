package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	usersCmd := &cobra.Command{Use: "users", Short: "User operations"}

	// create
	var userId, email, fullName, tz string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userId == "" || email == "" {
				return fmt.Errorf("--userId and --email required")
			}
			payload := map[string]interface{}{"userId": userId, "email": email}
			if fullName != "" {
				payload["displayName"] = fullName
			}
			if tz != "" {
				payload["timeZone"] = tz
			}
			url := fmt.Sprintf("%s/api/users", apiFlag)
			data, err := doPostJSON(url, payload)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(os.Stdout, string(data))
			return nil
		},
	}
	createCmd.Flags().StringVarP(&userId, "userId", "u", "", "UserID (required)")
	createCmd.Flags().StringVarP(&email, "email", "e", "", "User email (required)")
	createCmd.Flags().StringVarP(&fullName, "name", "n", "", "Full name")
	createCmd.Flags().StringVarP(&tz, "tz", "t", "", "Time zone (defaults UTC)")
	_ = createCmd.MarkFlagRequired("userId")
	usersCmd.AddCommand(createCmd)

	// get
	getCmd := &cobra.Command{
		Use:   "get USER_ID",
		Short: "Get user by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := fmt.Sprintf("%s/api/users/%s", apiFlag, args[0])
			data, err := doGet(url)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(os.Stdout, string(data))
			return nil
		},
	}
	usersCmd.AddCommand(getCmd)

	rootCmd.AddCommand(usersCmd)
}
