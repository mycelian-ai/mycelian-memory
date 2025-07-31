package indexer

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// State manages persisted high-watermark timestamps per memory.
// Data is stored as JSON files under ~/.indexer-prototype/watermarks.
//   filename: <userId>_<memoryId>.json
//   contents: {"watermark":"RFC3339Nano"}

const defaultStateDir = ".indexer-prototype/watermarks"

// Watermark represents a stored timestamp in RFC3339Nano format.

type Watermark struct {
	Timestamp string `json:"watermark"`
}

// State handles loading and saving watermarks.

type State struct {
	dir string
}

// NewState creates a new State. If dir is empty, $HOME/.indexer-prototype/watermarks is used.
func NewState(dir string) (*State, error) {
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		dir = filepath.Join(home, defaultStateDir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &State{dir: dir}, nil
}

// key returns filename for the given user+memory combination.
func (s *State) key(userID, memoryID string) string {
	return filepath.Join(s.dir, fmt.Sprintf("%s_%s.json", userID, memoryID))
}

// Load returns the stored watermark; if no file found, returns zero time and no error.
func (s *State) Load(userID, memoryID string) (time.Time, error) {
	path := s.key(userID, memoryID)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	var w Watermark
	if err := json.Unmarshal(data, &w); err != nil {
		return time.Time{}, err
	}
	ts, err := time.Parse(time.RFC3339Nano, w.Timestamp)
	if err != nil {
		return time.Time{}, err
	}
	return ts, nil
}

// Save writes the watermark timestamp.
func (s *State) Save(userID, memoryID string, ts time.Time) error {
	w := Watermark{Timestamp: ts.UTC().Format(time.RFC3339Nano)}
	data, err := json.MarshalIndent(&w, "", "  ")
	if err != nil {
		return err
	}
	path := s.key(userID, memoryID)
	return os.WriteFile(path, data, 0o644)
}
