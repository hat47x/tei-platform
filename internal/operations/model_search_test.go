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
	sum := 0
	for _, kc := range full.KindCounts {
		sum += kc.Count
	}
	if sum != full.Total {
		t.Fatalf("kindCounts sum %d != total %d", sum, full.Total)
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

// An empty hit list is a real answer, not a failure and not "the text is
// nowhere": coverage names what was not searched.
func TestModelSearchZeroHitsIsASuccessWithCoverage(t *testing.T) {
	got := searchIn(t, SearchQuery{Term: "no-such-text"}, 50, 0)
	if got.Total != 0 || len(got.Hits) != 0 || len(got.KindCounts) != 0 || got.Omitted != 0 {
		t.Fatalf("zero-hit answer is not clean: %+v", got)
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
