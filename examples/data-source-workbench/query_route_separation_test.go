package workbenchfixture

import (
	"bytes"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Query IDs are qualified inside a Workspace, not across every TEI Source.
// The actual reference/workspace and xtea-lowcode workspaces both use
// order_entry.recent_orders, but target different physical relations and
// supply different SQL inputs. This test deliberately executes no SQL.
func TestRecentOrdersSameIDHasTwoDistinctSourceAndParameterContracts(t *testing.T) {
	read := func(path string) string {
		t.Helper()
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return string(contents)
	}
	referenceSQL := read("../../reference/workspace/applications/order-entry/queries/recent_orders.sql")
	referenceDecl := read("../../reference/workspace/applications/order-entry/queries/recent_orders.tei")
	xteaSQL := read("../xtea-lowcode/applications/order-entry/queries/recent_orders.sql")
	xteaDecl := read("../xtea-lowcode/applications/order-entry/queries/recent_orders.tei")
	xteaPattern := read("../xtea-lowcode/applications/order-entry/patterns/recent-orders.tei")
	productSeed := read("recent_orders_product_seed.sql")

	for name, declaration := range map[string]string{"reference": referenceDecl, "xtea": xteaDecl} {
		if !strings.Contains(declaration, "query order_entry.recent_orders {") {
			t.Fatalf("%s Query ID changed; comparison is no longer the same-ID negative case", name)
		}
	}
	if bytes.Equal([]byte(referenceSQL), []byte(xteaSQL)) {
		t.Fatal("different Workspaces unexpectedly contain identical SQL")
	}
	// Independent expected route contracts; neither can serve as the
	// other's real PostgreSQL seed or selected physical Query.
	for _, tc := range []struct {
		name    string
		source  string
		must    []string
		mustNot []string
	}{
		{
			name: "reference",
			source: referenceSQL,
			must: []string{
				"FROM public.orders AS o", "o.ordered_at >= :from",
				"ORDER BY o.ordered_at DESC", "o.total_amount",
			},
			mustNot: []string{"from sales.order", ":status_operator", ":page_offset"},
		},
		{
			name: "xtea",
			source: xteaSQL,
			must: []string{
				"select order_id, customer_id", ":status",
			},
			mustNot: []string{"FROM public.orders", ":from", "o.total_amount"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, part := range tc.must {
				if !strings.Contains(tc.source, part) {
					t.Errorf("SQL does not contain independent expected clause %q", part)
				}
			}
			for _, part := range tc.mustNot {
				if strings.Contains(tc.source, part) {
					t.Errorf("SQL contains clause from the other Workspace %q", part)
				}
			}
		})
	}
	// Keep the logical XTEA relation distinct from public.orders without
	// making its current *unquoted* PostgreSQL reserved identifier a permanent
	// expected syntax. "order" must be quoted in executable PostgreSQL SQL.
	xteaLower := strings.ToLower(xteaSQL)
	if !strings.Contains(xteaLower, "from sales.order") &&
		!strings.Contains(xteaLower, `from sales."order"`) &&
		!strings.Contains(xteaLower, `from "sales"."order"`) {
		t.Fatal("xtea SQL no longer targets the independently selected sales.order relation")
	}
	if !strings.Contains(productSeed, "CREATE TABLE public.orders (") ||
		strings.Contains(productSeed, "CREATE TABLE sales.order") ||
		!strings.Contains(xteaPattern, "filter sales.order.status { eq ne }") ||
		!strings.Contains(xteaPattern, "page-size 20") {
		t.Fatal("product seed and xtea Pattern contracts were silently conflated")
	}

	// Source-static SQL-parameter discovery is only a mismatch detector,
	// NOT a SQL parser, runtime parameter Binder, or actual DB observation.
	//
	// The reference SQL currently requires :from, but its .tei declaration
	// has no input clause for it. Record this as an explicit unresolved
	// qualification. Changing the declaration must update this Expected.
	// XTEA main currently has only :status. Draft #4076 adds operator and
	// page controls; this main test must not require an unmerged draft.
	if missing := undeclaredNativeSQLInputs(referenceSQL, referenceDecl); !reflect.DeepEqual(missing, []string{"from"}) {
		t.Fatalf("reference recent_orders input gap changed: got %v want [from]; review R0 qualification", missing)
	}
	if missing := undeclaredNativeSQLInputs(xteaSQL, xteaDecl); len(missing) != 0 {
		t.Fatalf("xtea recent_orders SQL requires undeclared parameters: %v", missing)
	}
}

var nativeParameterName = regexp.MustCompile(`:[a-z][a-z0-9_]*`)
var queryDeclaredInput = regexp.MustCompile(`(?m)^\s*input\s+([a-z][a-z0-9_]*)\s+`)

// undeclaredNativeSQLInputs intentionally checks only these simple reference
// SQLs, which contain no quoted :name literals or PostgreSQL cast syntax.
func undeclaredNativeSQLInputs(sql, declaration string) []string {
	declared := make(map[string]struct{})
	for _, match := range queryDeclaredInput.FindAllStringSubmatch(declaration, -1) {
		declared[match[1]] = struct{}{}
	}
	missing := make(map[string]struct{})
	for _, name := range nativeParameterName.FindAllString(sql, -1) {
		if _, ok := declared[name[1:]]; !ok {
			missing[name[1:]] = struct{}{}
		}
	}
	names := make([]string, 0, len(missing))
	for name := range missing {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
