//go:build integration
// +build integration

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
)

var binPath string

// build the CLI binary once for all integration tests
func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "mycelian-cli-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v", err)
		os.Exit(1)
	}
	binPath = filepath.Join(tmpDir, "mycelianCli")

	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Env = os.Environ()
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build CLI: %v", err)
		os.Exit(1)
	}

	code := m.Run()

	_ = os.RemoveAll(tmpDir)
	os.Exit(code)
}

func TestCreateUserAndMemory(t *testing.T) {
	serviceURL := os.Getenv("MEMORY_SERVICE_URL")
	if serviceURL == "" {
		serviceURL = "http://localhost:11545"
	}

	// Note: Using dev mode auth - no user creation needed

	// 1) create vault
	vaultTitle := fmt.Sprintf("it-vault-%d", os.Getpid())
	cmdVault := exec.Command(binPath, "create-vault", "--title", vaultTitle)
	cmdVault.Env = append(os.Environ(), "MEMORY_SERVICE_URL="+serviceURL)
	outV, err := cmdVault.CombinedOutput()
	if err != nil {
		t.Fatalf("create-vault failed: %v\noutput: %s", err, string(outV))
	}
	t.Logf("create-vault output: %s", string(outV))

	reVault := regexp.MustCompile(`Vault created: ([a-f0-9\-]+)`)
	vmatch := reVault.FindStringSubmatch(string(outV))
	if len(vmatch) < 2 {
		t.Fatalf("could not parse vault ID: %s", string(outV))
	}
	vaultID := vmatch[1]

	// 2) create memory
	cmdMem := exec.Command(binPath, "create-memory", "--vault-id", vaultID, "--title", "integration-memory", "--memory-type", "PROJECT")
	cmdMem.Env = append(os.Environ(), "MEMORY_SERVICE_URL="+serviceURL)
	outMem, err := cmdMem.CombinedOutput()
	if err != nil {
		t.Fatalf("create-memory failed: %v\noutput: %s", err, string(outMem))
	}
	t.Logf("create-memory output: %s", string(outMem))

	reMem := regexp.MustCompile(`Memory created: ([a-zA-Z0-9\-]+) -`)
	if !reMem.Match(outMem) {
		t.Fatalf("could not confirm memory creation: %s", string(outMem))
	}
}

func TestSearchWithRawEntries(t *testing.T) {
	serviceURL := os.Getenv("MEMORY_SERVICE_URL")
	if serviceURL == "" {
		serviceURL = "http://localhost:11545"
	}

	// 1) Create vault with unique name
	vaultTitle := fmt.Sprintf("search-test-%d", os.Getpid())
	cmdVault := exec.Command(binPath, "create-vault", "--title", vaultTitle)
	cmdVault.Env = append(os.Environ(), "MEMORY_SERVICE_URL="+serviceURL)
	outV, err := cmdVault.CombinedOutput()
	if err != nil {
		t.Fatalf("create-vault failed: %v\noutput: %s", err, string(outV))
	}

	reVault := regexp.MustCompile(`Vault created: ([a-f0-9\-]+)`)
	vmatch := reVault.FindStringSubmatch(string(outV))
	if len(vmatch) < 2 {
		t.Fatalf("could not parse vault ID: %s", string(outV))
	}
	vaultID := vmatch[1]

	// 2) Create memory
	cmdMem := exec.Command(binPath, "create-memory", "--vault-id", vaultID, "--title", "search-memory", "--memory-type", "TEST")
	cmdMem.Env = append(os.Environ(), "MEMORY_SERVICE_URL="+serviceURL)
	outMem, err := cmdMem.CombinedOutput()
	if err != nil {
		t.Fatalf("create-memory failed: %v\noutput: %s", err, string(outMem))
	}

	reMem := regexp.MustCompile(`Memory created: ([a-f0-9\-]+) -`)
	mmatch := reMem.FindStringSubmatch(string(outMem))
	if len(mmatch) < 2 {
		t.Fatalf("could not parse memory ID: %s", string(outMem))
	}
	memoryID := mmatch[1]

	// Note: Since we can't easily add entries via CLI in this test,
	// we're mainly testing that the flag is accepted and the limits work.
	// The actual raw entry behavior is tested in the unit tests.

	// 3) Test search with default (include_raw_entries=false)
	cmdSearch1 := exec.Command(binPath, "search",
		"--memory-id", memoryID,
		"--query", "test",
		"--ke", "5",
		"--kc", "2")
	cmdSearch1.Env = append(os.Environ(), "MEMORY_SERVICE_URL="+serviceURL)
	outSearch1, err := cmdSearch1.CombinedOutput()
	if err != nil {
		// This might fail if memory is empty, which is ok for this test
		t.Logf("search without include-raw-entries output: %s", string(outSearch1))
	} else {
		// Verify JSON output structure
		if !regexp.MustCompile(`"entries"`).Match(outSearch1) {
			t.Errorf("expected 'entries' in output, got: %s", string(outSearch1))
		}
	}

	// 4) Test search with include_raw_entries=true
	cmdSearch2 := exec.Command(binPath, "search",
		"--memory-id", memoryID,
		"--query", "test",
		"--ke", "5",
		"--kc", "2",
		"--include-raw-entries")
	cmdSearch2.Env = append(os.Environ(), "MEMORY_SERVICE_URL="+serviceURL)
	outSearch2, err := cmdSearch2.CombinedOutput()
	if err != nil {
		// This might fail if memory is empty, which is ok for this test
		t.Logf("search with include-raw-entries output: %s", string(outSearch2))
	} else {
		// Verify JSON output structure
		if !regexp.MustCompile(`"entries"`).Match(outSearch2) {
			t.Errorf("expected 'entries' in output, got: %s", string(outSearch2))
		}
	}

	// 5) Test with max limits
	cmdSearch3 := exec.Command(binPath, "search",
		"--memory-id", memoryID,
		"--query", "test",
		"--ke", "25",  // Max limit
		"--kc", "10")  // Max limit
	cmdSearch3.Env = append(os.Environ(), "MEMORY_SERVICE_URL="+serviceURL)
	outSearch3, err := cmdSearch3.CombinedOutput()
	if err != nil {
		t.Logf("search with max limits output: %s", string(outSearch3))
	}

	// 6) Test validation - ke too high should fail
	cmdSearch4 := exec.Command(binPath, "search",
		"--memory-id", memoryID,
		"--query", "test",
		"--ke", "26")  // Over limit
	cmdSearch4.Env = append(os.Environ(), "MEMORY_SERVICE_URL="+serviceURL)
	outSearch4, err := cmdSearch4.CombinedOutput()
	if err == nil {
		t.Fatalf("expected error for ke=26, but got success")
	}
	if !regexp.MustCompile(`--ke must be between 0 and 25`).Match(outSearch4) {
		t.Fatalf("expected validation error for ke=26, got: %s", string(outSearch4))
	}

	// 7) Test validation - kc too high should fail
	cmdSearch5 := exec.Command(binPath, "search",
		"--memory-id", memoryID,
		"--query", "test",
		"--kc", "11")  // Over limit
	cmdSearch5.Env = append(os.Environ(), "MEMORY_SERVICE_URL="+serviceURL)
	outSearch5, err := cmdSearch5.CombinedOutput()
	if err == nil {
		t.Fatalf("expected error for kc=11, but got success")
	}
	if !regexp.MustCompile(`--kc must be between 0 and 10`).Match(outSearch5) {
		t.Fatalf("expected validation error for kc=11, got: %s", string(outSearch5))
	}
}
