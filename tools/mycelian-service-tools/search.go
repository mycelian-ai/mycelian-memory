package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

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
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, apiURL+"/api/search", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
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
