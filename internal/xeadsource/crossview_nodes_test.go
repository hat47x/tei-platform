package xeadsource

import "testing"

// Every token a projection edge uses must be one CrossViewSourceNodes lists,
// so a consumer can lead each end of a relation back to a source position.
func TestCrossViewSourceNodesCarryTheProjectionTokens(t *testing.T) {
	doc := parseCrossViewFixture(t, `<System>
  <Table ID="T1" Name="Order"/>
  <Function ID="F1"><IOTable ID="IO1" TableID="T1" OpR="+"/></Function>
  <Task ID="TASK1"><TaskAction ID=""><TaskFunctionIO FunctionID="F1" IOID="0"/></TaskAction></Task>
  <x:Private xmlns:x="urn:private" ID="P1"/>
</System>`)
	nodes := CrossViewSourceNodes(doc)
	byToken := map[string]CrossViewSourceNode{}
	byLocator := map[string]CrossViewSourceNode{}
	for _, node := range nodes {
		if _, dup := byToken[node.Token]; dup {
			t.Fatalf("token %s names two elements", node.Token)
		}
		byToken[node.Token] = node
		byLocator[node.Locator] = node
	}
	projection, err := buildCrossViewProjection(Inspection{
		Document: doc, TranscriptionContractFingerprint: "sha256:" + doc.SHA256,
	})
	if err != nil {
		t.Fatal(err)
	}
	edges := append(append([]CrossViewEdge(nil), projection.SourceEdges...), projection.DerivedEdges...)
	if len(edges) == 0 {
		t.Fatal("the fixture produced no edges")
	}
	for _, edge := range edges {
		for _, token := range append([]string{edge.FromToken, edge.ToToken}, edge.SourceTokens...) {
			if _, ok := byToken[token]; !ok {
				t.Fatalf("edge %s uses token %s, which names no listed element", edge.Kind, token)
			}
		}
	}

	table := byLocator["/System[1]/Table[1]"]
	if table.Name != "Table" || table.LegacyID != "T1" || !table.LegacyIDPresent ||
		table.DisplayName != "Order" || table.ParentToken != byLocator["/System[1]"].Token {
		t.Fatalf("table node = %+v", table)
	}
	// A blank ID is absent. It is not filled in from the element's position.
	action := byLocator["/System[1]/Task[3]/TaskAction[1]"]
	if action.LegacyIDPresent || action.LegacyID != "" ||
		action.ParentToken != byLocator["/System[1]/Task[3]"].Token {
		t.Fatalf("action node = %+v", action)
	}
	// An element of a foreign namespace is listed, and none of its attributes
	// is read as a source ID.
	private := byLocator["/System[1]/{urn:private}Private[4]"]
	if private.Token == "" || private.Known || private.LegacyIDPresent {
		t.Fatalf("foreign node = %+v (locators: %v)", private, locators(nodes))
	}
}

func locators(nodes []CrossViewSourceNode) []string {
	out := make([]string, 0, len(nodes))
	for _, node := range nodes {
		out = append(out, node.Locator)
	}
	return out
}
