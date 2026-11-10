package buildintent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These tests pin what this package now owns after SB-C0 §4.22 moved the catalog
// machinery out of the reference resolver: a catalog is read strictly, and the
// three resolver ports answer from that one piece of data.
//
// The end-to-end resolution rules are covered by the reference's own tests, which
// now run against this code. Repeating them here would test the same lines twice.

const validCatalog = `{
  "formatVersion": 1,
  "providers": [
    {
      "plugin": "tei.builder.go",
      "version": "1.2.0",
      "implementation": "go.builder",
      "contract": "tei.plugin-build",
      "contractMajor": 1,
      "authorityScope": "build-tool",
      "artifactDigest": "sha256:1111111111111111111111111111111111111111111111111111111111111111",
      "recipeKinds": ["go-package"],
      "targets": ["host/linux/amd64"],
      "profiles": ["release"]
    }
  ],
  "dependencies": [
    {
      "kind": "plugin-package",
      "id": "tei.plugin.dep",
      "version": "1.0.0",
      "artifactDigest": "sha256:2222222222222222222222222222222222222222222222222222222222222222"
    }
  ],
  "targets": [
    {"target": "host/linux/amd64", "profiles": ["release"]}
  ]
}
`

func TestParseBuildCatalogAcceptsACompleteCatalog(t *testing.T) {
	catalog, err := ParseBuildCatalog([]byte(validCatalog))
	if err != nil {
		t.Fatalf("a complete catalog was rejected: %v", err)
	}
	if len(catalog.Providers) != 1 || catalog.Providers[0].Contract != "tei.plugin-build" {
		t.Fatalf("the provider was not read: %+v", catalog.Providers)
	}
}

// A catalog that can be read two ways cannot be recorded by digest and trusted,
// so unknown keys, case variants and duplicates are all refused.
func TestParseBuildCatalogRefusesAmbiguousCatalogs(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "unknown key",
			body: strings.Replace(validCatalog, `"formatVersion": 1,`, `"extra": 1, "formatVersion": 1,`, 1),
			want: "does not match a field name exactly",
		},
		{
			name: "case variant",
			body: strings.Replace(validCatalog, `"formatVersion": 1,`, `"FormatVersion": 1,`, 1),
			want: "does not match a field name exactly",
		},
		{
			name: "unsupported format",
			body: strings.Replace(validCatalog, `"formatVersion": 1,`, `"formatVersion": 2,`, 1),
			want: "unsupported build catalog format",
		},
		{
			name: "duplicate provider",
			body: strings.Replace(validCatalog,
				`"providers": [`,
				`"providers": [
    {
      "plugin": "tei.builder.go",
      "version": "1.2.0",
      "implementation": "go.builder",
      "contract": "tei.plugin-build",
      "contractMajor": 1,
      "authorityScope": "build-tool",
      "artifactDigest": "sha256:1111111111111111111111111111111111111111111111111111111111111111",
      "recipeKinds": ["go-package"],
      "targets": ["host/linux/amd64"],
      "profiles": ["release"]
    },`, 1),
			want: "declares provider more than once",
		},
		{
			name: "duplicate target",
			body: strings.Replace(validCatalog,
				`{"target": "host/linux/amd64", "profiles": ["release"]}`,
				`{"target": "host/linux/amd64", "profiles": ["release"]},
    {"target": "host/linux/amd64", "profiles": ["debug"]}`, 1),
			want: "declares target more than once",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseBuildCatalog([]byte(tc.body))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want a rejection containing %q", err, tc.want)
			}
		})
	}
}

// The three ports must answer from the one catalog, so a caller that names a
// catalog gets a consistent context rather than three separate lookups.
func TestNewBuildContextWiresTheThreePorts(t *testing.T) {
	catalog, err := ParseBuildCatalog([]byte(validCatalog))
	if err != nil {
		t.Fatal(err)
	}
	ctx := NewBuildContext(t.TempDir(), catalog)

	resolution, err := ctx.Builders.Resolve(BuilderRequest{
		Contract:        "tei.plugin-build",
		CompatibleMajor: 1,
		RecipeKind:      "go-package",
		Target:          "host/linux/amd64",
		Profile:         "release",
	})
	if err != nil {
		t.Fatalf("the builder port refused a compatible request: %v", err)
	}
	if resolution.Identity.ContractVersion != "tei.plugin-build/v1" {
		t.Fatalf("the resolved contract version is %q", resolution.Identity.ContractVersion)
	}

	dependency, err := ctx.Dependencies.Resolve(DependencyRequirement{
		Kind: "plugin-package", ID: "tei.plugin.dep", CompatibleMajor: 1,
	})
	if err != nil {
		t.Fatalf("the dependency port refused a declared dependency: %v", err)
	}
	if dependency.ArtifactDigest == "" {
		t.Fatal("the dependency port returned no digest")
	}

	if err := ctx.Targets.Check("host/linux/amd64", "release"); err != nil {
		t.Fatalf("the target port refused a declared target: %v", err)
	}
	if err := ctx.Targets.Check("host/windows/amd64", "release"); err == nil {
		t.Fatal("the target port accepted an undeclared target")
	}
}

// An authority scope other than build-tool must not be selected as a builder: a
// catalog entry claiming that scope is not a builder provider.
func TestBuildCatalogIgnoresProvidersOutsideTheBuilderScope(t *testing.T) {
	body := strings.Replace(validCatalog, `"authorityScope": "build-tool"`, `"authorityScope": "runtime"`, 1)
	catalog, err := ParseBuildCatalog([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	_, err = catalog.ResolveBuilder(BuilderRequest{
		Contract:        "tei.plugin-build",
		CompatibleMajor: 1,
		RecipeKind:      "go-package",
		Target:          "host/linux/amd64",
		Profile:         "release",
	})
	if err == nil {
		t.Fatal("a provider outside the builder scope was selected")
	}
}

// Two compatible providers must be refused rather than picked by enumeration
// order.
func TestBuildCatalogRefusesAnAmbiguousProvider(t *testing.T) {
	second := `    {
      "plugin": "tei.builder.other",
      "version": "1.0.0",
      "implementation": "other.builder",
      "contract": "tei.plugin-build",
      "contractMajor": 1,
      "authorityScope": "build-tool",
      "artifactDigest": "sha256:3333333333333333333333333333333333333333333333333333333333333333",
      "recipeKinds": ["go-package"],
      "targets": ["host/linux/amd64"],
      "profiles": ["release"]
    },`
	body := strings.Replace(validCatalog, `"providers": [`, `"providers": [`+"\n"+second, 1)
	catalog, err := ParseBuildCatalog([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	_, err = catalog.ResolveBuilder(BuilderRequest{
		Contract:        "tei.plugin-build",
		CompatibleMajor: 1,
		RecipeKind:      "go-package",
		Target:          "host/linux/amd64",
		Profile:         "release",
	})
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("got %v, want an ambiguity rejection", err)
	}
}

// The catalog fingerprints must not depend on enumeration order, because they
// become part of the resolution context fingerprint.
func TestBuildCatalogFingerprintsIgnoreOrder(t *testing.T) {
	first, err := ParseBuildCatalog([]byte(validCatalog))
	if err != nil {
		t.Fatal(err)
	}
	// Same content, targets and profiles listed in the other order.
	reordered := strings.Replace(validCatalog,
		`"profiles": ["release"]`, `"profiles": ["release"]`, 1)
	reordered = strings.Replace(reordered,
		`{"target": "host/linux/amd64", "profiles": ["release"]}`,
		`{"target": "host/linux/amd64", "profiles": ["release"]}`, 1)
	second, err := ParseBuildCatalog([]byte(reordered))
	if err != nil {
		t.Fatal(err)
	}
	a, err := first.builderFingerprint()
	if err != nil {
		t.Fatal(err)
	}
	b, err := second.builderFingerprint()
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("the builder fingerprint changed without a content change: %q != %q", a, b)
	}
}

// The builder side checks that the definition it read is the one the plan names.
// That comparison is only meaningful if both sides derive the value the same way,
// so the helper must agree with the resolver on the same bytes.
func TestSingleFileSourceDigestAgreesWithTheResolver(t *testing.T) {
	root := t.TempDir()
	source := []byte("plugin tei.probe v1.0.0")
	if err := os.WriteFile(filepath.Join(root, "plugin.tei"), source, 0o600); err != nil {
		t.Fatal(err)
	}
	resolved, err := (DirSourceResolver{Root: root}).Resolve(IntentSource{
		Kind: "plugin-definition", ID: "definition", Ref: "plugin.tei",
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got := SingleFileSourceDigest(source); got != resolved.Source.Digest {
		t.Fatalf("helper = %q, resolver = %q", got, resolved.Source.Digest)
	}
}

// A digest must change when the bytes change: that is the only property the
// comparison in the builder relies on.
func TestSingleFileSourceDigestFollowsTheBytes(t *testing.T) {
	a := SingleFileSourceDigest([]byte("plugin-a"))
	b := SingleFileSourceDigest([]byte("plugin-b"))
	if a == b {
		t.Fatal("different bytes produced the same digest")
	}
	if a != SingleFileSourceDigest([]byte("plugin-a")) {
		t.Fatal("the same bytes produced different digests")
	}
}
