package localstate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDataDir_Override(t *testing.T) {
	tmp := t.TempDir()
	os.Setenv(envHome, tmp)
	defer os.Unsetenv(envHome)

	dir, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir error: %v", err)
	}
	if dir != tmp {
		t.Fatalf("expected dir %s, got %s", tmp, dir)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("dir not created: %v", err)
	}
}

func TestDBPath(t *testing.T) {
	tmp := t.TempDir()
	os.Setenv(envHome, tmp)
	defer os.Unsetenv(envHome)

	p, err := DBPath()
	if err != nil {
		t.Fatalf("DBPath error: %v", err)
	}
	expected := filepath.Join(tmp, dbFilename)
	if p != expected {
		t.Fatalf("expected path %s, got %s", expected, p)
	}
}
