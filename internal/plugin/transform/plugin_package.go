package transform

import (
	"encoding/json"

	"tei/internal/core"
)

func PluginPackageSourceV1() Source {
	normalizedDefinition := normalizePluginDefinitionExpr()
	normalizedPlan := normalizeBuildPlanExpr()
	definitionFingerprint := sha256Expr(canonicalJSONExpr(normalizedDefinition))
	planFingerprint := sha256Expr(canonicalJSONExpr(normalizedPlan))

	return Source{
		FormatVersion: SourceFormatVersion,
		Identity: SourceIdentity{
			ID:              core.MustCanonicalID("tei.builder.plugin_package"),
			Version:         "1.0.0",
			ContractVersion: "tei.plugin-transform/v1",
		},
		Profile: ProfileV1,
		Limits: Limits{
			MaxSteps:        100_000,
			MaxCollection:   10_000,
			MaxOutputBytes:  8 << 20,
			MaxNestingDepth: 96,
		},
		Inputs: []InputSpec{
			{Name: "buildPlan", Kind: InputObject},
			{Name: "pluginDefinition", Kind: InputObject},
		},
		Program: Program{
			Assertions: []Expr{
				assertEqual(selectRequired(inputExpr("buildPlan"), "formatVersion"), literalExpr("1")),
				assertEnum(selectRequired(inputExpr("buildPlan"), "outputKind"), "\"plugin-package\""),
				assertEqual(
					selectRequired(selectRequired(inputExpr("buildPlan"), "plugin"), "id"),
					selectRequired(inputExpr("pluginDefinition"), "id"),
				),
				assertEqual(
					selectRequired(selectRequired(inputExpr("buildPlan"), "plugin"), "version"),
					selectRequired(inputExpr("pluginDefinition"), "version"),
				),
				assertEnum(selectRequired(inputExpr("pluginDefinition"), "artifact"), "{}"),
				assertEqual(
					selectRequired(inputExpr("buildPlan"), "pluginDefinitionFingerprint"),
					definitionFingerprint,
				),
			},
			Output: appendBytesExpr(
				canonicalJSONExpr(constructObjectExpr(
					fieldExpr("formatVersion", literalExpr("1")),
					fieldExpr("plugin", selectRequired(inputExpr("buildPlan"), "plugin")),
					fieldExpr("buildPlanFingerprint", planFingerprint),
					fieldExpr("pluginDefinitionFingerprint", definitionFingerprint),
					fieldExpr("definition", normalizedDefinition),
				)),
				"\n",
			),
		},
	}
}

func normalizePluginDefinitionExpr() Expr {
	capability := inputExpr("capability")
	normalizedCapability := capability
	for _, field := range []string{
		"actions",
		"observations",
		"evidence",
		"databaseProviders",
		"migrationTools",
		"features",
	} {
		normalizedCapability = replaceFieldExpr(
			normalizedCapability,
			field,
			sortArrayExpr(
				selectRequired(capability, field),
				"scalar",
				inputExpr("scalar"),
			),
		)
	}
	normalizedCapability = replaceFieldExpr(
		normalizedCapability,
		"permissions",
		sortArrayExpr(
			selectRequired(capability, "permissions"),
			"permission",
			joinStringsExpr(
				"|",
				selectRequired(inputExpr("permission"), "kind"),
				selectRequired(inputExpr("permission"), "access"),
				selectRequired(inputExpr("permission"), "scope"),
			),
		),
	)

	definition := inputExpr("pluginDefinition")
	normalized := definition
	normalized = replaceFieldExpr(
		normalized,
		"provides",
		sortArrayExpr(
			mapArrayExpr(
				selectRequired(definition, "provides"),
				"capability",
				normalizedCapability,
			),
			"provided",
			selectRequired(inputExpr("provided"), "kind"),
			selectRequired(inputExpr("provided"), "id"),
		),
	)
	normalized = replaceFieldExpr(
		normalized,
		"requires",
		sortArrayExpr(
			selectRequired(definition, "requires"),
			"requirement",
			selectRequired(inputExpr("requirement"), "plugin"),
		),
	)
	normalized = replaceFieldExpr(
		normalized,
		"externalRequirements",
		sortArrayExpr(
			selectRequired(definition, "externalRequirements"),
			"external",
			selectRequired(inputExpr("external"), "scope"),
			selectRequired(inputExpr("external"), "kind"),
			selectRequired(inputExpr("external"), "name"),
		),
	)
	return normalized
}

func normalizeBuildPlanExpr() Expr {
	plan := inputExpr("buildPlan")
	normalized := plan
	normalized = replaceFieldExpr(
		normalized,
		"sources",
		sortArrayExpr(
			selectRequired(plan, "sources"),
			"source",
			selectRequired(inputExpr("source"), "kind"),
			selectRequired(inputExpr("source"), "id"),
			selectRequired(inputExpr("source"), "locator"),
		),
	)
	normalized = replaceFieldExpr(
		normalized,
		"dependencies",
		sortArrayExpr(
			selectRequired(plan, "dependencies"),
			"dependency",
			selectRequired(inputExpr("dependency"), "kind"),
			selectRequired(inputExpr("dependency"), "id"),
			selectRequired(inputExpr("dependency"), "version"),
		),
	)
	normalized = replaceFieldExpr(
		normalized,
		"parameters",
		sortArrayExpr(
			selectRequired(plan, "parameters"),
			"parameter",
			selectRequired(inputExpr("parameter"), "name"),
		),
	)
	return normalized
}

func inputExpr(name string) Expr {
	return Expr{Op: OpInput, Name: name}
}

func literalExpr(raw string) Expr {
	return Expr{Op: OpLiteral, Literal: json.RawMessage(raw)}
}

func selectRequired(from Expr, field string) Expr {
	return Expr{Op: OpSelectField, From: exprPtr(from), Field: field}
}

func constructObjectExpr(fields ...ObjectFieldExpr) Expr {
	return Expr{Op: OpConstructObject, Fields: fields}
}

func fieldExpr(name string, value Expr) ObjectFieldExpr {
	return ObjectFieldExpr{Name: name, Value: value}
}

func mapArrayExpr(from Expr, item string, body Expr) Expr {
	return Expr{Op: OpMapArray, From: exprPtr(from), Item: item, Body: exprPtr(body)}
}

func sortArrayExpr(from Expr, item string, keys ...Expr) Expr {
	return Expr{Op: OpSortArrayBy, From: exprPtr(from), Item: item, Keys: keys}
}

func replaceFieldExpr(from Expr, field string, value Expr) Expr {
	return Expr{Op: OpReplaceField, From: exprPtr(from), Field: field, Value: exprPtr(value)}
}

func assertEqual(actual, expected Expr) Expr {
	return Expr{Op: OpAssertEqual, Actual: exprPtr(actual), Expected: exprPtr(expected)}
}

func assertEnum(actual Expr, allowed ...string) Expr {
	values := make([]json.RawMessage, len(allowed))
	for i := range allowed {
		values[i] = json.RawMessage(allowed[i])
	}
	return Expr{Op: OpAssertEnum, Actual: exprPtr(actual), Allowed: values}
}

func canonicalJSONExpr(from Expr) Expr {
	return Expr{Op: OpCanonicalJSON, From: exprPtr(from)}
}

func sha256Expr(from Expr) Expr {
	return Expr{Op: OpSHA256, From: exprPtr(from)}
}

func appendBytesExpr(from Expr, suffix string) Expr {
	return Expr{Op: OpAppendBytes, From: exprPtr(from), Suffix: suffix}
}

func joinStringsExpr(separator string, values ...Expr) Expr {
	return Expr{Op: OpJoinStrings, Values: values, Separator: separator}
}

func exprPtr(expr Expr) *Expr {
	copy := expr
	return &copy
}
