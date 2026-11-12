package documentdraft

import (
	"encoding/json"
	"strings"

	"tei/internal/operations"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
)

// DocumentKindSpecification bundles several elements and their relations into
// one chapter-structured document. It is the only stage that reads more than one
// element, and it keeps the same discipline as the single-element stage: every
// chapter is one element the accepted model declares, and a chapter is never
// written for a subject the model does not declare.
//
// The chapter list is an explicit, ordered input. The caller's order is the
// document's order, so the same request always produces the same document.
const DocumentKindSpecification = "specification"

// SectionSpec is one element or relation in a specification document. The order
// of the slice is the order of the chapters: it is an explicit input, so the same
// request always produces the same document and a reader can rely on the chapter
// order they asked for.
type SectionSpec struct {
	// Subject is the element this chapter is about.
	Subject sourceindex.Subject
	// RelationID optionally names one declared relation of that element, which
	// becomes a section inside the chapter. It is skipped with a reason when the
	// element does not declare it, exactly as the single-element stage refuses it.
	RelationID string
}

// projectSpecification fills one draft with one chapter per requested subject.
//
// The order is deterministic: the caller's order is the chapter order. Nothing is
// completed. A subject the model does not declare is recorded in
// Scope.SkippedSubjects with its reason and does not produce a chapter, so a
// partial request never silently yields a complete-looking document.
func (d *Draft) projectSpecification(read workspaceRead, sections []SectionSpec) error {
	if len(sections) == 0 {
		return draftError(CodeInvalidArgument,
			"document kind %s requires at least one section", DocumentKindSpecification)
	}
	first, err := read.inspect(sections[0].Subject)
	if err != nil {
		return err
	}
	digests := inputDigests(first, read)

	// Phase 1 resolves every requested section, so the chapter count in the
	// document counts the chapters that exist. Counting the requested sections
	// instead would write "全 3 章" into a document that holds two, and a reader
	// would trust a number the model does not support.
	resolved, err := resolveSpecificationSections(read, sections)
	if err != nil {
		return err
	}
	d.Scope.SkippedSubjects = resolved.skipped
	if len(resolved.chapters) == 0 {
		return draftError(CodeSubjectNotFound,
			"no requested subject is declared in this workspace")
	}

	// Phase 2 emits one chapter per resolved section, in the requested order.
	numbering := &occurrenceNumbering{}
	d.Subjects = []SubjectRef{}
	for index, chapter := range resolved.chapters {
		d.Subjects = append(d.Subjects, chapter.ref)
		d.Occurrences = append(d.Occurrences, numbering.add(OccurrenceExplanation,
			[]sourceindex.Subject{chapter.subject},
			conditions(digests, "specification-chapter-v0"),
			OccurrenceContent{Text: specificationChapter(
				chapter.subject, index+1, len(resolved.chapters), chapter.inspection.Canonical)}))
		if chapter.relation != nil {
			d.Relations = append(d.Relations, *chapter.relation)
			d.Subjects = append(d.Subjects, chapter.relationTarget)
			d.Occurrences = append(d.Occurrences, numbering.add(OccurrenceRelation,
				[]sourceindex.Subject{chapter.subject, chapter.relation.To},
				conditions(digests, "relation-summary-v0"),
				OccurrenceContent{Relation: chapter.relation}))
		}
		for _, artifact := range chapter.inspection.Sources.Artifacts {
			d.Occurrences = append(d.Occurrences, numbering.add(OccurrenceCitation,
				[]sourceindex.Subject{chapter.subject},
				conditions(digests, "source-citation-v0"),
				OccurrenceContent{Citation: citationOf(chapter.inspection, artifact)}))
		}
	}
	return nil
}

// specChapter is one resolved chapter: the element, its source evidence and,
// when the element declares the requested relation, the relation and its target.
type specChapter struct {
	subject        sourceindex.Subject
	inspection     operations.SubjectInspection
	ref            SubjectRef
	relation       *RelationRef
	relationTarget SubjectRef
}

// resolvedSpecification is the outcome of resolving every requested section.
type resolvedSpecification struct {
	chapters []specChapter
	skipped  []SkippedSubject
}

// resolveSpecificationSections reads every requested section from the same load.
// A subject the model does not declare is recorded in skipped and produces no
// chapter; a relation the element does not declare is recorded too, and the
// chapter stays without its relation section.
func resolveSpecificationSections(
	read workspaceRead, sections []SectionSpec,
) (resolvedSpecification, error) {
	var out resolvedSpecification
	for _, section := range orderedSections(sections) {
		inspection, err := read.inspect(section.Subject)
		if err != nil {
			out.skipped = append(out.skipped, SkippedSubject{
				Subject: section.Subject,
				Reason:  "subject-not-declared",
			})
			continue
		}
		chapter := specChapter{
			subject:    section.Subject,
			inspection: inspection,
			ref:        subjectRefOf(inspection, read, section.Subject),
		}
		if section.RelationID != "" {
			var data semantic.Data
			if err := json.Unmarshal(inspection.Canonical, &data); err != nil {
				return resolvedSpecification{}, draftError(CodeInvalidArgument,
					"canonical subject %s:%s is not a readable data declaration: %v",
					section.Subject.Kind, section.Subject.ID, err)
			}
			relation, target, err := resolveRelation(read, data, section.Subject, section.RelationID)
			if err != nil {
				// A relation the element does not declare is a skipped section,
				// not a silent omission: the chapter stays, the section does not.
				out.skipped = append(out.skipped, SkippedSubject{
					Subject: section.Subject,
					Reason:  "relation-not-declared:" + section.RelationID,
				})
				out.chapters = append(out.chapters, chapter)
				continue
			}
			chapter.relation = &relation
			chapter.relationTarget = target
		}
		out.chapters = append(out.chapters, chapter)
	}
	return out, nil
}

// orderedSections returns the sections in chapter order. The caller's order is
// the document's order: it is an explicit, ordered input, and the same input
// always yields the same document because the order is part of the input rather
// than something this function decides.
func orderedSections(sections []SectionSpec) []SectionSpec {
	return append([]SectionSpec(nil), sections...)
}

// specificationChapter generates the prose of one chapter. It names the position
// among the chapters the document actually holds, what the chapter covers and the
// declared shape of that subject, so a change to the element shows up in the
// chapter rather than only in its citations. It never asserts the chapter is
// complete: the model may declare more than one chapter's worth of meaning.
func specificationChapter(
	subject sourceindex.Subject, position, total int, canonical json.RawMessage,
) string {
	lines := []string{
		"第 " + itoa(position) + " 章（全 " + itoa(total) + " 章）: " + string(subject.ID) + "。",
		"対象は " + subject.Kind + ":" + subject.ID + " です。",
	}
	if symbol := symbolOf(canonical); symbol != "" {
		lines = append(lines, "宣言された記号は "+symbol+" です。")
	}
	if subject.Kind == "semantic-data" {
		var data semantic.Data
		if err := json.Unmarshal(canonical, &data); err == nil {
			lines = append(lines, "属性は "+itoa(len(data.Attributes))+
				" 件、関係は "+itoa(len(data.Relations))+" 件宣言されています。")
		}
	}
	lines = append(lines,
		"この文書は受理済みモデルからの投影であり、章の立て方・粒度・網羅性は保証しません。")
	return strings.Join(lines, "\n")
}
