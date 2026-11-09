package messagelocalization

import (
	"strings"
	"testing"

	"tei/internal/environment"
	"tei/internal/plugin"
)

func TestLocalizationPluginRequiresStandardMessageContract(t *testing.T) {
	in := fixture(t, true)
	for i := range in.Plugins.Plugins {
		if in.Plugins.Plugins[i].ID == jp.ID {
			in.Plugins.Plugins[i].Requires = nil
		}
	}
	var err error
	in.Lock, err = plugin.BuildLock(in.Plugins, []environment.Environment{in.Environment})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(in); err == nil || !strings.Contains(err.Error(), "missing-message-plugin-dependency") {
		t.Fatalf("locale plugin without explicit std.message dependency was applied: %v", err)
	}
}

func TestDuplicateTranslationStoreBindingRefused(t *testing.T) {
	in := fixture(t, true)
	in.Bindings = append(in.Bindings, in.Bindings[0])
	if _, err := Resolve(in); err == nil || !strings.Contains(err.Error(), "unapplied-plugin-store") {
		t.Fatalf("duplicated app binding silently selected: %v", err)
	}
}
