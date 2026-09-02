package realization_test

import (
	"strings"
	"testing"

	"tei/internal/realization"
)

func TestCustomGoBindingIdentityAndResolution(t *testing.T) {
	binding := realization.CustomGoBinding{
		Application: "order_entry",
		Capability:  "sales.order.submit",
	}
	bindings := []realization.CustomGoBinding{
		{Application: "other_app", Capability: "sales.order.submit"},
		binding,
	}

	if got, want := binding.Identity(), "order_entry/sales.order.submit"; got != want {
		t.Fatalf("binding identity=%q want=%q", got, want)
	}

	resolved, err := realization.ResolveCustomGoBinding(bindings, "order_entry", "sales.order.submit")
	if err != nil {
		t.Fatalf("Bindingを解決できませんでした: %v", err)
	}
	if resolved != binding {
		t.Fatalf("resolved=%#v want=%#v", resolved, binding)
	}
}

func TestResolveCustomGoBindingDoesNotChooseByEnumerationOrder(t *testing.T) {
	bindings := []realization.CustomGoBinding{
		{Application: "order_entry", Capability: "sales.order.submit"},
		{Application: "order_entry", Capability: "sales.order.submit"},
	}

	_, err := realization.ResolveCustomGoBinding(bindings, "order_entry", "sales.order.submit")
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("複数Bindingを暗黙選択せず曖昧さを報告する必要があります: %v", err)
	}
}

func TestResolveCustomGoBindingReportsMissingBindingAsExecutionPreparationFailure(t *testing.T) {
	_, err := realization.ResolveCustomGoBinding(nil, "order_entry", "sales.order.submit")
	if err == nil || !strings.Contains(err.Error(), "is not defined") {
		t.Fatalf("未定義Bindingを業務結果として扱わずerrorにする必要があります: %v", err)
	}
}
