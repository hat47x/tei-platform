package providerroutes

import "testing"

func readFixture() (ReadSlice, ReadSlice) {
	primary := ReadSlice{
		Route: Path{Provider: "pg", Endpoint: "primary", SemanticData: "sales.order", Scope: "tenant-A"},
		Scope: "tenant-A", Grain: "entity", QueryContract: "sales.order.read@v1",
		SnapshotIssuer: "db-1", SnapshotToken: "s7", Complete: true,
		Rows: []ReadRow{{Identity: "1001", Occurrence: "entity:1001", Presence: "value", Value: "A"}},
	}
	replica := primary
	replica.Route = Path{Provider: "pg", Endpoint: "replica", SemanticData: "sales.order", Scope: "tenant-A"}
	replica.SnapshotIssuer = "replica-db"
	replica.Rows = append([]ReadRow(nil), primary.Rows...)
	return primary, replica
}
func validCohort() AssertedCommonCohort {
	return AssertedCommonCohort{EstablishedByFixture: true, EvidenceLabel: "oracle:fixture-1"}
}
func expectReadStatus(t *testing.T, want string, a, b ReadSlice, premise AssertedCommonCohort) {
	t.Helper()
	x := CompareReadSlices(a, b, premise)
	if x.Status != want {
		t.Fatalf("want %s, got %#v", want, x)
	}
}
func TestRC01SameTokenAcrossProvidersIsNotProof(t *testing.T) {
	a, b := readFixture()
	expectReadStatus(t, "not_established", a, b, AssertedCommonCohort{})
}
func TestRC02IndependentFixturePremiseAllowsEqualBag(t *testing.T) {
	a, b := readFixture()
	expectReadStatus(t, "fixture_consistent", a, b, validCohort())
}
func TestRC03CompleteSameCohortButValueChanged(t *testing.T) {
	a, b := readFixture()
	b.Rows[0].Value = "B"
	expectReadStatus(t, "contradicted", a, b, validCohort())
}
func TestRC04PartialReadCannotEstablishEquivalence(t *testing.T) {
	a, b := readFixture()
	b.Complete = false
	b.Rows = nil
	expectReadStatus(t, "not_established", a, b, validCohort())
}
func TestRC05CompleteMissingRecordContradicts(t *testing.T) {
	a, b := readFixture()
	b.Rows = nil
	expectReadStatus(t, "contradicted", a, b, validCohort())
}
func TestRC06JoinFanoutIsABagNotASet(t *testing.T) {
	a, b := readFixture()
	a.Grain = "joined-occurrence"
	b.Grain = "joined-occurrence"
	a.Rows = append(a.Rows, a.Rows[0])
	b.Rows = append(b.Rows, b.Rows[0])
	b.Rows = b.Rows[:1]
	expectReadStatus(t, "contradicted", a, b, validCohort())
}
func TestRC07NullVersusUnobservedIsUnresolved(t *testing.T) {
	a, b := readFixture()
	a.Rows[0].Presence = "null"
	a.Rows[0].Value = ""
	b.Rows[0].Presence = "absent"
	b.Rows[0].Value = ""
	expectReadStatus(t, "not_established", a, b, validCohort())
}
func TestRC08DifferentGrainNotComparable(t *testing.T) {
	a, b := readFixture()
	b.Grain = "aggregate"
	expectReadStatus(t, "not_established", a, b, validCohort())
}
func TestRC09DifferentScopeNotComparable(t *testing.T) {
	a, b := readFixture()
	b.Scope = "tenant-B"
	expectReadStatus(t, "not_established", a, b, validCohort())
}
func TestRC10DifferentQueryMeaningNotComparable(t *testing.T) {
	a, b := readFixture()
	b.QueryContract = "sales.order.active@v1"
	expectReadStatus(t, "not_established", a, b, validCohort())
}
func TestRC11SameDataWithoutIndependentWitnessNotEstablished(t *testing.T) {
	a, b := readFixture()
	b.SnapshotIssuer = a.SnapshotIssuer
	b.SnapshotToken = a.SnapshotToken
	expectReadStatus(t, "not_established", a, b, AssertedCommonCohort{EvidenceLabel: "some-label"})
}
func TestRC12MissingRowIdentityCannotCompare(t *testing.T) {
	a, b := readFixture()
	b.Rows[0].Occurrence = ""
	expectReadStatus(t, "not_established", a, b, validCohort())
}
func TestRC13OrderIndependentButBagSensitive(t *testing.T) {
	a, b := readFixture()
	x := ReadRow{Identity: "1002", Occurrence: "entity:1002", Presence: "value", Value: "B"}
	a.Rows = append(a.Rows, x)
	b.Rows = append([]ReadRow{x}, b.Rows...)
	expectReadStatus(t, "fixture_consistent", a, b, validCohort())
}
func TestRC14DelimiterInValuesDoesNotCollapseFields(t *testing.T) {
	a, b := readFixture()
	a.Rows[0].Identity = "A\nB"
	a.Rows[0].Value = "C"
	b.Rows[0].Identity = "A"
	b.Rows[0].Value = "B\nC"
	expectReadStatus(t, "contradicted", a, b, validCohort())
}
func TestRC15UnrecognizedPresenceNotEquivalent(t *testing.T) {
	a, b := readFixture()
	b.Rows[0].Presence = "redacted"
	expectReadStatus(t, "not_established", a, b, validCohort())
}

func TestRC16DifferentSemanticSubjectsNotComparable(t *testing.T) {
	a, b := readFixture()
	b.Route.SemanticData = "sales.customer"
	expectReadStatus(t, "not_established", a, b, validCohort())
}
func TestRC17RouteScopeAndReadScopeDisagree(t *testing.T) {
	a, b := readFixture()
	b.Route.Scope = "tenant-B"
	expectReadStatus(t, "not_established", a, b, validCohort())
}

func TestRC18BothUnobservedValuesCannotEstablishEquivalence(t *testing.T) {
	a, b := readFixture()
	a.Rows[0].Presence = "absent"
	b.Rows[0].Presence = "absent"
	expectReadStatus(t, "not_established", a, b, validCohort())
}
func TestRC19ObservedNullDiffersFromObservedValue(t *testing.T) {
	a, b := readFixture()
	a.Rows[0].Presence = "null"
	a.Rows[0].Value = ""
	expectReadStatus(t, "contradicted", a, b, validCohort())
}
func TestRC20BothObservedNullHaveSameFixturePresence(t *testing.T) {
	a, b := readFixture()
	a.Rows[0].Presence = "null"
	a.Rows[0].Value = ""
	b.Rows[0].Presence = "null"
	b.Rows[0].Value = ""
	expectReadStatus(t, "fixture_consistent", a, b, validCohort())
}
