//go:build server

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/operations"
	"tei/internal/studiohost"
	"tei/internal/threeviewadapter"
	xteathreeview "tei/plugins/xtea-three-view"
)

const studioThreeViewSource = "design/order.xead"

// Independent expectations, written by hand.
const (
	studioWantThreeViewProvider = "tei.plugin.xtea-three-view"
	studioWantThreeViewEntry    = "tei/plugins/xtea-three-view.Navigate"
	studioWantThreeViewDecoy    = "tei/cmd/tei-studio.studioThreeViewDecoy"
)

var studioThreeViewDecoyCalls int

// studioThreeViewDecoy answers exactly as the plugin does, from other code.
func studioThreeViewDecoy(root string, r operations.ThreeViewRequest) (operations.ThreeViewNavigation, error) {
	studioThreeViewDecoyCalls++
	return xteathreeview.Navigate(root, r)
}

func studioThreeViewFixture(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "plugins", "xtea-three-view", "testdata", "workspace")
}

func studioThreeViewTrustedRoot(t *testing.T, root string) string {
	t.Helper()
	absolute, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	trusted, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		t.Fatal(err)
	}
	return trusted
}

// studioThreeViewBridge composes the Bridge the way newStudioHost does, with
// a Host-owned observer on the three-view Host.
func studioThreeViewBridge(t *testing.T, roots []string, threeView *threeviewadapter.Host) (*StudioBridge, *[]threeviewadapter.Invocation) {
	t.Helper()
	seen := &[]threeviewadapter.Invocation{}
	threeView.Observe(func(i threeviewadapter.Invocation) { *seen = append(*seen, i) })
	host, err := studiohost.NewReadOnly(roots)
	if err != nil {
		t.Fatal(err)
	}
	return &StudioBridge{host: composeStudioThreeView(host, threeView)}, seen
}

// The normal Studio Host, as main() composes it, reaches the X-TEA three-view
// plugin: the Bridge method returns what the plugin's own entry returns for
// the same root and request.
func TestNormalStudioBridgeReachesTheThreeViewPlugin(t *testing.T) {
	root := studioThreeViewFixture(t)
	host, err := newStudioHost([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	bridge := &StudioBridge{host: host}
	got := bridge.NavigateThreeView("workspace-1", studioThreeViewSource, "", "")
	want, err := xteathreeview.Navigate(studioThreeViewTrustedRoot(t, root),
		operations.ThreeViewRequest{SourcePath: studioThreeViewSource})
	if err != nil {
		t.Fatal(err)
	}
	if !got.OK || got.Value == nil || got.Failure != nil || !reflect.DeepEqual(*got.Value, want) {
		t.Fatalf("the Bridge answer is not the plugin's answer: %+v", got)
	}
	if got.Value.SchemaVersion != "tei.operations.three-view-navigation/v0" ||
		len(got.Value.Subjects) != 4 || len(got.Value.CRUD) != 1 {
		t.Fatalf("unexpected three-view answer: %+v", got.Value)
	}
}

// The inventory and selection the normal Studio uses enter the plugin's own
// linked function. Selecting a decoy with an identical answer enters the
// decoy instead, so the answer alone does not show which code ran.
func TestStudioThreeViewSelectionDecidesTheEnteredCode(t *testing.T) {
	studioThreeViewDecoyCalls = 0
	root := studioThreeViewFixture(t)
	normal, err := newStudioThreeViewHost()
	if err != nil {
		t.Fatal(err)
	}
	bridge, seen := studioThreeViewBridge(t, []string{root}, normal)
	fromPlugin := bridge.NavigateThreeView("workspace-1", studioThreeViewSource, "", "")
	if !fromPlugin.OK || len(*seen) != 1 {
		t.Fatalf("the normal selection failed: %+v seen=%+v", fromPlugin, *seen)
	}
	call := (*seen)[0]
	if call.SelectedProviderID != studioWantThreeViewProvider || call.RegisteredProviderID != studioWantThreeViewProvider ||
		call.ObservedEntry != studioWantThreeViewEntry || !call.OriginVerified || call.Err != nil ||
		call.HostRoot != studioThreeViewTrustedRoot(t, root) || call.Request.SourcePath != studioThreeViewSource {
		t.Fatalf("the normal Studio did not enter the plugin on the allowlisted root: %+v", call)
	}
	if studioThreeViewDecoyCalls != 0 {
		t.Fatal("the decoy ran under the normal selection")
	}

	inventory := append(studioThreeViewInventory(), threeviewadapter.Provider{
		ID: "three-view-decoy", Entry: studioWantThreeViewDecoy, Navigate: studioThreeViewDecoy,
	})
	decoy, err := threeviewadapter.NewHostSelecting("three-view-decoy", inventory)
	if err != nil {
		t.Fatal(err)
	}
	decoyBridge, decoySeen := studioThreeViewBridge(t, []string{root}, decoy)
	fromDecoy := decoyBridge.NavigateThreeView("workspace-1", studioThreeViewSource, "", "")
	a, _ := json.Marshal(fromPlugin)
	b, _ := json.Marshal(fromDecoy)
	if !fromDecoy.OK || !bytes.Equal(a, b) {
		t.Fatalf("the decoy was expected to answer identically:\n%s\n%s", a, b)
	}
	if studioThreeViewDecoyCalls != 1 || len(*decoySeen) != 1 ||
		(*decoySeen)[0].SelectedProviderID != "three-view-decoy" ||
		(*decoySeen)[0].ObservedEntry != studioWantThreeViewDecoy {
		t.Fatalf("selecting the decoy did not enter it: %+v", *decoySeen)
	}
}

func TestStudioThreeViewEnforcesTheWorkspaceAllowlist(t *testing.T) {
	normal, err := newStudioThreeViewHost()
	if err != nil {
		t.Fatal(err)
	}
	bridge, seen := studioThreeViewBridge(t, []string{studioThreeViewFixture(t)}, normal)
	for _, id := range []string{"workspace-2", "", "../workspace", "workspace-1/../"} {
		r := bridge.NavigateThreeView(id, studioThreeViewSource, "", "")
		if r.OK || r.Failure == nil || r.Failure.Code != "WORKSPACE_NOT_ALLOWED" {
			t.Fatalf("three-view accepted workspace %q: %+v", id, r)
		}
	}
	if len(*seen) != 0 {
		t.Fatalf("a refused workspace reached the Provider: %+v", *seen)
	}
}

// A host composed without a three-view Provider says so. It does not read the
// source some other way.
func TestStudioWithoutAThreeViewProviderIsUnavailable(t *testing.T) {
	host, err := studiohost.NewReadOnly([]string{studioThreeViewFixture(t)})
	if err != nil {
		t.Fatal(err)
	}
	r := (&StudioBridge{host: host}).NavigateThreeView("workspace-1", studioThreeViewSource, "", "")
	if r.OK || r.Value != nil || r.Failure == nil || r.Failure.Code != "THREE_VIEW_UNAVAILABLE" ||
		r.Failure.Kind != "business_rejection" {
		t.Fatalf("want THREE_VIEW_UNAVAILABLE, got %+v", r)
	}
}

// Failures carry the Operation's closed codes and nothing of the root.
func TestStudioThreeViewFailuresAreTypedAndDoNotDiscloseTheRoot(t *testing.T) {
	root := studioThreeViewFixture(t)
	normal, err := newStudioThreeViewHost()
	if err != nil {
		t.Fatal(err)
	}
	bridge, _ := studioThreeViewBridge(t, []string{root}, normal)
	trusted := studioThreeViewTrustedRoot(t, root)
	for path, want := range map[string]string{
		"design/absent.xead": "THREE_VIEW_SOURCE_READ_FAILED",
		"tei.workspace.yaml": "THREE_VIEW_UNSUPPORTED_SOURCE",
		"../order.xead":      "THREE_VIEW_SOURCE_READ_FAILED",
	} {
		r := bridge.NavigateThreeView("workspace-1", path, "", "")
		raw, _ := json.Marshal(r)
		if r.OK || r.Failure == nil || r.Failure.Code != want {
			t.Fatalf("%s: want %s, got %s", path, want, raw)
		}
		if bytes.Contains(raw, []byte(trusted)) || bytes.Contains(raw, []byte(filepath.Base(trusted))) {
			t.Fatalf("%s: the failure discloses the root: %s", path, raw)
		}
	}
}

// The pin travels through the Bridge unchanged. Reading again, or reading
// without it, does not alter it; a changed source makes it stale and it is
// not moved to the element that now carries the same ID.
func TestStudioThreeViewPinSurvivesReselectionAndGoesStaleOnSourceChange(t *testing.T) {
	root := t.TempDir()
	fixture, err := os.ReadFile(filepath.Join(studioThreeViewFixture(t), "design", "order.xead"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "design"), 0755); err != nil {
		t.Fatal(err)
	}
	manifest := "workspace: pin\nformatVersion: 1\nsources:\n  - design\n"
	for path, body := range map[string][]byte{
		"tei.workspace.yaml": []byte(manifest), filepath.Join("design", "order.xead"): fixture,
	} {
		if err := os.WriteFile(filepath.Join(root, path), body, 0644); err != nil {
			t.Fatal(err)
		}
	}
	normal, err := newStudioThreeViewHost()
	if err != nil {
		t.Fatal(err)
	}
	bridge, _ := studioThreeViewBridge(t, []string{root}, normal)
	first := bridge.NavigateThreeView("workspace-1", studioThreeViewSource, "", "")
	if !first.OK {
		t.Fatalf("first read failed: %+v", first)
	}
	var pinned operations.ThreeViewSubject
	for _, subject := range first.Value.Subjects {
		if subject.View == operations.ThreeViewFunction {
			pinned = subject
		}
	}
	digest := first.Value.Source.ArtifactDigest
	held := bridge.NavigateThreeView("workspace-1", studioThreeViewSource, pinned.Occurrence, digest)
	// A read without the pin is the user looking elsewhere; the pin is the
	// caller's value and the next read with it still finds it held.
	_ = bridge.NavigateThreeView("workspace-1", studioThreeViewSource, "", "")
	again := bridge.NavigateThreeView("workspace-1", studioThreeViewSource, pinned.Occurrence, digest)
	for _, r := range []studiohost.ThreeViewResponse{held, again} {
		if !r.OK || r.Value.Pin == nil || r.Value.Pin.Status != operations.ThreeViewPinHeld ||
			r.Value.Pin.Occurrence != pinned.Occurrence || r.Value.Pin.SourceArtifactDigest != digest {
			t.Fatalf("the pin was not held: %+v", r.Value)
		}
	}

	changed := bytes.Replace(fixture, []byte(`Name="Order entry"`), []byte(`Name="Order entry (revised)"`), 1)
	if bytes.Equal(changed, fixture) {
		t.Fatal("the fixture was not changed")
	}
	if err := os.WriteFile(filepath.Join(root, "design", "order.xead"), changed, 0644); err != nil {
		t.Fatal(err)
	}
	stale := bridge.NavigateThreeView("workspace-1", studioThreeViewSource, pinned.Occurrence, digest)
	if !stale.OK || stale.Value.Pin == nil || stale.Value.Pin.Status != operations.ThreeViewPinStaleSourceChanged ||
		stale.Value.Pin.Occurrence != pinned.Occurrence || stale.Value.Pin.SourceArtifactDigest != digest {
		t.Fatalf("a pin on changed bytes was not reported stale and unchanged: %+v", stale.Value)
	}
	for _, subject := range stale.Value.Subjects {
		if subject.Occurrence == pinned.Occurrence {
			t.Fatal("the pinned occurrence still resolves after the source changed")
		}
	}
}
