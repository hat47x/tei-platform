package hostadmission

import (
	"context"
	"errors"
	"strings"
	"testing"
)

const (
	r           = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	tBrowser    = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	tAPI        = "cccccccccccccccccccccccccccccccccccccccc"
	hashBrowser = "sha256:" + "d" + "ddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	hashAPI = "sha256:" + "e" + "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
)

type fixtureHost struct {
	snapshot    Snapshot
	err         error
	reads       int
	cancel      context.CancelFunc
	mutateQuery bool
}

func (f *fixtureHost) ReadCurrentHostAcceptance(
	_ context.Context, requested Selection,
) (Snapshot, error) {
	f.reads++
	if f.cancel != nil {
		f.cancel()
	}
	if f.mutateQuery {
		requested.Units[0].ArtifactDigest = "sha256:" + strings.Repeat("f", 64)
	}
	if f.err != nil {
		return Snapshot{}, f.err
	}
	return f.snapshot, nil
}

func fixtureSelection() Selection {
	return Selection{
		ApplicationID:              "order_entry",
		HandoverBaselineID:         "B",
		BaselineRepositoryRevision: r,
		Units: []Unit{
			{ID: BrowserUnit, TargetBuildRevision: tBrowser, ArtifactDigest: hashBrowser, ArtifactBytes: 250},
			{ID: APIUnit, TargetBuildRevision: tAPI, ArtifactDigest: hashAPI, ArtifactBytes: 75000},
		},
	}
}

func fixtureHostSource() *fixtureHost {
	s := fixtureSelection()
	return &fixtureHost{snapshot: Snapshot{
		Selection:           s,
		SourceAccepted:      true,
		SourceAcceptanceRef: "host-source-acceptance-from-current-store",
		Builds: []BuildAcceptance{
			{UnitID: BrowserUnit, ArtifactDigest: hashBrowser, Accepted: true, EvidenceRef: "current-browser-build-record"},
			{UnitID: APIUnit, ArtifactDigest: hashAPI, Accepted: true, EvidenceRef: "current-api-build-record"},
		},
		DeploymentAccepted:  true,
		DeploymentRef:       "current-host-deployment-decision",
		AuthorityRevision:   "current-authority-v1",
	}}
}

func TestExactCurrentHostAdmissionForTwoIndependentTargets(t *testing.T) {
	source := fixtureHostSource()
	got, err := CheckCurrent(context.Background(), source, fixtureSelection())
	if err != nil {
		t.Fatal(err)
	}
	if source.reads != 1 || got.AuthorityRevision != source.snapshot.AuthorityRevision ||
		got.BuildEvidenceRefs[BrowserUnit] != source.snapshot.Builds[0].EvidenceRef ||
		got.BuildEvidenceRefs[APIUnit] != source.snapshot.Builds[1].EvidenceRef ||
		got.DeploymentRef != source.snapshot.DeploymentRef {
		t.Fatalf("exact per-unit Host evidence was not preserved: %+v", got)
	}
	// A previous successful read is not a capability. The next call MUST
	// re-read the Host's current Source, Builder and Deployment decisions.
	source.snapshot.Builds[1].Accepted = false
	_, err = CheckCurrent(context.Background(), source, fixtureSelection())
	if err == nil || !strings.Contains(err.Error(), "Builder acceptance") || source.reads != 2 {
		t.Fatalf("Host revocation was cached: reads = %d error = %v", source.reads, err)
	}
	source.snapshot.Builds[1].Accepted = true
	source.snapshot.DeploymentAccepted = false
	_, err = CheckCurrent(context.Background(), source, fixtureSelection())
	if err == nil || !strings.Contains(err.Error(), "deployment acceptance") || source.reads != 3 {
		t.Fatalf("Host deployment revocation was cached: reads = %d error = %v", source.reads, err)
	}
}

func TestCurrentHostAdmissionRejectsDriftRevocationAndIncompleteEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Snapshot)
		want   string
	}{
		{"other-Application", func(s *Snapshot) {s.Selection.ApplicationID = "other"}, "Application B/R"},
		{"different-Handover-B", func(s *Snapshot) {s.Selection.HandoverBaselineID = "different"}, "Application B/R"},
		{"different-Handover-R", func(s *Snapshot) {s.Selection.BaselineRepositoryRevision = strings.Repeat("f", 40)}, "Application B/R"},
		{"different-Browser-T", func(s *Snapshot) {s.Selection.Units[0].TargetBuildRevision = tAPI}, "exact T/digest/bytes"},
		{"different-API-T", func(s *Snapshot) {s.Selection.Units[1].TargetBuildRevision = tBrowser}, "exact T/digest/bytes"},
		{"different-API-bytes", func(s *Snapshot) {s.Selection.Units[1].ArtifactBytes++}, "exact T/digest/bytes"},
		{"different-Browser-digest", func(s *Snapshot) {s.Selection.Units[0].ArtifactDigest = hashAPI}, "exact T/digest/bytes"},
		{"extra-Host-unit", func(s *Snapshot) {s.Selection.Units = append(s.Selection.Units, s.Selection.Units[0])}, "unit selection invalid"},
		{"missing-Host-unit", func(s *Snapshot) {s.Selection.Units = s.Selection.Units[:1]}, "unit selection invalid"},
		{"source-revoked", func(s *Snapshot) {s.SourceAccepted = false}, "Source acceptance"},
		{"source-ref-missing", func(s *Snapshot) {s.SourceAcceptanceRef = ""}, "Source acceptance"},
		{"build-api-revoked", func(s *Snapshot) {s.Builds[1].Accepted = false}, "Builder acceptance"},
		{"build-browser-mismatched", func(s *Snapshot) {s.Builds[0].ArtifactDigest = hashAPI}, "Builder acceptance"},
		{"build-evidence-missing", func(s *Snapshot) {s.Builds[0].EvidenceRef = ""}, "Builder acceptance"},
		{"build-duplicate", func(s *Snapshot) {s.Builds[1].UnitID = BrowserUnit}, "duplicate unit"},
		{"build-missing", func(s *Snapshot) {s.Builds = s.Builds[:1]}, "both exact unit builds"},
		{"deployment-revoked", func(s *Snapshot) {s.DeploymentAccepted = false}, "deployment acceptance"},
		{"deployment-ref-missing", func(s *Snapshot) {s.DeploymentRef = ""}, "deployment acceptance"},
		{"authority-revision-missing", func(s *Snapshot) {s.AuthorityRevision = ""}, "authority revision"},
	}{
		t.Run(tc.name, func(t *testing.T) {
			source := fixtureHostSource()
			tc.change(&source.snapshot)
			_, err := CheckCurrent(context.Background(), source, fixtureSelection())
			if err == nil || !strings.Contains(err.Error(), tc.want) || source.reads != 1{
				t.Fatalf("unaccepted Host state admitted: %+v reads = %d error = %v", source.snapshot, source.reads, err)
			}
		})
	}
}

func TestCurrentHostAdmissionFailClosedForSourceFailuresAndInvalidSelections(t *testing.T) {
	if _, err := CheckCurrent(context.Background(), nil, fixtureSelection()); err == nil {
		t.Fatal("missing Host-owned Source accepted")
	}
	source := fixtureHostSource()
	source.err = errors.New("Host source unavailable")
	_, err := CheckCurrent(context.Background(), source, fixtureSelection())
	if err == nil || !strings.Contains(err.Error(), "Host current acceptance unavailable") {
		t.Fatalf("source failure was not refused: %v", err)
	}
	for _, tc := range []struct {
		name   string
		change func(*Selection)
	}{
		{"empty-B", func(s *Selection) {s.HandoverBaselineID = ""}},
		{"invalid-R", func(s *Selection) {s.BaselineRepositoryRevision = "rev"}},
		{"missing-API", func(s *Selection) {s.Units = s.Units[:1]}},
		{"duplicate-API", func(s *Selection) {s.Units[0] = s.Units[1]}},
		{"unknown-unit", func(s *Selection) {s.Units[1].ID = "desktop"}},
		{"unbound-T", func(s *Selection) {s.Units[1].TargetBuildRevision = "T"}},
		{"incorrect-digest", func(s *Selection) {s.Units[0].ArtifactDigest = "not-a-digest"}},
		{"zero-bytes", func(s *Selection) {s.Units[0].ArtifactBytes = 0}},
	}{
		t.Run(tc.name, func(t *testing.T) {
			selected := fixtureSelection()
			tc.change(&selected)
			f := fixtureHostSource()
			if _, err := CheckCurrent(context.Background(), f, selected);err == nil || f.reads != 0{
				t.Fatalf("invalid user selection reached Host read: err = %v reads = %d", err, f.reads)
			}
		})
	}
}

func TestCurrentHostAdmissionContextAndQueryIsolation(t *testing.T) {
	t.Run("before-read", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		source := fixtureHostSource()
		if _, err := CheckCurrent(ctx, source, fixtureSelection());!errors.Is(err, context.Canceled) || source.reads != 0{
			t.Fatalf("cancelled context reached Host: err = %v reads = %d", err, source.reads)
		}
	})
	t.Run("after-read", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		source := fixtureHostSource()
		source.cancel = cancel
		if _, err := CheckCurrent(ctx, source, fixtureSelection());!errors.Is(err, context.Canceled) || source.reads != 1{
			t.Fatalf("cancelled current Host read admitted: err = %v reads = %d", err, source.reads)
		}
	})
	t.Run("Host-cannot-mutate-selection", func(t *testing.T) {
		selected := fixtureSelection()
		source := fixtureHostSource()
		source.mutateQuery = true
		_, err := CheckCurrent(context.Background(), source, selected)
		if err != nil || selected.Units[0].ArtifactDigest != hashBrowser{
			t.Fatalf("Host mutated caller request: err = %v selected = %+v", err, selected)
		}
	})
}

func TestExistingE1TwoUnitMeasurementMapsToSeparateHostSelection(t *testing.T) {
	pair := MeasuredE1Pair{
		ApplicationID:               "order_entry",
		HandoverBaselineID:          "B",
		BaselineRepositoryRevision:  r,
		BrowserAndAPISourceRevision: tAPI,
		BrowserSHA256:               strings.TrimPrefix(hashBrowser, "sha256:"),
		BrowserBytes:                250,
		APISHA256:                   strings.TrimPrefix(hashAPI, "sha256:"),
		APIBytes:                    75000,
	}
	got, err := FromMeasuredE1Pair(pair)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Units) != 2 || got.Units[0].TargetBuildRevision != tAPI ||
		got.Units[1].TargetBuildRevision != tAPI ||
		got.Units[0].ArtifactDigest != hashBrowser ||
		got.Units[1].ArtifactDigest != hashAPI {
		t.Fatalf("existing measured E1 pair was not mapped to exact Host query: %+v", got)
	}
	pair.APISHA256 = "other-digest"
	if _, err := FromMeasuredE1Pair(pair); err == nil {
		t.Fatal("invalid diagnostic measurement produced Host query")
	}
}
