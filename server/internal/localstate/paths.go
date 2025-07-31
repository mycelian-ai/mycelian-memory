package localstate

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	envHome    = "LOCALSTATE_HOME" // override for tests
	dirName    = ".synapse-memory" // default under $HOME
	dbFilename = "memory.db"
)

// DataDir returns the directory where local state is stored (~/.synapse-memory).
// It creates the directory with 0700 permissions if it does not exist.
func DataDir() (string, error) {
	if custom := os.Getenv(envHome); custom != "" {
		if err := os.MkdirAll(custom, 0o700); err != nil {
			return "", err
		}
		return custom, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine user home: %w", err)
	}
	dir := filepath.Join(home, dirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

// DBPath returns the absolute path to the SQLite database file.
func DBPath() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, dbFilename), nil
}
