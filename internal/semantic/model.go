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
	// Operators are the operator contracts this namespace declares. A
	// declaration is data: it names what a rule may write after `is` and what
	// the operator reads. It is not an implementation.
	Operators   []Operator   `json:"operators,omitempty"`
	Periods     []Period     `json:"periods,omitempty"`
	Calendars   []Calendar   `json:"calendars,omitempty"`
	EffectRules []EffectRule `json:"effectRules,omitempty"`
	RunPlans    []RunPlan    `json:"runPlans,omitempty"`
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
	// JoinTableは多対多Relationでの中間表を指定する。
	// 空の場合は直接Relation、値がある場合はJoinTableを経由する。
	JoinTable string `json:"joinTable,omitempty"`
}

// ValidationKindは、業務上の不変条件として実Referenceで確認できた
// 最小のrule種別だけを表す。汎用式言語やrule engineを先回りして導入しない。
type ValidationKind string

const (
	ValidationPositive       ValidationKind = "positive"
	ValidationRelationUnique ValidationKind = "relation-unique"
	// ValidationOperator states that the rule is decided by a declared
	// Operator. The kind never changes when an operator is added: the
	// operator's identity is data in Validation.Operator.
	ValidationOperator ValidationKind = "operator"
	// ValidationNotAfter states that one calendar date of a record is not
	// after another calendar date of the same record. The second date is
	// Validation.Compared.
	ValidationNotAfter ValidationKind = "not-after"
	// ValidationNoOverlap states that the periods of the rows of a relation
	// do not overlap. A period runs from the date of Attribute to the date
	// of Compared, both days included. With GroupBy, only rows that share
	// that attribute's value are compared with each other.
	ValidationNoOverlap ValidationKind = "no-overlap"
	// ValidationRequiredIf states that Attribute holds a value whenever the
	// attribute of Compared holds one. It constrains presence, not the value:
	// a record whose trigger attribute has no value satisfies the rule
	// whatever Attribute holds, because the rule speaks only about records
	// that have the trigger. Whether Attribute is required in every record is
	// the attribute definition's business, not this rule's.
	ValidationRequiredIf ValidationKind = "required-if"
)

// ValidationKinds lists every kind once. Valid() and the diagnostics that quote
// the vocabulary read this list, so a kind added here is accepted and named
// without a second place to update.
var ValidationKinds = []ValidationKind{
	ValidationPositive,
	ValidationRelationUnique,
	ValidationOperator,
	ValidationNotAfter,
	ValidationNoOverlap,
	ValidationRequiredIf,
}

func (k ValidationKind) Valid() bool {
	for _, valid := range ValidationKinds {
		if k == valid {
			return true
		}
	}
	return false
}

// OperatorInput states what an operator reads. It fixes the read demand of
// every rule that uses the operator, independent of the implementation.
type OperatorInput string

const (
	// OperatorInputNumber is one numeric value of the constrained attribute.
	OperatorInputNumber OperatorInput = "number"
	// OperatorInputPresence is whether an attribute holds a value, not the
	// value itself.
	OperatorInputPresence OperatorInput = "presence"
)

// Valid reports whether this is a known operator input kind. Only the kinds
// something in this program actually reads are valid: a word accepted at
// authoring time that no reader handles would build a contract nothing can
// decide. date, text, truth, and any are stated by the design but have no
// reader here yet, so they stay out of this list until one exists.
func (i OperatorInput) Valid() bool {
	switch i {
	case OperatorInputNumber, OperatorInputPresence:
		return true
	}
	return false
}

// OperatorInputDeclaration is one input of an operator signature: its name,
// type, whether it reads every row, and how an absent value is read.
type OperatorInputDeclaration struct {
	// Name is how a rule assigns a value to this input.
	Name string `json:"name"`
	// Type is the kind of value this input reads.
	Type OperatorInput `json:"type"`
	// OfEachRow is true when the input reads every row of the relation.
	OfEachRow bool `json:"ofEachRow,omitempty"`
	// AbsentAs is the name an absent value is read as. Empty means no mapping.
	AbsentAs string `json:"absentAs,omitempty"`
}

// Operator is a declared operator contract. A rule refers to it by ID. The
// result is a truth value: the rule is satisfied or not satisfied. The
// contract carries no implementation and grants no execution authority.
type Operator struct {
	ID     core.CanonicalID `json:"id"`
	Symbol string           `json:"symbol"`
	// Input is the first input's type, kept for backward compatibility with
	// single-input operators. The full signature is in Inputs.
	Input OperatorInput `json:"input"`
	// InputName is the name the author gave the first input, as in `takes low: number`.
	// The single-input form `takes number` names it "value".
	InputName string `json:"inputName,omitempty"`
	// AbsentAs is the name the first input's absent value is read as.
	AbsentAs string `json:"absentAs,omitempty"`
	// Inputs is the full list of inputs the operator declares, in signature
	// order. The first input is also represented by Input/InputName/AbsentAs
	// for backward compatibility.
	Inputs []OperatorInputDeclaration `json:"inputs,omitempty"`
	// RefutesOnSomeRows states that a witnessed counterexample among the rows
	// read is enough to say the rule does not hold, written
	// `refutes on some rows`. Without it, a relation that was only partly read
	// cannot be answered not-satisfied.
	RefutesOnSomeRows bool `json:"refutesOnSomeRows,omitempty"`
	// Cases are the examples the contract states.
	Cases []OperatorCase `json:"cases,omitempty"`
}

// ReadsEveryRow reports whether this operator declares an input that reads
// every row of a relation, written `of each row`. A rule that uses such an
// operator is asked once for the whole relation instead of once per row, so a
// reader that can only answer for one row must refuse it rather than answer it
// row by row.
func (o Operator) ReadsEveryRow() bool {
	for _, input := range o.Inputs {
		if input.OfEachRow {
			return true
		}
	}
	return false
}

// ValidationInputAssignment is one resolved input assignment of a rule: the
// input name, the value assigned (a canonical attribute id or a constant), and
// the absent-value mapping if the rule stated one.
type ValidationInputAssignment struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	AbsentAs string `json:"absentAs,omitempty"`
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
	// Operator is set only for ValidationOperator and names a declared Operator.
	Operator core.CanonicalID `json:"operator,omitempty"`
	// Compared names a second attribute. For ValidationNotAfter it is the
	// attribute Attribute is compared with. For ValidationNoOverlap it is
	// the last day of the period that starts at Attribute. For
	// ValidationRequiredIf it is the attribute whose value makes Attribute
	// required.
	Compared core.CanonicalID `json:"compared,omitempty"`
	// IfPresent is set only for ValidationNotAfter. It states that a record
	// that holds no value for Compared satisfies the rule: the rule speaks
	// only about records that have that value. Without it, such a record is
	// not decided.
	IfPresent bool `json:"ifPresent,omitempty"`
	// GroupBy is set only for ValidationNoOverlap. Rows are compared only
	// with rows that hold the same value for it. When empty, every row of
	// the relation is compared with every other.
	GroupBy core.CanonicalID `json:"groupBy,omitempty"`
	// OpenEnded is set only for ValidationNoOverlap. It states that a row
	// without a Compared date is a period with no last day. Without it, such
	// a row is not decided.
	OpenEnded bool `json:"openEnded,omitempty"`
	// Inputs is set only for ValidationOperator. It carries the input
	// assignments the rule stated, in the operator's signature order.
	Inputs []ValidationInputAssignment `json:"inputs,omitempty"`
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
