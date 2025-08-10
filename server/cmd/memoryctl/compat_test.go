package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Test-only shims to satisfy existing tests after moving the CLI to tools/.
// These are minimal re-implementations of the previous helpers.

var vaultFlag string

func runContextPut(api, user, memory, ctxJSON string, out io.Writer) error {
	if user == "" || memory == "" {
		return fmt.Errorf("--user and --memory required")
	}
	if ctxJSON == "" {
		return fmt.Errorf("--json payload required")
	}

	var tmp map[string]interface{}
	if err := json.Unmarshal([]byte(ctxJSON), &tmp); err != nil {
		return fmt.Errorf("context must be a JSON object: %w", err)
	}

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

func httpPutJSON(url string, payload []byte) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	cli := &http.Client{}
	return cli.Do(req)
}

func runSearch(apiURL, userID, memoryID, query string, topK int, out io.Writer) error {
	if query == "" {
		return fmt.Errorf("query cannot be empty")
	}
	payload := map[string]interface{}{
		"userId":   userID,
		"memoryId": memoryID,
		"query":    query,
		"topK":     topK,
	}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(apiURL+"/api/search", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(data))
	}
	_, err = io.Copy(out, resp.Body)
	return err
}
