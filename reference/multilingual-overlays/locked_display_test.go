package multilingualoverlays

import (
	"reflect"
	"strings"
	"testing"

	"tei/internal/environment"
	"tei/internal/modelref"
	"tei/internal/plugin"
)

func TestO2RealPluginLockSelectsLabelsWithoutModifyingOrdinaryPlugin(t *testing.T) {
	c, env, lock, graph, surfaces, packs, grants, request, err := ReferenceOrderCase()
	if err != nil {
		t.Fatal(err)
	}
	original := graph.Nodes[0]
	for _, tc := range []struct {
		locale string
		text   string
		source bool
	}{
		{"ja-JP", "受注を確定", true},
		{"en-US", "Confirm order", false},
	} {
		request.Locale = tc.locale
		result, err := BuildLockedDisplay(c, env, lock, graph, surfaces, packs, grants, nil, request)
		if err != nil {
			t.Fatalf("%s: %v", tc.locale, err)
		}
		if result.Kind != DisplayManifestKind ||
			result.Environment != string(env.ID) ||
			result.Text != tc.text || result.FromSource != tc.source ||
			result.Locale != tc.locale || result.LockFingerprint == "" ||
			result.SourceRevision != Fingerprint("受注を確定") ||
			result.Target.ID != string(request.Target.ID) || result.Target.Version != request.Target.Version || result.Resource != request.Key {
			t.Fatalf("%s: invalid display manifest: %+v", tc.locale, result)
		}
	}
	if !reflect.DeepEqual(graph.Nodes[0], original) || len(c.Plugins[0].Provides) != 0 {
		t.Fatal("ordinary plugin was modified to enable localization")
	}
}

func TestO2LockedSelectionRejectsStalePluginAndMutation(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*plugin.Catalog, *environment.Environment, *plugin.Lock, *modelref.Graph, *[]Surface, *[]Pack, *[]Grant, *Lookup)
		want string
	}{
		{"plugin fingerprint changed", func(c *plugin.Catalog, _ *environment.Environment, _ *plugin.Lock, _ *modelref.Graph, _ *[]Surface, _ *[]Pack, _ *[]Grant, _ *Lookup) {
			c.Plugins[1].Version = "1.0.1"
		}, "plugin-lock:"},
		{"plugin missing from catalog", func(c *plugin.Catalog, _ *environment.Environment, _ *plugin.Lock, _ *modelref.Graph, _ *[]Surface, _ *[]Pack, _ *[]Grant, _ *Lookup) {
			c.Plugins = c.Plugins[:1]
		}, "plugin-lock:"},
		{"lock targets another environment", func(_ *plugin.Catalog, env *environment.Environment, _ *plugin.Lock, _ *modelref.Graph, _ *[]Surface, _ *[]Pack, _ *[]Grant, _ *Lookup) {
			env.ID = "env.other"
		}, "plugin-lock:"},
		{"source label revised", func(_ *plugin.Catalog, _ *environment.Environment, _ *plugin.Lock, graph *modelref.Graph, _ *[]Surface, _ *[]Pack, _ *[]Grant, _ *Lookup) {
			graph.Nodes[0].DisplayName = "注文を確定"
		}, "stale-translation"},
		{"no environment grant", func(_ *plugin.Catalog, _ *environment.Environment, _ *plugin.Lock, _ *modelref.Graph, _ *[]Surface, _ *[]Pack, grants *[]Grant, _ *Lookup) {
			*grants = nil
		}, "overlay-not-permitted"},
		{"wrong target version", func(_ *plugin.Catalog, _ *environment.Environment, _ *plugin.Lock, _ *modelref.Graph, _ *[]Surface, packs *[]Pack, _ *[]Grant, _ *Lookup) {
			(*packs)[0].Target.Version = "1.3.0"
		}, "incompatible-plugin-version"},
		{"unsupported language", func(_ *plugin.Catalog, _ *environment.Environment, _ *plugin.Lock, _ *modelref.Graph, _ *[]Surface, _ *[]Pack, _ *[]Grant, request *Lookup) {
			request.Locale = "fr-FR"
		}, "missing-translation"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, env, lock, graph, surfaces, packs, grants, request, err := ReferenceOrderCase()
			if err != nil {
				t.Fatal(err)
			}
			tc.edit(&c, &env, &lock, &graph, &surfaces, &packs, &grants, &request)
			manifest, err := BuildLockedDisplay(c, env, lock, graph, surfaces, packs, grants, nil, request)
			if err == nil || !strings.Contains(err.Error(), tc.want) || manifest.Kind != "" {
				t.Fatalf("expected %s fail closed: manifest=%+v, err=%v", tc.want, manifest, err)
			}
		})
	}
}

func TestO2UnselectedLocaleIsNotAHiddenDependency(t *testing.T) {
	c, env, lock, graph, surfaces, packs, grants, request, err := ReferenceOrderCase()
	if err != nil {
		t.Fatal(err)
	}
	// An environment without the locale plugin cannot use its pack even if
	// the untranslated ordinary plugin is still resolvable.
	env.Plugins = env.Plugins[:1]
	lock, err = plugin.BuildLock(c, []environment.Environment{env})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = BuildLockedDisplay(c, env, lock, graph, surfaces, packs, grants, nil, request); err == nil ||
		!strings.Contains(err.Error(), "unselected-locale-plugin") {
		t.Fatalf("missing locale plugin silently used: %v", err)
	}
	// Disabling the locale overlay allows the normal source-language view.
	request.Locale = "ja-JP"
	manifest, err := BuildLockedDisplay(c, env, lock, graph, surfaces, nil, nil, nil, request)
	if err != nil || manifest.Text != "受注を確定" {
		t.Fatalf("ordinary plugin without locale pack was broken: %+v, %v", manifest, err)
	}
}

func TestO2EnvironmentLockRejectsCatalogMutationEvenWithSameVersion(t *testing.T) {
	c, env, lock, graph, surfaces, packs, grants, request, err := ReferenceOrderCase()
	if err != nil {
		t.Fatal(err)
	}
	// The version string is unchanged but the pinned package definition is not.
	c.Plugins[0].Runtime.Kind = plugin.RuntimeProcess
	c.Plugins[0].Runtime.Command = "untrusted"
	if _, err := BuildLockedDisplay(c, env, lock, graph, surfaces, packs, grants, nil, request); err == nil ||
		!strings.Contains(err.Error(), "fingerprint diverged") {
		t.Fatalf("changed descriptor bypassed lock: %v", err)
	}
}
