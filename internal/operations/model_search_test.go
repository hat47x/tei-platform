package operations

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
)

// searchSource is one small Workspace Source. The line layout is deliberate:
// line 1 is a comment no declaration owns, line 6 is a relation key that
// carries no identifier of its own, and line 7 is the closing brace.
const searchSource = `# authored by the reference set
namespace sales

data Order {
    attribute orderId type sales.order_id
    relation customer to customer.customer one
        key sales.order.customer_id -> customer.customer.customer_id
}
`

// searchAnchor returns the range of the first occurrence of want, the way the
// Source Index records a declaration identifier.
func searchAnchor(t *testing.T, content, want string) sourceindex.Range {
	t.Helper()
	start := strings.Index(content, want)
	if start < 0 {
		t.Fatalf("%q is not in the fixture source", want)
	}
	rng, err := sourceindex.RangeFromOffsets([]byte(content), start, start+len(want))
	if err != nil {
		t.Fatalf("range for %q: %v", want, err)
	}
	return rng
}

func searchFixture(t *testing.T) (ModelCatalog, sourceindex.Index, map[string][]byte) {
	t.Helper()
	content := []byte(searchSource)
	path := "semantic/sales/order.tei"
	ref := func(kind, id, anchor string) sourceindex.Reference {
		return sourceindex.Reference{
			Subject:      sourceindex.Subject{Kind: kind, ID: id},
			ArtifactPath: path,
			Range:        searchAnchor(t, searchSource, anchor),
			Role:         sourceindex.RoleDefinition,
		}
	}
	index := sourceindex.Index{
		Artifacts: []sourceindex.Artifact{{Path: path, Fingerprint: sourceindex.Fingerprint(content)}},
		References: []sourceindex.Reference{
			ref("semantic", "sales", "sales"),
			ref("semantic-data", "sales.order", "Order"),
			ref("semantic-attribute", "sales.order.order_id", "orderId"),
			ref("semantic-relation", "sales.order.customer", "customer"),
		},
	}
	index.Normalize()
	// The World declares exactly what the fixture source writes, because the
	// searched subjects come from the catalog traversal over this World.
	world := ir.World{Semantics: []semantic.Namespace{{
		ID: "sales",
		Data: []semantic.Data{{
			ID:     "sales.order",
			Symbol: "Order",
			Attributes: []semantic.Attribute{
				{ID: "sales.order.order_id", Symbol: "orderId", TypeRef: "sales.order_id"},
			},
			Relations: []semantic.Relation{
				{ID: "sales.order.customer", Symbol: "customer", Target: "customer.customer"},
			},
		}},
	}}}
	catalog := CatalogWorld(world, index, "", "world-fp", "source-fp")
	return catalog, index, map[string][]byte{path: content}
}

// searchPagingSource is a Workspace Source whose first subject matches on two
// lines and two later subjects match on one each, so a two-hit page names one
// subject while the whole answer names three. The comment line is owned by no
// declaration, like searchSource's.
const searchPagingSource = `# authored by the reference set
namespace sales

data Order {
    attribute orderId type sales.order_id
    relation customer to customer.customer one
        key sales.order.customer_id -> customer.customer.customer_id
}

data Customer {
    attribute customerId type sales.customer_id
}
`

func searchPagingFixture(t *testing.T) (ModelCatalog, sourceindex.Index, map[string][]byte) {
	t.Helper()
	content := []byte(searchPagingSource)
	path := "semantic/sales/customer.tei"
	ref := func(kind, id, anchor string) sourceindex.Reference {
		return sourceindex.Reference{
			Subject:      sourceindex.Subject{Kind: kind, ID: id},
			ArtifactPath: path,
			Range:        searchAnchor(t, searchPagingSource, anchor),
			Role:         sourceindex.RoleDefinition,
		}
	}
	index := sourceindex.Index{
		Artifacts: []sourceindex.Artifact{{Path: path, Fingerprint: sourceindex.Fingerprint(content)}},
		References: []sourceindex.Reference{
			ref("semantic", "sales", "sales"),
			ref("semantic-data", "sales.order", "Order"),
			ref("semantic-attribute", "sales.order.order_id", "orderId"),
			ref("semantic-relation", "sales.order.customer", "customer"),
			ref("semantic-data", "sales.customer", "Customer"),
			ref("semantic-attribute", "sales.customer.customer_id", "customerId"),
		},
	}
	index.Normalize()
	world := ir.World{Semantics: []semantic.Namespace{{
		ID: "sales",
		Data: []semantic.Data{
			{
				ID:     "sales.order",
				Symbol: "Order",
				Attributes: []semantic.Attribute{
					{ID: "sales.order.order_id", Symbol: "orderId", TypeRef: "sales.order_id"},
				},
				Relations: []semantic.Relation{
					{ID: "sales.order.customer", Symbol: "customer", Target: "customer.customer"},
				},
			},
			{
				ID:     "sales.customer",
				Symbol: "Customer",
				Attributes: []semantic.Attribute{
					{ID: "sales.customer.customer_id", Symbol: "customerId", TypeRef: "sales.customer_id"},
				},
			},
		},
	}}}
	return CatalogWorld(world, index, "", "world-fp", "source-fp"), index,
		map[string][]byte{path: content}
}

func searchIn(t *testing.T, query SearchQuery, limit, offset int) ModelSearch {
	t.Helper()
	catalog, index, sources := searchFixture(t)
	got, failure := SearchInSources(catalog, index, sources, query, limit, offset, defaultSearchBounds())
	if failure != nil {
		t.Fatalf("search %+v: %s %s", query, failure.Code, failure.Message)
	}
	return got
}

// defaultSearchBounds is the work bound SearchWorkspace applies, so the
// fixture-sized searches are never stopped early.
func defaultSearchBounds() SearchBounds { return SearchBounds{} }

// distinctSearchSubjects counts the subjects a hit list names, the way the
// projection's kindCounts must: one subject is one count however many lines of
// its declaration matched.
func distinctSearchSubjects(hits []ModelSearchHit) int {
	seen := map[sourceindex.Subject]bool{}
	for _, hit := range hits {
		seen[hit.Subject] = true
	}
	return len(seen)
}

func searchFailure(t *testing.T, query SearchQuery, limit, offset int) *CatalogError {
	t.Helper()
	catalog, index, sources := searchFixture(t)
	_, failure := SearchInSources(catalog, index, sources, query, limit, offset, defaultSearchBounds())
	if failure == nil {
		t.Fatalf("search %+v succeeded, want a typed failure", query)
	}
	return failure
}

// The answer is a projection of the same load as the catalog, so it reports the
// catalog's model version and a caller can tie a hit to one Workspace state.
func TestModelSearchReportsTheCatalogModelVersion(t *testing.T) {
	got := searchIn(t, SearchQuery{Term: "orderId"}, 50, 0)
	if got.SchemaVersion != "tei.operations.model-search/v0" || got.Effect != "read-only" {
		t.Fatalf("header: %+v", got)
	}
	if got.WorldFingerprint != "world-fp" || got.SourceIndexFingerprint != "source-fp" {
		t.Fatalf("fingerprints: %+v", got)
	}
	if got.SnapshotFingerprint != InspectionSnapshotFingerprint("world-fp", "source-fp") {
		t.Fatalf("snapshotFingerprint=%q", got.SnapshotFingerprint)
	}
	if got.ModelRevision == "" {
		t.Fatal("modelRevision is empty")
	}
	if got.Query.Term != "orderId" || got.Query.Kind != "" {
		t.Fatalf("echoed query: %+v", got.Query)
	}
	if got.Limit != 50 || got.Offset != 0 || got.Total != 1 || len(got.Hits) != 1 || got.Omitted != 0 {
		t.Fatalf("paging header: %+v", got)
	}
	if got.Truncated {
		t.Fatal("truncated is true although the work bound was never reached")
	}
	if got.NextOffset != nil {
		t.Fatalf("nextOffset=%d on the last page", *got.NextOffset)
	}
}

// A hit names the subject, the Source, the line and the line as written: the
// evidence that lets a caller open the match instead of trusting it.
func TestModelSearchHitCarriesSubjectPathLineAndText(t *testing.T) {
	got := searchIn(t, SearchQuery{Term: "orderId"}, 50, 0)
	want := ModelSearchHit{
		Subject:      sourceindex.Subject{Kind: "semantic-attribute", ID: "sales.order.order_id"},
		ArtifactPath: "semantic/sales/order.tei",
		Line:         5,
		Text:         "    attribute orderId type sales.order_id",
	}
	if got.Hits[0] != want {
		t.Fatalf("hit=%+v want %+v", got.Hits[0], want)
	}
}

// Matching is case-insensitive and literal: "ORDERID" finds the declaration
// "attribute orderId ..." and no pattern syntax is interpreted.
func TestModelSearchMatchesLiterallyAndCaseInsensitively(t *testing.T) {
	got := searchIn(t, SearchQuery{Term: "ORDERID"}, 50, 0)
	if got.Total != 1 || got.Hits[0].Subject.Kind != "semantic-attribute" {
		t.Fatalf("hits=%+v", got.Hits)
	}
	if got.Hits[0].Line != 5 || got.Hits[0].Text != "    attribute orderId type sales.order_id" {
		t.Fatalf("hit=%+v", got.Hits[0])
	}
	// A line with no identifier of its own belongs to the nearest preceding
	// declaration, so the relation key is found under the relation, not lost.
	key := searchIn(t, SearchQuery{Term: "customer_id"}, 50, 0)
	if key.Total != 1 || key.Hits[0].Subject.Kind != "semantic-relation" ||
		key.Hits[0].Subject.ID != "sales.order.customer" || key.Hits[0].Line != 7 {
		t.Fatalf("relation key hit=%+v", key.Hits)
	}
}

// A line before the first declaration of a Source belongs to no subject, so it
// is not searched: the fixture's comment line carries "sales" and returns
// nothing, while the namespace declaration on the next line does.
func TestModelSearchSkipsLinesBeforeTheFirstDeclaration(t *testing.T) {
	got := searchIn(t, SearchQuery{Term: "authored by"}, 50, 0)
	if got.Total != 0 || len(got.Hits) != 0 {
		t.Fatalf("comment line was searched: %+v", got.Hits)
	}
	namespace := searchIn(t, SearchQuery{Term: "namespace"}, 50, 0)
	if namespace.Total != 1 || namespace.Hits[0].Subject.Kind != "semantic" {
		t.Fatalf("namespace hit=%+v", namespace.Hits)
	}
}

// The kind filter restricts the search to one subject kind, and a kind outside
// the closed set is an argument failure rather than an empty answer.
func TestModelSearchKindFilterAndUnknownKind(t *testing.T) {
	got := searchIn(t, SearchQuery{Kind: "semantic-relation", Term: "customer_id"}, 50, 0)
	if got.Total != 1 || got.Hits[0].Subject.Kind != "semantic-relation" {
		t.Fatalf("kind filter: %+v", got.Hits)
	}
	// The same line under a data filter: the line is attributed to the nearest
	// preceding declaration among the searched kinds, so a data search sees the
	// mention inside the data element's own block.
	data := searchIn(t, SearchQuery{Kind: "semantic-data", Term: "customer_id"}, 50, 0)
	if data.Total != 1 || data.Hits[0].Subject.Kind != "semantic-data" ||
		data.Hits[0].Subject.ID != "sales.order" || data.Hits[0].Line != 7 {
		t.Fatalf("data filter: %+v", data.Hits)
	}
	// Without a term the answer is the kind's own Sources, blank lines aside.
	listed := searchIn(t, SearchQuery{Kind: "semantic-data"}, 50, 0)
	if listed.Total != 5 {
		t.Fatalf("kind listing total=%d, want the 5 non-blank lines of the data block", listed.Total)
	}
	for _, hit := range listed.Hits {
		if hit.Subject.Kind != "semantic-data" || hit.Subject.ID != "sales.order" {
			t.Fatalf("kind listing leaked %+v", hit.Subject)
		}
	}
	failure := searchFailure(t, SearchQuery{Term: "order", Kind: "semantic-nope"}, 50, 0)
	if failure.Code != SearchCodeInvalidArgument || !strings.Contains(failure.Message, "semantic-nope") {
		t.Fatalf("unknown kind: %s %q", failure.Code, failure.Message)
	}
}

// The limit and offset page one deterministic answer; total, kindCounts and
// omitted describe the whole set, and a clamped offset stays observable.
func TestModelSearchPaging(t *testing.T) {
	// "sales" appears on the namespace line, the attribute line and the
	// relation key line: three matches, three different subjects.
	full := searchIn(t, SearchQuery{Term: "sales"}, 50, 0)
	if full.Total != 3 || len(full.Hits) != 3 || full.Omitted != 0 {
		t.Fatalf("total=%d hits=%d", full.Total, len(full.Hits))
	}
	if full.SubjectCount != 3 || len(full.Subjects) != full.SubjectCount {
		t.Fatalf("subjectCount=%d subjects=%d, want the three distinct subjects",
			full.SubjectCount, len(full.Subjects))
	}
	sum := 0
	for _, kc := range full.KindCounts {
		sum += kc.Count
	}
	// kindCounts counts hits, so its sum is total; here every subject matched
	// exactly once, which is why it also equals the subject count.
	if sum != full.Total {
		t.Fatalf("kindCounts sum %d != total %d", sum, full.Total)
	}
	if full.SubjectCount != distinctSearchSubjects(full.Hits) {
		t.Fatalf("subjectCount=%d, want the %d distinct subjects",
			full.SubjectCount, distinctSearchSubjects(full.Hits))
	}
	if full.DistinctSubjectsOmitted != 0 {
		t.Fatalf("distinctSubjectsOmitted=%d, want 0 when the page holds every match",
			full.DistinctSubjectsOmitted)
	}
	// The subject list is the same answer without the repetition: one entry per
	// distinct subject, in the order each first matched.
	if len(full.Subjects) != full.SubjectCount {
		t.Fatalf("subjects=%d, want subjectCount %d", len(full.Subjects), full.SubjectCount)
	}
	order := map[sourceindex.Subject]int{}
	for i, subject := range full.Subjects {
		if _, dup := order[subject.Subject]; dup {
			t.Fatalf("subject %+v is listed twice", subject.Subject)
		}
		order[subject.Subject] = i
		if subject.MatchCount < 1 {
			t.Fatalf("subject %+v counted %d matching lines", subject.Subject, subject.MatchCount)
		}
	}
	for _, hit := range full.Hits {
		at, ok := order[hit.Subject]
		if !ok {
			t.Fatalf("hit %+v has no subject entry", hit)
		}
		first := full.Subjects[at]
		if first.ArtifactPath != hit.ArtifactPath || first.Line > hit.Line {
			t.Fatalf("subject entry %+v is after its hit %+v", first, hit)
		}
	}
	page := searchIn(t, SearchQuery{Term: "sales"}, 2, 0)
	if len(page.Hits) != 2 || page.Omitted != 1 || page.NextOffset == nil || *page.NextOffset != 2 {
		t.Fatalf("first page: hits=%d omitted=%d next=%v", len(page.Hits), page.Omitted, page.NextOffset)
	}
	if page.Hits[0] != full.Hits[0] || page.Hits[1] != full.Hits[1] {
		t.Fatal("the first page is not a prefix of the full answer")
	}
	last := searchIn(t, SearchQuery{Term: "sales"}, 2, 2)
	if len(last.Hits) != 1 || last.Offset != 2 || last.NextOffset != nil {
		t.Fatalf("last page: hits=%d offset=%d next=%v", len(last.Hits), last.Offset, last.NextOffset)
	}
	clamped := searchIn(t, SearchQuery{Term: "sales"}, 2, 9)
	if len(clamped.Hits) != 0 || clamped.Offset != 3 || clamped.RequestedOffset != 9 {
		t.Fatalf("clamped page: hits=%d offset=%d requested=%d",
			len(clamped.Hits), clamped.Offset, clamped.RequestedOffset)
	}
}

// The two count units answer different questions. "customer" matches two lines
// of one relation, so kindCounts -- a breakdown of the hits by kind -- reports
// two hits while subjectCount reports one subject: the hit count must not be
// read as a subject count, because a declaration spans several lines.
func TestModelSearchKindCountsCountHitsWhileSubjectCountCountsSubjects(t *testing.T) {
	// "customer" is on the relation's declaration line and on its key line,
	// and both lines are attributed to the one relation.
	got := searchIn(t, SearchQuery{Term: "customer"}, 50, 0)
	if got.Total != 2 || len(got.Hits) != 2 {
		t.Fatalf("total=%d hits=%d, want the two matching lines", got.Total, len(got.Hits))
	}
	for _, hit := range got.Hits {
		if hit.Subject.Kind != "semantic-relation" || hit.Subject.ID != "sales.order.customer" {
			t.Fatalf("hit=%+v, want the one relation subject", hit)
		}
	}
	// kindCounts counts hits, so its sum is total and its relation count is the
	// relation's two lines, not its one subject.
	if len(got.KindCounts) != 1 || got.KindCounts[0].Kind != "semantic-relation" ||
		got.KindCounts[0].Count != 2 {
		t.Fatalf("kindCounts=%+v, want two semantic-relation hits", got.KindCounts)
	}
	sum := 0
	for _, kc := range got.KindCounts {
		sum += kc.Count
	}
	if sum != got.Total {
		t.Fatalf("kindCounts sum %d != total %d", sum, got.Total)
	}
	// subjectCount counts distinct subjects, so the two lines are one subject,
	// and the subject list names it once with both of its lines counted.
	if got.SubjectCount != 1 || len(got.Subjects) != 1 {
		t.Fatalf("subjectCount=%d subjects=%d, want one subject", got.SubjectCount, len(got.Subjects))
	}
	if got.DistinctSubjectsOmitted != 0 {
		t.Fatalf("distinctSubjectsOmitted=%d, want 0 when the page holds every match",
			got.DistinctSubjectsOmitted)
	}
	wantSubject := ModelSearchSubject{
		Subject:      sourceindex.Subject{Kind: "semantic-relation", ID: "sales.order.customer"},
		ArtifactPath: "semantic/sales/order.tei",
		Line:         6,
		MatchCount:   2,
	}
	if got.Subjects[0] != wantSubject {
		t.Fatalf("subject=%+v want %+v", got.Subjects[0], wantSubject)
	}
	named := 0
	for _, limit := range got.AuthorityLimits {
		switch limit {
		case "total-and-kindCounts-count-hits-subjectCount-counts-distinct-subjects",
			"distinctSubjectsOmitted-counts-subjects-among-the-matches-this-page-does-not-return":
			named++
		}
	}
	if named != 2 {
		t.Fatalf("authorityLimits do not state both counting rules: %v", got.AuthorityLimits)
	}
}

// A page of one hit must not shrink the subject answer: subjectCount still
// reports the full count and distinctSubjectsOmitted reports how many subjects
// the page does not return. "sales" matches the namespace line first, and the
// namespace is the only line that subject matches, so the two omitted matches
// carry the other two subjects and distinctSubjectsOmitted is subjectCount-1.
func TestModelSearchSubjectCountSurvivesAPageOfOneHit(t *testing.T) {
	full := searchIn(t, SearchQuery{Term: "sales"}, 50, 0)
	if full.SubjectCount != 3 {
		t.Fatalf("full subjectCount=%d, want 3", full.SubjectCount)
	}
	page := searchIn(t, SearchQuery{Term: "sales"}, 1, 0)
	if len(page.Hits) != 1 || page.Omitted != 2 {
		t.Fatalf("page: hits=%d omitted=%d", len(page.Hits), page.Omitted)
	}
	if page.SubjectCount != full.SubjectCount {
		t.Fatalf("page subjectCount=%d, want the full count %d", page.SubjectCount, full.SubjectCount)
	}
	if len(page.Subjects) != full.SubjectCount {
		t.Fatalf("page subjects=%d, want the full count %d", len(page.Subjects), full.SubjectCount)
	}
	// The returned hit is the namespace line, and the namespace subject matches
	// nowhere else, so the omitted matches carry the other two subjects only.
	if page.Hits[0].Subject.Kind != "semantic" || page.Hits[0].Subject.ID != "sales" {
		t.Fatalf("page hit=%+v, want the namespace line", page.Hits[0])
	}
	if page.DistinctSubjectsOmitted != page.SubjectCount-1 {
		t.Fatalf("distinctSubjectsOmitted=%d, want subjectCount-1 %d",
			page.DistinctSubjectsOmitted, page.SubjectCount-1)
	}
	// The count is of the omitted matches, not an inference from the paging
	// numbers: asking for the matches after the page returns the attribute line
	// and the relation key line, which are two different subjects.
	rest := searchIn(t, SearchQuery{Term: "sales"}, 2, 1)
	if distinctSearchSubjects(rest.Hits) != page.DistinctSubjectsOmitted {
		t.Fatalf("distinctSubjectsOmitted=%d, want the %d distinct subjects of the matches after the page",
			page.DistinctSubjectsOmitted, distinctSearchSubjects(rest.Hits))
	}
}

// A page cut by -limit can show one subject's lines and name nothing about how
// many subjects the whole answer holds. subjectCount is that number, and
// distinctSubjectsOmitted counts the subjects among the matches left out: the
// two-hit page below holds both of the relation's lines, so the two later
// subjects are the omitted ones and the relation is not counted there.
func TestModelSearchSubjectCountAnswersHowManySubjectsTheAnswerNames(t *testing.T) {
	search := func(limit int) ModelSearch {
		t.Helper()
		catalog, index, sources := searchPagingFixture(t)
		got, failure := SearchInSources(catalog, index, sources,
			SearchQuery{Term: "customer"}, limit, 0, defaultSearchBounds())
		if failure != nil {
			t.Fatalf("search limit=%d: %s %s", limit, failure.Code, failure.Message)
		}
		return got
	}
	whole := search(50)
	if whole.Total != 4 || whole.SubjectCount != 3 || len(whole.Subjects) != 3 {
		t.Fatalf("whole: total=%d subjectCount=%d subjects=%d, want 4 lines and 3 subjects",
			whole.Total, whole.SubjectCount, len(whole.Subjects))
	}
	sum := 0
	for _, kc := range whole.KindCounts {
		sum += kc.Count
	}
	// kindCounts counts hits, so its sum is the four matching lines, not the
	// three subjects: that is the whole point of the two units.
	if sum != whole.Total {
		t.Fatalf("kindCounts sum %d != total %d", sum, whole.Total)
	}
	page := search(2)
	if len(page.Hits) != 2 || page.Omitted != 2 {
		t.Fatalf("page: hits=%d omitted=%d", len(page.Hits), page.Omitted)
	}
	if distinctSearchSubjects(page.Hits) != 1 {
		t.Fatalf("page hits name %d subjects, want the one relation",
			distinctSearchSubjects(page.Hits))
	}
	if page.SubjectCount != 3 || len(page.Subjects) != 3 {
		t.Fatalf("page: subjectCount=%d subjects=%d, want the whole subject answer",
			page.SubjectCount, len(page.Subjects))
	}
	// The page holds both of the relation's lines, so the omitted matches are
	// the data element's line and the attribute's line: two subjects, and the
	// relation is not one of them because none of its lines was left out.
	if page.DistinctSubjectsOmitted != 2 {
		t.Fatalf("distinctSubjectsOmitted=%d, want the two subjects after the page",
			page.DistinctSubjectsOmitted)
	}
	if page.Subjects[0].Subject.ID != "sales.order.customer" || page.Subjects[0].MatchCount != 2 {
		t.Fatalf("page subjects[0]=%+v, want the relation with its two lines",
			page.Subjects[0])
	}
}

// Distinct is by subject, not by (subject, Source): the same data element is
// declared in one Source and authored again in a second, and "order" matches
// three of its lines across the two. The subject answer holds one entry for it,
// counted once in subjectCount, with its first location in the search's path
// order and every one of its lines counted.
func TestModelSearchSubjectsAreDistinctBySubjectNotBySource(t *testing.T) {
	first := "semantic/sales/order.tei"
	// The name sorts after first, so the definition's Source is searched first
	// and the entry's location is the declaration rather than the second Source.
	second := "semantic/sales/order_notes.tei"
	notes := `# notes for the order element
data Order {
    attribute note type sales.order_id
}
`
	index := sourceindex.Index{
		Artifacts: []sourceindex.Artifact{
			{Path: first, Fingerprint: sourceindex.Fingerprint([]byte(searchSource))},
			{Path: second, Fingerprint: sourceindex.Fingerprint([]byte(notes))},
		},
		References: []sourceindex.Reference{
			{Subject: sourceindex.Subject{Kind: "semantic", ID: "sales"}, ArtifactPath: first, Range: searchAnchor(t, searchSource, "sales"), Role: sourceindex.RoleDefinition},
			{Subject: sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}, ArtifactPath: first, Range: searchAnchor(t, searchSource, "Order"), Role: sourceindex.RoleDefinition},
			{Subject: sourceindex.Subject{Kind: "semantic-attribute", ID: "sales.order.order_id"}, ArtifactPath: first, Range: searchAnchor(t, searchSource, "orderId"), Role: sourceindex.RoleDefinition},
			{Subject: sourceindex.Subject{Kind: "semantic-relation", ID: "sales.order.customer"}, ArtifactPath: first, Range: searchAnchor(t, searchSource, "customer"), Role: sourceindex.RoleDefinition},
			{Subject: sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}, ArtifactPath: second, Range: searchAnchor(t, notes, "Order"), Role: sourceindex.RoleAuthoring},
		},
	}
	index.Normalize()
	world := ir.World{Semantics: []semantic.Namespace{{
		ID: "sales",
		Data: []semantic.Data{{
			ID:     "sales.order",
			Symbol: "Order",
			Attributes: []semantic.Attribute{
				{ID: "sales.order.order_id", Symbol: "orderId", TypeRef: "sales.order_id"},
			},
			Relations: []semantic.Relation{
				{ID: "sales.order.customer", Symbol: "customer", Target: "customer.customer"},
			},
		}},
	}}}
	catalog := CatalogWorld(world, index, "", "world-fp", "source-fp")
	sources := map[string][]byte{first: []byte(searchSource), second: []byte(notes)}
	got, failure := SearchInSources(catalog, index, sources,
		SearchQuery{Term: "order"}, 50, 0, defaultSearchBounds())
	if failure != nil {
		t.Fatalf("search: %s %s", failure.Code, failure.Message)
	}
	// "order" matches the data declaration in the first Source, its attribute
	// and relation key lines there, and the two data lines of the second.
	if got.Total != 5 {
		t.Fatalf("total=%d, want the five matching lines", got.Total)
	}
	if got.SubjectCount != 3 || len(got.Subjects) != 3 {
		t.Fatalf("subjectCount=%d subjects=%d, want three distinct subjects",
			got.SubjectCount, len(got.Subjects))
	}
	want := ModelSearchSubject{
		Subject:      sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"},
		ArtifactPath: first,
		Line:         4,
		MatchCount:   3,
	}
	if got.Subjects[0] != want {
		t.Fatalf("subjects[0]=%+v want %+v", got.Subjects[0], want)
	}
	counted := 0
	for _, subject := range got.Subjects {
		counted += subject.MatchCount
	}
	if counted != got.Total {
		t.Fatalf("matchCounts sum %d != total %d", counted, got.Total)
	}
}

// An empty hit list is a real answer, not a failure and not "the text is
// nowhere": coverage names what was not searched.
func TestModelSearchZeroHitsIsASuccessWithCoverage(t *testing.T) {
	got := searchIn(t, SearchQuery{Term: "no-such-text"}, 50, 0)
	if got.Total != 0 || len(got.Hits) != 0 || len(got.KindCounts) != 0 || got.Omitted != 0 {
		t.Fatalf("zero-hit answer is not clean: %+v", got)
	}
	if got.Subjects == nil || len(got.Subjects) != 0 {
		t.Fatalf("subjects=%+v, want an empty list, not null", got.Subjects)
	}
	if got.SubjectCount != 0 {
		t.Fatalf("subjectCount=%d on a zero-hit answer", got.SubjectCount)
	}
	if got.DistinctSubjectsOmitted != 0 {
		t.Fatalf("distinctSubjectsOmitted=%d on a zero-hit answer", got.DistinctSubjectsOmitted)
	}
	if len(got.Coverage.Derived) == 0 || len(got.Coverage.NotDerived) == 0 {
		t.Fatalf("coverage=%+v", got.Coverage)
	}
	named := false
	for _, limit := range got.AuthorityLimits {
		if limit == "a-line-is-attributed-to-the-nearest-preceding-declaration-among-the-searched-kinds" {
			named = true
		}
	}
	if !named {
		t.Fatalf("authorityLimits do not state the attribution rule: %v", got.AuthorityLimits)
	}
}

func TestModelSearchRefusesAnUnapplicableQuery(t *testing.T) {
	cases := []struct {
		name   string
		query  SearchQuery
		limit  int
		offset int
		code   string
	}{
		{"neither term nor kind", SearchQuery{}, 50, 0, SearchCodeInvalidArgument},
		{"padded term", SearchQuery{Term: " order "}, 50, 0, SearchCodeInvalidArgument},
		{"unknown kind", SearchQuery{Kind: "semantic-nope"}, 50, 0, SearchCodeInvalidArgument},
		{"limit zero", SearchQuery{Term: "sales"}, 0, 0, SearchCodeInvalidLimit},
		{"limit over max", SearchQuery{Term: "sales"}, SearchMaxLimit + 1, 0, SearchCodeInvalidLimit},
		{"negative offset", SearchQuery{Term: "sales"}, 50, -1, SearchCodeInvalidOffset},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			failure := searchFailure(t, tc.query, tc.limit, tc.offset)
			if failure.Code != tc.code {
				t.Fatalf("code=%q want %q (%s)", failure.Code, tc.code, failure.Message)
			}
		})
	}
}

// The Workspace read answers with the catalog's fingerprints and every hit it
// returns is re-checked against the file on disk: the line number, the text and
// the term are verified independently of the projection.
func TestModelSearchWorkspaceAgreesWithCatalogAndReportsRealLines(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	catalog, err := CatalogWorkspace(root, "")
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}
	got, err := SearchWorkspace(root, SearchQuery{Term: "namespace"}, 500, 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if got.WorldFingerprint != catalog.WorldFingerprint ||
		got.SourceIndexFingerprint != catalog.SourceIndexFingerprint ||
		got.ModelRevision != catalog.ModelRevision {
		t.Fatal("search and catalog disagree on the model version")
	}
	if got.Total == 0 || len(got.Hits) == 0 {
		t.Fatal("the reference workspace declares no namespace line")
	}
	for _, hit := range got.Hits {
		if !SupportedInspectionKind(hit.Subject.Kind) {
			t.Fatalf("hit kind %q is outside the closed set", hit.Subject.Kind)
		}
		if hit.ArtifactPath != filepath.ToSlash(filepath.Clean(hit.ArtifactPath)) ||
			strings.HasPrefix(hit.ArtifactPath, "/") || strings.Contains(hit.ArtifactPath, "..") {
			t.Fatalf("hit path %q is not workspace-relative", hit.ArtifactPath)
		}
		content, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(hit.ArtifactPath)))
		if readErr != nil {
			t.Fatalf("hit points at %s, which cannot be read: %v", hit.ArtifactPath, readErr)
		}
		lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
		if hit.Line < 1 || hit.Line > len(lines) {
			t.Fatalf("hit line %d is outside %s", hit.Line, hit.ArtifactPath)
		}
		if lines[hit.Line-1] != hit.Text {
			t.Fatalf("hit text %q is not line %d of %s (%q)",
				hit.Text, hit.Line, hit.ArtifactPath, lines[hit.Line-1])
		}
		if !strings.Contains(strings.ToLower(hit.Text), "namespace") {
			t.Fatalf("hit %+v does not carry the term", hit)
		}
	}
	// The subject list accounts for every hit: one entry per distinct subject,
	// each counting its own matching lines, and the counts add up to the total
	// because a subject is counted once however many lines it matched on.
	counted := 0
	listed := map[string]bool{}
	for _, subject := range got.Subjects {
		key := subject.Subject.Kind + "\x00" + subject.Subject.ID
		if listed[key] {
			t.Fatalf("subject %+v is listed twice", subject.Subject)
		}
		listed[key] = true
		if subject.MatchCount < 1 {
			t.Fatalf("subject %+v counted no matching line", subject.Subject)
		}
		counted += subject.MatchCount
	}
	if counted != got.Total {
		t.Fatalf("subject matchCounts sum %d != total %d", counted, got.Total)
	}
	if got.SubjectCount != len(got.Subjects) || got.SubjectCount > got.Total {
		t.Fatalf("subjectCount=%d subjects=%d total=%d", got.SubjectCount, len(got.Subjects), got.Total)
	}
	// This search asked for every hit in one page, so no subject was left out
	// and the hit breakdown still sums to the hit total.
	if got.DistinctSubjectsOmitted != 0 {
		t.Fatalf("distinctSubjectsOmitted=%d, want 0 when the page holds every match",
			got.DistinctSubjectsOmitted)
	}
	hitsByKind := 0
	for _, kc := range got.KindCounts {
		hitsByKind += kc.Count
	}
	if hitsByKind != got.Total {
		t.Fatalf("kindCounts sum %d != total %d", hitsByKind, got.Total)
	}
	// The kind filter narrows the same answer to one kind.
	filtered, err := SearchWorkspace(root, SearchQuery{Term: "namespace", Kind: "semantic"}, 500, 0)
	if err != nil {
		t.Fatalf("filtered search: %v", err)
	}
	if filtered.Total == 0 || filtered.Total > got.Total {
		t.Fatalf("filtered total=%d, unfiltered=%d", filtered.Total, got.Total)
	}
	for _, hit := range filtered.Hits {
		if hit.Subject.Kind != "semantic" {
			t.Fatalf("filtered hit kind=%q", hit.Subject.Kind)
		}
	}
	// A term no Source carries is a clean zero, and an unusable query or an
	// unreadable Workspace is a typed failure.
	none, err := SearchWorkspace(root, SearchQuery{Term: "zzz-no-such-text"}, 50, 0)
	if err != nil {
		t.Fatalf("zero-hit search: %v", err)
	}
	if none.Total != 0 || len(none.Hits) != 0 {
		t.Fatalf("zero-hit total=%d", none.Total)
	}
	if none.Truncated {
		t.Fatal("zero-hit search reports truncated")
	}
	if _, err := SearchWorkspace(filepath.Join(root, "missing"), SearchQuery{Term: "namespace"}, 50, 0); err == nil {
		t.Fatal("missing workspace accepted")
	}
	var failure *ValidationFailure
	if _, err := SearchWorkspace(filepath.Join(root, "missing"), SearchQuery{Term: "namespace"}, 50, 0); errors.As(err, &failure) {
		if failure.Stage != StageLoad {
			t.Fatalf("stage=%v", failure.Stage)
		}
	}
}

// A term several lines of ONE subject carry is several hits and one subject:
// `sales_core` is written on three lines of the order_entry application, so
// subjectCount is 1 while total is 3, and the hit breakdown by kind reports the
// three hits rather than the one subject. The subject answer must not be an
// artifact of how many lines the declaration occupies.
func TestModelSearchSalesCoreIsThreeHitsAndOneSubject(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	got, err := SearchWorkspace(root, SearchQuery{Term: "sales_core"}, 500, 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if got.Total != 3 || len(got.Hits) != 3 {
		t.Fatalf("total=%d hits=%d, want the three matching lines", got.Total, len(got.Hits))
	}
	for _, hit := range got.Hits {
		if hit.Subject.Kind != "application" || hit.Subject.ID != "order_entry" {
			t.Fatalf("hit=%+v, want the one order_entry application", hit)
		}
	}
	if got.SubjectCount != 1 || len(got.Subjects) != 1 {
		t.Fatalf("subjectCount=%d subjects=%d, want the one application",
			got.SubjectCount, len(got.Subjects))
	}
	if got.Subjects[0].MatchCount != got.Total {
		t.Fatalf("matchCount=%d, want the %d matching lines", got.Subjects[0].MatchCount, got.Total)
	}
	// kindCounts counts hits, so the application count is the three lines.
	if len(got.KindCounts) != 1 || got.KindCounts[0].Kind != "application" ||
		got.KindCounts[0].Count != 3 {
		t.Fatalf("kindCounts=%+v, want three application hits", got.KindCounts)
	}
	if got.DistinctSubjectsOmitted != 0 {
		t.Fatalf("distinctSubjectsOmitted=%d, want 0 when the page holds every match",
			got.DistinctSubjectsOmitted)
	}
	// A page of one hit keeps the full subject answer: still the one subject.
	// The two omitted matches are that same subject's other lines, so one
	// distinct subject exists among them: the count is of the subjects the
	// omitted matches name, not of subjects the page has not shown yet.
	page, err := SearchWorkspace(root, SearchQuery{Term: "sales_core"}, 1, 0)
	if err != nil {
		t.Fatalf("paged search: %v", err)
	}
	if page.SubjectCount != 1 || len(page.Subjects) != 1 {
		t.Fatalf("page subjectCount=%d subjects=%d, want the one subject",
			page.SubjectCount, len(page.Subjects))
	}
	if page.DistinctSubjectsOmitted != 1 {
		t.Fatalf("page distinctSubjectsOmitted=%d, want the one subject the omitted matches name",
			page.DistinctSubjectsOmitted)
	}
}

// The indexed bytes are the evidence: a Source whose content no longer matches
// the fingerprint the index recorded fails the read instead of answering from
// bytes the model revision does not describe.
func TestModelSearchRefusesASourceThatChangedUnderTheRead(t *testing.T) {
	catalog, index, sources := searchFixture(t)
	path := "semantic/sales/order.tei"
	tampered := map[string][]byte{path: []byte(strings.Replace(searchSource, "namespace sales", "namespace other", 1))}
	if _, failure := SearchInSources(catalog, index, tampered, SearchQuery{Term: "namespace"}, 50, 0, defaultSearchBounds()); failure == nil ||
		failure.Code != SearchCodeSourceChanged {
		t.Fatalf("changed source: %+v", failure)
	}
	// The untampered read still answers.
	if _, failure := SearchInSources(catalog, index, sources, SearchQuery{Term: "namespace"}, 50, 0, defaultSearchBounds()); failure != nil {
		t.Fatalf("untampered read: %s %s", failure.Code, failure.Message)
	}
	// A Source the index binds but the read did not return is a read failure,
	// not a silently smaller answer.
	partial := map[string][]byte{}
	if _, failure := SearchInSources(catalog, index, partial, SearchQuery{Term: "namespace"}, 50, 0, defaultSearchBounds()); failure == nil ||
		failure.Code != SearchCodeSourceUnreadable {
		t.Fatalf("missing source bytes: %+v", failure)
	}
}

// The work bound stops the scan early instead of reading the Workspace
// unbounded, and a search that stopped reports truncated so a caller cannot
// read the hits, or an empty hit list, as the whole Workspace's answer.
func TestModelSearchWorkBoundReportsTruncated(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	whole, err := SearchWorkspace(root, SearchQuery{Term: "namespace"}, 500, 0)
	if err != nil {
		t.Fatalf("whole search: %v", err)
	}
	if whole.Truncated {
		t.Fatal("the reference workspace reached the documented work bound")
	}
	if whole.Total == 0 {
		t.Fatal("the reference workspace declares no namespace line")
	}
	// One Source is enough to find a match, but the bound must say the answer is
	// partial rather than let the smaller hit count stand for the whole scan.
	bounded, err := searchWorkspace(root, SearchQuery{Term: "namespace"}, 500, 0,
		SearchBounds{MaxFiles: 1})
	if err != nil {
		t.Fatalf("bounded search: %v", err)
	}
	if !bounded.Truncated {
		t.Fatal("a search stopped at its file bound does not report truncated")
	}
	if bounded.Total > whole.Total {
		t.Fatalf("bounded total=%d exceeds the whole total=%d", bounded.Total, whole.Total)
	}
	// A byte bound stops the same way, before opening a Source that would take
	// the scanned bytes past it.
	byBytes, err := searchWorkspace(root, SearchQuery{Term: "namespace"}, 500, 0,
		SearchBounds{MaxBytes: 1})
	if err != nil {
		t.Fatalf("byte-bounded search: %v", err)
	}
	if !byBytes.Truncated || byBytes.Total != 0 || len(byBytes.Hits) != 0 {
		t.Fatalf("byte bound: truncated=%v total=%d", byBytes.Truncated, byBytes.Total)
	}
}
