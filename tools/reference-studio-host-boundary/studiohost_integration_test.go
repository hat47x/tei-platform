//go:build tei_studio_host_integration

// This test imports the unmerged #1832 Studio Host through a verification
// snapshot. The default #1827 branch remains independently testable against
// main until that dependency is integrated.
package hostboundary

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"

	"tei/internal/studiohost"
)

func referenceAndEmptyWorkspace(t *testing.T) (*studiohost.ReadOnly, string, string) {
	t.Helper()
	// Paths are Host-owned configuration, not browser input.
	reference := filepath.Join("..", "..", "reference", "workspace")
	empty := t.TempDir()
	host, err := studiohost.NewReadOnly([]string{reference, empty})
	if err != nil {
		t.Fatal(err)
	}
	return host, reference, empty
}

// This characterizes the current one-user, fixed-allowlist semantics of
// #1832; these goroutines are NOT browser sessions or Wails event consumers.
func TestStudioReadOnlyTwoCallersHaveNoSharedWorkspaceSelection(t *testing.T) {
	host, _, _ := referenceAndEmptyWorkspace(t)
	items := host.ListWorkspaces()
	if !items.OK || len(items.Value) != 2 ||
		items.Value[0].ID != "workspace-1" || items.Value[1].ID != "workspace-2" {
		t.Fatalf("unexpected fixed allowlist: %+v", items)
	}
	items.Value[0].ID = "tampered-by-client"
	if got := host.ListWorkspaces().Value[0].ID; got != "workspace-1" {
		t.Fatalf("client modified allowlist: %q", got)
	}

	const requestsPerCaller = 4
	failures := make(chan error, 2*requestsPerCaller)
	var wg sync.WaitGroup
	for caller := 0; caller < 2; caller++ {
		wg.Add(1)
		go func(caller int) {
			defer wg.Done()
			for n := 0; n < requestsPerCaller; n++ {
				if caller == 0 {
					got := host.ValidateWorkspace("workspace-1")
					if !got.OK || got.Failure != nil || got.Value == nil {
						failures <- fmt.Errorf("reference caller %d: %+v", n, got)
					}
				} else {
					got := host.ValidateWorkspace("workspace-2")
					if got.OK || got.Failure == nil ||
						got.Failure.Code != "WORKSPACE_LOAD_FAILED" ||
						got.Failure.Stage != "load" {
						failures <- fmt.Errorf("empty Workspace caller %d: %+v", n, got)
					}
				}
			}
		}(caller)
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}

	// Critical LIMITATION of the initial PoC: both callers reach the SAME
	// startup allowlist. The bridge cannot infer a client/user authority
	// from a Workspace ID. A shared multiuser Wails Server Build would
	// need host-side authenticated, request-scoped resource restrictions.
	fromCallerB := host.CatalogWorkspace("workspace-1", "")
	if !fromCallerB.OK || fromCallerB.Failure != nil || fromCallerB.Value == nil ||
		fromCallerB.Value.Effect != "read-only" {
		t.Fatalf("allowlisted Workspace inaccessible from another caller: %+v", fromCallerB)
	}
}

func TestStudioReadOnlyRejectsPathShapedWorkspaceIDsWithoutHostRootDisclosure(t *testing.T) {
	host, _, emptyRoot := referenceAndEmptyWorkspace(t)
	resolvedEmpty, err := filepath.EvalSymlinks(emptyRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, attemptedID := range []string{
		"", "workspace-0", "workspace-3", "WORKSPACE-1",
		"../workspace-1", "../../reference/workspace",
		"workspace-1/../workspace-2", "workspace-1/../../outside",
		"/etc/passwd", "..\\workspace-1", "workspace-1\x00",
	} {
		t.Run(fmt.Sprintf("%q", attemptedID), func(t *testing.T) {
			validation := host.ValidateWorkspace(attemptedID)
			catalog := host.CatalogWorkspace(attemptedID, "")
			inspection := host.InspectSubject(attemptedID, "semantic", "any")
			for _, failure := range []*studiohost.Failure{validation.Failure, catalog.Failure, inspection.Failure} {
				if failure == nil || failure.Code != "WORKSPACE_NOT_ALLOWED" {
					t.Fatalf("untrusted Workspace ID %q admitted: %+v", attemptedID, failure)
				}
				if strings.Contains(failure.Message, resolvedEmpty) {
					t.Fatalf("host-local root disclosed in failure: %+v", failure)
				}
			}
			if validation.OK || catalog.OK || inspection.OK {
				t.Fatalf("untrusted Workspace ID %q was accepted", attemptedID)
			}
		})
	}
}

// A change of the public service surface needs explicit requalification:
// Wails renderer arguments must not become arbitrary filesystem or command
// operations. This does NOT prove the safety of the Workspace loader itself.
func TestStudioReadOnlyExportsOnlyFourReadMethods(t *testing.T) {
	host, _, _ := referenceAndEmptyWorkspace(t)
	typ := reflect.TypeOf(host)
	actual := make([]string, 0, typ.NumMethod())
	for i := 0; i < typ.NumMethod(); i++ {
		actual = append(actual, typ.Method(i).Name)
	}
	expected := []string{
		"CatalogWorkspace", "InspectSubject", "ListWorkspaces", "ValidateWorkspace",
	}
	if !slices.Equal(actual, expected) {
		t.Fatalf("Studio Host exported method surface changed: %v (expected %v)", actual, expected)
	}
}

// Test failure bodies and the public inventory: a Host-owned absolute root
// must not be rendered as an incidental implementation detail. The permitted
// Workspace label is intentionally visible; labels themselves should not be
// considered confidential by a caller provisioning the Host.
func TestStudioReadOnlyResponsesDoNotExposeHostRoot(t *testing.T) {
	privateParent := filepath.Join(t.TempDir(), "HOST_PRIVATE_ROOT_SENTINEL")
	root := filepath.Join(privateParent, "public-workspace")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	host, err := studiohost.NewReadOnly([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	physical, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	// Unknown IDs are denied before any Source read and do not cause
	// the root or the directory-structure sentinel to be disclosed.
	results := []struct {
		name string
		data any
	}{
		{"list", host.ListWorkspaces()},
		{"unknown-catalog", host.CatalogWorkspace("workspace-99", "")},
		{"unknown-validate", host.ValidateWorkspace("workspace-99")},
		{"unknown-inspect", host.InspectSubject("workspace-99", "semantic", "x")},
		{"allowed-but-empty-catalog", host.CatalogWorkspace("workspace-1", "")},
		{"allowed-but-empty-validate", host.ValidateWorkspace("workspace-1")},
		{"allowed-but-empty-inspect", host.InspectSubject("workspace-1", "semantic", "x")},
	}
	for _, result := range results {
		t.Run(result.name, func(t *testing.T) {
			raw, err := json.Marshal(result.data)
			if err != nil {
				t.Fatal(err)
			}
			for _, leak := range []string{"HOST_PRIVATE_ROOT_SENTINEL", physical, filepath.ToSlash(physical)} {
				if strings.Contains(string(raw), leak) {
					t.Fatalf("Host private root exposed in %s response: %s", result.name, raw)
				}
			}
		})
	}
	list := host.ListWorkspaces()
	if !list.OK || len(list.Value) != 1 ||
		list.Value[0].Label != "public-workspace" || list.Value[0].ID != "workspace-1" {
		t.Fatalf("unexpected sanitized Workspace inventory: %+v", list)
	}
	if got := host.ValidateWorkspace(physical); got.OK || got.Failure == nil || got.Failure.Code != "WORKSPACE_NOT_ALLOWED" {
		t.Fatalf("OS path accepted as Workspace ID: %+v", got)
	}
}

// Parse the Wails entry point itself (not only the Go-neutral ReadOnly
// type). Detect accidentally exported mutators on StudioBridge before any
// Desktop WebView is available. This is a source/API contract check, not a
// live Wails-IPC or privilege-escalation test.
func TestWailsBridgePublishesOnlyFourReadOnlyMethods(t *testing.T) {
	const path = "../../cmd/tei-studio/main.go"
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("extract #1832 exact main.go into the test snapshot: %v", err)
	}
	var methods []string
	bridgeFound := false
	for _, decl := range file.Decls {
		switch item := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range item.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if ok && typeSpec.Name.Name == "StudioBridge" {
					bridgeFound = true
				}
			}
		case *ast.FuncDecl:
			if item.Recv == nil || len(item.Recv.List) != 1 {
				continue
			}
			recv := item.Recv.List[0].Type
			if p, ok := recv.(*ast.StarExpr); ok {
				recv = p.X
			}
			ident, ok := recv.(*ast.Ident)
			if ok && ident.Name == "StudioBridge" && item.Name.IsExported() {
				methods = append(methods, item.Name.Name)
			}
		}
	}
	slices.Sort(methods)
	expected := []string{"CatalogWorkspace", "InspectSubject", "ListWorkspaces", "ValidateWorkspace"}
	if !bridgeFound || !slices.Equal(methods, expected) {
		t.Fatalf("Wails StudioBridge exposed methods changed: found=%v actual=%v want=%v", bridgeFound, methods, expected)
	}
}
