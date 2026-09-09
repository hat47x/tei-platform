package main

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/workspace"
)

func TestBusinessActionDataProjectionIsInvariantToBusinessWordingOnlyChange(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	before, err := deriveBusinessActionDataProjectionProbe(loaded.World, f)
	if err != nil {
		t.Fatal(err)
	}

	changed := f
	changed.Steps = append([]step(nil), f.Steps...)
	for i := range changed.Steps {
		if changed.Steps[i].ID == "BP-00" {
			changed.Steps[i].BusinessAction = changed.Steps[i].BusinessAction + " — wording changed for Business review"
		}
	}
	after, err := deriveBusinessActionDataProjectionProbe(loaded.World, changed)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(before, after) {
		t.Fatalf("Business wording-only changeからData dependencyを推測しました: before=%+v after=%+v", before, after)
	}
}

func TestBusinessActionDataProjectionTracksReferencedCapabilityContractChange(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	before, err := deriveBusinessActionDataProjectionProbe(loaded.World, f)
	if err != nil {
		t.Fatal(err)
	}
	beforeRow := businessActionDataRowsByStep(before)["BP-07"]
	if contains(beforeRow.Data, "sales.order_line") {
		t.Fatalf("test前提が崩れています: BP-07=%+v", beforeRow)
	}

	changedWorld := cloneWorldForBusinessActionDataProjection(loaded.World)
	changed := false
	for ni := range changedWorld.Semantics {
		for ci := range changedWorld.Semantics[ni].Capabilities {
			capability := &changedWorld.Semantics[ni].Capabilities[ci]
			if string(capability.ID) == "sales.order.submit" {
				capability.Output = "sales.order_line"
				changed = true
			}
		}
	}
	if !changed {
		t.Fatal("sales.order.submit Capabilityをtest用に変更できません")
	}

	after, err := deriveBusinessActionDataProjectionProbe(changedWorld, f)
	if err != nil {
		t.Fatal(err)
	}
	afterRow := businessActionDataRowsByStep(after)["BP-07"]
	if !contains(afterRow.Data, "sales.order_line") {
		t.Fatalf("referenced Capability contract changeをData projectionへ反映できません: before=%+v after=%+v", beforeRow, afterRow)
	}
	if reflect.DeepEqual(beforeRow, afterRow) {
		t.Fatalf("referenced authority changeをderived projection差として観測できません: before=%+v after=%+v", beforeRow, afterRow)
	}
}

func TestBusinessActionDataProjectionFailsClosedWhenReferencedPatternDataIsMissing(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	changedWorld := cloneWorldForBusinessActionDataProjection(loaded.World)
	changed := false
	for ai := range changedWorld.Applications {
		for pi := range changedWorld.Applications[ai].Patterns {
			pattern := &changedWorld.Applications[ai].Patterns[pi]
			if string(pattern.ID) == "order_entry.recent_orders" {
				pattern.Data = "sales.missing_order_projection_target"
				changed = true
			}
		}
	}
	if !changed {
		t.Fatal("order_entry.recent_orders Patternをtest用に変更できません")
	}

	_, err = deriveBusinessActionDataProjectionProbe(changedWorld, f)
	if err == nil {
		t.Fatal("missing Pattern Data authorityを名称や近接Dataで補完しました")
	}
	if !strings.Contains(err.Error(), "exactに解決できません") {
		t.Fatalf("missing authorityのfail-closed reasonが不明確です: %v", err)
	}
}

func cloneWorldForBusinessActionDataProjection(world ir.World) ir.World {
	result := world
	result.Applications = append([]application.Application(nil), world.Applications...)
	for i := range result.Applications {
		result.Applications[i].Patterns = append([]application.Pattern(nil), world.Applications[i].Patterns...)
	}
	result.Semantics = append([]semantic.Namespace(nil), world.Semantics...)
	for i := range result.Semantics {
		result.Semantics[i].Capabilities = append([]semantic.Capability(nil), world.Semantics[i].Capabilities...)
		result.Semantics[i].Validations = append([]semantic.Validation(nil), world.Semantics[i].Validations...)
		result.Semantics[i].Data = append([]semantic.Data(nil), world.Semantics[i].Data...)
	}
	return result
}
