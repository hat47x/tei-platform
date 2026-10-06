package xeadsource

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"strings"
)

type Format string

const (
	FormatXead Format = "xead"
	FormatXeaf Format = "xeaf"
)

type Attribute struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Node struct {
	Locator    string      `json:"locator"`
	Name       string      `json:"name"`
	Attributes []Attribute `json:"attributes,omitempty"`
	Text       string      `json:"text,omitempty"`
	Known      bool        `json:"known"`
	Children   []Node      `json:"children,omitempty"`
}

type InventoryEntry struct {
	Name       string `json:"name"`
	Count      int    `json:"count"`
	Known      bool   `json:"known"`
}

type Document struct {
	Format       Format           `json:"format"`
	SHA256       string           `json:"sha256"`
	Root         Node             `json:"root"`
	Inventory    []InventoryEntry `json:"inventory"`
	UnknownNames []string         `json:"unknownNames,omitempty"`
}

var known = map[Format]map[string]struct{}{
	FormatXead: set(
		"System", "Subsystem", "Table", "TableField", "Relationship",
		"FunctionType", "Function", "IOTable", "IOPanel", "IOSpool",
		"Role", "Task", "TaskAction", "TaskFunctionIO", "SubjectArea",
		"DataflowNode", "DataflowLine", "Instance",
	),
	FormatXeaf: set("System", "Function", "Script", "Field", "Column", "Button", "Menu"),
}

func set(names ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(names))
	for _, name := range names {
		out[name] = struct{}{}
	}
	return out
}

func Parse(format Format, source []byte) (Document, error) {
	knownNames, ok := known[format]
	if !ok {
		return Document{}, fmt.Errorf("unsupported X-TEA source format %q", format)
	}
	sum := sha256.Sum256(source)
	decoder := xml.NewDecoder(bytes.NewReader(source))
	var root *Node
	stack := []*Node{}
	childOrdinals := []int{}
	counts := map[string]int{}
	unknown := map[string]struct{}{}

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return Document{}, fmt.Errorf("parse %s source: %w", format, err)
		}
		switch t := token.(type) {
		case xml.StartElement:
			name := qualifiedName(t.Name)
			_, isKnown := knownNames[t.Name.Local]
			n := Node{Name: name, Known: isKnown}
			n.Attributes = make([]Attribute, 0, len(t.Attr))
			for _, attr := range t.Attr {
				n.Attributes = append(n.Attributes, Attribute{Name: qualifiedName(attr.Name), Value: attr.Value})
			}
			sort.Slice(n.Attributes, func(i, j int) bool {
				if n.Attributes[i].Name != n.Attributes[j].Name {
					return n.Attributes[i].Name < n.Attributes[j].Name
				}
				return n.Attributes[i].Value < n.Attributes[j].Value
			})
			if len(stack) == 0 {
				n.Locator = "/" + name + "[1]"
				root = &n
			} else {
				parent := stack[len(stack)-1]
				childOrdinals[len(childOrdinals)-1]++
				n.Locator = fmt.Sprintf("%s/%s[%d]", parent.Locator, name, childOrdinals[len(childOrdinals)-1])
				parent.Children = append(parent.Children, n)
				n = parent.Children[len(parent.Children)-1]
			}
			counts[name]++
			if !isKnown {
				unknown[name] = struct{}{}
			}
			stack = append(stack, &n)
			childOrdinals = append(childOrdinals, 0)
		case xml.CharData:
			if len(stack) > 0 {
				stack[len(stack)-1].Text += string(t)
			}
		case xml.EndElement:
			if len(stack) == 0 {
				return Document{}, fmt.Errorf("unexpected closing element %q", t.Name.Local)
			}
			stack[len(stack)-1].Text = strings.TrimSpace(stack[len(stack)-1].Text)
			stack = stack[:len(stack)-1]
			childOrdinals = childOrdinals[:len(childOrdinals)-1]
		}
	}
	if root == nil {
		return Document{}, fmt.Errorf("source contains no root element")
	}
	inventory := make([]InventoryEntry, 0, len(counts))
	for name, count := range counts {
		_, isKnown := knownNames[localName(name)]
		inventory = append(inventory, InventoryEntry{Name: name, Count: count, Known: isKnown})
	}
	sort.Slice(inventory, func(i, j int) bool { return inventory[i].Name < inventory[j].Name })
	unknownNames := make([]string, 0, len(unknown))
	for name := range unknown {
		unknownNames = append(unknownNames, name)
	}
	sort.Strings(unknownNames)
	return Document{
		Format: format, SHA256: hex.EncodeToString(sum[:]), Root: *root,
		Inventory: inventory, UnknownNames: unknownNames,
	}, nil
}

func qualifiedName(name xml.Name) string {
	if name.Space == "" {
		return name.Local
	}
	return "{" + name.Space + "}" + name.Local
}

func localName(name string) string {
	if i := strings.LastIndex(name, "}"); i >= 0 {
		return name[i+1:]
	}
	return name
}
