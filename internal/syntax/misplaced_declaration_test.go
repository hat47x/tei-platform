package syntax

import (
	"strings"
	"testing"
)

// A declaration keyword written in the wrong place is a placement mistake, not a
// typo. The parser now names the declaration instead of reporting an unknown
// token, in three places: after a declaration ends, inside a body, and inside a
// semantic namespace (#4169). Every case below is a token the parser itself
// reads somewhere, so the phrase it promises is one the parser really accepts.
func TestMisplacedDeclarationNamesTheDeclaration(t *testing.T) {
	cases := []struct {
		name  string
		parse func(string) error
		src   string
		want  string
	}{
		{
			name:  "after an application declaration",
			parse: parseApplicationForTest,
			src:   "application order_entry {\n}\n\nmigration 20261111_001 {\n    database sales_core\n    recovery forward-fix\n}\n",
			want:  `"migration" starts a migration, which cannot be declared after the end of this declaration; declare it under databases/<database>/migrations, for example <database>/migrations/<id>.tei`,
		},
		{
			name:  "after a database declaration",
			parse: parseDatabaseForTest,
			src:   "database sales_core {\n    kind relational\n    ownership managed-here\n}\n\ndatabase-release sales_core 5.0.0 {\n    data-readiness ready\n}\n",
			want:  `"database-release" starts a database release, which cannot be declared after the end of this declaration; declare it under databases/<database>/releases, for example <database>/releases/<version>.tei`,
		},
		{
			name:  "inside the application body",
			parse: parseApplicationForTest,
			src:   "application order_entry {\n    realize Order\n}\n",
			want:  `"realize" starts a realization, which cannot be declared in the application body; declare it under applications/<application>/realizations, for example <application>/realizations/<capability>.tei`,
		},
		{
			name:  "inside the database body",
			parse: parseDatabaseForTest,
			src:   "database sales_core {\n    kind relational\n    migration 20261111_001\n}\n",
			want:  `"migration" starts a migration, which cannot be declared in the database body; declare it under databases/<database>/migrations, for example <database>/migrations/<id>.tei`,
		},
		{
			name:  "inside a semantic namespace",
			parse: parseSemanticFileForTest,
			src:   "namespace sales\n\nmigration 20261111_001 {\n}\n",
			want:  `"migration" starts a migration, which cannot be declared in a semantic namespace; declare it under databases/<database>/migrations, for example <database>/migrations/<id>.tei`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.parse(c.src)
			if err == nil {
				t.Fatal("expected a parse error")
			}
			message, ok := ErrorMessage(err)
			if !ok {
				t.Fatal("error has no message")
			}
			if !strings.Contains(message, c.want) {
				t.Errorf("message does not name the declaration: %s", message)
			}
			if !strings.Contains(message, "declare it under") {
				t.Errorf("message does not say where it belongs: %s", message)
			}
		})
	}
}

// A token that is not a declaration keyword must keep the narrower message, so
// the new branch cannot hide a genuine typo behind a placement hint.
func TestUnknownTokenKeepsItsNarrowerMessage(t *testing.T) {
	cases := []struct {
		parse func(string) error
		src   string
		want  string
	}{
		{parseApplicationForTest, "application order_entry {\n    frobnicate\n}\n", "expected database, uses, or binds"},
		{parseDatabaseForTest, "database sales_core {\n    kind relational\n    frobnicate\n}\n", "expected kind, ownership, or uses"},
		{parseSemanticFileForTest, "namespace sales\n\nfrobnicate\n", "expected type, data, capability, validation, calendar, period, effect-rule, or run-plan"},
		{parseDatabaseForTest, "database sales_core {\n    kind relational\n}\n\nfrobnicate\n", `unexpected trailing token "frobnicate"`},
	}
	for _, c := range cases {
		err := c.parse(c.src)
		if err == nil {
			t.Fatalf("%q: expected a parse error", c.src)
		}
		message, ok := ErrorMessage(err)
		if !ok || message != c.want {
			t.Errorf("%q: message=%q want=%q", c.src, message, c.want)
		}
	}
}

func parseApplicationForTest(src string) error {
	_, err := NewParser(src).ParseApplication()
	return err
}

func parseDatabaseForTest(src string) error {
	_, err := NewParser(src).ParseDatabase()
	return err
}

func parseSemanticFileForTest(src string) error {
	_, err := NewParser(src).ParseSemanticFile()
	return err
}
