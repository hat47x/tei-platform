package main

import (
	"fmt"
	"regexp"
	"strings"

	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
)

const (
	opAddValidation               = "add-validation"
	opAddData                     = "add-data"
	opAddRelation                 = "add-relation"
	opAddEnvironmentBinding       = "add-environment-binding"
	opAddApplication              = "add-application"
	opAddReferencePattern         = "add-reference-pattern"
	opAddApplicationValidationUse = "add-application-validation-use"
)

// Operationは、型付きの最小semantic deltaの一単位である。
// 任意のTEI textやfile patchは運ばず、意味とplacementだけを持ち、sourceは意味から決定論的に生成する。
// add-validationは既存fileへ追記し、それ以外は新しいfileを作る。
type Operation struct {
	Op        string `json:"op"`
	Namespace string `json:"namespace,omitempty"`
	ID        string `json:"id,omitempty"`
	Kind      string `json:"kind,omitempty"`
	Attribute string `json:"attribute,omitempty"`
	Relation  string `json:"relation,omitempty"`
	Placement string `json:"placement,omitempty"`

	Data        *DataSpec        `json:"data,omitempty"`
	AddRelation *AddRelationSpec `json:"addRelation,omitempty"`
	Application *ApplicationSpec `json:"application,omitempty"`
	Pattern     *PatternSpec     `json:"pattern,omitempty"`

	EnvironmentBinding *EnvironmentBindingSpec `json:"environmentBinding,omitempty"`

	ApplicationValidationUse *ApplicationValidationUseSpec `json:"applicationValidationUse,omitempty"`
}

// ApplicationValidationUseSpecは、既存のApplicationが評価するValidationを一件宣言する（`uses validation`）。
// 宣言順が評価順になる（ADR-0029）ため、既存の宣言の後ろへ追加する。
type ApplicationValidationUseSpec struct {
	Application string `json:"application"`
	Validation  string `json:"validation"`
}

// EnvironmentBindingSpecは、Applicationが宣言したdatasource aliasを、Environmentが持つ
// database instanceへ結ぶ決定を表す。Application側の意味ではなくEnvironment側のauthorityである。
type EnvironmentBindingSpec struct {
	Environment string `json:"environment"`
	Application string `json:"application"`
	Alias       string `json:"alias"`
	Instance    string `json:"instance"`
}

// DataSpecは、新しいsemantic namespaceに一つのDataを追加する意味を表す。
type DataSpec struct {
	Namespace  string          `json:"namespace"`
	ID         string          `json:"id"`
	Symbol     string          `json:"symbol"`
	Attributes []AttributeSpec `json:"attributes,omitempty"`
	Relations  []RelationSpec  `json:"relations,omitempty"`
}

type AttributeSpec struct {
	ID     string `json:"id"`
	Symbol string `json:"symbol"`
	Type   string `json:"type"`
}

type RelationSpec struct {
	ID          string            `json:"id"`
	Symbol      string            `json:"symbol"`
	Target      string            `json:"target"`
	Cardinality string            `json:"cardinality"`
	Keys        []RelationKeySpec `json:"keys,omitempty"`
}

type RelationKeySpec struct {
	SourceAttribute string `json:"sourceAttribute"`
	TargetAttribute string `json:"targetAttribute"`
}

// AddRelationSpecは、既存のDataへRelationを一つ追加する意味を表す。
// sourceは対象Dataを宣言したfileのdata blockへ挿入するため、placementは持たない。
type AddRelationSpec struct {
	Data        string            `json:"data"`
	ID          string            `json:"id"`
	Symbol      string            `json:"symbol"`
	Target      string            `json:"target"`
	Cardinality string            `json:"cardinality"`
	Keys        []RelationKeySpec `json:"keys,omitempty"`
}

// ApplicationSpecは、Environment bindingを含まないApplication定義の意味を表す。
// usesSemanticとdatabasesの順序は、authoredされたsourceの順序としてそのまま保つ。
type ApplicationSpec struct {
	ID           string         `json:"id"`
	UsesSemantic []string       `json:"usesSemantic"`
	Databases    []DatabaseSpec `json:"databases,omitempty"`
}

type DatabaseSpec struct {
	Alias     string                    `json:"alias"`
	Target    string                    `json:"target"`
	Access    string                    `json:"access"`
	Contracts []ContractRequirementSpec `json:"requires,omitempty"`
}

type ContractRequirementSpec struct {
	Contract string `json:"contract"`
	Major    int    `json:"major"`
}

// PatternSpecは、reference Patternの意味を表す。browse / record / command / header-detailは扱わない。
type PatternSpec struct {
	ID            string            `json:"id"`
	Application   string            `json:"application"`
	Relation      string            `json:"relation"`
	Datasource    string            `json:"datasource"`
	ContractQuery ContractQuerySpec `json:"contractQuery"`
}

type ContractQuerySpec struct {
	Contract string           `json:"contract"`
	Query    string           `json:"query"`
	Inputs   []QueryInputSpec `json:"inputs,omitempty"`
}

type QueryInputSpec struct {
	Attribute string `json:"attribute"`
	Input     string `json:"input"`
}

var symbolPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

func canonical(field, value string) (core.CanonicalID, error) {
	id, err := core.ParseCanonicalID(value)
	if err != nil {
		return "", fmt.Errorf("%s: %w", field, err)
	}
	return id, nil
}

func localName(field, value string) (string, error) {
	name, err := core.ParseLocalName(value)
	if err != nil {
		return "", fmt.Errorf("%s: %w", field, err)
	}
	return string(name), nil
}

func normalizePlacement(op Operation) (string, error) {
	if strings.TrimSpace(op.Placement) == "" {
		return "", nil
	}
	p, err := sourceindex.NormalizeWorkspacePath(op.Placement)
	if err != nil {
		return "", fmt.Errorf("placement: %w", err)
	}
	return p, nil
}

func onlyFields(op Operation, allowed ...string) error {
	present := map[string]bool{
		"namespace": op.Namespace != "", "id": op.ID != "", "kind": op.Kind != "",
		"attribute": op.Attribute != "", "relation": op.Relation != "",
		"data": op.Data != nil, "addRelation": op.AddRelation != nil, "application": op.Application != nil, "pattern": op.Pattern != nil,
	}
	for _, name := range allowed {
		delete(present, name)
	}
	for name, has := range present {
		if has {
			return fmt.Errorf("operation %q は %s を受け付けません", op.Op, name)
		}
	}
	return nil
}

func normalizeOperation(op Operation) (Operation, error) {
	op.Op = strings.TrimSpace(op.Op)
	switch op.Op {
	case opAddValidation:
		return normalizeAddValidation(op)
	case opAddData:
		return normalizeAddData(op)
	case opAddRelation:
		return normalizeAddRelation(op)
	case opAddEnvironmentBinding:
		return normalizeAddEnvironmentBinding(op)
	case opAddApplicationValidationUse:
		return normalizeAddApplicationValidationUse(op)
	case opAddApplication:
		return normalizeAddApplication(op)
	case opAddReferencePattern:
		return normalizeAddReferencePattern(op)
	default:
		return op, fmt.Errorf("operation %q は未対応です（%s / %s / %s / %s / %s / %s / %s だけを扱います）",
			op.Op, opAddValidation, opAddData, opAddRelation, opAddApplication, opAddReferencePattern, opAddEnvironmentBinding, opAddApplicationValidationUse)
	}
}

func normalizeAddValidation(op Operation) (Operation, error) {
	if err := onlyFields(op, "namespace", "id", "kind", "attribute", "relation"); err != nil {
		return op, err
	}
	namespace, err := canonical("namespace", op.Namespace)
	if err != nil {
		return op, err
	}
	id, err := canonical("id", op.ID)
	if err != nil {
		return op, err
	}
	if !strings.HasPrefix(string(id), string(namespace)+".") {
		return op, fmt.Errorf("validation id %s はnamespace %s に属していません", id, namespace)
	}
	kind := semantic.ValidationKind(strings.TrimSpace(op.Kind))
	if !kind.Valid() {
		return op, fmt.Errorf("validation kind %q は未対応です", op.Kind)
	}
	attribute, err := canonical("attribute", op.Attribute)
	if err != nil {
		return op, err
	}
	relation := core.CanonicalID("")
	switch kind {
	case semantic.ValidationPositive:
		if strings.TrimSpace(op.Relation) != "" {
			return op, fmt.Errorf("positive validation %s はrelationを受け付けません", id)
		}
	case semantic.ValidationRelationUnique:
		relation, err = canonical("relation", op.Relation)
		if err != nil {
			return op, fmt.Errorf("relation-unique validation %s にはrelationが必要です: %w", id, err)
		}
	}
	placement, err := normalizePlacement(op)
	if err != nil {
		return op, err
	}
	return Operation{
		Op: op.Op, Namespace: string(namespace), ID: string(id), Kind: string(kind),
		Attribute: string(attribute), Relation: string(relation), Placement: placement,
	}, nil
}

func segmentOf(symbol string) (string, error) {
	if !symbolPattern.MatchString(symbol) {
		return "", fmt.Errorf("symbol %q は英字で始まる英数字とアンダースコアだけで書く必要があります", symbol)
	}
	return core.SymbolToSegment(symbol)
}

func normalizeAddData(op Operation) (Operation, error) {
	if err := onlyFields(op, "data"); err != nil {
		return op, err
	}
	if op.Data == nil {
		return op, fmt.Errorf("add-data にはdataが必要です")
	}
	placement, err := normalizePlacement(op)
	if err != nil {
		return op, err
	}
	if placement == "" {
		return op, fmt.Errorf("add-data にはplacement（新しいsemantic fileのpath）が必要です")
	}
	d := *op.Data
	namespace, err := canonical("data.namespace", d.Namespace)
	if err != nil {
		return op, err
	}
	id, err := canonical("data.id", d.ID)
	if err != nil {
		return op, err
	}
	segment, err := segmentOf(d.Symbol)
	if err != nil {
		return op, fmt.Errorf("data.symbol: %w", err)
	}
	if string(id) != string(namespace)+"."+segment {
		return op, fmt.Errorf("data.id %s は namespace %s とsymbol %s から導かれるid %s.%s と一致しません", id, namespace, d.Symbol, namespace, segment)
	}

	seen := map[string]bool{}
	out := DataSpec{Namespace: string(namespace), ID: string(id), Symbol: d.Symbol}
	for i, a := range d.Attributes {
		aid, err := canonical(fmt.Sprintf("data.attributes[%d].id", i), a.ID)
		if err != nil {
			return op, err
		}
		aseg, err := segmentOf(a.Symbol)
		if err != nil {
			return op, fmt.Errorf("data.attributes[%d].symbol: %w", i, err)
		}
		if string(aid) != string(id)+"."+aseg {
			return op, fmt.Errorf("data.attributes[%d].id %s はsymbol %s から導かれるid %s.%s と一致しません", i, aid, a.Symbol, id, aseg)
		}
		typ, err := canonical(fmt.Sprintf("data.attributes[%d].type", i), a.Type)
		if err != nil {
			return op, err
		}
		if seen[string(aid)] {
			return op, fmt.Errorf("data.attributes[%d]: %s が重複しています", i, aid)
		}
		seen[string(aid)] = true
		out.Attributes = append(out.Attributes, AttributeSpec{ID: string(aid), Symbol: a.Symbol, Type: string(typ)})
	}
	for i, r := range d.Relations {
		rid, err := canonical(fmt.Sprintf("data.relations[%d].id", i), r.ID)
		if err != nil {
			return op, err
		}
		rseg, err := segmentOf(r.Symbol)
		if err != nil {
			return op, fmt.Errorf("data.relations[%d].symbol: %w", i, err)
		}
		if string(rid) != string(id)+"."+rseg {
			return op, fmt.Errorf("data.relations[%d].id %s はsymbol %s から導かれるid %s.%s と一致しません", i, rid, r.Symbol, id, rseg)
		}
		target, err := canonical(fmt.Sprintf("data.relations[%d].target", i), r.Target)
		if err != nil {
			return op, err
		}
		if !semantic.Cardinality(r.Cardinality).Valid() {
			return op, fmt.Errorf("data.relations[%d].cardinality %q は one / many のいずれかである必要があります", i, r.Cardinality)
		}
		if seen[string(rid)] {
			return op, fmt.Errorf("data.relations[%d]: %s が重複しています", i, rid)
		}
		seen[string(rid)] = true
		spec := RelationSpec{ID: string(rid), Symbol: r.Symbol, Target: string(target), Cardinality: r.Cardinality}
		for k, key := range r.Keys {
			source, err := canonical(fmt.Sprintf("data.relations[%d].keys[%d].sourceAttribute", i, k), key.SourceAttribute)
			if err != nil {
				return op, err
			}
			targetAttr, err := canonical(fmt.Sprintf("data.relations[%d].keys[%d].targetAttribute", i, k), key.TargetAttribute)
			if err != nil {
				return op, err
			}
			spec.Keys = append(spec.Keys, RelationKeySpec{SourceAttribute: string(source), TargetAttribute: string(targetAttr)})
		}
		out.Relations = append(out.Relations, spec)
	}
	return Operation{Op: op.Op, Placement: placement, Data: &out}, nil
}

func normalizeAddRelation(op Operation) (Operation, error) {
	if err := onlyFields(op, "addRelation"); err != nil {
		return op, err
	}
	if op.AddRelation == nil {
		return op, fmt.Errorf("add-relation にはaddRelationが必要です")
	}
	if strings.TrimSpace(op.Placement) != "" {
		return op, fmt.Errorf("add-relation はplacementを受け付けません（sourceは対象Dataを宣言したfileのdata blockへ入ります）")
	}
	r := *op.AddRelation
	data, err := canonical("addRelation.data", r.Data)
	if err != nil {
		return op, err
	}
	id, err := canonical("addRelation.id", r.ID)
	if err != nil {
		return op, err
	}
	segment, err := segmentOf(r.Symbol)
	if err != nil {
		return op, fmt.Errorf("addRelation.symbol: %w", err)
	}
	if string(id) != string(data)+"."+segment {
		return op, fmt.Errorf("addRelation.id %s は data %s とsymbol %s から導かれるid %s.%s と一致しません", id, data, r.Symbol, data, segment)
	}
	target, err := canonical("addRelation.target", r.Target)
	if err != nil {
		return op, err
	}
	if !semantic.Cardinality(r.Cardinality).Valid() {
		return op, fmt.Errorf("addRelation.cardinality %q は one / many のいずれかである必要があります", r.Cardinality)
	}
	out := AddRelationSpec{Data: string(data), ID: string(id), Symbol: r.Symbol, Target: string(target), Cardinality: r.Cardinality}
	seen := map[RelationKeySpec]bool{}
	for k, key := range r.Keys {
		source, err := canonical(fmt.Sprintf("addRelation.keys[%d].sourceAttribute", k), key.SourceAttribute)
		if err != nil {
			return op, err
		}
		targetAttr, err := canonical(fmt.Sprintf("addRelation.keys[%d].targetAttribute", k), key.TargetAttribute)
		if err != nil {
			return op, err
		}
		spec := RelationKeySpec{SourceAttribute: string(source), TargetAttribute: string(targetAttr)}
		if seen[spec] {
			return op, fmt.Errorf("addRelation.keys[%d]: 同じkey対応が重複しています", k)
		}
		seen[spec] = true
		out.Keys = append(out.Keys, spec)
	}
	return Operation{Op: op.Op, AddRelation: &out}, nil
}

func normalizeAddApplication(op Operation) (Operation, error) {
	if err := onlyFields(op, "application"); err != nil {
		return op, err
	}
	if op.Application == nil {
		return op, fmt.Errorf("add-application にはapplicationが必要です")
	}
	placement, err := normalizePlacement(op)
	if err != nil {
		return op, err
	}
	if placement == "" {
		return op, fmt.Errorf("add-application にはplacement（新しいapplication.tei）が必要です")
	}
	a := *op.Application
	id, err := canonical("application.id", a.ID)
	if err != nil {
		return op, err
	}
	if len(a.UsesSemantic) == 0 {
		return op, fmt.Errorf("application.usesSemantic が空です")
	}
	out := ApplicationSpec{ID: string(id)}
	seen := map[string]bool{}
	for i, u := range a.UsesSemantic {
		ns, err := canonical(fmt.Sprintf("application.usesSemantic[%d]", i), u)
		if err != nil {
			return op, err
		}
		if seen[string(ns)] {
			return op, fmt.Errorf("application.usesSemantic[%d]: %s が重複しています", i, ns)
		}
		seen[string(ns)] = true
		out.UsesSemantic = append(out.UsesSemantic, string(ns))
	}
	aliases := map[string]bool{}
	for i, db := range a.Databases {
		alias, err := localName(fmt.Sprintf("application.databases[%d].alias", i), db.Alias)
		if err != nil {
			return op, err
		}
		target, err := localName(fmt.Sprintf("application.databases[%d].target", i), db.Target)
		if err != nil {
			return op, err
		}
		if !database.Access(db.Access).Valid() {
			return op, fmt.Errorf("application.databases[%d].access %q は read / write / read-write のいずれかである必要があります", i, db.Access)
		}
		if aliases[alias] {
			return op, fmt.Errorf("application.databases[%d]: alias %s が重複しています", i, alias)
		}
		aliases[alias] = true
		spec := DatabaseSpec{Alias: alias, Target: target, Access: db.Access}
		for k, c := range db.Contracts {
			contract, err := canonical(fmt.Sprintf("application.databases[%d].requires[%d].contract", i, k), c.Contract)
			if err != nil {
				return op, err
			}
			if c.Major < 1 {
				return op, fmt.Errorf("application.databases[%d].requires[%d].major は1以上である必要があります", i, k)
			}
			spec.Contracts = append(spec.Contracts, ContractRequirementSpec{Contract: string(contract), Major: c.Major})
		}
		out.Databases = append(out.Databases, spec)
	}
	return Operation{Op: op.Op, Placement: placement, Application: &out}, nil
}

func normalizeAddReferencePattern(op Operation) (Operation, error) {
	if err := onlyFields(op, "pattern"); err != nil {
		return op, err
	}
	if op.Pattern == nil {
		return op, fmt.Errorf("add-reference-pattern にはpatternが必要です")
	}
	placement, err := normalizePlacement(op)
	if err != nil {
		return op, err
	}
	if placement == "" {
		return op, fmt.Errorf("add-reference-pattern にはplacement（新しいpattern file）が必要です")
	}
	p := *op.Pattern
	id, err := canonical("pattern.id", p.ID)
	if err != nil {
		return op, err
	}
	app, err := canonical("pattern.application", p.Application)
	if err != nil {
		return op, err
	}
	relation, err := canonical("pattern.relation", p.Relation)
	if err != nil {
		return op, err
	}
	datasource, err := localName("pattern.datasource", p.Datasource)
	if err != nil {
		return op, err
	}
	contract, err := canonical("pattern.contractQuery.contract", p.ContractQuery.Contract)
	if err != nil {
		return op, err
	}
	query, err := localName("pattern.contractQuery.query", p.ContractQuery.Query)
	if err != nil {
		return op, err
	}
	out := PatternSpec{
		ID: string(id), Application: string(app), Relation: string(relation), Datasource: datasource,
		ContractQuery: ContractQuerySpec{Contract: string(contract), Query: query},
	}
	for i, in := range p.ContractQuery.Inputs {
		attribute, err := canonical(fmt.Sprintf("pattern.contractQuery.inputs[%d].attribute", i), in.Attribute)
		if err != nil {
			return op, err
		}
		input, err := localName(fmt.Sprintf("pattern.contractQuery.inputs[%d].input", i), in.Input)
		if err != nil {
			return op, err
		}
		out.ContractQuery.Inputs = append(out.ContractQuery.Inputs, QueryInputSpec{Attribute: string(attribute), Input: input})
	}
	return Operation{Op: op.Op, Placement: placement, Pattern: &out}, nil
}

func normalizeAddEnvironmentBinding(op Operation) (Operation, error) {
	if err := onlyFields(op, "placement", "environmentBinding"); err != nil {
		return op, err
	}
	if op.EnvironmentBinding == nil {
		return op, fmt.Errorf("%s にはenvironmentBindingが必要です", opAddEnvironmentBinding)
	}
	spec := *op.EnvironmentBinding
	env, err := canonical("environmentBinding.environment", spec.Environment)
	if err != nil {
		return op, err
	}
	application, err := canonical("environmentBinding.application", spec.Application)
	if err != nil {
		return op, err
	}
	instance, err := canonical("environmentBinding.instance", spec.Instance)
	if err != nil {
		return op, err
	}
	alias := strings.TrimSpace(spec.Alias)
	if alias == "" {
		return op, fmt.Errorf("environmentBinding.aliasが空です")
	}
	if strings.ContainsAny(alias, " \t.") {
		return op, fmt.Errorf("environmentBinding.alias %q はApplication内のlocal nameです", alias)
	}
	placement := strings.TrimSpace(op.Placement)
	if placement == "" {
		return op, fmt.Errorf("%s にはplacement（Environment source file）が必要です", opAddEnvironmentBinding)
	}
	return Operation{Op: op.Op, Placement: placement, EnvironmentBinding: &EnvironmentBindingSpec{
		Environment: string(env), Application: string(application), Alias: alias, Instance: string(instance),
	}}, nil
}

func normalizeAddApplicationValidationUse(op Operation) (Operation, error) {
	if err := onlyFields(op, "placement", "applicationValidationUse"); err != nil {
		return op, err
	}
	if op.ApplicationValidationUse == nil {
		return op, fmt.Errorf("%s にはapplicationValidationUseが必要です", opAddApplicationValidationUse)
	}
	spec := *op.ApplicationValidationUse
	application, err := canonical("applicationValidationUse.application", spec.Application)
	if err != nil {
		return op, err
	}
	validation, err := canonical("applicationValidationUse.validation", spec.Validation)
	if err != nil {
		return op, err
	}
	placement := strings.TrimSpace(op.Placement)
	if placement == "" {
		return op, fmt.Errorf("%s にはplacement（既存のApplication source file）が必要です", opAddApplicationValidationUse)
	}
	return Operation{Op: op.Op, Placement: placement, ApplicationValidationUse: &ApplicationValidationUseSpec{
		Application: string(application), Validation: string(validation),
	}}, nil
}

// operationKeyは、deltaの中でoperationを一意に識別する。同じsubjectを二度追加するdeltaを拒否するために使う。
func operationKey(op Operation) string {
	switch op.Op {
	case opAddValidation:
		return "validation:" + op.ID
	case opAddData:
		return "data:" + op.Data.ID
	case opAddRelation:
		return "relation:" + op.AddRelation.ID
	case opAddEnvironmentBinding:
		b := op.EnvironmentBinding
		return "environment-binding:" + b.Environment + "/" + b.Application + "." + b.Alias
	case opAddApplicationValidationUse:
		u := op.ApplicationValidationUse
		return "application-validation-use:" + u.Application + "/" + u.Validation
	case opAddApplication:
		return "application:" + op.Application.ID
	case opAddReferencePattern:
		return "application-pattern:" + op.Pattern.ID
	}
	return op.Op
}
