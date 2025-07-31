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
	"time"
)

var binPath string

// build the CLI binary once for all integration tests
func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "synapse-cli-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v", err)
		os.Exit(1)
	}
	binPath = filepath.Join(tmpDir, "synapse")

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
		serviceURL = "http://localhost:8080"
	}

	// 1) create user
	email := fmt.Sprintf("inttest_%d@example.com", time.Now().UnixNano())
	uid := fmt.Sprintf("u%d", time.Now().UnixNano()%1e8)
	cmdUser := exec.Command(binPath, "create-user", "--user-id", uid, "--email", email)
	cmdUser.Env = append(os.Environ(), "MEMORY_SERVICE_URL="+serviceURL)
	outUser, err := cmdUser.CombinedOutput()
	if err != nil {
		t.Fatalf("create-user failed: %v\noutput: %s", err, string(outUser))
	}
	t.Logf("create-user output: %s", string(outUser))

	reUser := regexp.MustCompile(`User created: ([a-zA-Z0-9\-]+)`)
	matches := reUser.FindStringSubmatch(string(outUser))
	if len(matches) < 2 {
		t.Fatalf("could not parse user ID from output: %s", string(outUser))
	}
	userID := matches[1]

	// 2) create vault
	vaultTitle := "it-vault"
	cmdVault := exec.Command(binPath, "create-vault", "--user-id", userID, "--title", vaultTitle)
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

	// 3) create memory
	cmdMem := exec.Command(binPath, "create-memory", "--user-id", userID, "--vault-id", vaultID, "--title", "integration-memory", "--memory-type", "PROJECT")
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
