package validation

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/database/persistence"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// A calendar date stored as a moment in time is refused. The cases are
// written from the rule, not from the validator: an attribute is a calendar
// date when its type is std.date or an alias that leads to it, and the only
// storage that is a moment in time is datetime-tz.

const calendarDateStoredAsMoment = "PERSISTENCE_CALENDAR_DATE_STORED_AS_MOMENT"

func membershipWorld(attributeType core.CanonicalID, storage persistence.StorageType) ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{{
			ID: "std",
			Types: []semantic.Type{
				{ID: "std.identifier", Symbol: "Identifier", Kind: semantic.TypePrimitive},
				{ID: "std.text", Symbol: "Text", Kind: semantic.TypePrimitive},
				{ID: "std.date", Symbol: "Date", Kind: semantic.TypePrimitive},
				{ID: "std.datetime", Symbol: "DateTime", Kind: semantic.TypePrimitive},
			},
		}, {
			ID: "organization",
			Types: []semantic.Type{
				{ID: "organization.business_day", Symbol: "BusinessDay", Kind: semantic.TypeAlias, Underlying: "std.date"},
				{ID: "organization.closing_day", Symbol: "ClosingDay", Kind: semantic.TypeAlias, Underlying: "organization.business_day"},
				{ID: "organization.registered_at", Symbol: "RegisteredAt", Kind: semantic.TypeAlias, Underlying: "std.datetime"},
			},
			Data: []semantic.Data{{
				ID: "organization.membership", Symbol: "Membership",
				Attributes: []semantic.Attribute{
					{ID: "organization.membership.person_id", Symbol: "personId", TypeRef: "std.identifier"},
					{ID: "organization.membership.starts_on", Symbol: "startsOn", TypeRef: attributeType},
				},
			}},
		}},
		Databases: []database.Database{{ID: "org_store", Kind: "relational", Ownership: database.OwnershipManagedHere, SemanticUses: []core.CanonicalID{"organization"}}},
		Persistence: []persistence.Mapping{{
			ID: "org_store.membership", Database: "org_store", SemanticData: "organization.membership", Table: "public.membership",
			Columns: []persistence.ColumnMapping{
				{Attribute: "organization.membership.person_id", Column: "person_id", Storage: persistence.StorageVarchar, Nullability: persistence.Required},
				{Attribute: "organization.membership.starts_on", Column: "starts_on", Storage: storage, Nullability: persistence.Optional},
			},
			PrimaryKey: []core.CanonicalID{"organization.membership.person_id"},
		}},
	}
}

func codes(world ir.World) []string {
	var out []string
	for _, d := range Validate(world) {
		out = append(out, d.Code)
	}
	return out
}

func TestACalendarDateIsNotStoredAsAMoment(t *testing.T) {
	cases := []struct {
		name    string
		typ     core.CanonicalID
		storage persistence.StorageType
		refused bool
	}{
		{"a date as a moment", "std.date", persistence.StorageDateTimeTZ, true},
		{"an alias of a date as a moment", "organization.business_day", persistence.StorageDateTimeTZ, true},
		{"an alias of an alias of a date as a moment", "organization.closing_day", persistence.StorageDateTimeTZ, true},
		{"a date as varchar", "std.date", persistence.StorageVarchar, false},
		{"a date as text", "std.date", persistence.StorageText, false},
		{"an alias of a date as text", "organization.closing_day", persistence.StorageText, false},
		{"a moment as a moment", "std.datetime", persistence.StorageDateTimeTZ, false},
		{"an alias of a moment as a moment", "organization.registered_at", persistence.StorageDateTimeTZ, false},
		{"a text as a moment", "std.text", persistence.StorageDateTimeTZ, false},
	}
	for _, c := range cases {
		world := membershipWorld(c.typ, c.storage)
		count := 0
		for _, code := range codes(world) {
			if code == calendarDateStoredAsMoment {
				count++
			}
		}
		want := 0
		if c.refused {
			want = 1
		}
		if count != want {
			t.Errorf("%s: reported %d times, want %d; all codes: %v", c.name, count, want, codes(world))
		}
	}
}

// Without the storage in question the world of these cases is accepted, so a
// case above is refused for the storage and for nothing else.
func TestTheMembershipWorldIsOtherwiseAccepted(t *testing.T) {
	for _, d := range Validate(membershipWorld("std.date", persistence.StorageText)) {
		if d.Severity == Error {
			t.Errorf("unexpected %s: %s", d.Code, d.Message)
		}
	}
	// With it, the one error is this one.
	var errors []string
	for _, d := range Validate(membershipWorld("std.date", persistence.StorageDateTimeTZ)) {
		if d.Severity == Error {
			errors = append(errors, d.Code)
		}
	}
	if len(errors) != 1 || errors[0] != calendarDateStoredAsMoment {
		t.Fatalf("errors=%v", errors)
	}
}

func TestTheMessageSaysWhatToWriteInstead(t *testing.T) {
	message := diagnosticMessage(t, membershipWorld("organization.business_day", persistence.StorageDateTimeTZ), calendarDateStoredAsMoment)
	for _, part := range []string{
		"org_store.membership", "organization.membership.starts_on", "datetime-tz",
		"varchar", "text", "YYYY-MM-DD", "std.datetime",
	} {
		if !strings.Contains(message, part) {
			t.Errorf("the message does not name %q: %s", part, message)
		}
	}
}

// The attribute of a column that names no known attribute is reported as
// unknown, and not also as a date.
func TestAnUnknownAttributeIsNotJudgedAsADate(t *testing.T) {
	world := membershipWorld("std.date", persistence.StorageDateTimeTZ)
	world.Persistence[0].Columns[1].Attribute = "organization.membership.ends_on"
	got := strings.Join(codes(world), " ")
	if strings.Contains(got, calendarDateStoredAsMoment) || !strings.Contains(got, "UNKNOWN_PERSISTENCE_ATTRIBUTE") {
		t.Fatalf("codes=%s", got)
	}
}
