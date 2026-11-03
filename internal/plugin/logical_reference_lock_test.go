package plugin_test

import (
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/modelref"
	"tei/internal/plugin"
	"tei/internal/subject"
)

// This compares two existing reference APIs, not a TEI Language/Role integration.
func TestLogicalReferenceAndEnvironmentPluginLockAreSeparate(t *testing.T) {
	key := modelref.Key{Scope: "sales", Subject: subject.Subject{Kind: "data", ID: "order"}}
	nodes := []modelref.Node{{Key: key, DisplayName: "Order"}}
	ref := modelref.Ref{Scope: key.Scope, Subject: key.Subject}
	resolve := func() {
		t.Helper()
		got, diagnostic := modelref.Resolve(nodes, ref)
		if diagnostic != "" || got != key {
			t.Fatalf("resolve = %v, %q", got, diagnostic)
		}
	}
	encoded, err := json.Marshal(ref)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(string(encoded)), "fingerprint") || strings.Contains(strings.ToLower(string(encoded)), "digest") {
		t.Fatalf("logical reference contains a content fingerprint: %s", encoded)
	}
	resolve()

	id := core.MustCanonicalID("tei.example")
	v1 := plugin.Definition{ID: id, Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}}
	env := environment.Environment{ID: core.MustCanonicalID("test"), Plugins: []environment.ExtensionRequirement{{ID: id, CompatibleMajor: 1}}}
	catalog := plugin.Catalog{Plugins: []plugin.Definition{v1}}
	oldLock, err := plugin.BuildLock(catalog, []environment.Environment{env})
	if err != nil {
		t.Fatal(err)
	}
	v2 := v1
	v2.Version = "1.1.0"
	catalog.Plugins = append(catalog.Plugins, v2)
	newLock, err := plugin.BuildLock(catalog, []environment.Environment{env})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		lock plugin.Lock
		want string
	}{{oldLock, "1.0.0"}, {newLock, "1.1.0"}} {
		resolved, err := catalog.ResolveEnvironmentWithLock(env, tc.lock)
		if err != nil || len(resolved.Plugins) != 1 || resolved.Plugins[0].Version != tc.want {
			t.Fatalf("locked resolution = %+v, %v; want %s", resolved, err, tc.want)
		}
		resolve() // The domain reference does not need to be rewritten for either lock.
	}
	changed := plugin.Catalog{Plugins: []plugin.Definition{v1, v2}}
	changed.Plugins[0].Runtime = plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "different"}
	if _, err := changed.ResolveEnvironmentWithLock(env, oldLock); err == nil {
		t.Fatal("modified pinned descriptor was accepted")
	}
	resolve() // Descriptor drift does not change the logical reference itself.
}
