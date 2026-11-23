package record_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"

	"tei/internal/codegen/diag"
	"tei/internal/codegen/record"
)

type schemaNode struct {
	Ref        string                 `json:"$ref"`
	Required   []string               `json:"required"`
	Properties map[string]*schemaNode `json:"properties"`
	Items      *schemaNode            `json:"items"`
	Const      string                 `json:"const"`
	Pattern    string                 `json:"pattern"`
	Enum       []string               `json:"enum"`
	Additional *bool                  `json:"additionalProperties"`
}

type schemaDoc struct {
	schemaNode
	Schema string                 `json:"$schema"`
	ID     string                 `json:"$id"`
	Defs   map[string]*schemaNode `json:"$defs"`
}

func loadSchema(t *testing.T, name string) schemaDoc {
	t.Helper()
	raw, err := record.SchemaFile(name)
	if err != nil {
		t.Fatal(err)
	}
	var doc schemaDoc
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(&doc); err != nil {
		t.Fatalf("%s schema is not JSON: %v", name, err)
	}
	return doc
}

var recordTypes = map[string]reflect.Type{
	"diagnostic": reflect.TypeFor[diag.Record](),
	"trace":      reflect.TypeFor[record.TraceLine](),
	"manifest":   reflect.TypeFor[record.Manifest](),
}

func TestEachSchemaIsDraft2020WithItsID(t *testing.T) {
	ids := map[string]string{
		"diagnostic": record.DiagnosticSchema,
		"trace":      record.TraceSchema,
		"manifest":   record.ManifestSchema,
	}
	for name, id := range ids {
		doc := loadSchema(t, name)
		if doc.Schema != "https://json-schema.org/draft/2020-12/schema" {
			t.Errorf("%s: $schema %q", name, doc.Schema)
		}
		if want := "tei.codegen." + name + "/draft.1"; doc.ID != want || id != want {
			t.Errorf("%s: $id %q, constant %q, want %q", name, doc.ID, id, want)
		}
	}
	if _, err := record.SchemaFile("missing"); err == nil {
		t.Fatal("an unknown schema name was found")
	}
	// The schemas state the same values the Go validators enforce.
	if got := loadSchema(t, "trace").Properties["schema"].Const; got != record.TraceSchema {
		t.Errorf("trace schema const %q", got)
	}
	if got := loadSchema(t, "manifest").Properties["schema"].Const; got != record.ManifestSchema {
		t.Errorf("manifest schema const %q", got)
	}
	d := loadSchema(t, "diagnostic")
	if got := d.Properties["code"].Pattern; got != diag.CodePattern.String() {
		t.Errorf("code pattern %q, Go %q", got, diag.CodePattern)
	}
	if got := strings.Join(d.Properties["severity"].Enum, ","); got != "error,warning,info" {
		t.Errorf("severity enum %s", got)
	}
	if got := strings.Join(d.Properties["fixer"].Enum, ","); got != "designer,developer,provider" {
		t.Errorf("fixer enum %s", got)
	}
}

// Every object in a schema lists exactly the JSON fields of its Go type, and
// requires exactly those not marked omitempty. A field added on one side only
// fails here.
func TestSchemasMatchTheGoTypes(t *testing.T) {
	for name, typ := range recordTypes {
		doc := loadSchema(t, name)
		compareObject(t, name, doc, &doc.schemaNode, typ)
	}
}

func compareObject(t *testing.T, where string, doc schemaDoc, node *schemaNode, typ reflect.Type) {
	t.Helper()
	node = resolve(t, doc, node)
	if node.Additional == nil || *node.Additional {
		t.Errorf("%s: additionalProperties is not false", where)
	}
	var fields, required []string
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := strings.Split(field.Tag.Get("json"), ",")
		fields = append(fields, tag[0])
		if len(tag) == 1 {
			required = append(required, tag[0])
		}
		property, ok := node.Properties[tag[0]]
		if !ok {
			continue
		}
		inner := field.Type
		for inner.Kind() == reflect.Pointer || inner.Kind() == reflect.Slice {
			if inner.Kind() == reflect.Slice {
				property = resolve(t, doc, property).Items
			}
			inner = inner.Elem()
		}
		if inner.Kind() == reflect.Struct {
			compareObject(t, where+"."+tag[0], doc, property, inner)
		}
	}
	var properties []string
	for key := range node.Properties {
		properties = append(properties, key)
	}
	sort.Strings(fields)
	sort.Strings(properties)
	sort.Strings(required)
	schemaRequired := append([]string(nil), node.Required...)
	sort.Strings(schemaRequired)
	if strings.Join(fields, ",") != strings.Join(properties, ",") {
		t.Errorf("%s: Go fields %v, schema properties %v", where, fields, properties)
	}
	if strings.Join(required, ",") != strings.Join(schemaRequired, ",") {
		t.Errorf("%s: Go required %v, schema required %v", where, required, schemaRequired)
	}
}

func resolve(t *testing.T, doc schemaDoc, node *schemaNode) *schemaNode {
	t.Helper()
	if node == nil {
		t.Fatal("missing schema node")
	}
	if node.Ref == "" {
		return node
	}
	def, ok := doc.Defs[strings.TrimPrefix(node.Ref, "#/$defs/")]
	if !ok || !strings.HasPrefix(node.Ref, "#/$defs/") {
		t.Fatalf("unresolved $ref %s", node.Ref)
	}
	return def
}

const digest = "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"

type validator interface{ Validate() error }

// decode reads one record strictly: an unknown field is an error.
func decode(t *testing.T, name, text string) (validator, error) {
	t.Helper()
	target := reflect.New(recordTypes[name])
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target.Interface()); err != nil {
		return nil, err
	}
	return target.Elem().Interface().(validator), nil
}

func TestGoodExamplesRoundTrip(t *testing.T) {
	examples := []struct{ name, text string }{
		{"diagnostic", `{"code":"NAME_COLLISION","severity":"error","message":"two rules give QuantityIsPositive"}`},
		{"diagnostic", `{"code":"goemit/NAME_INVALID","severity":"warning","file":"semantic/sales.tei","range":{"start":4,"end":12},"line":1,"column":5,"element":"sales.order_line.quantity_positive","message":"m","related":[{"file":"semantic/other.tei","range":{"start":0,"end":3},"element":"inventory.item.quantity_positive","role":"conflict"}],"fixer":"developer","provider":"goemit"}`},
		{"trace", `{"schema":"tei.codegen.trace/draft.1","file":"quantity_is_positive.go","lines":[5,10],"symbol":"QuantityIsPositive","element":"sales.order_line.quantity_positive","role":"decision","provider":"goemit"}`},
		{"trace", `{"schema":"tei.codegen.trace/draft.1","file":"a.go","lines":[1,1],"symbol":"A","element":"x.y","role":"check"}`},
		{"manifest", `{"schema":"tei.codegen.manifest/draft.1","files":[]}`},
		{"manifest", `{"schema":"tei.codegen.manifest/draft.1","input":"` + digest + `","files":[{"path":"a.go","sha256":"` + digest + `","writer":"goemit","hook":"rename"},{"path":"b.go","sha256":"` + digest + `","writer":"goemit"}]}`},
	}
	for _, example := range examples {
		value, err := decode(t, example.name, example.text)
		if err != nil {
			t.Fatalf("%s: %v", example.text, err)
		}
		if err := value.Validate(); err != nil {
			t.Fatalf("%s: %v", example.text, err)
		}
		again, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if string(again) != example.text {
			t.Fatalf("round trip changed the record\n got %s\nwant %s", again, example.text)
		}
	}
}

func TestBadExamplesAreRejected(t *testing.T) {
	trace := `"file":"a.go","lines":[1,2],"symbol":"A","element":"x.y","role":"check"`
	file := `{"path":"a.go","sha256":"` + digest + `","writer":"goemit"}`
	examples := []struct{ name, why, text string }{
		{"diagnostic", "no message", `{"code":"C","severity":"error"}`},
		{"diagnostic", "no severity", `{"code":"C","message":"m"}`},
		{"diagnostic", "no code", `{"severity":"error","message":"m"}`},
		{"diagnostic", "unknown field", `{"code":"C","severity":"error","message":"m","hint":"h"}`},
		{"diagnostic", "reversed range", `{"code":"C","severity":"error","message":"m","range":{"start":3,"end":2}}`},
		{"trace", "no schema", `{` + trace + `}`},
		{"trace", "wrong schema", `{"schema":"tei.codegen.trace/draft.2",` + trace + `}`},
		{"trace", "manifest schema", `{"schema":"tei.codegen.manifest/draft.1",` + trace + `}`},
		{"trace", "no symbol", `{"schema":"tei.codegen.trace/draft.1","file":"a.go","lines":[1,2],"element":"x.y","role":"check"}`},
		{"trace", "no element", `{"schema":"tei.codegen.trace/draft.1","file":"a.go","lines":[1,2],"symbol":"A","role":"check"}`},
		{"trace", "no lines", `{"schema":"tei.codegen.trace/draft.1","file":"a.go","symbol":"A","element":"x.y","role":"check"}`},
		{"trace", "lines reversed", `{"schema":"tei.codegen.trace/draft.1","file":"a.go","lines":[3,2],"symbol":"A","element":"x.y","role":"check"}`},
		{"trace", "capital role", `{"schema":"tei.codegen.trace/draft.1","file":"a.go","lines":[1,2],"symbol":"A","element":"x.y","role":"Check"}`},
		{"manifest", "no schema", `{"files":[` + file + `]}`},
		{"manifest", "wrong schema", `{"schema":"tei.codegen.trace/draft.1","files":[` + file + `]}`},
		{"manifest", "no files", `{"schema":"tei.codegen.manifest/draft.1"}`},
		{"manifest", "no writer", `{"schema":"tei.codegen.manifest/draft.1","files":[{"path":"a.go","sha256":"` + digest + `"}]}`},
		{"manifest", "upper-case digest", `{"schema":"tei.codegen.manifest/draft.1","files":[{"path":"a.go","sha256":"` + strings.ToUpper(digest) + `","writer":"w"}]}`},
		{"manifest", "short input", `{"schema":"tei.codegen.manifest/draft.1","input":"abc","files":[]}`},
		{"manifest", "unsorted", `{"schema":"tei.codegen.manifest/draft.1","files":[` + strings.Replace(file, "a.go", "b.go", 1) + `,` + file + `]}`},
		{"manifest", "repeated path", `{"schema":"tei.codegen.manifest/draft.1","files":[` + file + `,` + file + `]}`},
	}
	for _, example := range examples {
		value, err := decode(t, example.name, example.text)
		if err == nil {
			err = value.Validate()
		}
		if err == nil {
			t.Errorf("%s %s accepted: %s", example.name, example.why, example.text)
		}
	}
}

func TestNewManifestSortsAndDigests(t *testing.T) {
	files := []record.FileContent{
		{Path: "result.go", Content: []byte("package p\n")},
		{Path: "decimal.go", Content: []byte("test")},
		{Path: "empty.go", Content: nil},
	}
	m := record.NewManifest("", files, "goemit", "")
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
	// The digest of "test" is typed in, not computed by the code under test.
	want := []record.ManifestFile{
		{Path: "decimal.go", SHA256: digest, Writer: "goemit"},
		{Path: "empty.go", SHA256: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", Writer: "goemit"},
		{Path: "result.go", SHA256: sum("package p\n"), Writer: "goemit"},
	}
	if !reflect.DeepEqual(m.Files, want) || m.Schema != record.ManifestSchema || m.Input != "" {
		t.Fatalf("manifest %+v", m)
	}
	reversed := record.NewManifest("", []record.FileContent{files[2], files[1], files[0]}, "goemit", "")
	if !reflect.DeepEqual(m, reversed) {
		t.Fatal("the order of the input changed the manifest")
	}
	withHook := record.NewManifest(digest, files[:1], "goemit", "rename")
	if withHook.Input != digest || withHook.Files[0].Hook != "rename" {
		t.Fatalf("input or hook lost: %+v", withHook)
	}
	if got := record.NewManifest("", nil, "goemit", ""); got.Files == nil || got.Validate() != nil {
		t.Fatalf("an empty manifest is not valid: %+v", got)
	}
}

func sum(text string) string {
	digest := sha256.Sum256([]byte(text))
	return hex.EncodeToString(digest[:])
}
