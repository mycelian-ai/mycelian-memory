package schema

import (
	"encoding/json"
	"testing"
)

func TestToolsSchemaIsValidJSON(t *testing.T) {
	var v interface{}
	if err := json.Unmarshal(ToolsSchema(), &v); err != nil {
		t.Fatalf("embedded tools schema is not valid JSON: %v", err)
	}
}
