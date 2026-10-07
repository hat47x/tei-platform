package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

type SourceFormat string

const (
	FormatXead SourceFormat = "xead"
	FormatXeaf SourceFormat = "xeaf"
)

type SourceLocator struct {
	Path        string            `json:"path"`
	LegacyHints map[string]string `json:"legacyHints,omitempty"`
}

type ObservedNode struct {
	Kind       string            `json:"kind"`
	Known      bool              `json:"known"`
	Locator    SourceLocator     `json:"locator"`
	Attributes map[string]string `json:"attributes,omitempty"`
	Text       string            `json:"text,omitempty"`
}

type SourceFact struct {
	Kind            string            `json:"kind"`
	Locator         string            `json:"locator"`
	FactType        string            `json:"factType"`
	Values          map[string]string `json:"values,omitempty"`
	AttributeNames  []string          `json:"attributeNames,omitempty"`
	AuthorityLimits []string          `json:"authorityLimits,omitempty"`
}

type ObservedDocument struct {
	Format         SourceFormat    `json:"format"`
	ArtifactSHA256 string          `json:"artifactSha256"`
	RootKind       string          `json:"rootKind"`
	Nodes          []ObservedNode  `json:"nodes"`
	Facts          []SourceFact    `json:"facts,omitempty"`
	Unresolved     []ObservedIssue `json:"unresolved,omitempty"`
}

type ObservedIssue struct {
	Code    string `json:"code"`
	Locator string `json:"locator"`
	Detail  string `json:"detail"`
}

type PublicationReceipt struct {
	SchemaVersion       int                       `json:"schemaVersion"`
	Format              SourceFormat              `json:"format"`
	ArtifactSHA256      string                    `json:"artifactSha256"`
	RootKind            string                    `json:"rootKind"`
	ElementCounts       map[string]int            `json:"elementCounts"`
	AttributeNameCounts map[string]map[string]int `json:"attributeNameCounts"`
	UnknownElements     map[string]int            `json:"unknownElements,omitempty"`
	LocatorCount          int                       `json:"locatorCount"`
	LocatorSetFingerprint string                    `json:"locatorSetFingerprint"`
	Facts                 []PublishedFact           `json:"facts,omitempty"`
	Unresolved          []PublishedIssue          `json:"unresolved,omitempty"`
	AuthorityLimits     []string                  `json:"authorityLimits"`
	ReceiptFingerprint  string                    `json:"receiptFingerprint"`
}

type PublishedFact struct {
	Kind            string   `json:"kind"`
	LocatorDigest   string   `json:"locatorDigest"`
	FactType        string   `json:"factType"`
	AttributeNames  []string `json:"attributeNames,omitempty"`
	AuthorityLimits []string `json:"authorityLimits,omitempty"`
}

type PublishedIssue struct {
	Code          string `json:"code"`
	LocatorDigest string `json:"locatorDigest"`
}

type parseFrame struct {
	kind        string
	path        string
	childCounts map[string]int
	nodeIndex   int
}

var knownElements = map[SourceFormat]map[string]struct{}{
	FormatXead: setOf(
		"System", "Subsystem", "Department", "DataType", "Table", "TableField", "TableKey", "TableKeyField",
		"Relationship", "FunctionType", "Function", "FunctionUsedByThis", "IOTable", "IOPanel", "IOPanelField", "IOPanelStyle", "IOSpool", "IOSpoolStyle", "Role",
		"Task", "TaskAction", "TaskFunctionIO", "SubjectArea", "DataflowNode", "DataflowLine", "Instance", "SubsystemTable", "SubsystemRelationship", "TableType", "MaintenanceLog",
	),
	FormatXeaf: setOf(
		"System", "Subsystem", "Table", "Function", "Script", "Field", "Column", "Button", "Filter", "Refer", "Detail", "Key", "Menu",
		"HeaderPhrase", "Phrase", "Option", "PrintFont", "BatchField", "AddRowListColumn", "AddRowListButton", "MaintenanceLog",
	),
}

func main() {
	formatFlag := flag.String("format", "", "source format: xead or xeaf")
	inputFlag := flag.String("input", "", "path to the source XML")
	flag.Parse()
	if *formatFlag == "" || *inputFlag == "" {
		fmt.Fprintln(os.Stderr, "usage: reference-xead-source-adapter -format xead|xeaf -input <file>")
		os.Exit(2)
	}
	content, err := os.ReadFile(*inputFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	doc, err := Observe(SourceFormat(*formatFlag), content)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(PublicReceipt(doc)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func Observe(format SourceFormat, content []byte) (ObservedDocument, error) {
	if _, ok := knownElements[format]; !ok {
		return ObservedDocument{}, fmt.Errorf("unsupported source format %q", format)
	}
	sum := sha256.Sum256(content)
	doc := ObservedDocument{Format: format, ArtifactSHA256: hex.EncodeToString(sum[:])}

	decoder := xml.NewDecoder(strings.NewReader(string(content)))
	var stack []*parseFrame
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return ObservedDocument{}, fmt.Errorf("parse %s: %w", format, err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			ordinal := 1
			parentPath := ""
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.childCounts[t.Name.Local]++
				ordinal = parent.childCounts[t.Name.Local]
				parentPath = parent.path
			}
			path := fmt.Sprintf("%s/%s[%d]", parentPath, t.Name.Local, ordinal)
			attrs := make(map[string]string, len(t.Attr))
			for _, attr := range t.Attr {
				name := attr.Name.Local
				if attr.Name.Space != "" {
					name = "{" + attr.Name.Space + "}" + attr.Name.Local
				}
				attrs[name] = attr.Value
			}
			_, known := knownElements[format][t.Name.Local]
			node := ObservedNode{
				Kind:       t.Name.Local,
				Known:      known,
				Locator:    SourceLocator{Path: path, LegacyHints: selectedLegacyHints(attrs)},
				Attributes: attrs,
			}
			doc.Nodes = append(doc.Nodes, node)
			if len(stack) == 0 {
				doc.RootKind = t.Name.Local
			}
			idx := len(doc.Nodes) - 1
			stack = append(stack, &parseFrame{kind: t.Name.Local, path: path, childCounts: map[string]int{}, nodeIndex: idx})
			doc.Facts = append(doc.Facts, factsForNode(format, doc.Nodes[idx])...)
			if !known {
				doc.Unresolved = append(doc.Unresolved, ObservedIssue{Code: "unknown-element", Locator: path, Detail: t.Name.Local})
			}
		case xml.CharData:
			if len(stack) == 0 {
				continue
			}
			idx := stack[len(stack)-1].nodeIndex
			doc.Nodes[idx].Text += string(t)
		case xml.EndElement:
			if len(stack) == 0 || stack[len(stack)-1].kind != t.Name.Local {
				return ObservedDocument{}, fmt.Errorf("unexpected closing element %s", t.Name.Local)
			}
			idx := stack[len(stack)-1].nodeIndex
			doc.Nodes[idx].Text = strings.TrimSpace(doc.Nodes[idx].Text)
			stack = stack[:len(stack)-1]
		}
	}
	if len(stack) != 0 {
		return ObservedDocument{}, fmt.Errorf("unclosed XML elements")
	}
	return doc, nil
}

func PublicReceipt(doc ObservedDocument) PublicationReceipt {
	receipt := PublicationReceipt{
		SchemaVersion:       1,
		Format:              doc.Format,
		ArtifactSHA256:      doc.ArtifactSHA256,
		RootKind:            doc.RootKind,
		ElementCounts:       map[string]int{},
		AttributeNameCounts: map[string]map[string]int{},
		UnknownElements:     map[string]int{},
		AuthorityLimits: []string{
			"source-observation-is-not-canonical-authority",
			"legacy-id-sortkey-and-xml-position-are-not-canonical-identity",
			"legacy-key-facts-are-not-business-identity-or-functional-dependency",
			"publication-receipt-does-not-contain-source-values-or-source-text",
		},
	}
	var locatorDigests []string
	for _, node := range doc.Nodes {
		receipt.ElementCounts[node.Kind]++
		if receipt.AttributeNameCounts[node.Kind] == nil {
			receipt.AttributeNameCounts[node.Kind] = map[string]int{}
		}
		for name := range node.Attributes {
			receipt.AttributeNameCounts[node.Kind][name]++
		}
		if !node.Known {
			receipt.UnknownElements[node.Kind]++
		}
		locatorDigests = append(locatorDigests, digestString(doc.ArtifactSHA256+"|"+node.Locator.Path))
	}
	for _, fact := range doc.Facts {
		receipt.Facts = append(receipt.Facts, PublishedFact{
			Kind:            fact.Kind,
			LocatorDigest:   digestString(doc.ArtifactSHA256 + "|" + fact.Locator),
			FactType:        fact.FactType,
			AttributeNames:  append([]string(nil), fact.AttributeNames...),
			AuthorityLimits: append([]string(nil), fact.AuthorityLimits...),
		})
	}
	for _, issue := range doc.Unresolved {
		receipt.Unresolved = append(receipt.Unresolved, PublishedIssue{Code: issue.Code, LocatorDigest: digestString(doc.ArtifactSHA256 + "|" + issue.Locator)})
	}
	if len(receipt.UnknownElements) == 0 {
		receipt.UnknownElements = nil
	}
	sort.Strings(locatorDigests)
	receipt.LocatorCount = len(locatorDigests)
	receipt.LocatorSetFingerprint = fingerprint(locatorDigests)
	sort.Slice(receipt.Facts, func(i, j int) bool {
		if receipt.Facts[i].LocatorDigest != receipt.Facts[j].LocatorDigest {
			return receipt.Facts[i].LocatorDigest < receipt.Facts[j].LocatorDigest
		}
		return receipt.Facts[i].FactType < receipt.Facts[j].FactType
	})
	sort.Slice(receipt.Unresolved, func(i, j int) bool {
		if receipt.Unresolved[i].LocatorDigest != receipt.Unresolved[j].LocatorDigest {
			return receipt.Unresolved[i].LocatorDigest < receipt.Unresolved[j].LocatorDigest
		}
		return receipt.Unresolved[i].Code < receipt.Unresolved[j].Code
	})
	copy := receipt
	copy.ReceiptFingerprint = ""
	receipt.ReceiptFingerprint = fingerprint(copy)
	return receipt
}

func factsForNode(format SourceFormat, node ObservedNode) []SourceFact {
	if format != FormatXead {
		return nil
	}
	attrs := node.Attributes
	switch node.Kind {
	case "TableKey":
		return []SourceFact{{
			Kind:           node.Kind,
			Locator:        node.Locator.Path,
			FactType:       "legacy-table-key",
			Values:         selectedValues(attrs, "ID", "Type", "KeyType", "SortKey"),
			AttributeNames: presentAttributeNames(attrs, "ID", "Type", "KeyType", "SortKey"),
			AuthorityLimits: []string{
				"legacy-key-is-not-business-identity",
				"legacy-key-is-not-candidate-key",
				"legacy-key-is-not-functional-dependency",
				"legacy-key-is-not-physical-key-authority",
			},
		}}
	case "TableKeyField":
		return []SourceFact{{
			Kind:           node.Kind,
			Locator:        node.Locator.Path,
			FactType:       "legacy-key-field-membership",
			Values:         selectedValues(attrs, "FieldID", "SortKey"),
			AttributeNames: presentAttributeNames(attrs, "FieldID", "SortKey"),
			AuthorityLimits: []string{
				"legacy-key-field-membership-is-not-business-identity",
				"legacy-key-field-membership-is-not-functional-dependency",
				"ambiguous-key-field-mapping-must-remain-unresolved",
			},
		}}
	case "Relationship":
		return []SourceFact{{
			Kind:           node.Kind,
			Locator:        node.Locator.Path,
			FactType:       "legacy-relationship-key-reference",
			Values:         selectedValues(attrs, "Table1ID", "TableKey1ID", "Table2ID", "TableKey2ID", "ExistWhen1", "ExistWhen2", "Type"),
			AttributeNames: presentAttributeNames(attrs, "Table1ID", "TableKey1ID", "Table2ID", "TableKey2ID", "ExistWhen1", "ExistWhen2", "Type"),
			AuthorityLimits: []string{
				"relationship-key-reference-is-not-business-identity",
				"relationship-key-reference-is-not-functional-dependency",
				"conditional-prose-is-not-executable-constraint",
			},
		}}
	case "Function", "Detail":
		if names := presentAttributeNames(attrs, "KeyFields"); len(names) > 0 {
			return []SourceFact{{
				Kind:            node.Kind,
				Locator:         node.Locator.Path,
				FactType:        "legacy-function-selection-key",
				Values:          selectedValues(attrs, "KeyFields"),
				AttributeNames:  names,
				AuthorityLimits: []string{"function-selection-key-is-not-business-identity", "function-selection-key-is-not-functional-dependency"},
			}}
		}
	}
	return nil
}

func selectedLegacyHints(attrs map[string]string) map[string]string {
	out := map[string]string{}
	for _, name := range []string{"ID", "SortKey"} {
		if value, ok := attrs[name]; ok && value != "" {
			out[name] = value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func presentAttributeNames(attrs map[string]string, names ...string) []string {
	var out []string
	for _, name := range names {
		if _, ok := attrs[name]; ok {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func selectedValues(attrs map[string]string, names ...string) map[string]string {
	out := map[string]string{}
	for _, name := range names {
		if value, ok := attrs[name]; ok {
			out[name] = value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func setOf(values ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}

func digestString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func fingerprint(value any) string {
	encoded, _ := json.Marshal(value)
	return digestString(string(encoded))
}
