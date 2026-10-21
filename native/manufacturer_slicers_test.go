package main

import (
	"encoding/json"
	"testing"
)

func TestManufacturerSlicersEndpointReturnsDetectionItems(t *testing.T) {
	items := (&App{}).ManufacturerSlicers()
	if items == nil {
		t.Fatal("endpoint must return a non-nil list")
	}
	raw, err := json.Marshal(items)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded) != len(items) || len(decoded) == 0 {
		t.Fatalf("want one item per supported slicer, got %d", len(decoded))
	}
	for _, item := range decoded {
		if _, ok := item["manufacturers"].([]any); !ok {
			t.Fatalf("manufacturers must be a JSON array: %v", item)
		}
		for _, key := range []string{"id", "name", "installed", "evidence", "sourceOnly"} {
			if _, ok := item[key]; !ok {
				t.Fatalf("missing field %q: %v", key, item)
			}
		}
	}
}
