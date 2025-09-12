package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func runSearch(apiURL, userID, memoryID, query string, topKE int, out io.Writer) error {
	if query == "" {
		return fmt.Errorf("query cannot be empty")
	}
	// Note: userID is no longer in the request body, it's handled via authorization
	if topKE <= 0 {
		topKE = 5 // default
	}
	topKC := 2 // default

	payload := map[string]interface{}{
		"memoryId": memoryID,
		"query":    query,
		"top_ke":   topKE,
		"top_kc":   topKC,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, apiURL+"/v0/search", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	// Add authorization header (using dev mode for now)
	req.Header.Set("Authorization", "Bearer LOCAL_DEV_MODE_NOT_FOR_PRODUCTION")
	resp, err := httpClient.Do(req)
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
