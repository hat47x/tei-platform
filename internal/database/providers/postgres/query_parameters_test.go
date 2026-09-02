package postgres

import (
	"reflect"
	"strings"
	"testing"
)

func TestBindNamedParametersPreservesSQLLexicalBoundaries(t *testing.T) {
	source := `SELECT
    ':literal' AS literal_value,
    "column:identifier",
    payload::text,
    $$:dollar_body$$,
    $tag$:tagged_body$tag$
FROM sample
WHERE created_at >= :from
  AND status = :status
  AND updated_at >= :from
  -- :line_comment
  AND note <> 'escaped '' :still_literal'
  /* :block_comment /* :nested_comment */ :still_comment */
  AND code = :code`

	bound, err := BindNamedParameters(source, map[string]any{
		"from":   "2026-09-01T00:00:00Z",
		"status": "OPEN",
		"code":   "A-01",
	})
	if err != nil {
		t.Fatalf("named parameter bindingに失敗しました: %v", err)
	}

	wantFragments := []string{
		"':literal'",
		`"column:identifier"`,
		"payload::text",
		"$$:dollar_body$$",
		"$tag$:tagged_body$tag$",
		"created_at >= $1",
		"status = $2",
		"updated_at >= $1",
		"-- :line_comment",
		"/* :block_comment /* :nested_comment */ :still_comment */",
		"code = $3",
	}
	for _, fragment := range wantFragments {
		if !strings.Contains(bound.SQL, fragment) {
			t.Fatalf("変換後SQLに期待した断片がありません: %q\nSQL:\n%s", fragment, bound.SQL)
		}
	}

	if want := []string{"from", "status", "code"}; !reflect.DeepEqual(bound.ParameterNames, want) {
		t.Fatalf("ParameterNames=%#v want=%#v", bound.ParameterNames, want)
	}
	if want := []any{"2026-09-01T00:00:00Z", "OPEN", "A-01"}; !reflect.DeepEqual(bound.Args, want) {
		t.Fatalf("Args=%#v want=%#v", bound.Args, want)
	}
}

func TestBindNamedParametersDistinguishesPrefixNames(t *testing.T) {
	bound, err := BindNamedParameters(
		"SELECT :id, :id2, :id",
		map[string]any{"id": int64(10), "id2": int64(20)},
	)
	if err != nil {
		t.Fatalf("named parameter bindingに失敗しました: %v", err)
	}
	if bound.SQL != "SELECT $1, $2, $1" {
		t.Fatalf("SQL=%q", bound.SQL)
	}
	if want := []any{int64(10), int64(20)}; !reflect.DeepEqual(bound.Args, want) {
		t.Fatalf("Args=%#v want=%#v", bound.Args, want)
	}
}

func TestBindNamedParametersRejectsMissingValue(t *testing.T) {
	_, err := BindNamedParameters("SELECT :from", map[string]any{})
	if err == nil || !strings.Contains(err.Error(), `parameter "from"に値がありません`) {
		t.Fatalf("不足parameterを明示的に拒否していません: %v", err)
	}
}

func TestBindNamedParametersRejectsUnusedValue(t *testing.T) {
	_, err := BindNamedParameters("SELECT :from", map[string]any{
		"from": "2026-09-01",
		"typo": "unused",
	})
	if err == nil || !strings.Contains(err.Error(), "使用されていないparameterがあります: typo") {
		t.Fatalf("未使用parameterを明示的に拒否していません: %v", err)
	}
}

func TestBindNamedParametersReportsUnusedValuesInStableOrder(t *testing.T) {
	_, err := BindNamedParameters("SELECT 1", map[string]any{
		"zeta":  1,
		"alpha": 2,
	})
	if err == nil || !strings.Contains(err.Error(), "alpha, zeta") {
		t.Fatalf("未使用parameter名が安定した順序ではありません: %v", err)
	}
}

func TestBindNamedParametersRejectsExistingPositionalPlaceholder(t *testing.T) {
	_, err := BindNamedParameters("SELECT $1, :from", map[string]any{"from": 1})
	if err == nil || !strings.Contains(err.Error(), "既存のpositional placeholder") {
		t.Fatalf("positional placeholderとの混在を拒否していません: %v", err)
	}
}

func TestBindNamedParametersRejectsUnclosedLexicalRegions(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{name: "string", source: "SELECT ':from", want: "文字列リテラルが閉じていません"},
		{name: "identifier", source: `SELECT "column:from`, want: "quoted identifierが閉じていません"},
		{name: "comment", source: "SELECT 1 /* :from", want: "block commentが閉じていません"},
		{name: "dollar", source: "SELECT $body$:from", want: "dollar-quoted body $body$が閉じていません"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BindNamedParameters(tt.source, map[string]any{"from": 1})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("閉じていない字句領域を拒否していません: %v", err)
			}
		})
	}
}

func TestBindNamedParametersDoesNotRecordRuntimeValuesInNames(t *testing.T) {
	secret := "runtime-secret-value"
	bound, err := BindNamedParameters("SELECT :credential", map[string]any{"credential": secret})
	if err != nil {
		t.Fatalf("named parameter bindingに失敗しました: %v", err)
	}
	if strings.Contains(strings.Join(bound.ParameterNames, ","), secret) {
		t.Fatalf("ParameterNamesに実行時の値が混入しています: %#v", bound.ParameterNames)
	}
	if strings.Contains(bound.SQL, secret) {
		t.Fatalf("SQL本文へ実行時の値を埋め込んでいます: %s", bound.SQL)
	}
}
