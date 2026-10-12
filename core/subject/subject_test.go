package subject

import (
	"encoding/json"
	"testing"
)

func TestSubjectIdentityAndJSON(t *testing.T) {
	if !(Subject{}).IsZero() || (Subject{Kind: "event"}).IsZero() ||
		(Subject{ID: "sales.order"}).IsZero() {
		t.Fatal("a subject is zero only when both fields are absent")
	}
	value := Subject{Kind: "event", ID: "sales.order"}
	if value.IsZero() {
		t.Fatal("populated subject was zero")
	}
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"kind":"event","id":"sales.order"}`
	if string(data) != want {
		t.Fatalf("JSON wire changed: %s", data)
	}
	var decoded Subject
	if err := json.Unmarshal(data, &decoded); err != nil || decoded != value {
		t.Fatalf("subject did not roundtrip: %+v %v", decoded, err)
	}
}
