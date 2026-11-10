package main

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"
)

// schemaArgumentNames returns the property names a tool declares in its
// inputSchema, independently of declaredArguments, so the two can be compared.
func schemaArgumentNames(t *testing.T, tool tool) []string {
	t.Helper()
	raw, err := json.Marshal(tool.InputSchema)
	if err != nil {
		t.Fatalf("%s: marshal inputSchema: %v", tool.Name, err)
	}
	var decoded struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("%s: inputSchema is not an object: %v", tool.Name, err)
	}
	names := make([]string, 0, len(decoded.Properties))
	for name := range decoded.Properties {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// The arguments a tool accepts must be exactly the ones its inputSchema
// declares. That list used to be a second hand-written table; when it drifted
// permissively no test noticed, and only a separate per-tool check happened to
// refuse the call (#3823). The list is now derived, so this test guards the
// derivation rather than a copy.
func TestDeclaredArgumentsMatchTheSchema(t *testing.T) {
	tools := toolList()
	if len(tools) == 0 {
		t.Fatal("no tools advertised")
	}
	for _, tool := range tools {
		want := schemaArgumentNames(t, tool)
		got := declaredArguments(tool)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("%s: declaredArguments=%v, schema properties=%v", tool.Name, got, want)
		}
	}
}

// A name that is not advertised must be unknown, and an advertised name must
// resolve to the same tool the list returns.
func TestToolLookupMatchesTheAdvertisedList(t *testing.T) {
	for _, tool := range toolList() {
		found, ok := toolByName(tool.Name)
		if !ok || found.Name != tool.Name {
			t.Fatalf("%s: toolByName returned %+v ok=%v", tool.Name, found, ok)
		}
	}
	if _, ok := toolByName("tei_not_advertised"); ok {
		t.Fatal("an unadvertised name resolved to a tool")
	}
}

// checkArguments must refuse an argument the schema does not declare, and
// accept every argument it does. This is the behaviour that used to depend on
// two lists agreeing by hand.
func TestCheckArgumentsFollowsTheSchema(t *testing.T) {
	for _, tool := range toolList() {
		for _, name := range schemaArgumentNames(t, tool) {
			if err := checkArguments(tool.Name, map[string]json.RawMessage{
				name: json.RawMessage(`"x"`),
			}); err != nil {
				t.Fatalf("%s: declared argument %q was refused: %v", tool.Name, name, err)
			}
		}
		if err := checkArguments(tool.Name, map[string]json.RawMessage{
			"undeclaredArgument": json.RawMessage(`"x"`),
		}); err == nil {
			t.Fatalf("%s: an undeclared argument was accepted", tool.Name)
		}
	}
}

// The schema must also declare what it requires, or a client cannot tell which
// arguments it must send. A required name that is not a property is a schema
// the client cannot satisfy.
func TestEveryRequiredArgumentIsADeclaredProperty(t *testing.T) {
	for _, tool := range toolList() {
		properties := map[string]bool{}
		for _, name := range schemaArgumentNames(t, tool) {
			properties[name] = true
		}
		raw, err := json.Marshal(tool.InputSchema)
		if err != nil {
			t.Fatal(err)
		}
		var decoded struct {
			Required []string `json:"required"`
		}
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatal(err)
		}
		for _, name := range decoded.Required {
			if !properties[name] {
				t.Fatalf("%s requires %q but does not declare it as a property", tool.Name, name)
			}
		}
	}
}
