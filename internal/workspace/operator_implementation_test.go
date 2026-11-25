package workspace

import (
	"reflect"
	"strings"
	"testing"
)

const operatorImplementationManifest = "workspace: tei_md_rule\nformatVersion: 1\nsources:\n  - semantic\nlanguages:\n  - tei.rule-authoring/v0\n"

func TestManifestImplementationsAreReadAsWritten(t *testing.T) {
	root := teiMDOperatorWorkspace(t)
	writeTEIMDTestFile(t, root, "tei.workspace.yaml", operatorImplementationManifest+
		"implementations:\n  - operator: sales.non_negative\n    use: tei.number.non-negative\n")
	loaded, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []OperatorImplementationSelection{{Operator: "sales.non_negative", Implementation: "tei.number.non-negative"}}
	if !reflect.DeepEqual(loaded.OperatorImplementations, want) {
		t.Fatalf("selections=%+v, want %+v", loaded.OperatorImplementations, want)
	}

	// Without the section nothing is selected, and nothing is inferred from
	// the operator's name.
	plain, err := Load(teiMDOperatorWorkspace(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(plain.OperatorImplementations) != 0 {
		t.Fatalf("a Workspace without the section has selections: %+v", plain.OperatorImplementations)
	}
	// Load does not decide whether this tei carries the implementation; a
	// name it has never heard of still loads and is refused at evaluation.
	writeTEIMDTestFile(t, root, "tei.workspace.yaml", operatorImplementationManifest+
		"implementations:\n  - operator: sales.non_negative\n    use: example.not-carried\n")
	if _, err := Load(root); err != nil {
		t.Fatalf("Load judged implementation availability: %v", err)
	}
}

func TestManifestImplementationsRefusals(t *testing.T) {
	cases := []struct {
		name, section, want string
	}{
		{"operator no source declares",
			"implementations:\n  - operator: sales.non_negativ\n    use: tei.number.non-negative\n", "TEI_OPERATOR_IMPLEMENTATION_UNBOUND"},
		{"attribute instead of an operator",
			"implementations:\n  - operator: sales.order_line.discount\n    use: tei.number.non-negative\n", "TEI_OPERATOR_IMPLEMENTATION_UNBOUND"},
		{"operator selected twice",
			"implementations:\n  - operator: sales.non_negative\n    use: tei.number.non-negative\n  - operator: sales.non_negative\n    use: tei.number.non-zero\n", "more than one implementation entry"},
		{"entry without use",
			"implementations:\n  - operator: sales.non_negative\n", "needs 'use: <implementation-id>'"},
		// The implementation name is a canonical word: lower case, digits,
		// dot, dash and underscore only. An upper-case letter is refused by
		// this diagnostic rather than by the selection check, so a reader is
		// told the name is malformed instead of that it is not carried.
		{"use with an upper-case letter",
			"implementations:\n  - operator: sales.non_negative\n    use: tei.number.NonNegative\n", "needs 'use: <implementation-id>'"},
		{"use with a space",
			"implementations:\n  - operator: sales.non_negative\n    use: tei.number.non negative\n", "needs 'use: <implementation-id>'"},
		{"use written twice",
			"implementations:\n  - operator: sales.non_negative\n    use: tei.number.non-negative\n    use: tei.number.non-zero\n", "duplicate implementation use"},
		{"use before an operator",
			"implementations:\n    use: tei.number.non-negative\n", "must begin with '- operator:'"},
		{"unknown key",
			"implementations:\n  - operator: sales.non_negative\n    version: 1\n", "unsupported implementation entry"},
		{"section written twice",
			"implementations:\n  - operator: sales.non_negative\n    use: tei.number.non-negative\nimplementations:\n", "duplicate implementations section"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := teiMDOperatorWorkspace(t)
			writeTEIMDTestFile(t, root, "tei.workspace.yaml", operatorImplementationManifest+tc.section)
			loaded, err := Load(root)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err=%v, want %q (selections=%+v)", err, tc.want, loaded.OperatorImplementations)
			}
		})
	}
}
