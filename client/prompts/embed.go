package prompts

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
)

// Version is incremented whenever default prompts change incompatibly.
const Version = "v1"

// defaultFS holds the embedded prompt assets.
//
//go:embed default/** system/context_summary_rules.md
var defaultFS embed.FS

// DefaultPromptResponse is the JSON-serialisable structure returned to callers.
type DefaultPromptResponse struct {
	Version             string            `json:"version"`
	ContextSummaryRules string            `json:"context_summary_rules"`
	Templates           map[string]string `json:"templates"`
}

// LoadDefaultPrompts returns the embedded prompt templates for the requested
// memoryType (e.g. "chat", "code"). It fails if the memoryType folder or any
// required template file is missing.
func LoadDefaultPrompts(memoryType string) (*DefaultPromptResponse, error) {
	if memoryType == "" {
		return nil, fmt.Errorf("memory type cannot be empty")
	}

	files := []string{"context_prompt.md", "entry_capture_prompt.md", "summary_prompt.md"}
	tmpl := make(map[string]string, len(files))

	for _, name := range files {
		b, err := fs.ReadFile(defaultFS, filepath.Join("default", memoryType, name))
		if err != nil {
			return nil, fmt.Errorf("unknown memory type %q or missing %s: %w", memoryType, name, err)
		}
		key := name[:len(name)-3] // strip .md
		tmpl[key] = string(b)
	}

	rules, err := fs.ReadFile(defaultFS, "system/context_summary_rules.md")
	if err != nil {
		return nil, fmt.Errorf("context_summary_rules missing: %w", err)
	}

	return &DefaultPromptResponse{
		Version:             Version,
		ContextSummaryRules: string(rules),
		Templates:           tmpl,
	}, nil
}

// ListMemoryTypes returns all memoryType directory names found under default/.
func ListMemoryTypes() ([]string, error) {
	entries, err := fs.ReadDir(defaultFS, "default")
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out, nil
}
