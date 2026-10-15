//go:build tei_studio_composition_boundary

// Standalone #7 qualification. Requires a fixed, unmerged snapshot containing
// #1832 Host and generated #1833 Go source, NOT production runtime authority.
package evidence7_test

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/operations"
	"tei/internal/sourceindex"
	"tei/internal/studiohost"
	staticservices "tei/tools/reference-studio-static-composition/generated"
)

const (
	allowedID = "workspace-1"
	deniedCode = "WORKSPACE_NOT_ALLOWED"
)

func fixtureHost(t *testing.T) (*studiohost.ReadOnly, string) {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "reference", "workspace"))
	if err != nil { t.Fatal(err) }
	h, err := studiohost.NewReadOnly([]string{root})
	if err != nil { t.Fatal(err) }
	return h, root
}

func TestGeneratedServicesPreserveOperationSemanticsButNotHostAuthority(t *testing.T) {
	h, root := fixtureHost(t)
	svcs := staticservices.Services(root)
	if len(svcs) != 2 { t.Fatalf("expected 2 generated services, got %d", len(svcs)) }

	type cataloger interface { Catalog(string) (operations.ModelCatalog, error) }
	type validator interface { Validate() (operations.WorkspaceValidation, error) }
	var catalogerFound cataloger
	var validatorFound validator
	for _, item := range svcs {
		if c, ok := item.(cataloger); ok {
			if catalogerFound != nil { t.Fatal("duplicate Catalog service") }
			catalogerFound = c
		}
		if v, ok := item.(validator); ok {
			if validatorFound != nil { t.Fatal("duplicate Validate service") }
			validatorFound = v
		}
	}
	if catalogerFound == nil || validatorFound == nil { t.Fatal("missing typed generated operations") }

	fromGeneratedCatalog, err := catalogerFound.Catalog("")
	if err != nil { t.Fatal(err) }
	fromHostCatalog := h.CatalogWorkspace(allowedID, "")
	if !fromHostCatalog.OK || fromHostCatalog.Value == nil || fromHostCatalog.Failure != nil {
		t.Fatalf("Host Catalog response contract: %+v", fromHostCatalog)
	}
	if !reflect.DeepEqual(fromGeneratedCatalog, *fromHostCatalog.Value) {
		t.Fatal("generated Catalog and allowed Host operation differ")
	}
	if fromGeneratedCatalog.Effect != "read-only" { t.Fatalf("unexpected generated catalog effect %q", fromGeneratedCatalog.Effect) }

	fromGeneratedValidation, err := validatorFound.Validate()
	if err != nil { t.Fatal(err) }
	fromHostValidation := h.ValidateWorkspace(allowedID)
	if !fromHostValidation.OK || fromHostValidation.Value == nil || fromHostValidation.Failure != nil {
		t.Fatalf("Host Validate response contract: %+v", fromHostValidation)
	}
	if !reflect.DeepEqual(fromGeneratedValidation, *fromHostValidation.Value) {
		t.Fatal("generated Validate and allowed Host validation differ")
	}
	// NOTE: The generated services accept a trusted root, not a Workspace ID.
	// Their deep equality does NOT entitle a caller to expose them as Wails RPCs.
}

func TestHostOnlyAllowsConfiguredWorkspaceAndDoesNotLeakPathInFailure(t *testing.T) {
	h, root := fixtureHost(t)
	refs := h.ListWorkspaces()
	if !refs.OK || len(refs.Value) != 1 || refs.Value[0].ID != allowedID {
		t.Fatalf("Host allowlist %+v", refs)
	}
	if strings.Contains(refs.Value[0].Label, root) { t.Fatal("Host label exposes configured absolute root") }
	for _, bad := range []string{"", "workspace-2", "../workspace", root, " workspace-1", "workspace-1/../workspace-1"} {
		for op, response := range map[string]any{
			"catalog": h.CatalogWorkspace(bad, ""),
			"validate": h.ValidateWorkspace(bad),
			"inspect": h.InspectSubject(bad, "semantic-data", "sales.order"),
		} {
			bytes, err := json.Marshal(response)
			if err != nil { t.Fatal(err) }
			var envelope map[string]json.RawMessage
			if err := json.Unmarshal(bytes, &envelope); err != nil { t.Fatal(err) }
			if string(envelope["ok"]) != "false" { t.Fatalf("%s accepted %q: %s", op, bad, bytes) }
			if _, ok := envelope["value"]; ok { t.Fatalf("%s leaked value on %q: %s", op, bad, bytes) }
			var failure struct { Code, Message string }
			if err := json.Unmarshal(envelope["failure"], &failure); err != nil { t.Fatalf("%s missing failure on %q: %v", op, bad, err) }
			if failure.Code != deniedCode || failure.Message == "" || strings.Contains(failure.Message, root) {
				t.Fatalf("%s incorrect or path-leaking failure on %q: %+v", op, bad, failure)
			}
		}
	}
	catalog := h.CatalogWorkspace(allowedID, "")
	if !catalog.OK || catalog.Value == nil || catalog.Value.Effect != "read-only" { t.Fatalf("catalog not read-only: %+v", catalog) }
	if len(catalog.Value.Entries) > 0 {
		s := catalog.Value.Entries[0].Subject
		var _ sourceindex.Subject = s
		insp := h.InspectSubject(allowedID, s.Kind, s.ID)
		if !insp.OK || insp.Value == nil || insp.Value.Effect != "read-only" { t.Fatalf("inspection not read-only: %+v", insp) }
	}
}

func TestBridgeDoesNotPublishGeneratedServiceMethods(t *testing.T) {
	path := filepath.Join("..", "..", "cmd", "tei-studio", "main.go")
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil { t.Fatal(err) }
	names := map[string]bool{}
	for _, node := range file.Decls {
		fn, ok := node.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || len(fn.Recv.List) != 1 { continue }
		typ, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
		if !ok { continue }
		id, ok := typ.X.(*ast.Ident)
		if !ok || id.Name != "StudioBridge" { continue }
		names[fn.Name.Name] = true
	}
	expected := map[string]bool{
		"ListWorkspaces": true, "CatalogWorkspace": true,
		"ValidateWorkspace": true, "InspectSubject": true,
	}
	if !reflect.DeepEqual(names, expected) { t.Fatalf("Wails Bridge exported method set changed: %v", names) }
	b, err := os.ReadFile(path)
	if err != nil { t.Fatal(err) }
	src := string(b)
	if !strings.Contains(src, "studiohost.NewReadOnly(paths)") ||
		!strings.Contains(src, "application.NewService(&StudioBridge{host: host})") ||
		strings.Contains(src, "staticservices.Services(") ||
		strings.Contains(src, "application.NewService(&staticservices.") {
		t.Fatal("Host composition must preserve the single allowlist-enforcing StudioBridge")
	}
	for _, method := range []string{"ListWorkspaces", "CatalogWorkspace", "ValidateWorkspace", "InspectSubject"} {
		if !strings.Contains(src, "return b.host."+method+"(") {
			t.Fatalf("StudioBridge.%s no longer delegates to allowed Host", method)
		}
	}
}
