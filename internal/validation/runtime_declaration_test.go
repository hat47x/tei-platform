package validation_test

import (
	"testing"

	"tei/internal/ir"
	"tei/internal/plugin"
	"tei/internal/validation"
)

// codes returns the diagnostic codes a World produces.
func codes(w ir.World) map[string]int {
	seen := map[string]int{}
	for _, d := range validation.Validate(w) {
		seen[d.Code]++
	}
	return seen
}

// A process runtime with no command has nothing to start. The declaration is
// accepted by the type system, so this is the only place that refuses it, and
// nothing pinned it: a plugin with an empty command would have reached the
// runtime plan and failed when it tried to run.
func TestAProcessRuntimeWithoutACommandIsRefused(t *testing.T) {
	w := ir.World{Plugins: []plugin.Definition{{
		ID:      cid("tei.process"),
		Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeProcess},
	}}}
	got := codes(w)
	if got["PLUGIN_PROCESS_COMMAND_REQUIRED"] != 1 {
		t.Fatalf("want one PLUGIN_PROCESS_COMMAND_REQUIRED, got %v", got)
	}
}

// Whitespace is not a command. The check trims before deciding, so a template
// that left a blank value is refused the same way as an absent one.
func TestAWhitespaceCommandIsNotACommand(t *testing.T) {
	w := ir.World{Plugins: []plugin.Definition{{
		ID:      cid("tei.process"),
		Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "   "},
	}}}
	if got := codes(w); got["PLUGIN_PROCESS_COMMAND_REQUIRED"] != 1 {
		t.Fatalf("want one PLUGIN_PROCESS_COMMAND_REQUIRED, got %v", got)
	}
}

// A command makes the declaration usable, so the refusal must not fire.
func TestAProcessRuntimeWithACommandIsAccepted(t *testing.T) {
	w := ir.World{Plugins: []plugin.Definition{{
		ID:      cid("tei.process"),
		Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "./plugin"},
	}}}
	if got := codes(w); got["PLUGIN_PROCESS_COMMAND_REQUIRED"] != 0 {
		t.Fatalf("a command was supplied, so the refusal must not fire: %v", got)
	}
}

// A sandbox policy describes how a separate process is confined. An in-process
// plugin has no process to confine, so naming one is a contradiction: the
// declaration would claim an isolation it does not have.
func TestASandboxPolicyOnANonProcessRuntimeIsRefused(t *testing.T) {
	w := ir.World{Plugins: []plugin.Definition{{
		ID:      cid("tei.builtin"),
		Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin, Sandbox: plugin.SandboxRequired},
	}}}
	got := codes(w)
	if got["PLUGIN_SANDBOX_REQUIRES_PROCESS_RUNTIME"] != 1 {
		t.Fatalf("want one PLUGIN_SANDBOX_REQUIRES_PROCESS_RUNTIME, got %v", got)
	}
}

// The empty policy and "none" both mean no confinement was asked for, so
// neither is a contradiction on a builtin runtime.
func TestNoSandboxOnABuiltinRuntimeIsAccepted(t *testing.T) {
	for _, policy := range []plugin.SandboxPolicy{"", plugin.SandboxNone} {
		w := ir.World{Plugins: []plugin.Definition{{
			ID:      cid("tei.builtin"),
			Version: "1.0.0",
			Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin, Sandbox: policy},
		}}}
		if got := codes(w); got["PLUGIN_SANDBOX_REQUIRES_PROCESS_RUNTIME"] != 0 {
			t.Fatalf("policy %q must not be a contradiction: %v", policy, got)
		}
	}
}

// A sandbox policy on a process runtime is the intended pairing and must be
// accepted, so the refusal above is about the runtime kind, not the policy.
func TestASandboxPolicyOnAProcessRuntimeIsAccepted(t *testing.T) {
	w := ir.World{Plugins: []plugin.Definition{{
		ID:      cid("tei.process"),
		Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "./plugin", Sandbox: plugin.SandboxPreferred},
	}}}
	if got := codes(w); got["PLUGIN_SANDBOX_REQUIRES_PROCESS_RUNTIME"] != 0 {
		t.Fatalf("a process runtime may name a sandbox policy: %v", got)
	}
}

// An unknown runtime kind is refused before either of the checks above can
// decide, so it reports its own code and not a misleading follow-on.
func TestAnUnknownRuntimeKindReportsItsOwnCode(t *testing.T) {
	w := ir.World{Plugins: []plugin.Definition{{
		ID:      cid("tei.wasm"),
		Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeKind("wasm")},
	}}}
	got := codes(w)
	if got["INVALID_PLUGIN_RUNTIME"] != 1 {
		t.Fatalf("want one INVALID_PLUGIN_RUNTIME, got %v", got)
	}
}
