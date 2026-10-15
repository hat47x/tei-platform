package derivationsnapshot

import (
	"errors"
	"math/big"
	"testing"

	"tei/internal/semantic/derivationref"
)

func present(decimal string) derivationref.Input {
	return derivationref.Input{State: derivationref.Present, Decimal: decimal}
}

// available computes inventory - reserved for the synthetic fixture. The rule
// body itself belongs to D1; the test only combines the checked reads.
func available(t *testing.T, result ReadSetResult) string {
	t.Helper()
	values := map[string]derivationref.Input{}
	for _, read := range result.Reads {
		values[read.InputID] = read.Value
	}
	inventory, ok := new(big.Rat).SetString(values["inventory"].Decimal)
	if !ok {
		t.Fatalf("inventory decimal: %+v", values["inventory"])
	}
	reserved, ok := new(big.Rat).SetString(values["reserved"].Decimal)
	if !ok {
		t.Fatalf("reserved decimal: %+v", values["reserved"])
	}
	return new(big.Rat).Sub(inventory, reserved).RatString()
}

// R0: inventory 120, reserved 100. R1: inventory 110, reserved 90.
func r0() []Read {
	return []Read{
		{InputID: "inventory", Provider: "db", Snapshot: "R0", Value: present("120")},
		{InputID: "reserved", Provider: "db", Snapshot: "R0", Value: present("100")},
	}
}

func r1() []Read {
	return []Read{
		{InputID: "inventory", Provider: "db", Snapshot: "R1", Value: present("110")},
		{InputID: "reserved", Provider: "db", Snapshot: "R1", Value: present("90")},
	}
}

func mixed() []Read {
	return []Read{
		{InputID: "inventory", Provider: "db", Snapshot: "R0", Value: present("120")},
		{InputID: "reserved", Provider: "db", Snapshot: "R1", Value: present("90")},
	}
}

func TestD11R0PinnedReadIsConfirmed(t *testing.T) {
	result, err := Evaluate(SameTransactionSnapshot, false, r0())
	if err != nil || result.Verdict != Guaranteed || !result.Confirmed {
		t.Fatalf("R0 pinned read: %+v, %v", result, err)
	}
	if got := available(t, result); got != "20" {
		t.Fatalf("R0 available = %s, want 20", got)
	}
}

func TestD11R1PinnedReadIsConfirmed(t *testing.T) {
	result, err := Evaluate(SameTransactionSnapshot, false, r1())
	if err != nil || !result.Confirmed {
		t.Fatalf("R1 pinned read: %+v, %v", result, err)
	}
	if got := available(t, result); got != "20" {
		t.Fatalf("R1 available = %s, want 20", got)
	}
}

func TestD11MixedSnapshotIsNotAConfirmedValue(t *testing.T) {
	result, err := Evaluate(SameTransactionSnapshot, false, mixed())
	if !errors.Is(err, ErrMixedSnapshot) {
		t.Fatalf("mixed snapshot was not rejected: %v", err)
	}
	if result.Confirmed || len(result.Reads) != 0 {
		t.Fatalf("rejected mix leaked a confirmed value: %+v", result)
	}
}

func TestD11SameProviderMixRejectedButCrossProviderUnsupported(t *testing.T) {
	if _, err := Evaluate(SameTransactionSnapshot, false, mixed()); !errors.Is(err, ErrMixedSnapshot) {
		t.Fatalf("same-provider mix: %v", err)
	}
	crossProvider := []Read{
		{InputID: "inventory", Provider: "A", Snapshot: "R0", Value: present("120")},
		{InputID: "reserved", Provider: "B", Snapshot: "R1", Value: present("90")},
	}
	result, err := Evaluate(SameTransactionSnapshot, false, crossProvider)
	if err != nil || result.Verdict != Unsupported || result.Confirmed {
		t.Fatalf("cross-provider strong guarantee was not Unsupported: %+v, %v", result, err)
	}
}

func TestD11BestEffortAndAcceptedNonAtomicAreNotConfirmed(t *testing.T) {
	best, err := Evaluate(BestEffort, false, mixed())
	if err != nil || best.Verdict != NonAtomic || best.Confirmed {
		t.Fatalf("best-effort claimed a strong guarantee: %+v, %v", best, err)
	}
	accepted, err := Evaluate(SameTransactionSnapshot, true, mixed())
	if err != nil || accepted.Verdict != NonAtomic || accepted.Confirmed {
		t.Fatalf("accepted non-atomic read was labelled confirmed: %+v, %v", accepted, err)
	}
}

func TestD11LostSnapshotIsUnavailableNotZero(t *testing.T) {
	reads := []Read{
		{InputID: "inventory", Provider: "db", Snapshot: "R0", Value: present("120")},
		{InputID: "reserved", Provider: "db", Snapshot: "R0", Err: ErrSnapshotLost},
	}
	result, err := Evaluate(SameTransactionSnapshot, false, reads)
	if !errors.Is(err, ErrInputUnavailable) || !errors.Is(err, ErrSnapshotLost) {
		t.Fatalf("lost snapshot was not reported as unavailable: %v", err)
	}
	if result.Confirmed || len(result.Reads) != 0 {
		t.Fatalf("lost snapshot leaked a value: %+v", result)
	}

	// A declared NULL input is legitimate and is not an unavailable read.
	if _, err := Evaluate(BestEffort, false, []Read{
		{InputID: "inventory", Provider: "db", Value: present("120")},
		{InputID: "reserved", Provider: "db", Value: derivationref.Input{State: derivationref.Null}},
	}); err != nil {
		t.Fatalf("declared NULL was treated as unavailable: %v", err)
	}
}

func TestD11VersionAlignedNeedsVersions(t *testing.T) {
	aligned := []Read{
		{InputID: "inventory", Provider: "db", Version: "v7", Value: present("120")},
		{InputID: "reserved", Provider: "db", Version: "v7", Value: present("100")},
	}
	result, err := Evaluate(VersionAligned, false, aligned)
	if err != nil || result.Verdict != Guaranteed || !result.Confirmed {
		t.Fatalf("version-aligned read: %+v, %v", result, err)
	}
	unversioned := r0()
	result, err = Evaluate(VersionAligned, false, unversioned)
	if err != nil || result.Verdict != Unsupported || result.Confirmed {
		t.Fatalf("unversioned set was treated as aligned: %+v, %v", result, err)
	}
}

func TestD11ReReadRequiresFullSetAndCurrentAuthorization(t *testing.T) {
	previous, err := Evaluate(SameTransactionSnapshot, false, r0())
	if err != nil || !previous.Confirmed {
		t.Fatalf("previous read: %+v, %v", previous, err)
	}

	if _, err := ReReadAfterConflict(previous, "a1", "a2", SameTransactionSnapshot, false, r1()); !errors.Is(err, ErrStaleAuthorization) {
		t.Fatalf("changed authorization was not re-checked: %v", err)
	}

	partial := []Read{
		{InputID: "inventory", Provider: "db", Snapshot: "R1", Value: present("110")},
	}
	if _, err := ReReadAfterConflict(previous, "a1", "a1", SameTransactionSnapshot, false, partial); !errors.Is(err, ErrPartialReadSet) {
		t.Fatalf("partial re-read was accepted: %v", err)
	}

	result, err := ReReadAfterConflict(previous, "a1", "a1", SameTransactionSnapshot, false, r1())
	if err != nil || !result.Confirmed {
		t.Fatalf("full re-read failed: %+v, %v", result, err)
	}
	if got := available(t, result); got != "20" {
		t.Fatalf("re-read available = %s, want 20", got)
	}
}

func TestD11InvalidInputsAreRejected(t *testing.T) {
	if _, err := Evaluate(Consistency("nonsense"), false, r0()); !errors.Is(err, ErrInvalidRequirement) {
		t.Fatalf("unknown requirement accepted: %v", err)
	}
	if _, err := Evaluate(SameTransactionSnapshot, false, nil); !errors.Is(err, ErrInputUnavailable) {
		t.Fatalf("empty read set accepted: %v", err)
	}
	duplicate := []Read{
		{InputID: "inventory", Provider: "db", Snapshot: "R0", Value: present("120")},
		{InputID: "inventory", Provider: "db", Snapshot: "R0", Value: present("110")},
	}
	if _, err := Evaluate(SameTransactionSnapshot, false, duplicate); !errors.Is(err, ErrInvalidRequirement) {
		t.Fatalf("duplicate input accepted: %v", err)
	}
}
