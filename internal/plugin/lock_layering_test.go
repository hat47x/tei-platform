package plugin_test

import (
	"strings"
	"testing"

	"tei/internal/environment"
	"tei/internal/plugin"
)

// When a lock is supplied the catalog is narrowed to the locked plugins and
// then resolved. If that narrowed catalog cannot satisfy the environment, the
// resolver names the reason, but the layered path replaces it with a mismatch
// built by fmt.Sprintf("%s"). The cause is dropped, so ResolveCode answers ""
// and a caller reading only IsLockMismatch cannot tell an unsatisfied
// requirement from a diverged fingerprint.
//
// A cycle cannot be used to show this: BuildLock refuses a cyclic catalog, so
// no lock that matches a cyclic declaration can exist. A stale lock is
// reachable, and shows the same information loss.
func TestAStaleLockLosesTheResolversOwnCode(t *testing.T) {
	// The lock was built when the environment selected only tei.a.
	stale := environment.Environment{ID: cid("prod")}
	stale.Plugins = []environment.ExtensionRequirement{{ID: cid("tei.a"), CompatibleMajor: 1}}
	a := plugin.Definition{ID: cid("tei.a"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}}
	b := plugin.Definition{ID: cid("tei.b"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}}
	catalog := plugin.Catalog{Plugins: []plugin.Definition{a, b}}
	lock, err := plugin.BuildLock(catalog, []environment.Environment{stale})
	if err != nil {
		t.Fatalf("BuildLock: %v", err)
	}

	// The environment now also selects tei.b, which the lock does not pin.
	current := environment.Environment{ID: cid("prod")}
	current.Plugins = []environment.ExtensionRequirement{
		{ID: cid("tei.a"), CompatibleMajor: 1},
		{ID: cid("tei.b"), CompatibleMajor: 1},
	}

	// Resolved directly, the reason is named.
	_, directErr := catalog.ResolveEnvironment(current)
	if directErr != nil {
		t.Fatalf("the full catalog satisfies the environment: %v", directErr)
	}

	// Resolved through the lock, the narrowed catalog cannot, and the reason is
	// replaced by the wrapper.
	_, lockedErr := catalog.ResolveEnvironmentWithLock(current, lock)
	if lockedErr == nil {
		t.Fatal("the stale lock must be refused")
	}
	if !plugin.IsLockMismatch(lockedErr) {
		t.Fatalf("the layered path reports a mismatch: %v", lockedErr)
	}
	if !strings.Contains(lockedErr.Error(), "locked resolution") {
		t.Fatalf("the message must say which stage failed: %v", lockedErr)
	}
	// This is the property worth pinning: the resolver's own reason does not
	// survive the wrapper.
	if got := plugin.ResolveCode(lockedErr); got != "" {
		t.Fatalf("ResolveCode = %q; if the cause is now wrapped, update this test", got)
	}
	// The message still quotes the resolver's text, so a reader is not blind to
	// the cause even though no code is attached.
	if !strings.Contains(lockedErr.Error(), "no plugin tei.b compatible with major 1") {
		t.Fatalf("the quoted reason must survive in the text: %v", lockedErr)
	}
}

// A lock that pins a plugin the catalog no longer has is reported before any
// resolution runs, so it keeps its own, more specific message.
func TestAMissingLockedPluginIsReportedBeforeResolution(t *testing.T) {
	env := environment.Environment{ID: cid("prod")}
	env.Plugins = []environment.ExtensionRequirement{{ID: cid("tei.a"), CompatibleMajor: 1}}
	definition := plugin.Definition{ID: cid("tei.a"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}}
	lock, err := plugin.BuildLock(plugin.Catalog{Plugins: []plugin.Definition{definition}}, []environment.Environment{env})
	if err != nil {
		t.Fatalf("BuildLock: %v", err)
	}

	// The catalog no longer offers the pinned version.
	_, err = (plugin.Catalog{}).ResolveEnvironmentWithLock(env, lock)
	if err == nil {
		t.Fatal("a lock naming an unavailable plugin must be refused")
	}
	if !plugin.IsLockMismatch(err) {
		t.Fatalf("want a mismatch, got %v", err)
	}
	if !strings.Contains(err.Error(), "is not available") {
		t.Fatalf("the message must name the missing plugin: %v", err)
	}
}

// A usable lock and a usable catalog resolve, so the failures above are about
// the specific defects and not about the layered path always failing.
func TestAUsableLockResolves(t *testing.T) {
	env := environment.Environment{ID: cid("prod")}
	env.Plugins = []environment.ExtensionRequirement{{ID: cid("tei.a"), CompatibleMajor: 1}}
	definition := plugin.Definition{ID: cid("tei.a"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}}
	catalog := plugin.Catalog{Plugins: []plugin.Definition{definition}}
	lock, err := plugin.BuildLock(catalog, []environment.Environment{env})
	if err != nil {
		t.Fatalf("BuildLock: %v", err)
	}
	set, err := catalog.ResolveEnvironmentWithLock(env, lock)
	if err != nil {
		t.Fatalf("a matching lock and catalog must resolve: %v", err)
	}
	if len(set.Plugins) != 1 || set.Plugins[0].ID != cid("tei.a") {
		t.Fatalf("resolved %+v", set.Plugins)
	}
}

// The lock format version is checked before anything else, so an unreadable
// lock is an input problem rather than a mismatch. The two classifications
// lead to different repairs and must not be confused.
func TestAnUnsupportedFormatIsAnInvalidLockNotAMismatch(t *testing.T) {
	lock := plugin.Lock{FormatVersion: 9}
	_, err := (plugin.Catalog{}).ResolveEnvironmentWithLock(environment.Environment{ID: cid("prod")}, lock)
	if err == nil {
		t.Fatal("an unsupported format must be refused")
	}
	if !plugin.IsInvalidLock(err) {
		t.Fatalf("want an invalid-lock classification, got %v", err)
	}
	if plugin.IsLockMismatch(err) {
		t.Fatalf("an unusable format is not a mismatch: %v", err)
	}
}
