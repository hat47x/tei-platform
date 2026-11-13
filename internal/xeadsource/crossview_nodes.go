package xeadsource

// CrossViewSourceNode is one element occurrence of an exact source document,
// addressed by the same source token the cross-view projection uses.
//
// The token is derived from the document digest and the element's position, so
// it names an occurrence in these exact bytes. It is NOT a Canonical identity,
// and it does not survive any change of the source. LegacyID is the literal
// `ID` attribute and stays a source fact: it may be missing, empty or repeated.
type CrossViewSourceNode struct {
	Token       string `json:"token"`
	ParentToken string `json:"parentToken,omitempty"`
	Locator     string `json:"locator"`
	Name        string `json:"name"`
	Known       bool   `json:"known"`
	// LegacyIDPresent is false when the attribute is absent or blank. A blank
	// value is never repaired from a sort key or from the element's position.
	LegacyID        string `json:"legacyId,omitempty"`
	LegacyIDPresent bool   `json:"legacyIdPresent"`
	// DisplayName is the literal `Name` attribute, when the element has one.
	DisplayName string `json:"displayName,omitempty"`
}

// CrossViewSourceNodes lists every element of doc in the order, and with the
// tokens, that BuildCrossViewProjection assigns for the same document. A
// consumer joins a projection edge back to a source position through it,
// instead of re-deriving tokens or matching elements by name.
func CrossViewSourceNodes(doc Document) []CrossViewSourceNode {
	flat := flattenCrossViewNodes(doc)
	out := make([]CrossViewSourceNode, 0, len(flat))
	// flattenCrossViewNodes walks in document order, so a parent is always
	// emitted before its children and a locator names exactly one element.
	tokenByLocator := make(map[string]string, len(flat))
	parentLocator := map[string]string{}
	var index func(Node)
	index = func(node Node) {
		for _, child := range node.Children {
			parentLocator[child.Locator] = node.Locator
			index(child)
		}
	}
	index(doc.Root)
	for _, ref := range flat {
		tokenByLocator[ref.node.Locator] = ref.token
	}
	for _, ref := range flat {
		node := CrossViewSourceNode{
			Token:   ref.token,
			Locator: ref.node.Locator,
			Name:    ref.node.Name,
			Known:   ref.node.Known,
		}
		if parent, ok := parentLocator[ref.node.Locator]; ok {
			node.ParentToken = tokenByLocator[parent]
		}
		if ref.node.Known {
			node.LegacyID, node.LegacyIDPresent = nonEmptyExactAttribute(ref.node, "ID")
			node.DisplayName, _ = exactAttribute(ref.node, "Name")
		}
		out = append(out, node)
	}
	return out
}
