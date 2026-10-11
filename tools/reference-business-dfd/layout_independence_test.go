package main

import (
	"reflect"
	"sort"
	"testing"
)

// Changing presentation order changes source pointers and DOT emission order,
// but not explicit control edges, information payloads, or Business Step IDs.
func TestStepArrayReorderDoesNotInventBusinessTransition(t *testing.T) {
	f,r,s,w:=sample()
	original:=build(f,r,s,w,"procedure.json","review.json","set.json")
	f.Steps[0],f.Steps[2]=f.Steps[2],f.Steps[0]
	reordered:=build(f,r,s,w,"procedure.json","review.json","set.json")
	if !reflect.DeepEqual(original.Flows,reordered.Flows) ||
		!reflect.DeepEqual(original.Triggers,reordered.Triggers) ||
		!reflect.DeepEqual(original.Handoffs,reordered.Handoffs) {
		t.Fatal("source list order incorrectly became control-flow or Handoff meaning")
	}
	a,b:=[]string{},[]string{}
	for _,step:=range original.Steps {a=append(a,step.ID)}
	for _,step:=range reordered.Steps {b=append(b,step.ID)}
	sort.Strings(a);sort.Strings(b)
	if !reflect.DeepEqual(a,b) {t.Fatal("authoring Step set changed from display order")}
	if original.Steps[0].Source.Pointer==reordered.Steps[2].Source.Pointer {
		t.Fatal("source location did not reflect changed authoring location")
	}
}
