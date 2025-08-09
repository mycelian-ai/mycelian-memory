package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

// runContextPut uploads a JSON context to the Memory API.
// ctxJSON must be a raw JSON string representing an object.
func runContextPut(api, user, memory, ctxJSON string, out io.Writer) error {
	if user == "" || memory == "" {
		return fmt.Errorf("--user and --memory required")
	}
	if ctxJSON == "" {
		return fmt.Errorf("--json payload required")
	}

	// Validate that ctxJSON is a valid JSON object
	var tmp map[string]interface{}
	if err := json.Unmarshal([]byte(ctxJSON), &tmp); err != nil {
		return fmt.Errorf("context must be a JSON object: %w", err)
	}

	// Build request body { "context": {..} }
	bodyMap := map[string]interface{}{"context": tmp}
	bodyBytes, _ := json.Marshal(bodyMap)

	if vaultFlag == "" {
		return fmt.Errorf("--vault required (set with -v)")
	}
	url := fmt.Sprintf("%s/api/users/%s/vaults/%s/memories/%s/contexts", api, user, vaultFlag, memory)
	resp, err := httpPutJSON(url, bodyBytes)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	_, _ = io.Copy(out, resp.Body)
	return nil
}

// runContextGet fetches the latest context snapshot.
func runContextGet(api, user, memory string, out io.Writer) error {
	if user == "" || memory == "" {
		return fmt.Errorf("--user and --memory required")
	}
	if vaultFlag == "" {
		return fmt.Errorf("--vault required (set with -v)")
	}
	url := fmt.Sprintf("%s/api/users/%s/vaults/%s/memories/%s/contexts", api, user, vaultFlag, memory)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	_, _ = io.Copy(out, resp.Body)
	return nil
}

// httpPutJSON is a helper to send PUT with application/json.
func httpPutJSON(url string, payload []byte) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	cli := &http.Client{}
	return cli.Do(req)
}

func init() {
	// Parent command
	contextCmd := &cobra.Command{
		Use:   "context",
		Short: "Manage memory context snapshots",
	}

	// put subcommand
	putCmd := &cobra.Command{
		Use:   "put",
		Short: "Upload a context JSON snapshot",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonStr, _ := cmd.Flags().GetString("json")
			return runContextPut(apiFlag, userFlag, memoryFlag, jsonStr, os.Stdout)
		},
	}
	putCmd.Flags().StringVarP(&userFlag, "user", "u", "", "User ID (required)")
	putCmd.Flags().StringVarP(&memoryFlag, "memory", "m", "", "Memory ID (required)")
	putCmd.Flags().StringP("json", "j", "", "Context JSON payload (required)")
	_ = putCmd.MarkFlagRequired("json")

	// get subcommand
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Fetch the latest context snapshot",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runContextGet(apiFlag, userFlag, memoryFlag, os.Stdout)
		},
	}
	getCmd.Flags().StringVarP(&userFlag, "user", "u", "", "User ID (required)")
	getCmd.Flags().StringVarP(&memoryFlag, "memory", "m", "", "Memory ID (required)")

	contextCmd.AddCommand(putCmd, getCmd)

	rootCmd.AddCommand(contextCmd)
}
