package main

import (
	"reflect"
	"testing"
)

func TestNormalizedBusinessModelProbeDoesNotClaimBusinessFlowSurface(t *testing.T) {
	modelType := reflect.TypeOf(normalizedBusinessModelProbe{})
	for _, forbidden := range []string{
		"BusinessFlows",
		"Flows",
		"Dataflows",
		"FlowNodes",
		"ProcessNodes",
		"ExternalParties",
		"InformationFlows",
		"Documents",
		"Stores",
	} {
		if _, found := modelType.FieldByName(forbidden); found {
			t.Fatalf("handoff用normalized Business modelがBusiness Flow surface %qまで所有しました", forbidden)
		}
	}
}

func TestThinHandoffProbeCarriesOnlyEndpointAndConditionMeaning(t *testing.T) {
	handoffType := reflect.TypeOf(canonicalBusinessHandoffProbe{})
	want := []string{"From", "To", "Condition"}
	if got := fieldNames(handoffType); !reflect.DeepEqual(got, want) {
		t.Fatalf("thin handoffの守備範囲が変化しました: got=%v want=%v", got, want)
	}

	endpointType := reflect.TypeOf(canonicalBusinessHandoffEndpointProbe{})
	endpointWant := []string{"Procedure", "Step"}
	if got := fieldNames(endpointType); !reflect.DeepEqual(got, endpointWant) {
		t.Fatalf("handoff endpointへBusiness Flow participant意味が侵入しました: got=%v want=%v", got, endpointWant)
	}
}

func TestThinHandoffDoesNotEncodeDFDParticipantOrPayloadSemantics(t *testing.T) {
	handoffType := reflect.TypeOf(canonicalBusinessHandoffProbe{})
	endpointType := reflect.TypeOf(canonicalBusinessHandoffEndpointProbe{})

	for _, forbidden := range []string{
		"Actor",
		"Role",
		"ExternalParty",
		"Process",
		"Data",
		"Information",
		"Document",
		"Store",
		"Payload",
		"FlowType",
	} {
		if _, found := handoffType.FieldByName(forbidden); found {
			t.Fatalf("thin handoff relationへDFD / Business Flow意味 %qが侵入しました", forbidden)
		}
		if _, found := endpointType.FieldByName(forbidden); found {
			t.Fatalf("thin handoff endpointへDFD / Business Flow participant意味 %qが侵入しました", forbidden)
		}
	}
}

func TestBusinessFlowPressureRemainsSeparateFromCurrentHandoffValidation(t *testing.T) {
	model := currentNormalizedBusinessModelProbe(t)
	if err := validateNormalizedBusinessModelProbe(model); err != nil {
		t.Fatalf("handoff-only normalized modelの前提が成立しません: %v", err)
	}

	// The current validator proves only Procedure/Step address integrity and a
	// non-empty handoff condition. It intentionally has no input for external
	// parties, stores, documents, information payloads, or DFD-style process
	// nodes. Passing this validation must therefore not be interpreted as
	// Business Flow / DFD conformance.
	if len(model.Handoffs) == 0 {
		t.Fatal("test setupにhandoffがありません")
	}
}
