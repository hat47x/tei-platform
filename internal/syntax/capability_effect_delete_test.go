package syntax_test

import (
	"testing"

	"tei/internal/semantic"
	"tei/internal/syntax"
)

func TestParseCapabilityDeleteEffect(t *testing.T) {
	src := `namespace sales

data order {
    attribute orderId type std.identifier
}

capability cancel {
    effect delete sales.order
}`
	namespace, err := syntax.NewParser(src).ParseSemanticFile()
	if err != nil {
		t.Fatal(err)
	}
	if len(namespace.Capabilities) != 1 || namespace.Capabilities[0].Effect == nil {
		t.Fatalf("capability=%+v", namespace.Capabilities)
	}
	if namespace.Capabilities[0].Effect.Kind != semantic.DataEffectDelete {
		t.Fatalf("effect=%s", namespace.Capabilities[0].Effect.Kind)
	}
}
