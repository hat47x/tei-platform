package studiohost

import (
	"os"
	"path/filepath"
	"testing"
)

// hookWorkspace writes a workspace whose record pattern declares one hook, so
// the Studio reference execution can be asked what it did with it.
//
// The namespaces, the data identifiers and the capability follow the syntax the
// reference workspace uses: a namespace line, a `data` block with attributes,
// and a `capability` block whose input names the data.
func hookWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "semantic", "sales"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "applications", "order_entry", "patterns"), 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := "workspace: studio_hook_fixture\nformatVersion: 1\nsources:\n  - semantic\n  - applications\n"
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	data := `namespace sales

data Order {
    attribute status type std.text
    attribute quantity type std.decimal
}

capability sales.submit {
    input order sales.order
    output sales.order
    effect update sales.order
}
`
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales", "order.tei"), []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	application := `application order_entry {
    uses semantic sales
}
`
	if err := os.WriteFile(filepath.Join(root, "applications", "order_entry", "application.tei"), []byte(application), 0o600); err != nil {
		t.Fatal(err)
	}
	pattern := `pattern order_entry.order_record record {
    data sales.order

    interaction {
        item sales.order.status

        command sales.submit {
            input order sales.order.status
        }

        hook before-update {
            reads sales.order.status
            uses sales.submit
            on-failure reject
        }
    }
}
`
	if err := os.WriteFile(filepath.Join(root, "applications", "order_entry", "patterns", "order-record.tei"), []byte(pattern), 0o600); err != nil {
		t.Fatal(err)
	}
	command := `pattern order_entry.submit command {
    capability sales.submit
}
`
	if err := os.WriteFile(filepath.Join(root, "applications", "order_entry", "patterns", "submit.tei"), []byte(command), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

// hookWorkspaceWithoutCall writes the same workspace, but the hook declares only
// reads. The reference execution can run a hook it has no caller for only when
// the hook does not promise a call.
func hookWorkspaceWithoutCall(t *testing.T) string {
	t.Helper()
	root := hookWorkspace(t)
	pattern := `pattern order_entry.order_record record {
    data sales.order

    interaction {
        item sales.order.status

        command sales.submit {
            input order sales.order.status
        }

        hook before-update {
            reads sales.order.status
            on-failure reject
        }
    }
}
`
	if err := os.WriteFile(filepath.Join(root, "applications", "order_entry", "patterns", "order-record.tei"), []byte(pattern), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}
