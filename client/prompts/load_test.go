package prompts

import "testing"

func TestLoadDefaultPrompts_OK(t *testing.T) {
	resp, err := LoadDefaultPrompts("chat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Version != Version {
		t.Fatalf("expected version %q, got %q", Version, resp.Version)
	}
	// Required keys
	wantKeys := []string{"context_prompt", "entry_capture_prompt", "summary_prompt"}
	for _, k := range wantKeys {
		if v, ok := resp.Templates[k]; !ok || v == "" {
			t.Fatalf("missing or empty template %q", k)
		}
	}
	if resp.ContextSummaryRules == "" {
		t.Fatalf("context summary rules empty")
	}
}

func TestLoadDefaultPrompts_Unknown(t *testing.T) {
	if _, err := LoadDefaultPrompts("unknown"); err == nil {
		t.Fatalf("expected error for unknown memory type")
	}
}
