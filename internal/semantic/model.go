package semantic

import "tei/internal/core"

// Namespace is a coarse-grained semantic dependency unit. It is independent
// from repository paths and may be consumed by Applications and Databases.
type Namespace struct {
	ID core.CanonicalID `json:"id"`
	// FormatVersion is the optional format version declared by the semantic
	// file itself ("formatVersion N"). Zero means the file did not declare one
	// and inherits the Workspace manifest version. It describes the syntax
	// generation, not the content version of a contract.
	FormatVersion int          `json:"formatVersion,omitempty"`
	Types         []Type       `json:"types,omitempty"`
	Data          []Data       `json:"data,omitempty"`
	Capabilities  []Capability `json:"capabilities,omitempty"`
	Validations   []Validation `json:"validations,omitempty"`
	Periods       []Period     `json:"periods,omitempty"`
	Calendars     []Calendar   `json:"calendars,omitempty"`
	EffectRules   []EffectRule `json:"effectRules,omitempty"`
	RunPlans      []RunPlan    `json:"runPlans,omitempty"`
}

// Calendar declares the business calendar a period follows: when the fiscal
// year starts, which weekday begins the week, and which data holds the
// exceptions (holidays). It declares meaning; it does not compute a date.
type Calendar struct {
	ID                   core.CanonicalID `json:"id"`
	FiscalYearStartMonth int              `json:"fiscalYearStartMonth,omitempty"`
	WeekStarts           string           `json:"weekStarts,omitempty"`
	BusinessDayExcludes  core.CanonicalID `json:"businessDayExcludes,omitempty"`
}

// Period binds one data to a business date and to the states a closing period
// passes through. It declares meaning; it does not execute a close.
type Period struct {
	ID            core.CanonicalID `json:"id"`
	Data          core.CanonicalID `json:"data"`
	BusinessDate  core.CanonicalID `json:"businessDate"`
	States        []string         `json:"states,omitempty"`
	CloseRequires core.CanonicalID `json:"closeRequires,omitempty"`
	ClosedRejects []string         `json:"closedRejects,omitempty"`
	Correction    core.CanonicalID `json:"correction,omitempty"`
	// Calendar names the business calendar that decides the period boundary.
	// An empty value means the period does not declare one, and a checker
	// must not assume a default.
	Calendar core.CanonicalID `json:"calendar,omitempty"`
	// ClosingDay splits every month into two segments, 1..28. Zero means the
	// period is the whole month. X-TEA moved a date after a last-day setting into
	// the next month (getMSeqOfDate, D:Session.java L2471–2484); here the month
	// does not move and the segment says which side of the closing the date is on.
	ClosingDay int `json:"closingDay,omitempty"`
}

type TypeKind string

const (
	TypePrimitive TypeKind = "primitive"
	TypeAlias     TypeKind = "alias"
)

// TypeKinds lists every accepted type kind in report order.
var TypeKinds = []TypeKind{TypePrimitive, TypeAlias}

func (k TypeKind) Valid() bool {
	for _, valid := range TypeKinds {
		if k == valid {
			return true
		}
	}
	return false
}

// Type is a semantic type identity. Primitive types are normally supplied by
// the TEI standard library; workspace definitions generally introduce aliases
// that preserve domain meaning while reusing a lower-level representation.
type Type struct {
	ID         core.CanonicalID `json:"id"`
	Symbol     string           `json:"symbol"`
	Kind       TypeKind         `json:"kind"`
	Underlying core.CanonicalID `json:"underlying,omitempty"`
	Values     []string         `json:"values,omitempty"`
}

type Data struct {
	ID                     core.CanonicalID       `json:"id"`
	Symbol                 string                 `json:"symbol"`
	Description            string                 `json:"description,omitempty"`
	Display                string                 `json:"display,omitempty"`
	Script                 string                 `json:"script,omitempty"`
	Attributes             []Attribute            `json:"attributes,omitempty"`
	Relations              []Relation             `json:"relations,omitempty"`
	CandidateKeys          []CandidateKey         `json:"candidateKeys,omitempty"`
	FunctionalDependencies []FunctionalDependency `json:"functionalDependencies,omitempty"`
}

// CandidateKey is an explicitly authored assertion of a logical business
// identity. It is not a physical PK, a generated row ID, or proof that the
// business assertion holds for all possible records.
type CandidateKey struct {
	ID         core.CanonicalID   `json:"id"`
	Symbol     string             `json:"symbol"`
	Attributes []core.CanonicalID `json:"attributes"`
}

// FunctionalDependency is an explicitly authored Data-scoped determinant
// assertion. It is distinct from relation join keys and physical constraints.
type FunctionalDependency struct {
	ID           core.CanonicalID   `json:"id"`
	Symbol       string             `json:"symbol"`
	Determinants []core.CanonicalID `json:"determinants"`
	Dependents   []core.CanonicalID `json:"dependents"`
}

type Attribute struct {
	ID          core.CanonicalID `json:"id"`
	Symbol      string           `json:"symbol"`
	Description string           `json:"description,omitempty"`
	Display     string           `json:"display,omitempty"`
	TypeRef     core.CanonicalID `json:"type"`
	Required    bool             `json:"required,omitempty"`
	Immutable   bool             `json:"immutable,omitempty"`
	Role        string           `json:"role,omitempty"`
	// Value is the optional declared value written as `value "..."`. A nil
	// pointer means the attribute has no value clause (Absent), a pointer to an
	// empty string means an explicitly empty value (Empty), and any other
	// pointer is a value (Value). This is the format-level tri-state, separate
	// from whether the attribute is required at runtime.
	Value *string `json:"value,omitempty"`
}

type Cardinality string

const (
	CardinalityOne  Cardinality = "one"
	CardinalityMany Cardinality = "many"
)

// Cardinalities lists every accepted cardinality in report order.
var Cardinalities = []Cardinality{CardinalityOne, CardinalityMany}

func (c Cardinality) Valid() bool {
	for _, valid := range Cardinalities {
		if c == valid {
			return true
		}
	}
	return false
}

// RelationKeyは、Relationを実データのinstance間で辿るために、
// source側とtarget側のどのSemantic attributeを対応付けるかを表す。
// 物理columnやDatabase Foreign Keyではなく、Semantic attributeのidentityだけを保持する。
type RelationKey struct {
	SourceAttribute core.CanonicalID `json:"sourceAttribute"`
	TargetAttribute core.CanonicalID `json:"targetAttribute"`
}

type Relation struct {
	ID          core.CanonicalID `json:"id"`
	Symbol      string           `json:"symbol"`
	Target      core.CanonicalID `json:"target"`
	Cardinality Cardinality      `json:"cardinality"`
	// Keysは複合keyを含むRelationも表現できるよう、対応関係を順序付きで保持する。
	// key対応をまだ定義していない概念Relationでは空のままとし、既存Canonical表現を変えない。
	Keys []RelationKey `json:"keys,omitempty"`
	// OnDeleteは、参照先の削除に対する振る舞いの宣言である。
	// restrict・cascade・preserveを持ち、実行時の強制はProviderに置く。
	OnDelete string `json:"onDelete,omitempty"`
}

// ValidationKindは、業務上の不変条件として実Referenceで確認できた
// 最小のrule種別だけを表す。汎用式言語やrule engineを先回りして導入しない。
type ValidationKind string

const (
	ValidationPositive       ValidationKind = "positive"
	ValidationRelationUnique ValidationKind = "relation-unique"
)

func (k ValidationKind) Valid() bool {
	return k == ValidationPositive || k == ValidationRelationUnique
}

// Validationは、Semantic上の業務ruleを機械可読な形で保持する。
// positiveはAttributeだけを、relation-uniqueはRelationとそのtarget側Attributeを参照する。
// 評価結果のsatisfied / not-satisfiedと、評価自体を完了できなかったerrorは実行境界で分離する。
type Validation struct {
	ID        core.CanonicalID `json:"id"`
	Symbol    string           `json:"symbol"`
	Kind      ValidationKind   `json:"kind"`
	Attribute core.CanonicalID `json:"attribute"`
	Relation  core.CanonicalID `json:"relation,omitempty"`
}

type Parameter struct {
	Name core.LocalName   `json:"name"`
	Type core.CanonicalID `json:"type"`
}

// Outcomeは、Capabilityを正常に実行した結果として得られる業務上の結末を表す。
// 実行そのものを完了できなかった状態はOutcomeへ変換せず、Invocation境界のerrorとして扱う。
type Outcome struct {
	Name    core.LocalName   `json:"name"`
	Payload core.CanonicalID `json:"payload"`
}

// DataEffectKind describes the business intent of one Capability. It does
// not authorize execution or assert that a Provider has changed durable data.
type DataEffectKind string

const (
	DataEffectCreate DataEffectKind = "create"
	DataEffectUpdate DataEffectKind = "update"
	DataEffectDelete DataEffectKind = "delete"
)

func (k DataEffectKind) Valid() bool {
	return k == DataEffectCreate || k == DataEffectUpdate || k == DataEffectDelete
}

type DataEffect struct {
	Kind DataEffectKind   `json:"kind"`
	Data core.CanonicalID `json:"data"`
}

type Capability struct {
	ID       core.CanonicalID `json:"id"`
	Symbol   string           `json:"symbol"`
	Inputs   []Parameter      `json:"inputs,omitempty"`
	Output   core.CanonicalID `json:"output,omitempty"`
	Outcomes []Outcome        `json:"outcomes,omitempty"`
	Effect   *DataEffect      `json:"effect,omitempty"`
	// EmptyPolicy is the Owner-declared handling of an empty target set for
	// this business action. Empty means "not declared; use the business-type
	// default" (see EffectiveEmptyPolicy).
	EmptyPolicy EmptyPolicy `json:"emptyPolicy,omitempty"`
}

// EffectRule declares a side effect that runs around a capability, and where it
// runs. It does not change what the capability returns.
type EffectRule struct {
	ID            core.CanonicalID `json:"id"`
	OnCapability  core.CanonicalID `json:"onCapability"`
	Stage         string           `json:"stage"`
	EffectKind    DataEffectKind   `json:"effectKind"`
	EffectData    core.CanonicalID `json:"effectData"`
	ViaCapability core.CanonicalID `json:"viaCapability,omitempty"`
	OnFailure     string           `json:"onFailure,omitempty"`
}

// RunPlan declares an ordered execution inside one application: which
// capabilities run, what identifies a run, and whether it may run unattended.
// The calendar, the scheduler, and credentials are deliberately absent.
type RunPlan struct {
	ID          core.CanonicalID   `json:"id"`
	Operations  []core.CanonicalID `json:"operations"`
	Key         []core.CanonicalID `json:"key,omitempty"`
	Concurrency string             `json:"concurrency,omitempty"`
	Resume      string             `json:"resume,omitempty"`
	Outputs     []core.CanonicalID `json:"outputs,omitempty"`
	Autonomy    string             `json:"autonomy,omitempty"`
}
