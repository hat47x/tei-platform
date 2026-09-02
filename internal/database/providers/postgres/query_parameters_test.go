package postgres

import (
	"reflect"
	"strings"
	"testing"
)

func TestBindNamedParameters(t *testing.T) {
	t.Run("named parameterをPostgreSQL placeholderへ束縛する", func(t *testing.T) {
		bound, err := BindNamedParameters(
			"SELECT * FROM orders WHERE ordered_at >= :from AND status = :status OR reopened_at >= :from",
			map[string]any{"from": "2026-09-01", "status": "OPEN"},
		)
		if err != nil {
			t.Fatalf("parameterを束縛できません: %v", err)
		}
		wantSQL := "SELECT * FROM orders WHERE ordered_at >= $1 AND status = $2 OR reopened_at >= $1"
		if bound.SQL != wantSQL {
			t.Fatalf("SQL=%q want=%q", bound.SQL, wantSQL)
		}
		if !reflect.DeepEqual(bound.ParameterNames, []string{"from", "status"}) {
			t.Fatalf("parameter names=%v", bound.ParameterNames)
		}
		if !reflect.DeepEqual(bound.Arguments, []any{"2026-09-01", "OPEN"}) {
			t.Fatalf("arguments=%#v", bound.Arguments)
		}
	})

	t.Run("SQL構文上のcolonをparameterと誤認しない", func(t *testing.T) {
		source := strings.Join([]string{
			"SELECT ':from' AS literal, \"column:from\" AS quoted, now()::date AS casted, :from AS actual",
			"-- :comment_parameter",
			"/* :block_parameter /* :nested_parameter */ */",
			"WHERE note = $$:dollar_parameter$$",
			"AND tagged = $tag$:tag_parameter$tag$",
		}, "\n")

		bound, err := BindNamedParameters(source, map[string]any{"from": "value"})
		if err != nil {
			t.Fatalf("parameterを束縛できません: %v", err)
		}
		if strings.Count(bound.SQL, "$1") != 1 {
			t.Fatalf("実parameter以外まで置換しています: %s", bound.SQL)
		}
		for _, preserved := range []string{
			"':from'",
			"\"column:from\"",
			"now()::date",
			"-- :comment_parameter",
			"/* :block_parameter /* :nested_parameter */ */",
			"$$:dollar_parameter$$",
			"$tag$:tag_parameter$tag$",
		} {
			if !strings.Contains(bound.SQL, preserved) {
				t.Fatalf("SQL構文を保持できていません: %q in %s", preserved, bound.SQL)
			}
		}
	})

	t.Run("parameter名のprefixを区別する", func(t *testing.T) {
		bound, err := BindNamedParameters(
			"SELECT :from, :from_date, :from",
			map[string]any{"from": 1, "from_date": 2},
		)
		if err != nil {
			t.Fatalf("parameterを束縛できません: %v", err)
		}
		if bound.SQL != "SELECT $1, $2, $1" {
			t.Fatalf("SQL=%q", bound.SQL)
		}
	})

	t.Run("Queryで必要なparameterがなければ失敗する", func(t *testing.T) {
		_, err := BindNamedParameters("SELECT :from", map[string]any{})
		if err == nil || !strings.Contains(err.Error(), `"from"`) {
			t.Fatalf("missing parameter error=%v", err)
		}
	})

	t.Run("Queryで使わないparameterを黙って無視しない", func(t *testing.T) {
		_, err := BindNamedParameters("SELECT :from", map[string]any{"from": 1, "unused": 2})
		if err == nil || !strings.Contains(err.Error(), "unused") {
			t.Fatalf("unused parameter error=%v", err)
		}
	})

	for _, tt := range []struct {
		name   string
		source string
	}{
		{name: "文字列リテラル", source: "SELECT ':from"},
		{name: "quoted identifier", source: "SELECT \"column:from"},
		{name: "block comment", source: "SELECT 1 /* :from"},
		{name: "dollar-quoted文字列", source: "SELECT $tag$:from"},
	} {
		t.Run(tt.name+"が閉じられていなければ失敗する", func(t *testing.T) {
			if _, err := BindNamedParameters(tt.source, nil); err == nil {
				t.Fatalf("閉じられていない%sを受理しています", tt.name)
			}
		})
	}
}
