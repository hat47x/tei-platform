// Command reference-sui-scope-conformance は、SUI の越境試験の行列を
// 言語非依存の契約として検査する read-only の Reference 実装である（パケット M6）。
//
// この tool は認可 Engine でも Product Gate でもない。行列の各セルが
// 許された状態語彙のどれかであり、空欄が pass として扱われていないことだけを
// 検査する。実試験の実行・実 Provider の観測・Owner の意味受理は扱わない。
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
)

const (
	matrixSchema  = "tai.reference.sui-scope-conformance/v0"
	authorityName = "reference-matrix"
)

// セルの状態。空文字は許さない。空欄を pass と読み替えない。
var allowedStates = []string{"pass", "fail", "unsupported", "not-run"}

// FailureCode は入力に依存しない安定した失敗の名前である。
type FailureCode string

const (
	FailureUnsupportedSchema FailureCode = "unsupported-schema"
	FailureInvalidMatrix     FailureCode = "invalid-matrix"
	FailureUnknownState      FailureCode = "unknown-cell-state"
	FailureEmptyCell         FailureCode = "empty-cell"
	FailureGapNotDeclared    FailureCode = "known-gap-not-declared"
	FailureAuthorized        FailureCode = "matrix-claims-authority"
)

type Row struct {
	Route             string   `json:"route"`
	Kind              string   `json:"kind"`
	TenantCrossing    string   `json:"tenantCrossing"`
	OperationBoundary string   `json:"operationBoundary"`
	Evidence          []string `json:"evidence"`
	Note              string   `json:"note"`
}

type Matrix struct {
	Schema          string         `json:"schema"`
	Authority       string         `json:"authority"`
	Source          map[string]any `json:"source"`
	CellStates      []string       `json:"cellStates"`
	RequiredColumns []string       `json:"requiredColumns"`
	Rows            []Row          `json:"rows"`
	KnownGaps       []string       `json:"knownGaps"`
}

type Report struct {
	OK          bool        `json:"ok"`
	Schema      string      `json:"schema,omitempty"`
	Failure     FailureCode `json:"failure,omitempty"`
	Problems    []string    `json:"problems,omitempty"`
	Rows        int         `json:"rows"`
	Gaps        int         `json:"declaredGaps"`
	NotRun      int         `json:"notRunCells"`
	Unsupported int         `json:"unsupportedCells"`
}

func main() {
	input := flag.String("matrix", "reference/sui-scope-conformance/matrix.json", "conformance matrix JSON")
	flag.Parse()

	payload, err := os.ReadFile(*input)
	if err != nil {
		fmt.Fprintln(os.Stderr, "reference-sui-scope-conformance: matrix cannot be read")
		os.Exit(1)
	}
	report := Check(payload)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		os.Exit(1)
	}
	if !report.OK {
		os.Exit(1)
	}
}

// Check は行列を検査する。
func Check(payload []byte) Report {
	if len(bytes.TrimSpace(payload)) == 0 {
		return Report{Failure: FailureInvalidMatrix}
	}
	var matrix Matrix
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&matrix); err != nil {
		return Report{Failure: FailureUnsupportedSchema}
	}

	if matrix.Schema != matrixSchema {
		return Report{Failure: FailureUnsupportedSchema}
	}
	if matrix.Authority != authorityName {
		// 行列は認可の正本ではない。authority を大きく名乗る行列を拒否する。
		return Report{Failure: FailureAuthorized}
	}

	problems := []string{}
	if len(matrix.Rows) == 0 {
		problems = append(problems, "matrix has no rows")
	}
	if len(matrix.KnownGaps) == 0 {
		problems = append(problems, "matrix declares no known gap")
	}

	seenKinds := map[string]bool{}
	notRun, unsupported := 0, 0
	for _, row := range matrix.Rows {
		if row.Kind == "" {
			problems = append(problems, "row has no kind")
			continue
		}
		if seenKinds[row.Kind] {
			problems = append(problems, "row kind is repeated")
		}
		seenKinds[row.Kind] = true

		for name, state := range map[string]string{
			"tenantCrossing":    row.TenantCrossing,
			"operationBoundary": row.OperationBoundary,
		} {
			if !allowed(state) {
				problems = append(problems, name+" is not an allowed cell state")
				continue
			}
			switch state {
			case "not-run":
				notRun++
			case "unsupported":
				unsupported++
			}
		}
		// 空欄を pass と読み替えない。空欄は許された状態語彙に無い。
		if row.TenantCrossing == "" || row.OperationBoundary == "" {
			problems = append(problems, "row has an empty cell")
		}
	}

	// 宣言された既知の不足が、行列のどこかに not-run として現れていること。
	// 空欄を pass と読み替えないため、kind の存在だけでは合格にしない。
	for _, gap := range matrix.KnownGaps {
		if !hasNotRunKind(matrix.Rows, gap) {
			problems = append(problems, "declared gap is not marked as not-run")
		}
	}

	if len(problems) > 0 {
		sort.Strings(problems)
		return Report{Failure: FailureInvalidMatrix, Problems: problems, Rows: len(matrix.Rows), Gaps: len(matrix.KnownGaps)}
	}
	return Report{OK: true, Schema: matrixSchema, Rows: len(matrix.Rows), Gaps: len(matrix.KnownGaps), NotRun: notRun, Unsupported: unsupported}
}

func allowed(state string) bool {
	for _, candidate := range allowedStates {
		if state == candidate {
			return true
		}
	}
	return false
}

// gapRoute は knownGaps の名前を、行列の kind 名へ写す。名前が一致する場合はそのまま。
func gapRoute(gap string) string {
	switch gap {
	case "import-operation-boundary":
		return "import"
	}
	return gap
}

func hasNotRunKind(rows []Row, gap string) bool {
	kind := gapRoute(gap)
	for _, row := range rows {
		if row.Kind != kind {
			continue
		}
		return row.TenantCrossing == "not-run" || row.OperationBoundary == "not-run"
	}
	return false
}
