package standard

import "testing"

// Two diagnostics answer "there is nothing to convert or parse". A code that no
// test names is a rule nothing proves can fire, so each is produced here (#4032).
func TestEmptyInputCodesAreProduced(t *testing.T) {
	t.Run("nil query", func(t *testing.T) {
		sql, diags := StandardToSQL(nil)
		if sql != "" {
			t.Fatalf("sql=%q", sql)
		}
		assertCode(t, diags, "EMPTY_QUERY")
	})

	t.Run("blank SQL", func(t *testing.T) {
		query, diags := ParseSQLToStandard("   \n\t", "sqlite")
		if query != nil {
			t.Fatalf("query=%+v", query)
		}
		assertCode(t, diags, "EMPTY_SQL")
	})
}

func assertCode(t *testing.T, diags []Diagnostic, want string) {
	t.Helper()
	for _, diagnostic := range diags {
		if diagnostic.Code == want {
			return
		}
	}
	var codes []string
	for _, diagnostic := range diags {
		codes = append(codes, diagnostic.Code)
	}
	t.Fatalf("expected %s, got %v", want, codes)
}
