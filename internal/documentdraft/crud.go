package documentdraft

import (
	"sort"
	"strconv"
	"strings"

	"tei/internal/operations"
	"tei/internal/sourceindex"
)

// DocumentKindCRUDTable projects the standard CRUD operations an application can
// perform on its mapped data. The derivation is the existing read-only
// operation surface (internal/operations), so the table comes from the same
// accepted meaning the other read commands answer from — not from a second model
// kept in the document layer.
const DocumentKindCRUDTable = "crud-table"

// SupportedDocumentKinds lists the document kinds this version implements. A
// kind not listed here is refused with UNSUPPORTED_DOCUMENT_KIND: a later stage
// is never half-implemented and silently treated as this one.
func SupportedDocumentKinds() []string {
	return []string{
		DocumentKindElementExplanation,
		DocumentKindCRUDTable,
	}
}

// projectCRUDTable fills one draft with the CRUD matrix of one application.
//
// The rows are the derived standard operations. Nothing is completed: an
// operation the accepted meaning does not derive is absent, and a gap the
// surface reports stays a gap in the document rather than becoming an assumed
// capability. CRUD here means "operations derived from the mapped data and the
// declared capability effects"; it does not assert that the mapping is the
// business truth, only that this is what the accepted model declares.
func (d *Draft) projectCRUDTable(read workspaceRead, application string) error {
	app, err := read.applicationRef(application)
	if err != nil {
		return err
	}
	d.Subjects = []SubjectRef{app.ref}
	d.Scope = &DraftScope{Application: application, CRUDRule: operations.OperationSurfaceSchemaVersion}

	surface := operations.DeriveOperationSurface(read.loaded.World, application)
	// The surface is derived from the same load, so the fingerprints are the
	// ones the draft records. Taking them from a second read could pair a
	// current World with an older Source Index.
	surface.WorldFingerprint = app.inspection.WorldFingerprint
	surface.ModelRevision = read.modelRevision

	digests := inputDigests(app.inspection, read)
	numbering := &occurrenceNumbering{}

	d.Occurrences = append(d.Occurrences, numbering.add(OccurrenceExplanation,
		[]sourceindex.Subject{app.subject},
		conditions(digests, "application-crud-overview-v0"),
		OccurrenceContent{Text: crudOverview(application, surface)}))

	table := &TableContent{
		Caption: application + " のCRUD操作",
		Header:  []string{"操作", "対象データ", "由来", "自律", "根拠", "実装"},
		Rows:    [][]string{},
	}
	for _, op := range surface.Operations {
		table.Rows = append(table.Rows, []string{
			string(op.Kind),
			string(op.Target),
			op.Origin,
			op.Autonomy,
			op.Basis,
			op.Binding,
		})
		// Each row carries the subjects it was derived from, so a reader can go
		// back to the data element and the application that produced it.
		table.RowSubjects = append(table.RowSubjects, []sourceindex.Subject{
			{Kind: "semantic-data", ID: string(op.Target)},
			app.subject,
		})
	}
	d.Occurrences = append(d.Occurrences, numbering.add(OccurrenceTable,
		[]sourceindex.Subject{app.subject},
		conditions(digests, "crud-matrix-v0"),
		OccurrenceContent{Table: table}))

	// The data every row targets is a referenced subject too, so a reader of the
	// draft sees which elements the document talks about without re-deriving it.
	for _, target := range tableTargets(surface) {
		subject := sourceindex.Subject{Kind: "semantic-data", ID: target}
		inspection, err := read.inspect(subject)
		if err != nil {
			// The surface only lists a target the mapping resolved, so this is
			// an unexpected read: keep the row, and do not invent a subject.
			continue
		}
		d.Subjects = append(d.Subjects, subjectRefOf(inspection, read, subject))
	}

	// Gaps and conflicts are reported as their own occurrence rather than being
	// folded into the matrix: "this operation is not derivable" is a fact about
	// the model, not a row of the table.
	if len(surface.Gaps) > 0 || len(surface.Conflicts) > 0 {
		d.Occurrences = append(d.Occurrences, numbering.add(OccurrenceTable,
			[]sourceindex.Subject{app.subject},
			conditions(digests, "crud-gaps-v0"),
			OccurrenceContent{Table: crudGapTable(application, surface)}))
	}

	for _, artifact := range app.inspection.Sources.Artifacts {
		d.Occurrences = append(d.Occurrences, numbering.add(OccurrenceCitation,
			[]sourceindex.Subject{app.subject},
			conditions(digests, "source-citation-v0"),
			OccurrenceContent{Citation: citationOf(app.inspection, artifact)}))
	}
	return nil
}

// applicationRef resolves one application subject from the same load and
// refuses an application the Workspace does not declare. The operation surface
// returns an empty result for an unknown filter, which a caller would read as
// "this application does nothing"; the draft refuses that instead.
func (read workspaceRead) applicationRef(application string) (resolvedApplication, error) {
	if strings.TrimSpace(application) == "" || application != strings.TrimSpace(application) {
		return resolvedApplication{}, draftError(CodeInvalidArgument,
			"crud-table requires an exact -application ID")
	}
	found := false
	for _, app := range read.loaded.World.Applications {
		if string(app.ID) == application {
			found = true
			break
		}
	}
	if !found {
		return resolvedApplication{}, draftError(CodeApplicationNotFound,
			"application %q is not declared in this workspace", application)
	}
	subject := sourceindex.Subject{Kind: "application", ID: application}
	inspection, err := read.inspect(subject)
	if err != nil {
		return resolvedApplication{}, err
	}
	return resolvedApplication{
		subject:    subject,
		inspection: inspection,
		ref:        subjectRefOf(inspection, read, subject),
	}, nil
}

type resolvedApplication struct {
	subject    sourceindex.Subject
	inspection operations.SubjectInspection
	ref        SubjectRef
}

// tableTargets returns the distinct data targets of a surface, sorted, so the
// draft's subject list is deterministic.
func tableTargets(surface operations.OperationSurface) []string {
	seen := map[string]bool{}
	var out []string
	for _, op := range surface.Operations {
		if op.Target == "" || seen[op.Target] {
			continue
		}
		seen[op.Target] = true
		out = append(out, op.Target)
	}
	sort.Strings(out)
	return out
}

// crudOverview generates the prose summary of one CRUD surface.
func crudOverview(application string, surface operations.OperationSurface) string {
	lines := []string{
		"アプリケーション " + application + " について、受理済みモデルから導出した標準操作の一覧です。",
		"操作は " + itoa(len(surface.Operations)) + " 件、うち導出 " +
			itoa(surface.Summary.Derived) + " 件・明示 " + itoa(surface.Summary.Declared) +
			" 件、無人実行可 " + itoa(surface.Summary.Unattended) + " 件です。",
		"導出規則は " + surface.Rule.ID + "(" + surface.Rule.Version + ") です。",
	}
	if surface.Summary.Gaps > 0 {
		lines = append(lines, "導出できなかった操作が "+itoa(surface.Summary.Gaps)+
			" 件あります。表ではなく別の一覧に理由を記しています。")
	}
	if len(surface.Conflicts) > 0 {
		lines = append(lines, "宣言と導出が食い違う箇所が "+itoa(len(surface.Conflicts))+" 件あります。")
	}
	lines = append(lines,
		"この表は受理済みモデルからの投影であり、CRUDの業務的な意味・網羅性・実行権限を保証しません。")
	return strings.Join(lines, "\n")
}

// crudGapTable lists operations that could not be derived and declarations that
// contradict the derivation. It does not guess the missing operation.
func crudGapTable(application string, surface operations.OperationSurface) *TableContent {
	table := &TableContent{
		Caption: application + " の未導出・矛盾",
		Header:  []string{"種別", "対象", "操作", "理由"},
		Rows:    [][]string{},
	}
	for _, gap := range surface.Gaps {
		table.Rows = append(table.Rows, []string{"gap", gap.Target, gap.Kind, gap.Reason + ": " + gap.Detail})
	}
	for _, conflict := range surface.Conflicts {
		table.Rows = append(table.Rows, []string{"conflict", conflict.Target, conflict.Kind, conflict.Detail})
	}
	return table
}

func itoa(value int) string {
	return strconv.Itoa(value)
}
