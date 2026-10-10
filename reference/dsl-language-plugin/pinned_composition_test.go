package languagereference

import (
	"errors"
	"testing"
)

func TestComposePinnedDeterministic(t *testing.T) {
	where := OperatorContract{ID: "std.where", Version: "1", Fingerprint: "where-v1", Effect: "pure", InputType: "rows/order", OutputType: "rows/order"}
	expand := OperatorContract{ID: "std.expand", Version: "1", Fingerprint: "expand-v1", Effect: "pure", InputType: "rows/order", OutputType: "rows/line"}
	calls := []PinnedCall{{Contract: where, InputType: "rows/order"}, {Contract: expand, InputType: "rows/order"}}
	for _, catalog := range [][]OperatorContract{{expand, where}, {where, expand}} {
		got, err := ComposePinned("rows/order", calls, catalog)
		if err != nil || got != "rows/line" {
			t.Fatalf("got %q: %v", got, err)
		}
	}
}
func TestComposePinnedRejectsDrift(t *testing.T) {
	pin := OperatorContract{ID: "std.where", Version: "1", Fingerprint: "accepted-v1", Effect: "pure", InputType: "rows/order", OutputType: "rows/order"}
	for _, tc := range []struct {
		name    string
		catalog []OperatorContract
		want    error
	}{
		{"missing", nil, ErrOperatorMissing},
		{"fingerprint drift", []OperatorContract{{ID: pin.ID, Version: pin.Version, Fingerprint: "new", Effect: "pure", InputType: pin.InputType, OutputType: pin.OutputType}}, ErrOperatorMissing},
		{"duplicate", []OperatorContract{pin, pin}, ErrOperatorAmbiguous},
		{"signature drift", []OperatorContract{{ID: pin.ID, Version: pin.Version, Fingerprint: pin.Fingerprint, Effect: "pure", InputType: "other", OutputType: pin.OutputType}}, ErrOperatorInvalid},
		{"effect drift", []OperatorContract{{ID: pin.ID, Version: pin.Version, Fingerprint: pin.Fingerprint, Effect: "write", InputType: pin.InputType, OutputType: pin.OutputType}}, ErrOperatorInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ComposePinned("rows/order", []PinnedCall{{Contract: pin, InputType: "rows/order"}}, tc.catalog)
			if !errors.Is(err, tc.want) {
				t.Fatalf("want %v, got %v", tc.want, err)
			}
		})
	}
}
