package exampleslide

import (
	"encoding/json"
	"testing"
)

func keyedInput() Input {
	return Input{
		Catalog: Catalog{
			Revision: "schema-r1",
			Tables: map[string]Table{
				"parent": {Fields: []Field{
					{ID:"tenant", Type:"integer", Visible:true},
					{ID:"number", Type:"string", Visible:true},
				}},
				"child": {Fields: []Field{
					{ID:"tenant_ref", Type:"integer", Visible:true},
					{ID:"number_ref", Type:"string", Visible:true},
				}},
			},
			Relations: []SemanticRelation{{
				ID:"child.parent", SourceTable:"child", TargetTable:"parent",
				Cardinality:"one",
				Keys:[]SemanticRelationKey{
					{SourceField:"tenant_ref", TargetField:"tenant"},
					{SourceField:"number_ref", TargetField:"number"},
				},
			}},
		},
		Example: Example{
			ID:"multi-key", ModelRevision:"schema-r1",
			Base: []Record{
				{ID:"a",Table:"parent",Values:map[string]json.RawMessage{
					"tenant":json.RawMessage("9007199254740993"),
					"number":json.RawMessage(`"A"`),
				}},
				{ID:"b",Table:"child",Values:map[string]json.RawMessage{
					"tenant_ref":json.RawMessage("9007199254740993.0"),
					"number_ref":json.RawMessage(`"A"`),
				}},
			},
			CompleteTables:[]string{"parent"},
		},
		Diagram: Diagram{
			ID:"diagram-1",
			Placements:[]Placement{
				{ID:"left",Table:"parent",X:15,Y:40},
				{ID:"right",Table:"child",X:310,Y:40},
			},
		},
		Presentation: Presentation{Frames:[]Frame{{ID:"slide-1",Stage:"base"}}},
	}
}

func oneFinding(t *testing.T, input Input) RelationFinding {
	t.Helper()
	projection,err:=Build(input)
	if err!=nil {t.Fatal(err)}
	if len(projection.Slides)!=1 || len(projection.Slides[0].RelationFindings)!=1 {
		t.Fatalf("unexpected findings: %+v",projection.Slides)
	}
	return projection.Slides[0].RelationFindings[0]
}

func TestCompleteCompositeRelationSupported(t *testing.T) {
	f:=oneFinding(t,keyedInput())
	if f.Result!="supported" || f.Reason!="keys-checked-in-claimed-complete-snapshot" ||
		len(f.MatchedTargetIDs)!=1 || f.MatchedTargetIDs[0]!="a" {
		t.Fatalf("unexpected complete join finding: %+v",f)
	}
}

func TestPartialDataNeverClaimsSupported(t *testing.T) {
	in:=keyedInput()
	in.Example.CompleteTables=nil
	f:=oneFinding(t,in)
	if f.Result!="unresolved" || f.Reason!="target-table-is-partial" {
		t.Fatalf("partial data wrongly accepted: %+v",f)
	}
}

func TestKnownMultipleTargetsContradictOneEvenInPartialDataset(t *testing.T) {
	in:=keyedInput()
	in.Example.CompleteTables=nil
	record:=in.Example.Base[0]
	record.ID="also-a"
	in.Example.Base=append(in.Example.Base,record)
	f:=oneFinding(t,in)
	if f.Result!="contradiction" || f.Reason!="multiple-targets-for-one" ||
		len(f.MatchedTargetIDs)!=2 {
		t.Fatalf("duplicate matches not detected: %+v",f)
	}
}

func TestUndeclaredKeyIsUnresolved(t *testing.T) {
	in:=keyedInput()
	in.Catalog.Relations[0].Keys=nil
	f:=oneFinding(t,in)
	if f.Result!="unresolved" || f.Reason!="relation-keys-not-declared" {
		t.Fatalf("keyless relation was guessed: %+v",f)
	}
}

func TestNullOrAbsentKeyRemainsUnresolved(t *testing.T) {
	in:=keyedInput()
	delete(in.Example.Base[1].Values,"number_ref")
	f:=oneFinding(t,in)
	if f.Result!="unresolved" || f.Reason!="source-key-missing-or-unusable" {
		t.Fatalf("missing key not deferred: %+v",f)
	}

	in=keyedInput()
	delete(in.Example.Base[0].Values,"number")
	f=oneFinding(t,in)
	if f.Result!="unresolved" || f.Reason!="unresolved-target-key" {
		t.Fatalf("missing target key was assumed absent: %+v",f)
	}
}

func TestNoMatchingTargetDoesNotInventRequiredness(t *testing.T) {
	in:=keyedInput()
	in.Example.Base[1].Values["number_ref"]=json.RawMessage(`"B"`)
	f:=oneFinding(t,in)
	if f.Result!="unresolved" || f.Reason!="target-absence-requires-business-review" {
		t.Fatalf("presence was assumed: %+v",f)
	}
}

func TestCompletenessMustDeclareKnownTables(t *testing.T) {
	in:=keyedInput()
	in.Example.CompleteTables=[]string{"not-in-catalog"}
	if _,err:=Build(in);err==nil {t.Fatal("unknown complete table accepted")}

	in=keyedInput()
	in.Example.CompleteTables=[]string{"parent","parent"}
	if _,err:=Build(in);err==nil {t.Fatal("duplicate complete table accepted")}
}

func TestNumericalScalarDoesNotLosePrecision(t *testing.T) {
	for _,pair:=range [][2]string{
		{"9007199254740993","9007199254740993.00"},
		{"1e2","100"},
	} {
		equal,comparable:=equalScalar(json.RawMessage(pair[0]),json.RawMessage(pair[1]))
		if !comparable || !equal {t.Fatalf("equal numeric values not recognized: %v",pair)}
	}
	equal,comparable:=equalScalar(json.RawMessage(`"2"`),json.RawMessage("2"))
	if comparable || equal {t.Fatal("automatically coerced string to number")}
}

func TestRelationFindingsFollowStageNotSlideOrder(t *testing.T) {
	in:=keyedInput()
	in.Example.Stages=[]Stage{{ID:"second", Changes:[]Change{{Op:"delete",Record:Record{ID:"a",Table:"parent"}}}}}
	// Only the child is left after removing its illustrative parent.
	in.Presentation.Frames=[]Frame{
		{ID:"later",Stage:"second"},
		{ID:"earlier",Stage:"base"},
	}
	projection,err:=Build(in)
	if err!=nil {t.Fatal(err)}
	if projection.Slides[0].RelationFindings[0].Result!="unresolved" ||
		projection.Slides[1].RelationFindings[0].Result!="supported" {
		t.Fatalf("findings followed presentation order: %+v",projection.Slides)
	}
}

func TestNumericKeyComparisonIsResourceBounded(t *testing.T) {
	equal, comparable := equalScalar(
		json.RawMessage("1e100000000"), json.RawMessage("1"),
	)
	if equal || comparable {
		t.Fatal("extreme numeric exponent was treated as comparable")
	}
	in := keyedInput()
	in.Example.Base[1].Values["tenant_ref"] = json.RawMessage("9e100000000")
	// The example's integer value is outside the ordinary scalar validator
	// range; this must not be silently accepted or cause an unbounded allocation.
	if _, err := Build(in); err == nil {
		t.Fatal("out-of-range integer was silently accepted")
	}
}

func TestDuplicateRelationDefinitionFailsClosed(t *testing.T) {
	in := keyedInput()
	in.Catalog.Relations = append(in.Catalog.Relations, in.Catalog.Relations[0])
	if _, err := Build(in); err == nil {
		t.Fatal("duplicated relation ID was accepted")
	}

	in = keyedInput()
	first := in.Catalog.Relations[0].Keys[0]
	in.Catalog.Relations[0].Keys = append(in.Catalog.Relations[0].Keys, first)
	if _, err := Build(in); err == nil {
		t.Fatal("duplicated composite key pair was accepted")
	}
}
