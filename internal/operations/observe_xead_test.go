package operations

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeObserveXeadFixture(t *testing.T, root, relativePath, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestObserveXeadSourceReturnsReadOnlyBoundedProjection(t *testing.T) {
	root := t.TempDir()
	writeObserveXeadFixture(t, root, "model.xead", `<System DatabasePassword="synthetic-system-secret"><Table ID="T1" Name="Order"><TableField ID="F1" Name="OrderID" DataTypeID="D1" Default="synthetic-default-secret"/><Instance Text="synthetic-instance-secret"/></Table></System>`)

	got, err := ObserveXeadSource(root, "model.xead")
	if err != nil {
		t.Fatal(err)
	}
	if got.SchemaVersion != "tei.operations.xead-observation/v0" || got.Effect != "read-only" {
		t.Fatalf("operation envelope = %+v", got)
	}
	if len(got.Observation.Tables) != 1 || got.Observation.Tables[0].Source.ID != "T1" || got.Observation.Tables[0].Name != "Order" {
		t.Fatalf("table projection = %+v", got.Observation.Tables)
	}
	fields := got.Observation.Tables[0].Fields
	if len(fields) != 1 || fields[0].Source.ID != "F1" || fields[0].DataType == nil || fields[0].DataType.ID != "D1" {
		t.Fatalf("field projection = %+v", fields)
	}
	if len(got.AuthorityLimits) == 0 {
		t.Fatal("authority limits missing")
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	output := string(encoded)
	for _, forbidden := range []string{
		"synthetic-system-secret", "synthetic-default-secret", "synthetic-instance-secret",
		"worldFingerprint", "sourceIndexFingerprint", `"world"`,
	} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("operation output contained forbidden value or authority surface %q", forbidden)
		}
	}
	writeObserveXeadFixture(t, root, "uppercase.XEAD", `<System/>`)
	if _, err := ObserveXeadSource(root, "uppercase.XEAD"); err != nil {
		t.Fatalf("upper-case .XEAD extension was not accepted: %v", err)
	}
}

func TestObserveXeadSourceUsesStaticErrorsForPathsLimitsAndXML(t *testing.T) {
	root := t.TempDir()
	writeObserveXeadFixture(t, root, "malformed.xead", `<System DatabasePassword="synthetic-parse-secret"><Table>`)
	tooLargePath := filepath.Join(root, "large.xead")
	if err := os.WriteFile(tooLargePath, []byte(strings.Repeat("x", int(XeadObservationInputLimitBytes+1))), 0600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name, path, content, code string
	}{
		{name: "traversal", path: "../synthetic-secret.xead", code: ObserveXeadSourceReadFailed},
		{name: "absolute", path: filepath.Join(root, "malformed.xead"), code: ObserveXeadSourceReadFailed},
		{name: "missing", path: "missing.xead", code: ObserveXeadSourceReadFailed},
		{name: "too large", path: "large.xead", code: ObserveXeadSourceTooLarge},
		{name: "malformed XML", path: "malformed.xead", code: ObserveXeadParseFailed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ObserveXeadSource(root, test.path)
			var operationErr *XeadObservationError
			if !errors.As(err, &operationErr) || operationErr.Code != test.code {
				t.Fatalf("error = %v; want static code %s", err, test.code)
			}
			if err.Error() != test.code || strings.Contains(err.Error(), "synthetic-") || strings.Contains(err.Error(), root) {
				t.Fatalf("failure included source details: %v", err)
			}
		})
	}
}

func TestObserveXeadSourceRejectsUnsupportedRootWithoutEchoingXML(t *testing.T) {
	root := t.TempDir()
	writeObserveXeadFixture(t, root, "wrong-root.xead", `<PrivateRoot Secret="synthetic-private-value"/>`)
	_, err := ObserveXeadSource(root, "wrong-root.xead")
	var operationErr *XeadObservationError
	if !errors.As(err, &operationErr) || operationErr.Code != ObserveXeadProjectionFailed {
		t.Fatalf("error = %v; want static projection error", err)
	}
	if strings.Contains(err.Error(), "synthetic-private-value") {
		t.Fatalf("projection failure leaked source value: %v", err)
	}
}

func TestObserveXeadSourceRejectsOtherFormatsBeforeReading(t *testing.T) {
	root := t.TempDir()
	for _, relativePath := range []string{"model.xeaf", "model.xml", "model"} {
		_, err := ObserveXeadSource(root, relativePath)
		var operationErr *XeadObservationError
		if !errors.As(err, &operationErr) || operationErr.Code != ObserveXeadUnsupportedFormat {
			t.Fatalf("path %q error = %v; want static unsupported-format code", relativePath, err)
		}
		if err.Error() != ObserveXeadUnsupportedFormat || strings.Contains(err.Error(), relativePath) {
			t.Fatalf("unsupported-format error leaked path: %v", err)
		}
	}
}

func TestObserveXeadSourceRejectsParserDepthAndElementBudgets(t *testing.T) {
	root := t.TempDir()
	deep := `<System>` + strings.Repeat(`<Nested>`, 128) + strings.Repeat(`</Nested>`, 128) + `</System>`
	writeObserveXeadFixture(t, root, "deep.xead", deep)
	tooManyElements := `<System>` + strings.Repeat(`<Unknown/>`, 100000) + `</System>`
	writeObserveXeadFixture(t, root, "elements.xead", tooManyElements)

	for _, path := range []string{"deep.xead", "elements.xead"} {
		_, err := ObserveXeadSource(root, path)
		var operationErr *XeadObservationError
		if !errors.As(err, &operationErr) || operationErr.Code != ObserveXeadParseFailed {
			t.Fatalf("over-budget source %q error = %v; want static parse-failure code", path, err)
		}
		if strings.Contains(err.Error(), path) {
			t.Fatalf("parser budget failure leaked source path: %v", err)
		}
	}
}
