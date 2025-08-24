package logger

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout runs f with os.Stdout redirected to a pipe and returns the output.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	f()

	_ = w.Close()
	b, _ := io.ReadAll(r)
	_ = r.Close()
	return string(b)
}

func lastNonEmptyLine(s string) string {
	lines := strings.Split(s, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			return lines[i]
		}
	}
	return ""
}

func TestLogger_IncludesStackAndServiceOnError(t *testing.T) {
	out := captureStdout(t, func() {
		log := New("test-service")
		err := errors.New("boom")
		log.Error().Stack().Err(err).Msg("something failed")
	})

	line := lastNonEmptyLine(out)
	if line == "" {
		t.Fatalf("no output captured")
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(line), &payload); err != nil {
		t.Fatalf("invalid json log: %v\n%s", err, line)
	}

	if svc, ok := payload["service"].(string); !ok || svc != "test-service" {
		t.Fatalf("expected service=\"test-service\", got %v", payload["service"])
	}
	if lvl, ok := payload["level"].(string); !ok || lvl != "error" {
		t.Fatalf("expected level=\"error\", got %v", payload["level"])
	}
	if _, ok := payload["stack"]; !ok {
		t.Fatalf("expected stack field in error log: %s", line)
	}
}
