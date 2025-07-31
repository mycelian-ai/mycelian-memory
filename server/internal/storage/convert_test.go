package storage

import "testing"

func TestConvertJSONToMap(t *testing.T) {
	// valid map passthrough
	m := map[string]interface{}{"a": 1}
	out, err := convertJSONToMap(m)
	if err != nil || out["a"].(int) != 1 {
		t.Fatalf("expected map passthrough, got error %v", err)
	}

	// valid JSON string
	s := `{"k":"v"}`
	out, err = convertJSONToMap(s)
	if err != nil || out["k"].(string) != "v" {
		t.Fatalf("expected parsed string json, err %v", err)
	}

	// invalid JSON string should error
	_, err = convertJSONToMap("not{json")
	if err == nil {
		t.Fatalf("expected error for invalid json string")
	}

	// unsupported type should error
	_, err = convertJSONToMap(42)
	if err == nil {
		t.Fatalf("expected error for unsupported type")
	}
}
