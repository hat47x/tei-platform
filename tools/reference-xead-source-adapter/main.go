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
	Namespace  string            `json:"namespace,omitempty"`
	Known      bool              `json:"known"`
	Locator    SourceLocator     `json:"locator"`
	Attributes map[string]string `json:"attributes,omitempty"`
	Text       string            `json:"text,omitempty"`
}

type ObservedMarkup struct {
	Kind          string `json:"kind"`
	ParentLocator string `json:"parentLocator,omitempty"`
	Target        string `json:"target,omitempty"`
	Data          string `json:"data,omitempty"`
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
	Format         SourceFormat     `json:"format"`
	ArtifactSHA256 string           `json:"artifactSha256"`
	RootKind       string           `json:"rootKind"`
	Nodes          []ObservedNode   `json:"nodes"`
	Markup         []ObservedMarkup `json:"markup,omitempty"`
	Facts          []SourceFact     `json:"facts,omitempty"`
	Unresolved     []ObservedIssue  `json:"unresolved,omitempty"`
}

type ObservedIssue struct {
	Code    string `json:"code"`
	Locator string `json:"locator"`
	Detail  string `json:"detail"`
}

type PublicationReceipt struct {
	SchemaVersion         int              `json:"schemaVersion"`
	Format                SourceFormat     `json:"format"`
	ArtifactSHA256        string           `json:"artifactSha256"`
	RootKind              string           `json:"rootKind"`
	ElementCounts         map[string]int   `json:"elementCounts"`
	AttributeCounts       map[string]int   `json:"attributeCounts,omitempty"`
	UnknownElementCount   int              `json:"unknownElementCount,omitempty"`
	UnknownAttributeCount int              `json:"unknownAttributeCount,omitempty"`
	MarkupCounts          map[string]int   `json:"markupCounts,omitempty"`
	LocatorCount          int              `json:"locatorCount"`
	LocatorSetFingerprint string           `json:"locatorSetFingerprint"`
	Facts                 []PublishedFact  `json:"facts,omitempty"`
	Unresolved            []PublishedIssue `json:"unresolved,omitempty"`
	AuthorityLimits       []string         `json:"authorityLimits"`
	ReceiptFingerprint    string           `json:"receiptFingerprint"`

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
	contractFlag := flag.String("contract", "", "optional transcription contract JSON")
	localCandidatesFlag := flag.Bool("local-candidates", false, "include source-locator candidate classification; local/private output only")
	flag.Parse()
	if *formatFlag == "" || *inputFlag == "" {
		fmt.Fprintln(os.Stderr, "usage: reference-xead-source-adapter -format xead|xeaf -input <file> [-contract transcription-contract.json] [-local-candidates]")
		os.Exit(2)
	}
	content, err := os.ReadFile(*inputFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	format := SourceFormat(*formatFlag)
	doc, err := Observe(format, content)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	receipt := PublicReceipt(doc)
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if *contractFlag == "" {
		if err := encoder.Encode(receipt); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		return
	}
	contractBytes, err := os.ReadFile(*contractFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	coverage, err := CheckClassificationCoverage(format, doc, contractBytes)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	classification, err := BuildClassificationProjection(format, doc, contractBytes)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	unresolved, err := BuildUnresolvedProjection(format, receipt, classification)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	var localCandidates []LocalTranscriptionCandidate
	if *localCandidatesFlag {
		localCandidates, err = ClassifyObservedDocument(format, doc, contractBytes)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	output := struct {
		Receipt           PublicationReceipt            `json:"receipt"`
		Coverage          ClassificationCoverage        `json:"classificationCoverage"`
		Classification    ClassificationProjection      `json:"classificationProjection"`
		Unresolved        UnresolvedProjection          `json:"unresolvedProjection"`
		LocalOutputNotice string                        `json:"localOutputNotice,omitempty"`
		LocalCandidates   []LocalTranscriptionCandidate `json:"localCandidates,omitempty"`
	}{
		Receipt:         receipt,
		Coverage:        coverage,
		Classification:  classification,
		Unresolved:      unresolved,
		LocalCandidates: localCandidates,
	}
	if *localCandidatesFlag {
		output.LocalOutputNotice = "local/private evidence: contains exact source structural locators; do not commit as public Reference Model evidence"
	}
	if err := encoder.Encode(output); err != nil {
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
			qualifiedKind := xmlElementName(t.Name)
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.childCounts[qualifiedKind]++
				ordinal = parent.childCounts[qualifiedKind]
				parentPath = parent.path
			}
			path := fmt.Sprintf("%s/%s[%d]", parentPath, qualifiedKind, ordinal)
			attrs := make(map[string]string, len(t.Attr))
			for _, attr := range t.Attr {
				name := attr.Name.Local
				if attr.Name.Space != "" {
					name = "{" + attr.Name.Space + "}" + attr.Name.Local
				}
				attrs[name] = attr.Value
			}
			_, known := knownElements[format][t.Name.Local]
			known = known && t.Name.Space == ""
			node := ObservedNode{
				Kind:       t.Name.Local,
				Namespace:  t.Name.Space,
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
				code := "unknown-element"
				detail := t.Name.Local
				if t.Name.Space != "" {
					code = "namespaced-element"
					detail = t.Name.Space + " " + t.Name.Local
				}
				doc.Unresolved = append(doc.Unresolved, ObservedIssue{Code: code, Locator: path, Detail: detail})
			}
		case xml.CharData:
			if len(stack) == 0 {
				continue
			}
			idx := stack[len(stack)-1].nodeIndex
			doc.Nodes[idx].Text += string(t)
		case xml.Comment:
			doc.Markup = append(doc.Markup, ObservedMarkup{Kind: "comment", ParentLocator: currentParentLocator(stack), Data: string(t)})
		case xml.ProcInst:
			doc.Markup = append(doc.Markup, ObservedMarkup{Kind: "processing-instruction", ParentLocator: currentParentLocator(stack), Target: t.Target, Data: string(t.Inst)})
		case xml.Directive:
			doc.Markup = append(doc.Markup, ObservedMarkup{Kind: "directive", ParentLocator: currentParentLocator(stack), Data: string(t)})
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

func publicRootKind(doc ObservedDocument) string {
	if len(doc.Nodes) == 0 {
		return ""
	}
	root := doc.Nodes[0]
	if root.Known {
		return root.Kind
	}
	return "unknown"
}

func PublicReceipt(doc ObservedDocument) PublicationReceipt {
	receipt := PublicationReceipt{
		SchemaVersion:         2,
		Format:                doc.Format,
		ArtifactSHA256:        doc.ArtifactSHA256,
		RootKind:              publicRootKind(doc),
		ElementCounts:         map[string]int{},
		AttributeCounts:       map[string]int{},
		MarkupCounts:          map[string]int{},
		AuthorityLimits: []string{
			"source-observation-is-not-canonical-authority",
			"legacy-id-sortkey-and-xml-position-are-not-canonical-identity",
			"legacy-key-facts-are-not-business-identity-or-functional-dependency",
			"publication-receipt-does-not-contain-source-values-or-source-text",
			"publication-receipt-does-not-disclose-arbitrary-unknown-element-or-attribute-names",
			"public-locator-digest-does-not-hash-source-derived-structural-names",
		},
	}
	var locatorDigests []string
	publicLocatorByPath := map[string]string{}
	for i, node := range doc.Nodes {
		if node.Known {
			receipt.ElementCounts[node.Kind]++
			receipt.AttributeCounts[node.Kind] += len(node.Attributes)
		} else {
			receipt.UnknownElementCount++
			receipt.UnknownAttributeCount += len(node.Attributes)
		}
		locatorDigest := digestString(fmt.Sprintf("%s|node-ordinal:%d", doc.ArtifactSHA256, i+1))
		publicLocatorByPath[node.Locator.Path] = locatorDigest
		locatorDigests = append(locatorDigests, locatorDigest)
	}
	for _, markup := range doc.Markup {
		receipt.MarkupCounts[markup.Kind]++
	}
	if len(receipt.MarkupCounts) == 0 {
		receipt.MarkupCounts = nil
	}
	for i, fact := range doc.Facts {
		locatorDigest, ok := publicLocatorByPath[fact.Locator]
		if !ok {
			locatorDigest = digestString(fmt.Sprintf("%s|unbound-fact-ordinal:%d", doc.ArtifactSHA256, i+1))
		}
		receipt.Facts = append(receipt.Facts, PublishedFact{
			Kind:            fact.Kind,
			LocatorDigest:   locatorDigest,
			FactType:        fact.FactType,
			AttributeNames:  append([]string(nil), fact.AttributeNames...),
			AuthorityLimits: append([]string(nil), fact.AuthorityLimits...),
		})
	}
	for i, issue := range doc.Unresolved {
		locatorDigest, ok := publicLocatorByPath[issue.Locator]
		if !ok {
			locatorDigest = digestString(fmt.Sprintf("%s|unbound-issue-ordinal:%d", doc.ArtifactSHA256, i+1))
		}
		receipt.Unresolved = append(receipt.Unresolved, PublishedIssue{Code: issue.Code, LocatorDigest: locatorDigest})
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
	if !node.Known {
		return nil
	}
	attrs := node.Attributes
	if format == FormatXead {
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
		}
	}
	if format == FormatXeaf {
		switch node.Kind {
		case "Function":
			names := presentAttributeNames(attrs, "KeyFields", "HeaderKeyFields", "DetailKeyFields", "BatchKeyFields", "BatchWithKeyFields", "StructureChildKeys", "StructureUpperKeys")
			if len(names) > 0 {
				return []SourceFact{{
					Kind:            node.Kind,
					Locator:         node.Locator.Path,
					FactType:        "legacy-function-selection-key",
					Values:          selectedValues(attrs, names...),
					AttributeNames:  names,
					AuthorityLimits: []string{"declared-key-fields-are-not-data-identity", "declared-key-fields-are-not-business-identity", "declared-key-fields-are-not-functional-dependency"},
				}}
			}
		case "Detail":
			names := presentAttributeNames(attrs, "KeyFields", "HeaderKeyFields")
			if len(names) > 0 {
				return []SourceFact{{
					Kind:            node.Kind,
					Locator:         node.Locator.Path,
					FactType:        "legacy-detail-key-binding",
					Values:          selectedValues(attrs, names...),
					AttributeNames:  names,
					AuthorityLimits: []string{"declared-key-fields-are-not-data-identity", "declared-key-fields-are-not-business-identity", "declared-key-fields-are-not-functional-dependency"},
				}}
			}
		case "Key":
			return []SourceFact{{
				Kind:            node.Kind,
				Locator:         node.Locator.Path,
				FactType:        "runtime-table-key",
				Values:          selectedValues(attrs, "Fields", "Type"),
				AttributeNames:  presentAttributeNames(attrs, "Fields", "Type"),
				AuthorityLimits: []string{"runtime-key-is-realization-not-business-identity", "runtime-key-is-not-functional-dependency", "runtime-key-does-not-replace-design-key-evidence"},
			}}
		case "Refer":
			return []SourceFact{{
				Kind:           node.Kind,
				Locator:        node.Locator.Path,
				FactType:       "runtime-reference-key-binding",
				Values:         selectedValues(attrs, "Fields", "ToKeyFields", "WithKeyFields", "ToTable", "TableAlias", "Optional"),
				AttributeNames: presentAttributeNames(attrs, "Fields", "ToKeyFields", "WithKeyFields", "ToTable", "TableAlias", "Optional"),
				AuthorityLimits: []string{
					"runtime-refer-is-realization-not-semantic-relation-authority",
					"runtime-refer-key-fields-are-not-business-identity",
					"runtime-pk-fallback-is-not-business-identity",
				},
			}}
		}
	}
	return nil
}

func currentParentLocator(stack []*parseFrame) string {
	if len(stack) == 0 {
		return ""
	}
	return stack[len(stack)-1].path
}

func xmlElementName(name xml.Name) string {
	if name.Space == "" {
		return name.Local
	}
	return "{" + name.Space + "}" + name.Local
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
