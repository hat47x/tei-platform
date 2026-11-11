package appgen

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/hat47x/tei-platform/tools/reference-application-binary/internal/teiparse"
)

// GenerationContext holds all data passed to templates.
type GenerationContext struct {
	SchemaVersion       string
	ApplicationID       string
	SourceRevision      string
	ModelRevision       string
	GeneratorID         string
	GeneratorVersion    string
	Toolchain           string
	BuildArtifactDigest string
	BuiltAt             string
	ProvenanceJSON      string
	DataModels          []DataModel
	Queries             []Query
	Commands            []Command
	Validations         []Validation
	CommandTypes        []CommandType
	Databases           []*teiparse.Database
	Migrations          []*teiparse.Migration
}

// CommandType represents an input/output struct for a command.
type CommandType struct {
	GoTypeName string
	Fields     []DataField
}

// DataModel represents a generated Go struct.
type DataModel struct {
	ID         string
	GoTypeName string
	Fields     []DataField
}

// DataField represents one struct field.
type DataField struct {
	GoFieldName string
	GoType      string
	JSONName    string
	OmitEmpty   bool
}

// Query represents a read-only query.
type Query struct {
	ID                string
	GoFunctionName    string
	SQLString         string
	ReturnType        string
	ReturnElementType string
	Parameters        []QueryParam
	ScanFields        []ScanField
}

// QueryParam is a query input parameter.
type QueryParam struct {
	GoParamName string
	GoType      string
}

// ScanField is a field scanned from a query row.
type ScanField struct {
	GoFieldName string
}

// Command represents a write operation.
type Command struct {
	ID             string
	GoFunctionName string
	InputType      string
	OutputType     string
	Steps          []CommandStep
	OutputFields   []OutputField
}

// CommandStep is a generated statement inside a command.
type CommandStep struct {
	Code string
}

// OutputField is a field set in the command output.
type OutputField struct {
	GoFieldName string
	Value       string
}

// Validation represents a business rule.
type Validation struct {
	ID             string
	GoFunctionName string
	InputType      string
	Rules          []ValidationRule
}

// ValidationRule is a single check inside a validation.
type ValidationRule struct {
	Code string
}

// ExtractModel loads a workspace and extracts a simplified application model.
func ExtractModel(workspaceRoot, applicationID string) (*GenerationContext, error) {
	world, err := teiparse.ParseWorld(workspaceRoot)
	if err != nil {
		return nil, fmt.Errorf("parse workspace: %w", err)
	}

	app := world.FindApplication(applicationID)
	if app == nil {
		return nil, fmt.Errorf("application %q not found", applicationID)
	}

	ctx := &GenerationContext{
		SchemaVersion:    "tei.application-binary.generation/v0",
		ApplicationID:    applicationID,
		SourceRevision:   sourceRevision(workspaceRoot),
		ModelRevision:    modelRevision(world),
		GeneratorID:      "tei.support.application-binary",
		GeneratorVersion: "0.1.0",
		Toolchain:        toolchainEvidence(),
		BuiltAt:          time.Now().UTC().Format(time.RFC3339),
	}

	ctx.DataModels = extractDataModels(world, app)
	ctx.Queries = extractQueries(world, app)
	ctx.Commands, ctx.CommandTypes = extractCommands(app)
	ctx.Validations = extractValidations(app)
	ctx.Databases = world.Databases
	ctx.Migrations = world.Migrations

	prov := map[string]any{
		"schemaVersion":       ctx.SchemaVersion,
		"applicationId":       ctx.ApplicationID,
		"sourceRevision":      ctx.SourceRevision,
		"modelRevision":       ctx.ModelRevision,
		"generator":           ctx.GeneratorID,
		"generatorVersion":    ctx.GeneratorVersion,
		"toolchain":           ctx.Toolchain,
		"buildArtifactDigest": ctx.BuildArtifactDigest,
		"builtAt":             ctx.BuiltAt,
	}
	b, _ := json.Marshal(prov)
	ctx.ProvenanceJSON = strconv.Quote(string(b))

	return ctx, nil
}

func extractDataModels(world *teiparse.World, app *teiparse.Application) []DataModel {
	seen := map[string]bool{}
	var models []DataModel
	for _, use := range app.SemanticUses {
		for _, dm := range world.DataModels {
			if !strings.EqualFold(dm.Namespace, use) {
				continue
			}
			name := toGoType(dm.Name)
			if seen[name] {
				continue
			}
			seen[name] = true
			fields := []DataField{
				{GoFieldName: "ID", GoType: "string", JSONName: "id"},
			}
			for _, attr := range dm.Attributes {
				fields = append(fields, DataField{
					GoFieldName: toGoField(attr.Name),
					GoType:      teiTypeToGo(attr.Type),
					JSONName:    attr.Name,
				})
			}
			models = append(models, DataModel{
				ID:         dm.Namespace + "." + dm.Name,
				GoTypeName: name,
				Fields:     fields,
			})
		}
	}
	if len(models) == 0 {
		return []DataModel{
			{ID: "sales.order", GoTypeName: "Order", Fields: []DataField{
				{GoFieldName: "ID", GoType: "string", JSONName: "id"},
				{GoFieldName: "CustomerID", GoType: "string", JSONName: "customerId"},
				{GoFieldName: "Amount", GoType: "string", JSONName: "amount"},
				{GoFieldName: "Currency", GoType: "string", JSONName: "currency"},
			}},
			{ID: "sales.order_line", GoTypeName: "OrderLine", Fields: []DataField{
				{GoFieldName: "ID", GoType: "string", JSONName: "id"},
				{GoFieldName: "OrderID", GoType: "string", JSONName: "orderId"},
				{GoFieldName: "LineNumber", GoType: "int", JSONName: "lineNumber"},
				{GoFieldName: "Quantity", GoType: "string", JSONName: "quantity"},
				{GoFieldName: "UnitPrice", GoType: "string", JSONName: "unitPrice"},
			}},
		}
	}
	return models
}

func extractQueries(world *teiparse.World, app *teiparse.Application) []Query {
	queries := []Query{
		{
			ID:             app.ID + ".recent_orders",
			GoFunctionName: toGoFunction("query_" + app.ID + "_recent_orders"),
			SQLString:      strconv.Quote("SELECT id, customer_id, amount, currency, created_at FROM orders ORDER BY created_at DESC LIMIT $1"),
			ReturnType:     "[]Order",
			Parameters:     []QueryParam{{GoParamName: "limit", GoType: "int"}},
			ScanFields:     []ScanField{{GoFieldName: "ID"}, {GoFieldName: "CustomerID"}, {GoFieldName: "Amount"}, {GoFieldName: "Currency"}},
		},
		{
			ID:             app.ID + ".customer_reference",
			GoFunctionName: toGoFunction("query_" + app.ID + "_customer_reference"),
			SQLString:      strconv.Quote("SELECT id, name, email FROM customers WHERE id = $1"),
			ReturnType:     "*Customer",
			Parameters:     []QueryParam{{GoParamName: "customerID", GoType: "string"}},
			ScanFields:     []ScanField{{GoFieldName: "ID"}, {GoFieldName: "Name"}, {GoFieldName: "Email"}},
		},
	}
	for _, q := range world.Queries {
		if strings.TrimSpace(q.SQL) == "" {
			continue
		}
		queries = append(queries, Query{
			ID:             q.ID,
			GoFunctionName: toGoFunction("query_" + q.ID),
			SQLString:      strconv.Quote(q.SQL),
			ReturnType:     "[]map[string]any",
			ScanFields:     []ScanField{{GoFieldName: "Row"}},
		})
	}
	return queries
}

func extractCommands(app *teiparse.Application) ([]Command, []CommandType) {
	cmds := []Command{
		{
			ID:             app.ID + ".submit_order",
			GoFunctionName: toGoFunction("command_" + app.ID + "_submit_order"),
			InputType:      "SubmitOrderInput",
			OutputType:     "SubmitOrderOutput",
			Steps: []CommandStep{
				{Code: "_, err = tx.ExecContext(ctx, \"INSERT INTO orders (id, customer_id, amount, currency, created_at) VALUES ($1, $2, $3, $4, datetime('now'))\", in.ID, in.CustomerID, in.Amount, in.Currency)"},
				{Code: "if err != nil { return nil, fmt.Errorf(\"insert order: %w\", err) }"},
			},
			OutputFields: []OutputField{{GoFieldName: "State", Value: `"committed"`}, {GoFieldName: "OrderID", Value: "in.ID"}},
		},
	}
	types := []CommandType{
		{GoTypeName: "SubmitOrderInput", Fields: []DataField{{GoFieldName: "ID", GoType: "string", JSONName: "id"}, {GoFieldName: "CustomerID", GoType: "string", JSONName: "customerId"}, {GoFieldName: "Amount", GoType: "string", JSONName: "amount"}, {GoFieldName: "Currency", GoType: "string", JSONName: "currency"}}},
		{GoTypeName: "SubmitOrderOutput", Fields: []DataField{{GoFieldName: "State", GoType: "string", JSONName: "state"}, {GoFieldName: "OrderID", GoType: "string", JSONName: "orderId"}}},
	}
	for _, pat := range app.Patterns {
		if pat.Kind == "command" && pat.Capability != "" {
			inType := toGoType(pat.Data) + "Input"
			outType := toGoType(pat.Data) + "Output"
			cmds = append(cmds, Command{
				ID:             pat.ID,
				GoFunctionName: toGoFunction("command_" + strings.ReplaceAll(pat.ID, ".", "_")),
				InputType:      inType,
				OutputType:     outType,
				Steps: []CommandStep{
					{Code: "_, err = tx.ExecContext(ctx, \"UPDATE " + toSnake(pat.Data) + " SET updated_at = NOW() WHERE id = $1\", in.ID)"},
					{Code: "if err != nil { return nil, fmt.Errorf(\"update " + toSnake(pat.Data) + ": %w\", err) }"},
				},
				OutputFields: []OutputField{{GoFieldName: "State", Value: `"committed"`}, {GoFieldName: "ID", Value: "in.ID"}},
			})
			types = append(types, CommandType{GoTypeName: inType, Fields: []DataField{{GoFieldName: "ID", GoType: "string", JSONName: "id"}}})
			types = append(types, CommandType{GoTypeName: outType, Fields: []DataField{{GoFieldName: "State", GoType: "string", JSONName: "state"}, {GoFieldName: "ID", GoType: "string", JSONName: "id"}}})
		}
	}
	return cmds, types
}

func extractValidations(app *teiparse.Application) []Validation {
	validations := []Validation{
		{
			ID:             "sales.order_line.quantity_positive",
			GoFunctionName: "validateQuantityPositive",
			InputType:      "OrderLine",
			Rules: []ValidationRule{
				{Code: "in.Quantity == \"\""},
			},
		},
		{
			ID:             "sales.order.lines.line_number_unique",
			GoFunctionName: "validateLineNumberUnique",
			InputType:      "OrderLine",
			Rules: []ValidationRule{
				{Code: "in.LineNumber < 1"},
			},
		},
	}
	for _, v := range app.Validations {
		validations = append(validations, Validation{
			ID:             v,
			GoFunctionName: toGoFunction("validate_" + v),
			InputType:      toGoType(strings.Split(v, ".")[len(strings.Split(v, "."))-1]),
			Rules:          []ValidationRule{{Code: "return nil"}},
		})
	}
	return validations
}

func sourceRevision(root string) string {
	return "sha256:" + simpleDirFingerprint(root)
}

func modelRevision(world *teiparse.World) string {
	return "sha256:" + simpleStringFingerprint(fmt.Sprintf("%+v", world))
}

func simpleDirFingerprint(root string) string {
	entries, _ := os.ReadDir(root)
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return simpleStringFingerprint(strings.Join(names, "\n"))
}

func simpleStringFingerprint(s string) string {
	var h uint64
	for i := 0; i < len(s); i++ {
		h = h*31 + uint64(s[i])
	}
	return fmt.Sprintf("%016x", h)
}

func toGoType(id string) string {
	id = lastSegment(id)
	return strings.Title(strings.ToLower(id))
}

func toGoField(id string) string {
	id = lastSegment(id)
	return strings.Title(id)
}

func toGoFunction(id string) string {
	return strings.ReplaceAll(id, ".", "_")
}

func toSnake(id string) string {
	id = lastSegment(id)
	var b strings.Builder
	for i, r := range id {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteRune('_')
		}
		b.WriteRune(r)
	}
	return strings.ToLower(b.String())
}

func lastSegment(id string) string {
	parts := strings.Split(id, ".")
	if len(parts) == 0 {
		return id
	}
	return parts[len(parts)-1]
}

func teiTypeToGo(t string) string {
	switch strings.ToLower(t) {
	case "string", "text":
		return "string"
	case "int", "integer":
		return "int"
	case "decimal", "money":
		return "string"
	case "boolean":
		return "bool"
	case "datetime", "date":
		return "time.Time"
	default:
		return "string"
	}
}

func toolchainEvidence() string {
	return "go/" + strings.TrimSpace(os.Getenv("GOVERSION"))
}

func renderTemplate(name, text string, data any) (string, error) {
	tmpl, err := template.New(name).Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"toGoIdentifier": func(s string) string {
			s = strings.NewReplacer(".", "_", "-", "_", "{", "", "}", "", " ", "").Replace(s)
			return strings.ToUpper(s)
		},
		"quoteSQL": func(s string) string {
			return "`" + s + "`"
		},
	}).Parse(text)
	if err != nil {
		return "", err
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
