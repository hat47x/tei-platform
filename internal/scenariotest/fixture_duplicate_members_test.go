package scenariotest

import (
	"strings"
	"testing"
)

// A fixture is test input, not an oracle. Duplicate JSON members must not
// silently overwrite the values on which a Scenario claims to have run.
func TestScenarioFixtureParsersRejectDuplicateJSONMembers(t *testing.T) {
	parseValidation := func(source []byte) error {
		_, err := ParseValidationFixture(source)
		return err
	}
	parseBrowse := func(source []byte) error {
		_, err := ParseBrowseFixture(source)
		return err
	}
	parseObservation := func(source []byte) error {
		_, err := ParseObservationFixture(source)
		return err
	}
	cases := []struct {
		name          string
		source        string
		parse         func([]byte) error
		wantDuplicate bool
	}{
		{name: "validation-nested-number", source: "{\"numbers\":{\"sales.order_line.quantity\":\"2.5\",\"sales.order_line.quantity\":\"0\"},\"relationKeys\":[]}", parse: parseValidation, wantDuplicate: true},
		{name: "validation-root-member", source: "{\"numbers\":{\"sales.order_line.quantity\":\"2.5\"},\"numbers\":{\"sales.order_line.quantity\":\"0\"},\"relationKeys\":[]}", parse: parseValidation, wantDuplicate: true},
		{name: "validation-relation-field", source: "{\"numbers\":{},\"relationKeys\":[{\"relation\":\"sales.order.lines\",\"attribute\":\"sales.order_line.line_number\",\"keys\":[\"1\",\"2\"],\"keys\":[\"1\",\"1\"]}]}", parse: parseValidation, wantDuplicate: true},
		{name: "browse-root-query", source: "{\"query\":\"order_entry.recent_orders\",\"query\":\"ignored\",\"datasourceAlias\":\"db\",\"targetDatabase\":\"postgres\",\"columns\":[\"id\"],\"rows\":[]}", parse: parseBrowse, wantDuplicate: true},
		{name: "browse-row-value", source: "{\"query\":\"order_entry.recent_orders\",\"datasourceAlias\":\"db\",\"targetDatabase\":\"postgres\",\"columns\":[\"id\"],\"rows\":[[{\"kind\":\"int64\",\"kind\":\"string\",\"data\":\"1\"}]]}", parse: parseBrowse, wantDuplicate: true},
		{name: "observation-read-identity", source: "{\"readIdentity\":\"a\",\"readIdentity\":\"b\",\"nature\":\"synthetic\",\"observed\":{\"kind\":\"string\",\"data\":\"pending\"}}", parse: parseObservation, wantDuplicate: true},
		{name: "observation-nested-observed", source: "{\"readIdentity\":\"a\",\"nature\":\"synthetic\",\"observed\":{\"kind\":\"string\",\"data\":\"pending\",\"data\":\"closed\"}}", parse: parseObservation, wantDuplicate: true},
		{name: "valid-validation", source: "{\"numbers\":{\"sales.order_line.quantity\":\"2.5\"},\"relationKeys\":[]}", parse: parseValidation, wantDuplicate: false},
		{name: "valid-browse", source: "{\"query\":\"order_entry.recent_orders\",\"datasourceAlias\":\"db\",\"targetDatabase\":\"postgres\",\"columns\":[\"id\"],\"rows\":[[{\"kind\":\"int64\",\"data\":\"1\"}]]}", parse: parseBrowse, wantDuplicate: false},
		{name: "valid-observation", source: "{\"readIdentity\":\"a\",\"nature\":\"synthetic\",\"observed\":{\"kind\":\"string\",\"data\":\"pending\"}}", parse: parseObservation, wantDuplicate: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.parse([]byte(tc.source))
			if tc.wantDuplicate {
				if err == nil || !strings.Contains(err.Error(), "duplicate JSON member") {
					t.Fatalf("duplicate member was not rejected explicitly: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("valid fixture was rejected: %v", err)
			}
		})
	}
}
