package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Size は合成Workspaceの寸法を表す。値は固定で、乱数を使わない。
// 同じSizeからは常にバイト単位で同一のWorkspaceができる。
type Size struct {
	Name         string
	Files        int // semantic配下の.teiファイル数（ハブ用の1ファイルは別）
	DataPerFile  int // 1ファイルのdata数
	AttrsPerData int // 1 dataのattribute数
}

// Sizes は計測に使う小・中・大の寸法。実案件の寸法ではない。
var Sizes = []Size{
	{Name: "small", Files: 4, DataPerFile: 5, AttrsPerData: 4},
	{Name: "medium", Files: 40, DataPerFile: 10, AttrsPerData: 8},
	{Name: "large", Files: 200, DataPerFile: 20, AttrsPerData: 10},
}

// HubScope・HubKind・HubID は、全attributeから参照される1つのType（ハブ）を指す。
// このSubjectへの入向きedge数が、fullの行数になる。
const (
	HubScope = "bench"
	HubKind  = "semantic-type"
	HubID    = "hub.hub_id"
)

// LeafKind・LeafID は、出向きedgeが1本しかないattribute。固定費の測定に使う。
const (
	LeafKind = "semantic-attribute"
	LeafID   = "f0000.d0000.a0000"
)

// AttributeCount は、ハブを参照するattribute（=ハブへの入向きedge）の件数。
func (s Size) AttributeCount() int { return s.Files * s.DataPerFile * s.AttrsPerData }

// Generate は、root配下へ合成Workspaceを書き出す。実案件のデータは含まない。
func Generate(root string, s Size) error {
	if err := os.MkdirAll(filepath.Join(root, "semantic", "hub"), 0o755); err != nil {
		return err
	}
	manifest := "workspace: cli_trace_benchmark\nsources:\n  - semantic\n"
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0o644); err != nil {
		return err
	}
	hub := "namespace hub\n\ntype HubId from std.identifier\n\ndata Hub {\n    attribute hubId type hub.hub_id\n}\n"
	if err := os.WriteFile(filepath.Join(root, "semantic", "hub", "hub.tei"), []byte(hub), 0o644); err != nil {
		return err
	}
	for f := 0; f < s.Files; f++ {
		dir := filepath.Join(root, "semantic", fmt.Sprintf("f%04d", f))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		var b strings.Builder
		fmt.Fprintf(&b, "namespace f%04d\n\n", f)
		for d := 0; d < s.DataPerFile; d++ {
			fmt.Fprintf(&b, "data D%04d {\n", d)
			for a := 0; a < s.AttrsPerData; a++ {
				// 偶数番目はハブのTypeを参照し、奇数番目は標準Typeを参照する。
				// ただし各dataの先頭attributeは必ずハブを参照する。
				ref := "hub.hub_id"
				if a%2 == 1 {
					ref = "std.text"
				}
				fmt.Fprintf(&b, "    attribute a%04d type %s\n", a, ref)
			}
			b.WriteString("}\n\n")
		}
		if err := os.WriteFile(filepath.Join(dir, "model.tei"), []byte(b.String()), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// HubEdgeCount は、ハブへ入る宣言済みedgeの件数。偶数番目のattributeだけが参照する。
func (s Size) HubEdgeCount() int {
	perData := (s.AttrsPerData + 1) / 2
	return s.Files * s.DataPerFile * perData
}
