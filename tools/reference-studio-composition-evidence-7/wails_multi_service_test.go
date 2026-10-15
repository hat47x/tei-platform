//go:build studio_generated && server && tei_wails_multi_probe

// Copy this #7-owned test into cmd/tei-studio/ in a NON-MERGE fixed snapshot.
// It exercises Wails v3.0.0-beta.29's actual application.Bindings with both
// #1833 fe8e37-generated Go services. This is NOT an instruction to expose
// either raw generated service to a production/browser-facing Wails app.
package main

import (
    "context"
    "encoding/json"
    "path/filepath"
    "reflect"
    "strings"
    "testing"

    "github.com/wailsapp/wails/v3/pkg/application"

    "tei/internal/operations"
    "tei/internal/studiohost/staticservices"
)

func generatedMethodName(instance any, method string) string {
    typ := reflect.TypeOf(instance).Elem()
    return typ.PkgPath() + "." + typ.Name() + "." + method
}

func TestWailsTwoGeneratedServicesRetainBothMethodsAndRejectDuplicate(t *testing.T) {
    // Wails Bindings.Add emits registration traces through globalApplication.
    // Create an app for the same initialisation used in Wails's own tests.
    _ = application.New(application.Options{})
    root, err := filepath.Abs(filepath.Join("..", "..", "reference", "workspace"))
    if err != nil { t.Fatal(err) }
    catalogSvc := staticservices.NewCatalogService(root)
    validateSvc := staticservices.NewValidateService(root)

    bindings := application.NewBindings(nil, nil)
    if err := bindings.Add(application.NewService(catalogSvc)); err != nil {
        t.Fatalf("Wails Catalog registration: %v", err)
    }
    if err := bindings.Add(application.NewService(validateSvc)); err != nil {
        t.Fatalf("Wails Validate registration: %v", err)
    }

    catalogName := generatedMethodName(catalogSvc, "Catalog")
    validateName := generatedMethodName(validateSvc, "Validate")
    if catalogName == validateName {
        t.Fatal("generated services resolved to the same fully qualified method name")
    }
    catalog := bindings.Get(&application.CallOptions{MethodName: catalogName})
    validate := bindings.Get(&application.CallOptions{MethodName: validateName})
    if catalog == nil || validate == nil {
        t.Fatalf("both Wails bound methods must survive registration: catalog=%v validate=%v", catalog != nil, validate != nil)
    }
    if catalog.ID == validate.ID || bindings.GetByID(catalog.ID) != catalog || bindings.GetByID(validate.ID) != validate {
        t.Fatal("Wails method ID resolution overwrote one generated service")
    }

    catalogResult, err := catalog.Call(context.Background(), []json.RawMessage{json.RawMessage(`""`)})
    if err != nil { t.Fatalf("actual generated Catalog Wails call: %v", err) }
    expectedCatalog, err := operations.CatalogWorkspace(root, "")
    if err != nil { t.Fatal(err) }
    if !reflect.DeepEqual(catalogResult, expectedCatalog) {
        t.Fatalf("Catalog result changed after two-service registration: actual=%T expected=%T", catalogResult, expectedCatalog)
    }
    validationResult, err := validate.Call(context.Background(), nil)
    if err != nil { t.Fatalf("actual generated Validate Wails call: %v", err) }
    expectedValidation, err := operations.ValidateWorkspace(root)
    if err != nil { t.Fatal(err) }
    if !reflect.DeepEqual(validationResult, expectedValidation) {
        t.Fatalf("Validate result changed after two-service registration: actual=%T expected=%T", validationResult, expectedValidation)
    }

    // Wails rejects an additional instance of the same Go service TYPE
    // rather than silently overwriting its FQN. The first two remain usable.
    err = bindings.Add(application.NewService(staticservices.NewCatalogService(root)))
    if err == nil || !strings.Contains(err.Error(), "bound method '") ||
        !strings.Contains(err.Error(), catalogName) ||
        !strings.Contains(err.Error(), "already registered") ||
        !strings.Contains(err.Error(), "at most one service of each type") {
        t.Fatalf("expected Wails duplicate-method registration error, got %v", err)
    }
    if bindings.Get(&application.CallOptions{MethodName: catalogName}) != catalog ||
       bindings.Get(&application.CallOptions{MethodName: validateName}) != validate {
        t.Fatal("duplicate registration damaged an existing binding")
    }
}

type sameShortNameA struct{}
func (*sameShortNameA) Status() string { return "a" }
type sameShortNameB struct{}
func (*sameShortNameB) Status() string { return "b" }

func TestWailsIdenticallyNamedMethodsOfDifferentServiceTypesDoNotConflict(t *testing.T) {
    _ = application.New(application.Options{})
    b := application.NewBindings(nil, nil)
    first, second := &sameShortNameA{}, &sameShortNameB{}
    if err := b.Add(application.NewService(first)); err != nil { t.Fatal(err) }
    if err := b.Add(application.NewService(second)); err != nil {
        t.Fatalf("same simple method name on a different type is not an FQN collision: %v", err)
    }
    for _, tc := range []struct{ instance any; expected string }{
        {first, "a"}, {second, "b"},
    } {
        name := generatedMethodName(tc.instance, "Status")
        method := b.Get(&application.CallOptions{MethodName: name})
        if method == nil { t.Fatalf("missing distinct service method %q", name) }
        result, err := method.Call(context.Background(), nil)
        if err != nil || result != tc.expected {
            t.Fatalf("method %q: result=%v err=%v", name, result, err)
        }
    }
}
