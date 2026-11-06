package xeadsource

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestExtractRelationalSourceFactsPreservesLegacyKeyAndRelationshipReferences(t *testing.T) {
	doc, err := Parse(FormatXead, []byte(`<System><Table ID="T1"><TableField ID="F1" AttributeType="DERIVABLE" DataTypeID="DT1"/><TableField ID="F2" AttributeType="NATIVE" DataTypeID="DT2"/><TableKey ID="K1" Type="PK"><TableKeyField FieldID="F1" SortKey="10"/><TableKeyField FieldID="F2" SortKey="20"/></TableKey></Table><Relationship ID="R1" Type="FAMILY" Table1ID="T1" TableKey1ID="K1" Table2ID="T2" TableKey2ID="K2"/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	got := ExtractRelationalSourceFacts(doc)
	if len(got.LegacyKeys) != 1 {
		t.Fatalf("legacy key count = %d: %#v", len(got.LegacyKeys), got)
	}
	key := got.LegacyKeys[0]
	if key.TableLegacyID != "T1" || key.KeyLegacyID != "K1" || key.KeyType != "PK" || key.Nature != SourceFactNature {
		t.Fatalf("legacy key source fact drifted: %#v", key)
	}
	if !reflect.DeepEqual(key.Members, []LegacyKeyMemberFact{
		{Locator: "/System[1]/Table[1]/TableKey[3]/TableKeyField[1]", FieldLegacyID: "F1", AttributeType: "DERIVABLE", DataTypeLegacyID: "DT1", SortKey: "10", SourceOrdinal: 0},
		{Locator: "/System[1]/Table[1]/TableKey[3]/TableKeyField[2]", FieldLegacyID: "F2", AttributeType: "NATIVE", DataTypeLegacyID: "DT2", SortKey: "20", SourceOrdinal: 1},
	}) {
		t.Fatalf("legacy composite key membership drifted: %#v", key.Members)
	}
	if !reflect.DeepEqual(key.AuthorityLimits, legacyKeyAuthorityLimits) {
		t.Fatalf("legacy key authority limits drifted: %#v", key.AuthorityLimits)
	}

	if len(got.RelationshipKeys) != 1 {
		t.Fatalf("relationship key count = %d: %#v", len(got.RelationshipKeys), got)
	}
	relation := got.RelationshipKeys[0]
	if relation.RelationshipID != "R1" ||
		relation.RelationshipType != "FAMILY" ||
		relation.Table1LegacyID != "T1" ||
		relation.Key1LegacyID != "K1" ||
		relation.Table2LegacyID != "T2" ||
		relation.Key2LegacyID != "K2" ||
		relation.Nature != SourceFactNature {
		t.Fatalf("relationship key source fact drifted: %#v", relation)
	}
	if !reflect.DeepEqual(relation.AuthorityLimits, relationshipKeyAuthorityLimits) {
		t.Fatalf("relationship authority limits drifted: %#v", relation.AuthorityLimits)
	}
}

func TestExtractRelationalSourceFactsPreservesRawRelationshipType(t *testing.T) {
	for _, kind := range []string{"FAMILY", "REFFER", "FUTURE_KIND"} {
		doc, err := Parse(FormatXead, []byte(`<System><Relationship ID="R1" Type="`+kind+`"/></System>`))
		if err != nil {
			t.Fatal(err)
		}
		facts := ExtractRelationalSourceFacts(doc)
		if len(facts.RelationshipKeys) != 1 {
			t.Fatalf("type %q: relationship count = %d", kind, len(facts.RelationshipKeys))
		}
		relation := facts.RelationshipKeys[0]
		if relation.RelationshipType != kind || relation.Nature != SourceFactNature {
			t.Fatalf("type %q lost its source value or authority limit: %#v", kind, relation)
		}
		encoded, err := json.Marshal(relation)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(encoded), `"relationshipType":"`+kind+`"`) {
			t.Fatalf("type %q absent from source-fact JSON: %s", kind, encoded)
		}
	}
}

func TestExtractRelationalSourceFactsIgnoresNamespacedRelationshipTypeLookalike(t *testing.T) {
	doc, err := Parse(FormatXead, []byte(`<System xmlns:x="urn:foreign"><Relationship ID="R1" x:Type="FAMILY"/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	got := ExtractRelationalSourceFacts(doc)
	if len(got.RelationshipKeys) != 1 || got.RelationshipKeys[0].RelationshipType != "" {
		t.Fatalf("foreign namespaced Type was promoted to source relationship type: %#v", got.RelationshipKeys)
	}
}

func TestExtractRelationalSourceFactsPreservesExistWhenPresenceAndLiteralValue(t *testing.T) {
	doc, err := Parse(FormatXead, []byte(`<System><Relationship ID="R1" Type="REFFER" ExistWhen1="" ExistWhen2="  A &amp; B  "/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	got := ExtractRelationalSourceFacts(doc)
	if len(got.RelationshipKeys) != 1 {
		t.Fatalf("relationship count = %d", len(got.RelationshipKeys))
	}
	relation := got.RelationshipKeys[0]
	if relation.ExistWhen1 == nil || *relation.ExistWhen1 != "" ||
		relation.ExistWhen2 == nil || *relation.ExistWhen2 != "  A & B  " {
		t.Fatalf("present empty or literal non-empty source attribute was lost: %#v", relation)
	}
	encoded, err := json.Marshal(relation)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatal(err)
	}
	if value, exists := object["existWhen1"]; !exists || value != "" {
		t.Fatalf("explicit empty ExistWhen1 not preserved in JSON: %s", encoded)
	}
	if value, exists := object["existWhen2"]; !exists || value != "  A & B  " {
		t.Fatalf("literal ExistWhen2 not preserved in JSON: %s", encoded)
	}
}

func TestExtractRelationalSourceFactsDoesNotInventExistWhenOnAbsentOrForeignAttribute(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
	}{
		{name: "both missing", source: `<System><Relationship ID="R1"/></System>`},
		{name: "foreign namespace", source: `<System xmlns:x="urn:foreign"><Relationship ID="R1" x:ExistWhen1="A" x:ExistWhen2="B"/></System>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := Parse(FormatXead, []byte(tc.source))
			if err != nil {
				t.Fatal(err)
			}
			got := ExtractRelationalSourceFacts(doc)
			if len(got.RelationshipKeys) != 1 {
				t.Fatalf("relationship count = %d", len(got.RelationshipKeys))
			}
			relation := got.RelationshipKeys[0]
			if relation.ExistWhen1 != nil || relation.ExistWhen2 != nil {
				t.Fatalf("absent or foreign namespaced attribute was promoted: %#v", relation)
			}
			encoded, err := json.Marshal(relation)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(encoded), `"existWhen1"`) || strings.Contains(string(encoded), `"existWhen2"`) {
				t.Fatalf("absent declarations were invented in JSON: %s", encoded)
			}
		})
	}
}

func TestExtractRelationalSourceFactsKeepsDeclaredKeyFieldsSeparateFromDataIdentity(t *testing.T) {
	doc, err := Parse(FormatXeaf, []byte(`<System><Function ID="F1" KeyFields="A; B;"><Detail ID="D1" KeyFields="H;L;"/></Function></System>`))
	if err != nil {
		t.Fatal(err)
	}
	got := ExtractRelationalSourceFacts(doc)
	if len(got.DeclaredKeyFields) != 2 {
		t.Fatalf("declared key field count = %d: %#v", len(got.DeclaredKeyFields), got)
	}
	want := []DeclaredKeyFieldsFact{
		{
			Locator:         "/System[1]/Function[1]",
			SourceName:      "Function",
			OwnerLegacyID:   "F1",
			Fields:          []string{"A", "B"},
			Nature:          SourceFactNature,
			AuthorityLimits: declaredKeyFieldsAuthorityLimits,
		},
		{
			Locator:         "/System[1]/Function[1]/Detail[1]",
			SourceName:      "Detail",
			OwnerLegacyID:   "D1",
			Fields:          []string{"H", "L"},
			Nature:          SourceFactNature,
			AuthorityLimits: declaredKeyFieldsAuthorityLimits,
		},
	}
	if !reflect.DeepEqual(got.DeclaredKeyFields, want) {
		t.Fatalf("declared key fields drifted: got=%#v want=%#v", got.DeclaredKeyFields, want)
	}
}

func TestExtractRelationalSourceFactsPreservesXeafRuntimeKeyAndReferMappings(t *testing.T) {
	doc, err := Parse(FormatXeaf, []byte(`<System><Table ID="T1"><Key ID="K1" Type="PK" Fields="A;B;"/><Refer ID="R1" ToTable="T2" TableAlias="REF" ToKeyFields="C;D;" WithKeyFields="A;B;"/></Table></System>`))
	if err != nil {
		t.Fatal(err)
	}
	got := ExtractRelationalSourceFacts(doc)
	if len(got.RuntimeKeys) != 1 {
		t.Fatalf("runtime key count = %d: %#v", len(got.RuntimeKeys), got)
	}
	key := got.RuntimeKeys[0]
	if key.TableLegacyID != "T1" ||
		key.KeyLegacyID != "K1" ||
		key.KeyType != "PK" ||
		!reflect.DeepEqual(key.Fields, []string{"A", "B"}) ||
		key.Nature != SourceFactNature ||
		!reflect.DeepEqual(key.AuthorityLimits, runtimeKeyAuthorityLimits) {
		t.Fatalf("runtime key fact drifted: %#v", key)
	}

	if len(got.RuntimeRefers) != 1 {
		t.Fatalf("runtime refer count = %d: %#v", len(got.RuntimeRefers), got)
	}
	refer := got.RuntimeRefers[0]
	if refer.ReferLegacyID != "R1" ||
		refer.SourceTableLegacyID != "T1" ||
		refer.ToTableLegacyID != "T2" ||
		refer.TableAlias != "REF" ||
		!reflect.DeepEqual(refer.ToKeyFields, []string{"C", "D"}) ||
		!reflect.DeepEqual(refer.WithKeyFields, []string{"A", "B"}) ||
		refer.Nature != SourceFactNature ||
		!reflect.DeepEqual(refer.AuthorityLimits, runtimeReferAuthorityLimits) {
		t.Fatalf("runtime refer fact drifted: %#v", refer)
	}
}

func TestExtractRelationalSourceFactsPreservesRawRuntimeReferDeclarations(t *testing.T) {
	literal := func(value string) *string { return &value }
	cases := []struct {
		name          string
		attributes    string
		wantTo        *string
		wantWith      *string
		wantToItems   []string
		wantWithItems []string
	}{
		{
			name:       "explicit blank both",
			attributes: `ToKeyFields="" WithKeyFields=""`,
			wantTo:     literal(""), wantWith: literal(""),
			wantToItems: []string{}, wantWithItems: []string{},
		},
		{
			name:        "absent both",
			attributes:  ``,
			wantToItems: []string{}, wantWithItems: []string{},
		},
		{
			name:       "literal component separators and whitespace",
			attributes: `ToKeyFields=" A ; B; " WithKeyFields=" T1.X; T1.Y;"`,
			wantTo:     literal(" A ; B; "), wantWith: literal(" T1.X; T1.Y;"),
			wantToItems: []string{"A", "B"}, wantWithItems: []string{"T1.X", "T1.Y"},
		},
		{
			name:        "mixed presence",
			attributes:  `WithKeyFields=""`,
			wantWith:    literal(""),
			wantToItems: []string{}, wantWithItems: []string{},
		},
		{
			name:        "foreign namespace lookalike",
			attributes:  `x:ToKeyFields="A" x:WithKeyFields="B"`,
			wantToItems: []string{}, wantWithItems: []string{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := `<System xmlns:x="urn:foreign"><Table ID="T1"><Refer ID="R1" ` + tc.attributes + `/></Table></System>`
			doc, err := Parse(FormatXeaf, []byte(input))
			if err != nil {
				t.Fatal(err)
			}
			facts := ExtractRelationalSourceFacts(doc)
			if len(facts.RuntimeRefers) != 1 {
				t.Fatalf("runtime refer count = %d", len(facts.RuntimeRefers))
			}
			refer := facts.RuntimeRefers[0]
			for _, pair := range []struct {
				name string
				got  *string
				want *string
			}{
				{"ToKeyFieldsRaw", refer.ToKeyFieldsRaw, tc.wantTo},
				{"WithKeyFieldsRaw", refer.WithKeyFieldsRaw, tc.wantWith},
			} {
				if (pair.got == nil) != (pair.want == nil) ||
					(pair.got != nil && *pair.got != *pair.want) {
					t.Errorf("%s: got %v, want %v", pair.name, pair.got, pair.want)
				}
			}
			if !reflect.DeepEqual(refer.ToKeyFields, tc.wantToItems) ||
				!reflect.DeepEqual(refer.WithKeyFields, tc.wantWithItems) {
				t.Fatalf("normalized refer fields changed: %#v", refer)
			}
			encoded, err := json.Marshal(refer)
			if err != nil {
				t.Fatal(err)
			}
			var object map[string]json.RawMessage
			if err := json.Unmarshal(encoded, &object); err != nil {
				t.Fatal(err)
			}
			for _, pair := range []struct {
				name string
				want *string
			}{
				{"toKeyFieldsRaw", tc.wantTo},
				{"withKeyFieldsRaw", tc.wantWith},
			} {
				value, exists := object[pair.name]
				if pair.want == nil {
					if exists {
						t.Errorf("absent %s invented in JSON: %s", pair.name, encoded)
					}
					continue
				}
				if !exists {
					t.Errorf("declared %s missing in JSON: %s", pair.name, encoded)
					continue
				}
				var actual string
				if err := json.Unmarshal(value, &actual); err != nil || actual != *pair.want {
					t.Errorf("%s = %q, want %q (err=%v)", pair.name, actual, *pair.want, err)
				}
			}
		})
	}
}

func TestExtractRelationalSourceFactsRejectsForeignNamespaceReferLookalikes(t *testing.T) {
	input := []byte(`<System xmlns:x="urn:foreign"><Table ID="T1">
		<x:Refer ID="FOREIGN_NODE" ToTable="T9" WithKeyFields="X" ToKeyFields="Y"/>
		<Refer x:ID="FOREIGN_ID" x:ToTable="FOREIGN_TABLE" x:TableAlias="FOREIGN_ALIAS"
			x:ToKeyFields="A" x:WithKeyFields="B"/>
		<Refer ID="R2" ToTable="T2" TableAlias="GOOD" WithKeyFields="A" ToKeyFields="B"/>
	</Table></System>`)
	doc, err := Parse(FormatXeaf, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Root.Children) != 1 || len(doc.Root.Children[0].Children) != 3 {
		t.Fatalf("unexpected raw source tree: %#v", doc.Root)
	}
	if doc.Root.Children[0].Children[0].Known {
		t.Fatal("foreign-namespaced Refer must remain unknown raw source")
	}
	facts := ExtractRelationalSourceFacts(doc)
	if len(facts.RuntimeRefers) != 2 {
		t.Fatalf("foreign-namespaced Refer was promoted to typed fact: %#v", facts.RuntimeRefers)
	}
	first := facts.RuntimeRefers[0]
	if first.ReferLegacyID != "" || first.ToTableLegacyID != "" || first.TableAlias != "" ||
		first.ToKeyFieldsRaw != nil || first.WithKeyFieldsRaw != nil ||
		len(first.ToKeyFields) != 0 || len(first.WithKeyFields) != 0 {
		t.Fatalf("foreign-namespaced attributes promoted to typed fact: %#v", first)
	}
	second := facts.RuntimeRefers[1]
	if second.ReferLegacyID != "R2" || second.ToTableLegacyID != "T2" ||
		second.TableAlias != "GOOD" || second.ToKeyFieldsRaw == nil ||
		*second.ToKeyFieldsRaw != "B" || second.WithKeyFieldsRaw == nil ||
		*second.WithKeyFieldsRaw != "A" {
		t.Fatalf("unqualified source declaration was changed: %#v", second)
	}
}

func TestExtractRelationalSourceFactsPreservesNamespaceBoundaryAcrossLegacyKeyAndRelation(t *testing.T) {
	source := []byte(`<System xmlns:x="urn:foreign">
		<x:Table ID="FOREIGN_TABLE"><TableField ID="F"/></x:Table>
		<Table ID="T1">
			<TableField ID="F1" DataTypeID="D1"/>
			<x:TableField ID="FOREIGN_FIELD" DataTypeID="D2"/>
			<TableKey ID="K1" x:Type="SK">
				<x:TableKeyField FieldID="FOREIGN_FIELD" SortKey="0"/>
				<TableKeyField FieldID="F1" x:SortKey="42"/>
			</TableKey>
			<x:TableKey ID="FOREIGN_KEY"><TableKeyField FieldID="F1"/></x:TableKey>
		</Table>
		<x:Relationship ID="FOREIGN_REL" Type="FAMILY" Table1ID="T1" TableKey1ID="K1"/>
		<Relationship x:ID="FOREIGN_REL_ID" Type="REFFER"
			x:Table1ID="T1" x:TableKey1ID="K1" Table2ID="T1" TableKey2ID="K1"/>
		<Function x:KeyFields="FOREIGN_KEY_FIELDS"/>
		<x:Function KeyFields="FOREIGN_UNKNOWN_NODE"/>
		<Function ID="REAL_FUNCTION" KeyFields="F1;"/>
	</System>`)
	doc, err := Parse(FormatXead, source)
	if err != nil {
		t.Fatal(err)
	}
	facts := ExtractRelationalSourceFacts(doc)
	if len(facts.LegacyKeys) != 1 || len(facts.RelationshipKeys) != 1 ||
		len(facts.DeclaredKeyFields) != 1 {
		t.Fatalf("foreign element or attribute promoted to typed fact: %#v", facts)
	}
	key := facts.LegacyKeys[0]
	if key.TableLegacyID != "T1" || key.KeyLegacyID != "K1" || key.KeyType != "" ||
		len(key.Members) != 1 || key.Members[0].FieldLegacyID != "F1" ||
		key.Members[0].SortKey != "" || key.Members[0].DataTypeLegacyID != "D1" {
		t.Fatalf("legacy key member/type contaminated by namespace: %#v", key)
	}
	relation := facts.RelationshipKeys[0]
	if relation.RelationshipID != "" || relation.Table1LegacyID != "" || relation.Key1LegacyID != "" ||
		relation.Table2LegacyID != "T1" || relation.Key2LegacyID != "K1" ||
		relation.RelationshipType != "REFFER" {
		t.Fatalf("namespace aliases promoted to relationship refs: %#v", relation)
	}
	declared := facts.DeclaredKeyFields[0]
	if declared.SourceName != "Function" || declared.OwnerLegacyID != "REAL_FUNCTION" ||
		!reflect.DeepEqual(declared.Fields, []string{"F1"}) {
		t.Fatalf("foreign KeyFields invented as standard declaration: %#v", declared)
	}
}

func TestExtractRelationalSourceFactsForeignTableDoesNotInheritParentOwnership(t *testing.T) {
	doc, err := Parse(FormatXeaf, []byte(`<System xmlns:x="urn:foreign">
		<Table ID="OUTER">
			<Field ID="LIVE"/>
			<x:Table ID="FOREIGN">
				<Field ID="HIDDEN"/>
				<Key ID="INSIDE" Fields="HIDDEN"/>
				<Refer ID="INSIDE_REFER" ToTable="OUTER" ToKeyFields="LIVE" WithKeyFields="HIDDEN"/>
			</x:Table>
			<Key ID="OUTSIDE" Fields="HIDDEN"/>
		</Table>
	</System>`))
	if err != nil {
		t.Fatal(err)
	}
	facts := ExtractRelationalSourceFacts(doc)
	if len(facts.RuntimeKeys) != 2 || len(facts.RuntimeRefers) != 1 {
		t.Fatalf("unexpected typed fact count: %#v", facts)
	}
	byID := map[string]RuntimeKeyFact{}
	for _, key := range facts.RuntimeKeys {
		byID[key.KeyLegacyID] = key
	}
	if inside := byID["INSIDE"]; inside.TableLegacyID != "" {
		t.Fatalf("foreign wrapper donated or inherited a Table identity: %#v", inside)
	}
	if outside := byID["OUTSIDE"]; outside.TableLegacyID != "OUTER" {
		t.Fatalf("legitimate enclosing Table changed: %#v", outside)
	}
	if refer := facts.RuntimeRefers[0]; refer.SourceTableLegacyID != "" ||
		refer.ReferLegacyID != "INSIDE_REFER" {
		t.Fatalf("foreign wrapper leaked parent table to nested Refer: %#v", refer)
	}
	got := DiagnoseRelationalSourceFacts(doc, facts)
	foundMissing := false
	for _, d := range got {
		if d.Code == "runtime-key-field-not-found" &&
			d.TableLegacyID == "OUTER" &&
			d.FieldLegacyID == "HIDDEN" {
			foundMissing = true
		}
	}
	if !foundMissing {
		t.Fatalf("foreign nested Field incorrectly satisfied enclosing Table Key: %#v", got)
	}
}

func TestExtractRelationalSourceFactsRejectsUnknownDefaultNamespace(t *testing.T) {
	for _, tc := range []struct {
		name   string
		format Format
		xml    string
	}{
		{"xead", FormatXead, `<System xmlns="urn:foreign"><Table ID="T"><TableKey ID="K"/></Table><Relationship ID="R"/></System>`},
		{"xeaf", FormatXeaf, `<System xmlns="urn:foreign"><Table ID="T"><Key ID="K" Fields="F"/><Refer ID="R" ToTable="T"/></Table></System>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := Parse(tc.format, []byte(tc.xml))
			if err != nil {
				t.Fatal(err)
			}
			facts := ExtractRelationalSourceFacts(doc)
			if len(facts.LegacyKeys) != 0 || len(facts.RelationshipKeys) != 0 ||
				len(facts.RuntimeKeys) != 0 || len(facts.RuntimeRefers) != 0 ||
				len(facts.DeclaredKeyFields) != 0 {
				t.Fatalf("default foreign namespace promoted typed facts: %#v", facts)
			}
		})
	}
}

func TestExtractRelationalSourceFactsPreservesRuntimeKeyOnlyFromExactFields(t *testing.T) {
	doc, err := Parse(FormatXeaf, []byte(`<System xmlns:x="urn:foreign"><Table ID="T1">
		<x:Key ID="FOREIGN_K" Fields="G"/>
		<Key x:ID="FOREIGN_ID" x:Type="PK" x:Fields="G"/>
		<Key ID="REAL_K" Type="SK" Fields="A;B;"/>
	</Table></System>`))
	if err != nil {
		t.Fatal(err)
	}
	facts := ExtractRelationalSourceFacts(doc)
	if len(facts.RuntimeKeys) != 2 {
		t.Fatalf("unknown Key was promoted to runtime fact: %#v", facts.RuntimeKeys)
	}
	if facts.RuntimeKeys[0].KeyLegacyID != "" || facts.RuntimeKeys[0].KeyType != "" ||
		len(facts.RuntimeKeys[0].Fields) != 0 {
		t.Fatalf("foreign attributes were promoted into runtime key: %#v", facts.RuntimeKeys[0])
	}
	if facts.RuntimeKeys[1].KeyLegacyID != "REAL_K" ||
		facts.RuntimeKeys[1].KeyType != "SK" ||
		!reflect.DeepEqual(facts.RuntimeKeys[1].Fields, []string{"A", "B"}) {
		t.Fatalf("valid runtime key changed: %#v", facts.RuntimeKeys[1])
	}
}

func TestExtractRelationalSourceFactsDoesNotInventFactsFromNamesOrTypes(t *testing.T) {
	doc, err := Parse(FormatXead, []byte(`<System><Table ID="ORDER"><TableField ID="ORDER_ID" Name="Primary Key"/></Table></System>`))
	if err != nil {
		t.Fatal(err)
	}
	got := ExtractRelationalSourceFacts(doc)
	if len(got.LegacyKeys) != 0 || len(got.RelationshipKeys) != 0 || len(got.DeclaredKeyFields) != 0 {
		t.Fatalf("key facts were inferred from names: %#v", got)
	}
}

func TestExtractRelationalSourceFactsDoesNotChooseMetadataFromDuplicateFieldID(t *testing.T) {
	doc, err := Parse(FormatXead, []byte(`<System><Table ID="T1"><TableField ID="A" AttributeType="NATIVE" DataTypeID="D1"/><TableField ID="A" AttributeType="DERIVABLE" DataTypeID="D2"/><TableKey ID="K1" Type="PK"><TableKeyField FieldID="A"/></TableKey></Table></System>`))
	if err != nil {
		t.Fatal(err)
	}
	got := ExtractRelationalSourceFacts(doc)
	if len(got.LegacyKeys) != 1 || len(got.LegacyKeys[0].Members) != 1 {
		t.Fatalf("legacy key source fact is missing: %#v", got)
	}
	member := got.LegacyKeys[0].Members[0]
	if member.FieldLegacyID != "A" {
		t.Fatalf("key member identity drifted: %#v", member)
	}
	if member.AttributeType != "" || member.DataTypeLegacyID != "" {
		t.Fatalf("duplicate field metadata was chosen implicitly: %#v", member)
	}
}
