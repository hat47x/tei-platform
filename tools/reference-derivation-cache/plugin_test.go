package derivationcache

import (
	"testing"
)

func TestReusePluginRejectsVersionChange(t *testing.T) {
	report := ReusePlugin(PluginInput{
		CacheRequested: true,
		CachedPlugin:   PluginIdentity{ID: "p", Version: "1.0.0", DefinitionHash: "abc"},
		CurrentPlugin:  PluginIdentity{ID: "p", Version: "1.0.1", DefinitionHash: "abc"},
	})
	if report.Accepted || report.Code != CodeStaleRule || !report.ValueNull {
		t.Fatalf("report=%+v want STALE_RULE with unknown value", report)
	}
}

func TestReusePluginRejectsDefinitionChange(t *testing.T) {
	report := ReusePlugin(PluginInput{
		CacheRequested: true,
		CachedPlugin:   PluginIdentity{ID: "p", Version: "1.0.0", DefinitionHash: "abc"},
		CurrentPlugin:  PluginIdentity{ID: "p", Version: "1.0.0", DefinitionHash: "def"},
	})
	if report.Accepted || report.Code != CodeStaleRule || !report.ValueNull {
		t.Fatalf("report=%+v want STALE_RULE with unknown value", report)
	}
}

func TestReusePluginRejectsLockFingerprintChange(t *testing.T) {
	report := ReusePlugin(PluginInput{
		CacheRequested:         true,
		CachedPlugin:           PluginIdentity{ID: "p", Version: "1.0.0", DefinitionHash: "abc"},
		CurrentPlugin:          PluginIdentity{ID: "p", Version: "1.0.0", DefinitionHash: "abc"},
		CachedLockFingerprint:  "sha256:old",
		CurrentLockFingerprint: "sha256:new",
	})
	if report.Accepted || report.Code != CodeStaleInput || !report.ValueNull {
		t.Fatalf("report=%+v want STALE_INPUT with unknown value", report)
	}
}

func TestReusePluginAllowsUnchangedPlugin(t *testing.T) {
	report := ReusePlugin(PluginInput{
		CacheRequested:         true,
		CachedPlugin:           PluginIdentity{ID: "p", Version: "1.0.0", DefinitionHash: "abc"},
		CurrentPlugin:          PluginIdentity{ID: "p", Version: "1.0.0", DefinitionHash: "abc"},
		CachedLockFingerprint:  "sha256:lock",
		CurrentLockFingerprint: "sha256:lock",
	})
	if !report.Accepted || report.Code != CodeCacheReused || report.ValueNull {
		t.Fatalf("report=%+v want CACHE_REUSED", report)
	}
}
