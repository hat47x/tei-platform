package runtimepackaging

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

const digest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func binding(unit, capability string) Binding {
	return Binding{Unit: unit, Capability: capability, Plugin: "tei.example", PluginVersion: "1.0.0", ArtifactDigest: digest, ContractFingerprint: "contract-v1"}
}
func fixture() Input {
	web := binding("browser", "ui")
	web.RemoteRequires = []RemoteCall{{Capability: "action", Unit: "api", Transport: "http-api-v1", TrustBinding: "session-auth-v1"}}
	action := binding("api", "action")
	action.Requires = []string{"db"}
	db := binding("api", "db")
	db.HostImports = []string{"private.database"}
	unused := binding("api", "unused.editor")
	return Input{Units: []Unit{
		{ID: "browser", Host: "web", Roots: []string{"ui"}},
		{ID: "api", Host: "linux/amd64", Roots: []string{"action"}, AvailableImports: []string{"private.database"}},
	}, Bindings: []Binding{unused, db, web, action}}
}
func TestClosureSeparatesBrowserAndAPI(t *testing.T) {
	p, err := Derive(fixture())
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Units) != 2 || len(p.Units[0].Included) != 2 || len(p.Units[1].Included) != 1 {
		t.Fatalf("unexpected plan %+v", p)
	}
	if p.Units[1].Included[0].Capability != "ui" {
		t.Fatalf("unexpected browser bindings %+v", p.Units[1])
	}
	for _, unit := range p.Units {
		for _, b := range unit.Included {
			if b.Capability == "unused.editor" {
				t.Fatal("unreachable plugin leaked")
			}
			if unit.Unit == "browser" && b.Capability == "db" {
				t.Fatal("server-only plugin leaked")
			}
		}
	}
}
func TestOrderIndependence(t *testing.T) {
	x := fixture()
	p1, err := Derive(x)
	if err != nil {
		t.Fatal(err)
	}
	for i, j := 0, len(x.Units)-1; i < j; i, j = i+1, j-1 {
		x.Units[i], x.Units[j] = x.Units[j], x.Units[i]
	}
	for i, j := 0, len(x.Bindings)-1; i < j; i, j = i+1, j-1 {
		x.Bindings[i], x.Bindings[j] = x.Bindings[j], x.Bindings[i]
	}
	p2, err := Derive(x)
	if err != nil {
		t.Fatal(err)
	}
	j1, _ := json.Marshal(p1)
	j2, _ := json.Marshal(p2)
	if !reflect.DeepEqual(j1, j2) {
		t.Fatal("input ordering changed build plan")
	}
}
func TestUnreachableInvalidHostImportNotIncluded(t *testing.T) {
	x := fixture()
	x.Bindings[0].HostImports = []string{"editor-only"}
	if _, err := Derive(x); err != nil {
		t.Fatal(err)
	}
}
func TestMissingRequiredCapability(t *testing.T) {
	x := fixture()
	x.Bindings = x.Bindings[:len(x.Bindings)-1]
	mustFail(t, x, "missing bound capability")
}
func TestNoImplicitRemoteBinding(t *testing.T) {
	x := fixture()
	x.Bindings[2].RemoteRequires[0].Unit = "missing"
	mustFail(t, x, "remote target unit")
}
func TestMissingRemoteTrustBinding(t *testing.T) {
	x := fixture()
	x.Bindings[2].RemoteRequires[0].TrustBinding = ""
	mustFail(t, x, "invalid remote call")
}
func TestHostImportViolation(t *testing.T) {
	x := fixture()
	x.Units[1].AvailableImports = nil
	mustFail(t, x, "host import")
}
func TestDynamicDependencyFails(t *testing.T) {
	x := fixture()
	x.Bindings[2].DynamicOpen = true
	mustFail(t, x, "unbounded dynamic dependency")
}
func TestAmbiguousImplementationFails(t *testing.T) {
	x := fixture()
	x.Bindings = append(x.Bindings, x.Bindings[2])
	mustFail(t, x, "ambiguous binding")
}
func TestInvalidArtifactDigestFails(t *testing.T) {
	x := fixture()
	x.Bindings[2].ArtifactDigest = "unknown"
	mustFail(t, x, "invalid binding")
}
func TestPrebuildPlanCanBeSelectedWithoutArtifact(t *testing.T) {
	x := fixture()
	x.Bindings[2].ArtifactDigest = ""
	x.Bindings[2].BuildPlanFingerprint = strings.Repeat("b", 64)
	p, err := Derive(x)
	if err != nil {
		t.Fatal(err)
	}
	if p.Units[1].Included[0].BuildPlanFingerprint != strings.Repeat("b", 64) {
		t.Fatal("prebuild identity lost")
	}
}
func TestMissingBuildInputAndArtifactFails(t *testing.T) {
	x := fixture()
	x.Bindings[2].ArtifactDigest = ""
	mustFail(t, x, "invalid binding")
}
func TestMalformedBuildFingerprintFails(t *testing.T) {
	x := fixture()
	x.Bindings[2].BuildPlanFingerprint = "unfixed"
	mustFail(t, x, "invalid binding")
}
func TestBuiltinDescriptorFingerprintWithoutArtifact(t *testing.T) {
	x := fixture()
	x.Bindings[2].ArtifactDigest = ""
	x.Bindings[2].PluginFingerprint = strings.Repeat("c", 64)
	p, err := Derive(x)
	if err != nil {
		t.Fatal(err)
	}
	if got := p.Units[1].Included[0].PluginFingerprint; got != strings.Repeat("c", 64) {
		t.Fatalf("locked builtin fingerprint lost: %q", got)
	}
}

func TestRejectsMalformedDescriptorFingerprint(t *testing.T) {
	x := fixture()
	x.Bindings[2].PluginFingerprint = "not-a-fingerprint"
	mustFail(t, x, "invalid binding")
}

func TestCycleClosure(t *testing.T) {
	x := fixture()
	x.Bindings[1].Requires = []string{"action"}
	p, err := Derive(x)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Units[0].Included) != 2 {
		t.Fatal("cycle caused missing or excess dependency")
	}
}
func TestDuplicateUnitFails(t *testing.T) {
	x := fixture()
	x.Units = append(x.Units, x.Units[0])
	mustFail(t, x, "duplicate unit")
}
func TestNoRootsFails(t *testing.T) {
	x := fixture()
	x.Units[0].Roots = nil
	mustFail(t, x, "invalid unit")
}
func TestLocalRemoteUnitConfusionFails(t *testing.T) {
	x := fixture()
	x.Bindings[2].RemoteRequires[0].Unit = "browser"
	mustFail(t, x, "invalid remote call")
}
func mustFail(t *testing.T, x Input, expected string) {
	t.Helper()
	_, err := Derive(x)
	if err == nil || !strings.Contains(err.Error(), expected) {
		t.Fatalf("want %q error; got %v", expected, err)
	}
}
