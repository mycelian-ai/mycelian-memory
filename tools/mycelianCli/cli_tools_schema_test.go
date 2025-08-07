package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestCLI_GetToolsSchema(t *testing.T) {
	root := NewRootCmd()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetArgs([]string{"get-tools-schema"})
	if err := root.Execute(); err != nil {
		t.Fatalf("get-tools-schema cmd failed: %v", err)
	}

	var v interface{}
	if err := json.Unmarshal(buf.Bytes(), &v); err != nil {
		t.Fatalf("CLI output is not valid JSON: %v", err)
	}
}
