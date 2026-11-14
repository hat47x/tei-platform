package buildintent_test

import (
	"strings"
	"testing"

	"tei/internal/buildintent"
	"tei/internal/plugin"
)

type changedSources struct {
	change func(*plugin.BuildSource)
}

func (s changedSources) Resolve(req buildintent.IntentSource) (buildintent.ResolvedSource, error) {
	r, err := (fakeSources{}).Resolve(req)
	s.change(&r.Source)
	return r, err
}

type changedBuilders struct {
	fakeBuilders
	change func(*buildintent.BuilderResolution)
}

func (b changedBuilders) Resolve(req buildintent.BuilderRequest) (buildintent.BuilderResolution, error) {
	r, err := b.fakeBuilders.Resolve(req)
	b.change(&r)
	return r, err
}

type changedDependencies struct {
	fakeDependencies
	change func(*plugin.BuildDependency)
}

func (d changedDependencies) Resolve(req buildintent.DependencyRequirement) (plugin.BuildDependency, error) {
	r, err := d.fakeDependencies.Resolve(req)
	r.Version = "1.2.0"
	d.change(&r)
	return r, err
}

// A port returning structurally valid data for a different request must not
// turn that substitution into a receipt for the caller's original intent.
func TestResolveRejectsPortSubstitution(t *testing.T) {
	tests := []struct {
		name   string
		change func(*buildintent.BuildIntent, *buildintent.ResolutionContext)
		want   string
	}{
		{"source kind", func(_ *buildintent.BuildIntent, c *buildintent.ResolutionContext) {
			c.Sources = changedSources{func(s *plugin.BuildSource) { s.Kind = "other" }}
		}, "source"},
		{"source id", func(_ *buildintent.BuildIntent, c *buildintent.ResolutionContext) {
			c.Sources = changedSources{func(s *plugin.BuildSource) { s.ID = "other" }}
		}, "source"},
		{"builder contract", func(_ *buildintent.BuildIntent, c *buildintent.ResolutionContext) {
			c.Builders = changedBuilders{change: func(b *buildintent.BuilderResolution) { b.Identity.ContractVersion = "other/v1" }}
		}, "contract"},
		{"builder provider", func(i *buildintent.BuildIntent, _ *buildintent.ResolutionContext) {
			i.Builder.Provider = &buildintent.ProviderWiring{Plugin: "tei.builder.requested"}
		}, "provider"},
		{"builder implementation", func(i *buildintent.BuildIntent, _ *buildintent.ResolutionContext) {
			i.Builder.Provider = &buildintent.ProviderWiring{Plugin: "tei.builder.resolve_probe", Implementation: "requested"}
		}, "implementation"},
		{"builder pin", func(i *buildintent.BuildIntent, c *buildintent.ResolutionContext) {
			i.Builder.Pin = &buildintent.ArtifactPin{ArtifactDigest: "sha256:" + strings.Repeat("d", 64)}
			c.Builders = changedBuilders{change: func(b *buildintent.BuilderResolution) { b.Pinned = true }}
		}, "pin"},
		{"unrequested pin claim", func(_ *buildintent.BuildIntent, c *buildintent.ResolutionContext) {
			c.Builders = changedBuilders{change: func(b *buildintent.BuilderResolution) { b.Pinned = true }}
		}, "pinned"},
		{"missing pin claim", func(i *buildintent.BuildIntent, _ *buildintent.ResolutionContext) {
			i.Builder.Pin = &buildintent.ArtifactPin{ArtifactDigest: "sha256:" + strings.Repeat("b", 64)}
		}, "pinned"},
		{"dependency kind", func(i *buildintent.BuildIntent, c *buildintent.ResolutionContext) {
			i.Dependencies = []buildintent.DependencyRequirement{{Kind: "library", ID: "dependency", CompatibleMajor: 1}}
			c.Dependencies = changedDependencies{change: func(d *plugin.BuildDependency) { d.Kind = "other" }}
		}, "dependency"},
		{"dependency id", func(i *buildintent.BuildIntent, c *buildintent.ResolutionContext) {
			i.Dependencies = []buildintent.DependencyRequirement{{Kind: "library", ID: "dependency", CompatibleMajor: 1}}
			c.Dependencies = changedDependencies{change: func(d *plugin.BuildDependency) { d.ID = "other" }}
		}, "dependency"},
		{"dependency major", func(i *buildintent.BuildIntent, c *buildintent.ResolutionContext) {
			i.Dependencies = []buildintent.DependencyRequirement{{Kind: "library", ID: "dependency", CompatibleMajor: 2}}
			c.Dependencies = changedDependencies{change: func(_ *plugin.BuildDependency) {}}
		}, "dependency"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i, err := buildintent.ParseIntent([]byte(probeDeclaration))
			if err != nil {
				t.Fatal(err)
			}
			c := probeContext()
			tt.change(&i, &c)
			r, err := buildintent.Resolve(i, c)
			if err == nil {
				t.Fatalf("substitution produced receipt %s", r.PlanFingerprint)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("unexpected error: %v", err)
			}
			if r.PlanFingerprint != "" {
				t.Fatal("failure returned a usable receipt")
			}
		})
	}
}

func TestResolveAcceptsMatchingExplicitRequirements(t *testing.T) {
	for _, major := range []int{0, 1} {
		i, err := buildintent.ParseIntent([]byte(probeDeclaration))
		if err != nil {
			t.Fatal(err)
		}
		i.Builder.Provider = &buildintent.ProviderWiring{Plugin: "tei.builder.resolve_probe", Implementation: "probe"}
		i.Builder.Pin = &buildintent.ArtifactPin{ArtifactDigest: "sha256:" + strings.Repeat("b", 64)}
		i.Dependencies = []buildintent.DependencyRequirement{{Kind: "library", ID: "dependency", CompatibleMajor: major}}
		c := probeContext()
		c.Builders = changedBuilders{change: func(b *buildintent.BuilderResolution) { b.Pinned = true }}
		c.Dependencies = changedDependencies{change: func(d *plugin.BuildDependency) {
			if major == 0 {
				d.Version = "0.2.0"
			}
		}}
		if _, err := buildintent.Resolve(i, c); err != nil {
			t.Fatalf("matching major %d: %v", major, err)
		}
	}
}
