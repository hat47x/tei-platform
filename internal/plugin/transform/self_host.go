package transform

import (
	"encoding/json"

	"tei/internal/core"
)

// SourceValue is the Runtime Engine's typed input projection for a validated,
// normalized transform Source. Normalization is part of the engine contract,
// not ambient builder state.
func SourceValue(source Source) (Value, error) {
	normalized, err := NormalizeSource(source)
	if err != nil {
		return Value{}, err
	}
	raw, err := json.Marshal(normalized)
	if err != nil {
		return Value{}, err
	}
	return ParseJSON(raw)
}

// TransformSourceRealizerV1 is the smallest meta-builder used for SB-3.
// It does not interpret filesystem paths or invoke another tool. Given the
// engine-validated normalized Source value, it emits the canonical realization
// bytes used as an interpreted builder artifact.
func TransformSourceRealizerV1() Source {
	sourceInput := inputExpr("transformSource")
	return Source{
		FormatVersion: SourceFormatVersion,
		Identity: SourceIdentity{
			ID:              core.MustCanonicalID("tei.builder.transform_source"),
			Version:         "1.0.0",
			ContractVersion: "tei.plugin-transform/v1",
		},
		Profile: ProfileV1,
		Limits: Limits{
			MaxSteps:        10_000,
			MaxCollection:   10_000,
			MaxInputNodes:   200_000,
			MaxInputBytes:   16 << 20,
			MaxOutputBytes:  8 << 20,
			MaxNestingDepth: 96,
		},
		Inputs: []InputSpec{{Name: "transformSource", Kind: InputObject}},
		Program: Program{
			Assertions: []Expr{
				assertEqual(selectRequired(sourceInput, "formatVersion"), literalExpr("1")),
				assertEnum(selectRequired(sourceInput, "profile"), ""deterministic-transform-v1""),
			},
			Output: appendBytesExpr(canonicalJSONExpr(sourceInput), "
"),
		},
	}
}
