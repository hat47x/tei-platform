package validation

import "testing"

func TestValidateHeaderDetailContractQueryBindingsRequiresSourceInputBinding(t *testing.T) {
	world := headerDetailContractQueryBindingWorld()
	world.HeaderDetailContractQueryBindings[0].Inputs = nil
	world.Databases[0].Contracts[0].Queries[0].Inputs = nil

	assertHeaderDetailBindingDiagnostic(
		t,
		ValidateHeaderDetailContractQueryBindings(world),
		"HEADER_DETAIL_BINDING_INPUTS_REQUIRED",
	)
}
