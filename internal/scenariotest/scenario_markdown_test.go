package scenariotest

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestParseMarkdownScenarioTablesRoundTripsToJSON(t *testing.T) {
	md := strings.Join([]string{
		"---",
		"application: order_entry",
		"tei-test-scenario: \"true\"",
		"---",
		"",
		"# Order entry scenario from Markdown authoring",
		"",
		"```tei-scenario-md",
		"| step id | operation | pattern | expect | fixture | expected result |",
		"|---|---|---|---|---|---|",
		"| browse-pattern | resolve-pattern | order_entry.recent_orders | browse:sales.order,query:order_entry.recent_orders | - | - |",
		"| validate-positive | evaluate-validations | - | - | fixtures/order_valid.json | sales.order_line.quantity_positive=satisfied,sales.order.lines.line_number_unique=satisfied,sales.order_line.line_number_positive=satisfied |",
		"| observe-synthetic | observe-synthetic-state | - | - | fixtures/order_state_pending.json | observation:pending |",
		"```",
		"",
	}, "\n")
	got, err := ParseMarkdownScenarioTables(md, "order-entry-smoke")
	if err != nil {
		t.Fatal(err)
	}
	want := Scenario{
		SchemaVersion: SchemaVersion,
		ID:            "order-entry-smoke",
		Application:   "order_entry",
		Steps: []Step{
			{ID: "browse-pattern", Operation: "resolve-pattern", Pattern: "order_entry.recent_orders", ExpectKind: "browse", ExpectData: "sales.order", ExpectQuery: "order_entry.recent_orders"},
			{ID: "validate-positive", Operation: "evaluate-validations", Fixture: "fixtures/order_valid.json", ExpectValidations: map[string]string{"sales.order_line.quantity_positive": "satisfied", "sales.order.lines.line_number_unique": "satisfied", "sales.order_line.line_number_positive": "satisfied"}},
			{ID: "observe-synthetic", Operation: "observe-synthetic-state", Fixture: "fixtures/order_state_pending.json", ReadIdentity: "synthetic.order-entry-smoke.observe-synthetic", ExpectObservation: &Value{Kind: "string", Data: "pending"}},
		},
	}
	if len(got.Steps) != len(want.Steps) {
		t.Fatalf("step count mismatch: got=%d want=%d", len(got.Steps), len(want.Steps))
	}
	for i := range want.Steps {
		if got.Steps[i].ID != want.Steps[i].ID || got.Steps[i].Operation != want.Steps[i].Operation {
			t.Fatalf("step %d identity mismatch: got=%#v want=%#v", i, got.Steps[i], want.Steps[i])
		}
		switch want.Steps[i].Operation {
		case "resolve-pattern":
			if got.Steps[i].ExpectKind != want.Steps[i].ExpectKind || got.Steps[i].ExpectData != want.Steps[i].ExpectData || got.Steps[i].ExpectQuery != want.Steps[i].ExpectQuery {
				t.Fatalf("step %d pattern expectation mismatch", i)
			}
		case "evaluate-validations":
			if !reflect.DeepEqual(got.Steps[i].ExpectValidations, want.Steps[i].ExpectValidations) {
				t.Fatalf("step %d validation mismatch", i)
			}
		case "observe-synthetic-state":
			if !reflect.DeepEqual(got.Steps[i].ExpectObservation, want.Steps[i].ExpectObservation) || got.Steps[i].ReadIdentity != want.Steps[i].ReadIdentity {
				t.Fatalf("step %d observation mismatch", i)
			}
		}
	}
}

func TestParseMarkdownScenarioTablesIgnoresOtherTables(t *testing.T) {
	md := strings.Join([]string{
		"---",
		"application: order_entry",
		"tei-test-scenario: \"true\"",
		"---",
		"",
		"# Authoring doc",
		"",
		"This table is documentation only.",
		"",
		"| topic | value |",
		"|---|---|",
		"| owner | platform |",
		"",
		"```tei-scenario-md",
		"| step id | operation | pattern | expect | fixture | expected result |",
		"|---|---|---|---|---|---|",
		"| record | resolve-pattern | order_entry.order_record | record:sales.order | - | - |",
		"```",
		"",
	}, "\n")
	got, err := ParseMarkdownScenarioTables(md, "record-only")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Steps) != 1 || got.Steps[0].ID != "record" {
		t.Fatalf("expected one record step, got %#v", got.Steps)
	}
}

func TestParseMarkdownScenarioTablesFailsClosed(t *testing.T) {
	cases := []struct {
		name    string
		md      string
		wantErr string
	}{
		{
			name:    "missing_block",
			md:      "# plain markdown\n",
			wantErr: "no fenced tei-scenario-md block found",
		},
		{
			name:    "missing_application",
			md:      "```tei-scenario-md\n| a | b |\n|---|---|\n```\n",
			wantErr: "markdown scenario missing application",
		},
		{
			name:    "unsupported_operation",
			md:      "```tei-scenario-md\n| step id | operation | pattern | expect | fixture | expected result |\n|---|---|---|---|---|---|\n| s1 | invoke-command | - | - | - | - |\n```\n",
			wantErr: `unsupported operation "invoke-command"`,
		},
		{
			name:    "browse_rows_not_supported",
			md:      "```tei-scenario-md\n| step id | operation | pattern | expect | fixture | expected result |\n|---|---|---|---|---|---|\n| s1 | execute-browse-query | order_entry.recent_orders | browse:sales.order,query:order_entry.recent_orders | fixtures/recent_orders.json | - |\n```\n",
			wantErr: "execute-browse-query row expectations are not supported in Markdown yet",
		},
		{
			name:    "duplicate_step",
			md:      "```tei-scenario-md\n| step id | operation | pattern | expect | fixture | expected result |\n|---|---|---|---|---|---|\n| s1 | resolve-pattern | p | browse:sales.order,query:q | - | - |\n| s1 | resolve-pattern | p | browse:sales.order,query:q | - | - |\n```\n",
			wantErr: `duplicate step id "s1"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseMarkdownScenarioTables(tc.md, "x")
			if err == nil {
				t.Fatal("accepted invalid markdown")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("got %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestParseMarkdownScenarioTablesGeneratedJSONIsValidScenario(t *testing.T) {
	md := strings.Join([]string{
		"---",
		"application: order_entry",
		"tei-test-scenario: \"true\"",
		"---",
		"",
		"```tei-scenario-md",
		"| step id | operation | pattern | expect | fixture | expected result |",
		"|---|---|---|---|---|---|",
		"| browse-pattern | resolve-pattern | order_entry.recent_orders | browse:sales.order,query:order_entry.recent_orders | - | - |",
		"```",
		"",
	}, "\n")
	got, err := ParseMarkdownScenarioTables(md, "browse")
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(b); err != nil {
		t.Fatalf("generated JSON is not a valid Scenario: %v\n%s", err, string(b))
	}
}
