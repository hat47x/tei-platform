package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/studiohost"
)

func writeXeadFixture(t *testing.T, root, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestStudioXeadObserverReturnsBoundedProjectionAndRedactsSourceValues(t *testing.T) {
	root := t.TempDir()
	writeXeadFixture(t, root, "model.xead", `<System DatabasePassword="synthetic-system-secret"><Table ID="T1" Name="Order"><TableField ID="F1" Name="OrderID" DataTypeID="D1" Default="synthetic-default-secret"/><Instance Text="synthetic-instance-secret"/></Table></System>`)
	host, err := newStudioHost([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	got := host.ObserveXead("workspace-1", "model.xead")
	if !got.OK || got.Value == nil || got.Value.Effect != "read-only" {
		t.Fatalf("observation = %+v", got)
	}
	if len(got.Value.Observation.Tables) != 1 || got.Value.Observation.Tables[0].Source.ID != "T1" || got.Value.Observation.Tables[0].Name != "Order" {
		t.Fatalf("tables = %+v", got.Value.Observation.Tables)
	}
	fields := got.Value.Observation.Tables[0].Fields
	if len(fields) != 1 || fields[0].Source.ID != "F1" || fields[0].DataType == nil || fields[0].DataType.ID != "D1" {
		t.Fatalf("fields = %+v", fields)
	}
	if len(got.Value.AuthorityLimits) == 0 {
		t.Fatal("authority limits missing")
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"synthetic-system-secret", "synthetic-default-secret", "synthetic-instance-secret"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("response leaked %q", secret)
		}
	}
}

func TestStudioXeadObserverUsesStableFailuresAndReadBudget(t *testing.T) {
	root := t.TempDir()
	writeXeadFixture(t, root, "malformed.xead", `<System Secret="synthetic-parse-secret"><Table>`)
	writeXeadFixture(t, root, "wrong-root.xead", `<PrivateRoot Secret="synthetic-private-value"/>`)
	if err := os.WriteFile(filepath.Join(root, "large.xead"), []byte(strings.Repeat("x", int(studiohost.XeadObservationInputLimitBytes+1))), 0600); err != nil {
		t.Fatal(err)
	}
	host, err := newStudioHost([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ path, code string }{
		{"../secret.xead", studiohost.XeadSourceReadFailed}, {filepath.Join(root, "malformed.xead"), studiohost.XeadSourceReadFailed},
		{"missing.xead", studiohost.XeadSourceReadFailed}, {"large.xead", studiohost.XeadSourceTooLarge},
		{"malformed.xead", xeadParseFailed}, {"wrong-root.xead", xeadProjectionFailed},
		{"model.xml", studiohost.XeadUnsupportedFormat},
	} {
		t.Run(tc.path, func(t *testing.T) {
			got := host.ObserveXead("workspace-1", tc.path)
			if got.OK || got.Failure == nil || got.Failure.Code != tc.code {
				t.Fatalf("response=%+v want stable %s", got, tc.code)
			}
			if strings.Contains(got.Failure.Message, root) || strings.Contains(got.Failure.Message, "synthetic-") {
				t.Fatalf("failure leaked source data: %+v", got.Failure)
			}
		})
	}
}

func TestStudioXeadObserverEnforcesParserBudgets(t *testing.T) {
	root := t.TempDir()
	deep := `<System>` + strings.Repeat(`<Nested>`, 128) + strings.Repeat(`</Nested>`, 128) + `</System>`
	writeXeadFixture(t, root, "deep.xead", deep)
	writeXeadFixture(t, root, "elements.xead", `<System>`+strings.Repeat(`<Unknown/>`, 100000)+`</System>`)
	host, err := newStudioHost([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	for _, sourcePath := range []string{"deep.xead", "elements.xead"} {
		got := host.ObserveXead("workspace-1", sourcePath)
		if got.OK || got.Failure == nil || got.Failure.Code != xeadParseFailed {
			t.Fatalf("over-budget source %q response=%+v", sourcePath, got)
		}
	}
}
