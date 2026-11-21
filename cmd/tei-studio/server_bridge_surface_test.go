//go:build server

package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// This is an inventory guard for the *existing loopback-only* Wails Server
// reference, not authorization for public exposure. Wails server mode binds
// exported Go service methods; the JavaScript client's method list is not an
// access-control list. Every added exported method requires a deliberate
// classification before a shared Web host can be considered.
func TestServerBridgeExportedMethodInventoryRequiresReview(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "main.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	registered, unreviewed, optionSets := studioServicesFromOptions(file)
	if registered != 1 || unreviewed != 0 || optionSets != 1 {
		t.Fatalf("Wails Server Services list changed; review actual registrations: StudioBridge=%d unreviewed=%d Options=%d", registered, unreviewed, optionSets)
	}

	// The labels are audit classifications, NOT grants. In particular,
	// "reference command" is still an exported method even when its current
	// implementation only writes to an in-memory reference store.
	expected := map[string]string{
		"ListWorkspaces":                   "read projection",
		"CatalogWorkspace":                 "read projection",
		"ValidateWorkspace":                "read projection",
		"InspectSubject":                   "read projection",
		"InspectSubjectAtSnapshot":         "read projection",
		"ObserveXead":                      "bounded source read",
		"ViewPattern":                      "read projection",
		"ViewApplication":                  "read projection",
		"ViewPackage":                      "read projection",
		"PlanCommand":                      "command plan",
		"ExecuteCommand":                   "reference command",
		"ExecuteCommandWithPeriods":        "reference command",
		"ExecuteCommandWithPolicy":         "reference command; client allowed list is not Host grant",
		"ExecuteCommandWithReferenceStore": "reference command with memory effect",
		"OpenShoDocument":                  "external child process / file path",
		"ResumeShoDocument":                "external child process / file path",
	}
	found := map[string]bool{}
	// Reflection inspects the effective Go method set, including methods
	// promoted from embedded types and methods declared in another file.
	// It is the candidate set, not proof of every Wails HTTP route.
	typ := reflect.TypeOf(&StudioBridge{})
	for i := 0; i < typ.NumMethod(); i++ {
		name := typ.Method(i).Name
		found[name] = true
		if _, classified := expected[name]; !classified {
			t.Errorf("new Wails Server exported method %q has no exposure classification; review before adding it to the shared-host surface", name)
		}
	}
	if len(found) == 0 {
		t.Fatal("no StudioBridge exported methods found: inventory check would pass vacuously")
	}
	for name, classification := range expected {
		if !found[name] {
			t.Errorf("reviewed StudioBridge method %q (%s) disappeared; update the inventory deliberately", name, classification)
		}
	}
	var elevated []string
	for name := range found {
		if classification := expected[name]; classification != "" && classification != "read projection" && classification != "bounded source read" {
			elevated = append(elevated, name+"="+classification)
		}
	}
	sort.Strings(elevated)
	if len(elevated) == 0 {
		t.Fatal("expected privileged/non-read methods are absent; re-evaluate this inventory test when the server bridge is split")
	}
	t.Logf("NOT PUBLIC-QUALIFIED: loopback Wails Server currently registers privileged/non-read surfaces: %s", strings.Join(elevated, ", "))
}

// studioServicesFromOptions inspects the Services member actually supplied to
// application.Options. Scanning NewService calls anywhere in main.go alone
// would miss a dynamically replaced list or an additional direct Service.
func studioServicesFromOptions(file *ast.File) (registered, unreviewed, optionSets int) {
	ast.Inspect(file, func(node ast.Node) bool {
		options, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		typ, ok := options.Type.(*ast.SelectorExpr)
		if !ok || typ.Sel.Name != "Options" {
			return true
		}
		pkg, ok := typ.X.(*ast.Ident)
		if !ok || pkg.Name != "application" {
			return true
		}
		optionSets++
		servicesFound := false
		for _, elt := range options.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				unreviewed++
				continue
			}
			name, ok := kv.Key.(*ast.Ident)
			if !ok || name.Name != "Services" {
				continue
			}
			servicesFound = true
			services, ok := kv.Value.(*ast.CompositeLit)
			if !ok {
				// The only reviewed dynamic form has a fixed StudioBridge
				// and a single, credential-gated read-only HTTP route.
				// Every other variable, append, or replacement remains
				// unreviewed so the public-surface inventory fails closed.
				if ident, isIdent := kv.Value.(*ast.Ident); isIdent && ident.Name == "services" &&
					reviewOptionalInspectionServiceList(file, ident.Name) {
					registered++
				} else {
					unreviewed++
				}
				continue
			}
			for _, entry := range services.Elts {
				call, ok := entry.(*ast.CallExpr)
				if !ok || len(call.Args) != 1 {
					unreviewed++
					continue
				}
				method, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || method.Sel.Name != "NewService" {
					unreviewed++
					continue
				}
				imported, ok := method.X.(*ast.Ident)
				if !ok || imported.Name != "application" {
					unreviewed++
					continue
				}
				addr, ok := call.Args[0].(*ast.UnaryExpr)
				if !ok || addr.Op != token.AND {
					unreviewed++
					continue
				}
				value, ok := addr.X.(*ast.CompositeLit)
				if !ok {
					unreviewed++
					continue
				}
				serviceType, ok := value.Type.(*ast.Ident)
				if !ok || serviceType.Name != "StudioBridge" {
					unreviewed++
					continue
				}
				registered++
			}
		}
		if !servicesFound {
			unreviewed++
		}
		return true
	})
	return
}

// reviewOptionalInspectionServiceList recognizes exactly the two changes to
// the Services variable in main: the original StudioBridge and one opt-in
// read-only HTTP route. Unknown registrations must not pass the inventory by
// hiding behind a variable or a helper.
func reviewOptionalInspectionServiceList(file *ast.File, variable string) bool {
	var body *ast.BlockStmt
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Recv == nil && fn.Name.Name == "main" {
			body = fn.Body
			break
		}
	}
	if body == nil {
		return false
	}
	var assigned []*ast.AssignStmt
	ast.Inspect(body, func(node ast.Node) bool {
		a, ok := node.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for _, lhs := range a.Lhs {
			if name, isIdent := lhs.(*ast.Ident); isIdent && name.Name == variable {
				assigned = append(assigned, a)
			}
		}
		return true
	})
	if len(assigned) != 2 {
		return false
	}
	bridge := false
	published := false
	for _, a := range assigned {
		if len(a.Lhs) != 1 || len(a.Rhs) != 1 {
			return false
		}
		if a.Tok == token.DEFINE {
			bridge = bridge || reviewedBridgeServiceInitializer(a.Rhs[0])
		} else if a.Tok == token.ASSIGN {
			published = published || reviewedInspectionHTTPAppend(a.Rhs[0], variable)
		} else {
			return false
		}
	}
	if !bridge || !published {
		return false
	}
	// A registered service must not be reachable without the explicit,
	// opt-in configuration. This check is structural: it does not infer
	// authorization or external-publication fitness from an env variable.
	guarded := false
	ast.Inspect(body, func(node ast.Node) bool {
		guard, ok := node.(*ast.IfStmt)
		if !ok || guard.Else != nil || !reviewedInspectionHTTPTokenGuard(guard) {
			return true
		}
		ast.Inspect(guard.Body, func(n ast.Node) bool {
			if n == assigned[1] && assigned[1].Tok == token.ASSIGN {
				guarded = true
			}
			return true
		})
		return true
	})
	return guarded
}

func reviewedBridgeServiceInitializer(expr ast.Expr) bool {
	lit, ok := expr.(*ast.CompositeLit)
	if !ok || len(lit.Elts) != 1 {
		return false
	}
	array, ok := lit.Type.(*ast.ArrayType)
	if !ok || array.Len != nil || !reviewedApplicationSelector(array.Elt, "Service") {
		return false
	}
	call, ok := lit.Elts[0].(*ast.CallExpr)
	if !ok || !reviewedApplicationSelector(call.Fun, "NewService") || len(call.Args) != 1 {
		return false
	}
	address, ok := call.Args[0].(*ast.UnaryExpr)
	if !ok || address.Op != token.AND {
		return false
	}
	bridge, ok := address.X.(*ast.CompositeLit)
	if !ok {
		return false
	}
	typ, ok := bridge.Type.(*ast.Ident)
	return ok && typ.Name == "StudioBridge"
}

func reviewedInspectionHTTPAppend(expr ast.Expr, variable string) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) != 2 {
		return false
	}
	fn, ok := call.Fun.(*ast.Ident)
	if !ok || fn.Name != "append" {
		return false
	}
	list, ok := call.Args[0].(*ast.Ident)
	if !ok || list.Name != variable {
		return false
	}
	service, ok := call.Args[1].(*ast.CallExpr)
	if !ok || !reviewedApplicationSelector(service.Fun, "NewServiceWithOptions") || len(service.Args) != 2 {
		return false
	}
	api, ok := service.Args[0].(*ast.Ident)
	if !ok || api.Name != "api" {
		return false
	}
	options, ok := service.Args[1].(*ast.CompositeLit)
	if !ok || !reviewedApplicationSelector(options.Type, "ServiceOptions") || len(options.Elts) != 1 {
		return false
	}
	route, ok := options.Elts[0].(*ast.KeyValueExpr)
	if !ok {
		return false
	}
	name, ok := route.Key.(*ast.Ident)
	if !ok || name.Name != "Route" {
		return false
	}
	value, ok := route.Value.(*ast.Ident)
	return ok && value.Name == "inspectionHTTPRoute"
}

func reviewedInspectionHTTPTokenGuard(guard *ast.IfStmt) bool {
	cond, ok := guard.Cond.(*ast.Ident)
	if !ok || cond.Name != "set" {
		return false
	}
	init, ok := guard.Init.(*ast.AssignStmt)
	if !ok || init.Tok != token.DEFINE || len(init.Lhs) != 2 || len(init.Rhs) != 1 {
		return false
	}
	tokenVar, tokenOK := init.Lhs[0].(*ast.Ident)
	setVar, setOK := init.Lhs[1].(*ast.Ident)
	if !tokenOK || !setOK || tokenVar.Name != "token" || setVar.Name != "set" {
		return false
	}
	call, ok := init.Rhs[0].(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return false
	}
	fn, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || fn.Sel.Name != "LookupEnv" {
		return false
	}
	pkg, ok := fn.X.(*ast.Ident)
	if !ok || pkg.Name != "os" {
		return false
	}
	name, ok := call.Args[0].(*ast.Ident)
	return ok && name.Name == "inspectionHTTPTokenEnv"
}

func reviewedApplicationSelector(expr ast.Expr, method string) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != method {
		return false
	}
	pkg, ok := selector.X.(*ast.Ident)
	return ok && pkg.Name == "application"
}

func TestServerBridgeServicesInventoryRejectsUnreviewedMutants(t *testing.T) {
	const good = `package main
func main() {
	_ = application.Options{Services: []application.Service{application.NewService(&StudioBridge{})}}
}`
	mutants := map[string]string{
		"extra service":       strings.Replace(good, "application.NewService(&StudioBridge{})}}", "application.NewService(&StudioBridge{}), application.NewService(&OtherBridge{})}}", 1),
		"second StudioBridge": strings.Replace(good, "application.NewService(&StudioBridge{})}}", "application.NewService(&StudioBridge{}), application.NewService(&StudioBridge{})}}", 1),
		"opaque variable":     strings.Replace(good, "Services: []application.Service{application.NewService(&StudioBridge{})}", "Services: registeredServices", 1),
		"empty list":          strings.Replace(good, "application.NewService(&StudioBridge{})", "", 1),
		"other bridge":        strings.Replace(good, "&StudioBridge{}", "&OtherBridge{}", 1),
		"missing Services":    strings.Replace(good, "Services: []application.Service{application.NewService(&StudioBridge{})}", "", 1),
	}
	parse := func(name, src string) (int, int, int) {
		t.Helper()
		file, err := parser.ParseFile(token.NewFileSet(), name, src, 0)
		if err != nil {
			t.Fatalf("%s: parse input: %v", name, err)
		}
		return studioServicesFromOptions(file)
	}
	registered, unreviewed, count := parse("valid.go", good)
	if registered != 1 || unreviewed != 0 || count != 1 {
		t.Fatalf("valid inventory not recognized: StudioBridge=%d unreviewed=%d Options=%d", registered, unreviewed, count)
	}
	for name, source := range mutants {
		t.Run(name, func(t *testing.T) {
			registered, unreviewed, count := parse(name+".go", source)
			if registered == 1 && unreviewed == 0 && count == 1 {
				t.Fatalf("unreviewed Services mutation escaped review: %s", name)
			}
		})
	}
}
