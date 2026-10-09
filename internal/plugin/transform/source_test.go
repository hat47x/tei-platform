package transform_test

import (
	"encoding/json"
	"testing"

	"tei/internal/plugin/transform"
)

func TestParseSourceJSONRejectsUnknownDuplicateAndTrailingData(t *testing.T) {
	source := transform.TransformSourceRealizerV1()
	raw, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("unknown field", func(t *testing.T) {
		var object map[string]json.RawMessage
		if err := json.Unmarshal(raw, &object); err != nil {
			t.Fatal(err)
		}
		object["unknown"] = json.RawMessage("true")
		changed, _ := json.Marshal(object)
		if _, err := transform.ParseSourceJSON(changed); err == nil {
			t.Fatal("expected unknown transform Source field to fail")
		}
	})

	t.Run("duplicate key", func(t *testing.T) {
		duplicate := []byte(`{"formatVersion":1,"formatVersion":1}`)
		if _, err := transform.ParseSourceJSON(duplicate); err == nil {
			t.Fatal("expected duplicate JSON key to fail")
		}
	})

	t.Run("trailing value", func(t *testing.T) {
		changed := append(append([]byte(nil), raw...), []byte(" {}")...)
		if _, err := transform.ParseSourceJSON(changed); err == nil {
			t.Fatal("expected trailing transform Source value to fail")
		}
	})
}
