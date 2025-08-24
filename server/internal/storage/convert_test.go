package storage

import (
	"encoding/json"
	"fmt"
	"testing"
)

func convertJSONToMap(v interface{}) (map[string]interface{}, error) {
	switch val := v.(type) {
	case map[string]interface{}:
		return val, nil
	case string:
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(val), &obj); err != nil {
			return nil, err
		}
		return obj, nil
	case []byte:
		var obj map[string]interface{}
		if err := json.Unmarshal(val, &obj); err != nil {
			return nil, err
		}
		return obj, nil
	default:
		return nil, fmt.Errorf("unsupported JSON type %T", v)
	}
}

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
