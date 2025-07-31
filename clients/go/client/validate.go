package client

import "fmt"

// requireUserID returns error if userID is empty.
func requireUserID(userID string) error {
	if userID == "" {
		return fmt.Errorf("userId is required")
	}
	return nil
}
