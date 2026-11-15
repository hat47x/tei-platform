package buildintent

import (
	"strings"
	"testing"
)

// The non-Plugin Studio Build Subject is NOT yet supported. This executable
// baseline prevents a go-command Recipe from being mistaken for a qualified
// non-Plugin Build Plan before the typed subject / artifact contract is wired.
// Replace this assertion with acceptance + a Stage1->Stage2 integration test
// only when the product owner has approved that complete binding.
func TestNonPluginCommandBuildIntentRequiresTypedSubjectPlan(t *testing.T) {
	const command = `{
        "formatVersion": 1,
        "subject": {
            "kind": "command",
            "id": "tei.studio.desktop",
            "version": "1.0.0",
            "definitionSource": "definition"
        },
        "sources": [{
            "kind": "go-package",
            "id": "definition",
            "ref": "cmd/tei-studio"
        }],
        "builder": {
            "contract": "tei.command-build",
            "compatibleMajor": 1
        },
        "recipe": {
            "kind": "go-command",
            "entrypoint": "./cmd/tei-studio"
        },
        "outputKind": "runtime-artifact",
        "target": "host/windows/amd64"
	}`
	if _, err := ParseIntent([]byte(command)); err == nil ||
		!strings.Contains(err.Error(), `subject kind "command" is not supported`) {
		t.Fatalf("non-Plugin command must not silently pass through the Plugin-only plan: %v", err)
	}
}
