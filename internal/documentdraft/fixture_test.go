package documentdraft

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/sourceindex"
)

// fixture is one synthetic Workspace on disk. It uses no customer data and no
// reference/workspace content: the point is the projection contract, not a real
// model.
type fixture struct {
	root string
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	root := t.TempDir()
	write := func(relative, body string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("tei.workspace.yaml", "workspace: docdraft_fixture\nformatVersion: 1\nsources:\n  - semantic\n")
	write("semantic/sales/order.tei", `namespace sales

type OrderId from std.identifier

data Order {
    attribute orderId type sales.order_id
    attribute customerId type customer.customer_id
    attribute status type std.text
    relation customer to customer.customer one
        key sales.order.customer_id -> customer.customer.customer_id
    relation lines to sales.order_line many
}
`)
	write("semantic/sales/order-line.tei", `namespace sales

data OrderLine {
    attribute orderLineId type sales.order_line_id
    attribute lineNumber type std.integer
}
`)
	write("semantic/customer/customer.tei", `namespace customer

data Customer {
    attribute customerId type customer.customer_id
    attribute name type std.text
}
`)
	write("semantic/sales/submit.tei", `namespace sales

capability sales.order.submit {
    input order sales.order
    output sales.order
}
`)
	return fixture{root: root}
}

func (f fixture) rewrite(t *testing.T, relative, body string) {
	t.Helper()
	path := filepath.Join(f.root, filepath.FromSlash(relative))
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func (f fixture) remove(t *testing.T, relative string) {
	t.Helper()
	if err := os.Remove(filepath.Join(f.root, filepath.FromSlash(relative))); err != nil {
		t.Fatal(err)
	}
}

func (f fixture) digestOf(t *testing.T, relative string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(f.root, filepath.FromSlash(relative)))
	if err != nil {
		t.Fatal(err)
	}
	return sourceindex.Fingerprint(body)
}

// readFile and writeFile move one Source between two paths without touching its
// bytes, which is how the Source-move case is built.
func readFile(root, relative string) ([]byte, error) {
	return os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
}

func writeFile(t *testing.T, root, relative string, body []byte) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
}

func orderSubject() sourceindex.Subject {
	return sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}
}

func occurrenceByID(draft Draft, id string) (Occurrence, bool) {
	for _, occurrence := range draft.Occurrences {
		if occurrence.OccurrenceID == id {
			return occurrence, true
		}
	}
	return Occurrence{}, false
}

// readSubject resolves one subject from the fixture's own Workspace, so a test
// that asserts a row's subject exists does not reimplement the loader.
func (f fixture) readSubject(subject sourceindex.Subject) (Draft, error) {
	return Build(Request{Root: f.root, Subject: subject})
}

func kindsOf(draft Draft) []string {
	out := make([]string, 0, len(draft.Occurrences))
	for _, occurrence := range draft.Occurrences {
		out = append(out, occurrence.Kind)
	}
	return out
}
