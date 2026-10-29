package ordermultilingualoperation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tei/internal/modelref"
	"tei/internal/subject"
	business "tei/reference/multilingual-business-data"
)

// Product is one product description in two languages.
type Product struct {
	Code      string `json:"code"`
	JA        string `json:"ja"`
	EN        string `json:"en"`
	JARevised string `json:"jaRevised,omitempty"`
}

// DescriptionField is the attribute the sample uses for product
// descriptions. It is not declared in reference/workspace (PlatformGaps).
func (u Unified) DescriptionField() modelref.Key {
	return modelref.Key{Scope: u.Scope, Subject: subject.Subject{Kind: "attribute", ID: "catalog.product.description"}}
}

// DescriptionKey maps an order line item code to the business key of its
// description. This mapping is the sample's, not an accepted relation.
func (u Unified) DescriptionKey(itemCode, locale string) business.Key {
	return business.Key{Tenant: u.Tenant, Record: "product:" + itemCode, Field: u.DescriptionField(), Locale: locale, Edition: u.Edition}
}

const insertRevision = `INSERT INTO tei_multilingual_revision
 (tenant_id, record_id, field_scope, field_kind, field_id, locale, edition, revision, body, origin_kind, review_state, source_locale, source_revision)
 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

const publish = `INSERT INTO tei_multilingual_publication
 (tenant_id, record_id, field_scope, field_kind, field_id, locale, edition, revision)
 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
 ON CONFLICT (tenant_id, record_id, field_scope, field_kind, field_id, locale, edition) DO UPDATE SET revision = excluded.revision`

func revisionStmt(k business.Key, rev, body, origin, review string, src *business.SourceRef) Stmt {
	var sl, sr any
	if src != nil {
		sl, sr = src.Key.Locale, src.Revision
	}
	return Stmt{SQL: insertRevision, Args: []any{k.Tenant, k.Record, k.Field.Scope, k.Field.Subject.Kind, k.Field.Subject.ID,
		k.Locale, k.Edition, rev, body, origin, review, sl, sr}}
}

func publishStmt(k business.Key, rev string) Stmt {
	return Stmt{SQL: publish, Args: []any{k.Tenant, k.Record, k.Field.Scope, k.Field.Subject.Kind, k.Field.Subject.ID, k.Locale, k.Edition, rev}}
}

func str(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

// ReadSnapshot reads every revision and publication of the tenant on a new
// connection and builds the asset's Snapshot.
func (u Unified) ReadSnapshot(ctx context.Context, st Store) (business.Snapshot, error) {
	snap := business.Snapshot{Model: modelref.Graph{Schema: "tei.modelref.v0", Nodes: []modelref.Node{{Key: u.DescriptionField()}}}}
	rows, err := st.Query(ctx, `SELECT record_id, field_scope, field_kind, field_id, locale, edition, revision, body, origin_kind,
 review_state, source_locale, source_revision FROM tei_multilingual_revision WHERE tenant_id = ? ORDER BY record_id, locale, revision`, u.Tenant)
	if err != nil {
		return snap, err
	}
	key := func(r []any, locale string) business.Key {
		return business.Key{Tenant: u.Tenant, Record: str(r[0]), Field: modelref.Key{Scope: str(r[1]),
			Subject: subject.Subject{Kind: str(r[2]), ID: str(r[3])}}, Locale: locale, Edition: str(r[5])}
	}
	for _, r := range rows {
		rev := business.Revision{Key: key(r, str(r[4])), Revision: str(r[6]), Text: str(r[7]), Origin: str(r[8]), Review: str(r[9])}
		if r[10] != nil {
			rev.BasedOn = &business.SourceRef{Key: key(r, str(r[10])), Revision: str(r[11])}
		}
		snap.Revisions = append(snap.Revisions, rev)
	}
	pubs, err := st.Query(ctx, `SELECT record_id, field_scope, field_kind, field_id, locale, edition, revision
 FROM tei_multilingual_publication WHERE tenant_id = ? ORDER BY record_id, locale`, u.Tenant)
	if err != nil {
		return snap, err
	}
	for _, p := range pubs {
		snap.Publications = append(snap.Publications, business.Publication{Key: key(p, str(p[4])), Revision: str(p[6])})
	}
	return snap, nil
}

// CompareSelectWithView compares the asset's Go Select with the SQL view
// tei_multilingual_current row by row on the same store.
func (u Unified) CompareSelectWithView(ctx context.Context, st Store) (map[string]any, bool, error) {
	snap, err := u.ReadSnapshot(ctx, st)
	if err != nil {
		return nil, false, err
	}
	goRows := map[string]string{}
	for _, p := range snap.Publications {
		r := business.Select(snap, p.Key)
		if r.Diagnostic == "" {
			goRows[p.Key.Record+" "+p.Key.Locale] = r.Revision + " " + r.Text
		}
	}
	view, err := st.Query(ctx, `SELECT record_id, locale, revision, body FROM tei_multilingual_current WHERE tenant_id = ?`, u.Tenant)
	if err != nil {
		return nil, false, err
	}
	sqlRows := map[string]string{}
	for _, v := range view {
		sqlRows[str(v[0])+" "+str(v[1])] = str(v[2]) + " " + str(v[3])
	}
	var diff []string
	for k, v := range goRows {
		if sqlRows[k] != v {
			diff = append(diff, "go-only "+k)
		}
	}
	for k, v := range sqlRows {
		if goRows[k] != v {
			diff = append(diff, "sql-only "+k)
		}
	}
	sort.Strings(diff)
	return map[string]any{"goRows": len(goRows), "viewRows": len(sqlRows), "differences": diff,
		"snapshotValid": len(business.Validate(snap)) == 0}, len(diff) == 0 && len(goRows) > 0, nil
}

var errOutcomeUnknown = errors.New("translator connection closed after the request was sent; outcome unknown")

// MockTranslator is the sample's translator. It sends nothing anywhere: a
// call is counted, an "external" flag only changes how the asset gates it.
type MockTranslator struct {
	Mode        string // ok, fail, cancel, unknown
	IsExternal  bool
	Output      string
	Cancel      context.CancelFunc
	During      func() error // another person's update while translating
	Calls       int
	Completions int // the provider processed the request
	Payloads    []business.TranslationContext
	LastErr     error
	DuringErr   error
}

func (m *MockTranslator) Name() string   { return "mock-translator" }
func (m *MockTranslator) External() bool { return m.IsExternal }
func (m *MockTranslator) Translate(ctx context.Context, in business.TranslationContext) (string, error) {
	m.Calls++
	m.Payloads = append(m.Payloads, in)
	if m.During != nil {
		m.DuringErr = m.During()
	}
	switch m.Mode {
	case "fail":
		m.LastErr = errors.New("translator returned 503")
	case "cancel":
		m.Cancel()
		m.LastErr = ctx.Err()
	case "unknown":
		m.Completions++
		m.LastErr = errOutcomeUnknown
	default:
		m.Completions++
		return m.Output, nil
	}
	return "", m.LastErr
}

// Attempt is the sample's record of one translation preparation. The
// sample never retries: an attempt is made once and its outcome recorded.
type Attempt struct {
	Scenario    string `json:"scenario"`
	Record      string `json:"record"`
	Diagnostic  string `json:"diagnostic,omitempty"`
	Kind        string `json:"kind,omitempty"`
	Outcome     string `json:"outcome"`
	Calls       int    `json:"translatorCalls"`
	Completions int    `json:"providerCompletions"`
	Stored      string `json:"stored"`
	DuringError string `json:"duringError,omitempty"`
}

func classify(a business.TranslationAttempt, m *MockTranslator) string {
	switch {
	case a.Kind == "candidate":
		return "candidate"
	case m != nil && errors.Is(m.LastErr, errOutcomeUnknown):
		return "outcome-unknown"
	case a.Diagnostic == "source-changed" || a.Diagnostic == "target-changed" || a.Diagnostic == "source-content-changed":
		return "conflict"
	case a.Diagnostic == "external-transfer-blocked":
		return "blocked"
	}
	return "failed"
}

// StoreResult is the stage 5 observation of one product.
type StoreResult struct {
	Store          string            `json:"store"`
	Info           map[string]string `json:"info"`
	Compare        map[string]any    `json:"selectVersusView"`
	CompareOK      bool              `json:"selectVersusViewEqual"`
	HistoryKept    bool              `json:"historyKept"`
	StaleHidden    bool              `json:"staleTranslationHidden"`
	Attempts       []Attempt         `json:"attempts"`
	CandidateRows  int               `json:"candidateRows"`
	PublishedAfter map[string]string `json:"publishedAfterAttempts"`
	Lists          map[string]List   `json:"lists"`
}

// List is one ordered list with its count and hash.
type List struct {
	Order OrderKey `json:"order"`
	Count int      `json:"count"`
	Hash  string   `json:"hash"`
	Rows  []string `json:"rows"`
}

func listHash(rows []string) string { return digestBytes([]byte(strings.Join(rows, "\n"))) }

func (u Unified) list(ctx context.Context, st Store, k OrderKey, after []string, limit int) ([]string, [][]any, error) {
	q := "SELECT body, record_id, locale, revision FROM tei_multilingual_current WHERE tenant_id = ?"
	args := []any{u.Tenant}
	if after != nil {
		q += " AND " + st.RowGreater(k)
		for _, a := range after {
			args = append(args, a)
		}
	}
	q += " ORDER BY " + st.Order(k)
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
	}
	raw, err := st.Query(ctx, q, args...)
	if err != nil {
		return nil, nil, err
	}
	var rows []string
	for _, r := range raw {
		rows = append(rows, str(r[1])+"\t"+str(r[2])+"\t"+str(r[3])+"\t"+str(r[0]))
	}
	return rows, raw, nil
}

// Paginate fetches pages alternately from the given stores using the last
// key of the previous page (keyset continuation), with the order key each
// store is given.
func (u Unified) Paginate(ctx context.Context, stores []Store, keys []OrderKey, size int) ([]string, error) {
	var out []string
	var after []string
	for page := 0; page < 100; page++ {
		i := page % len(stores)
		rows, raw, err := u.list(ctx, stores[i], keys[i], after, size)
		if err != nil {
			return nil, err
		}
		out = append(out, rows...)
		if len(raw) < size {
			return out, nil
		}
		last := raw[len(raw)-1]
		after = []string{str(last[0]), str(last[1]), str(last[2])}
	}
	return out, fmt.Errorf("pagination did not end")
}

// PageDefects counts duplicates and missing rows of a paginated result
// against the full set.
func PageDefects(full, paged []string) (dup, missing int) {
	seen := map[string]int{}
	for _, r := range paged {
		seen[r]++
	}
	for _, n := range seen {
		if n > 1 {
			dup += n - 1
		}
	}
	for _, r := range full {
		if seen[r] == 0 {
			missing++
		}
	}
	return dup, missing
}

// Binary is the declared order key of the lists.
var Binary = OrderKey{Columns: []string{"body", "record_id", "locale"}, Comparison: "binary-utf8"}

func (u Unified) translationJob(code, srcRev, expectedTarget string) business.TranslationJob {
	return business.TranslationJob{
		Source:                 business.SourceRef{Key: u.DescriptionKey(code, "ja-JP"), Revision: srcRev},
		Target:                 u.DescriptionKey(code, "en-US"),
		ExpectedTargetRevision: expectedTarget, ReadAuthorized: true,
		ExternalTransferAuthorized: true, Classification: "public",
	}
}

// RunStore runs the stage 5 sequence on one store.
func (c *Chain) RunStore(ctx context.Context, st Store, products []Product) (StoreResult, error) {
	u := c.U
	res := StoreResult{Store: st.Name(), Info: st.Info(ctx), Lists: map[string]List{}}
	schema, err := os.ReadFile(filepath.Join(c.Repo, "reference", "multilingual-business-data", "testdata", "schema.sql"))
	if err != nil {
		return res, err
	}
	if err := st.Setup(ctx, string(schema)); err != nil {
		return res, err
	}
	var seed []Stmt
	for _, p := range products {
		ja, en := u.DescriptionKey(p.Code, "ja-JP"), u.DescriptionKey(p.Code, "en-US")
		src := business.SourceRef{Key: ja, Revision: "ja-1"}
		seed = append(seed, revisionStmt(ja, "ja-1", p.JA, "original", "approved", nil),
			revisionStmt(en, "en-1", p.EN, "translation", "approved", &src), publishStmt(ja, "ja-1"), publishStmt(en, "en-1"))
	}
	if err := st.Exec(ctx, seed...); err != nil {
		return res, err
	}
	// Revise the Japanese sources (P-100 to P-102): their English
	// translations become stale but stay in the history table.
	var revise []Stmt
	for _, p := range products {
		if p.JARevised != "" {
			ja := u.DescriptionKey(p.Code, "ja-JP")
			revise = append(revise, revisionStmt(ja, "ja-2", p.JARevised, "original", "approved", nil), publishStmt(ja, "ja-2"))
		}
	}
	if err := st.Exec(ctx, revise...); err != nil {
		return res, err
	}
	hist, err := st.Query(ctx, `SELECT count(*) FROM tei_multilingual_revision WHERE tenant_id = ? AND record_id = ? AND locale = 'en-US'`, u.Tenant, "product:P-100")
	if err != nil {
		return res, err
	}
	vis, err := st.Query(ctx, `SELECT count(*) FROM tei_multilingual_current WHERE tenant_id = ? AND record_id = ? AND locale = 'en-US'`, u.Tenant, "product:P-100")
	if err != nil {
		return res, err
	}
	res.HistoryKept = str(hist[0][0]) == "1"
	res.StaleHidden = str(vis[0][0]) == "0"
	if res.Compare, res.CompareOK, err = u.CompareSelectWithView(ctx, st); err != nil {
		return res, err
	}
	reader := func(ctx context.Context) (business.Snapshot, error) { return u.ReadSnapshot(ctx, st) }

	attempt := func(scenario, code string, job business.TranslationJob, m *MockTranslator, actx context.Context) {
		a := business.PrepareTranslation(actx, reader, m, job)
		at := Attempt{Scenario: scenario, Record: "product:" + code, Diagnostic: a.Diagnostic, Kind: a.Kind,
			Outcome: classify(a, m), Calls: m.Calls, Completions: m.Completions, Stored: "nothing"}
		if m.DuringErr != nil {
			at.DuringError = m.DuringErr.Error()
		}
		if a.Kind == "candidate" {
			// The proposal is stored as an unapproved candidate, never published.
			src := a.Proposal.Source
			if err := st.Exec(ctx, revisionStmt(a.Proposal.Target, "en-2-candidate", a.Text, "translation", "candidate", &src)); err != nil {
				at.Stored = "store failed: " + err.Error()
			} else {
				at.Stored = "candidate"
			}
		}
		res.Attempts = append(res.Attempts, at)
	}
	bg := context.Background()
	job := u.translationJob("P-100", "ja-2", "en-1")
	restricted := job
	restricted.Classification = "restricted"
	attempt("restricted-text-to-external", "P-100", restricted, &MockTranslator{IsExternal: true, Output: "x"}, bg)
	unauth := job
	unauth.ExternalTransferAuthorized = false
	attempt("external-without-authorization", "P-100", unauth, &MockTranslator{IsExternal: true, Output: "x"}, bg)
	attempt("translator-fails", "P-100", job, &MockTranslator{Mode: "fail", IsExternal: true}, bg)
	cctx, cancel := context.WithCancel(bg)
	attempt("translator-canceled", "P-100", job, &MockTranslator{Mode: "cancel", IsExternal: true, Cancel: cancel}, cctx)
	cancel()
	attempt("translator-outcome-unknown", "P-100", job, &MockTranslator{Mode: "unknown", IsExternal: true}, bg)
	cand, err := st.Query(ctx, `SELECT count(*) FROM tei_multilingual_revision WHERE tenant_id = ? AND review_state = 'candidate'`, u.Tenant)
	if err != nil {
		return res, err
	}
	failedLeftNothing := str(cand[0][0]) == "0"
	attempt("translator-succeeds", "P-100", job, &MockTranslator{IsExternal: true,
		Output: "Store at room temperature away from direct sunlight."}, bg)
	// Another person publishes an English description of P-101 while the
	// proposal is being prepared.
	p101 := products[1]
	en101 := u.DescriptionKey(p101.Code, "en-US")
	src101 := business.SourceRef{Key: u.DescriptionKey(p101.Code, "ja-JP"), Revision: "ja-2"}
	attempt("target-updated-meanwhile", p101.Code, u.translationJob(p101.Code, "ja-2", "en-1"), &MockTranslator{IsExternal: true,
		Output: "Handle with care.", During: func() error {
			return st.Exec(ctx, revisionStmt(en101, "en-2", "Fragile item.", "translation", "approved", &src101), publishStmt(en101, "en-2"))
		}}, bg)
	// Another person revises the Japanese source of P-102 meanwhile.
	p102 := products[2]
	ja102 := u.DescriptionKey(p102.Code, "ja-JP")
	attempt("source-updated-meanwhile", p102.Code, u.translationJob(p102.Code, "ja-2", "en-1"), &MockTranslator{IsExternal: true,
		Output: "Refrigerate.", During: func() error {
			return st.Exec(ctx, revisionStmt(ja102, "ja-3", "要冷蔵(10℃以下)。", "original", "approved", nil), publishStmt(ja102, "ja-3"))
		}}, bg)
	cand, err = st.Query(ctx, `SELECT count(*) FROM tei_multilingual_revision WHERE tenant_id = ? AND review_state = 'candidate'`, u.Tenant)
	if err != nil {
		return res, err
	}
	// -1 records that a failed attempt left a candidate behind.
	res.CandidateRows = -1
	if failedLeftNothing {
		fmt.Sscan(str(cand[0][0]), &res.CandidateRows)
	}
	pubs, err := st.Query(ctx, `SELECT record_id, locale, revision FROM tei_multilingual_publication WHERE tenant_id = ? AND record_id IN (?, ?, ?) ORDER BY record_id, locale`,
		u.Tenant, "product:P-100", "product:P-101", "product:P-102")
	if err != nil {
		return res, err
	}
	res.PublishedAfter = map[string]string{}
	for _, p := range pubs {
		res.PublishedAfter[str(p[0])+" "+str(p[1])] = str(p[2])
	}
	// Select versus view again after the concurrent updates.
	again, okAgain, err := u.CompareSelectWithView(ctx, st)
	if err != nil {
		return res, err
	}
	res.Compare["afterAttempts"] = again
	res.CompareOK = res.CompareOK && okAgain
	for name, k := range map[string]OrderKey{"binary": Binary,
		"databaseDefault": {Columns: Binary.Columns, Comparison: "database-default"},
		"icu":             {Columns: Binary.Columns, Comparison: "icu-und"}} {
		if st.Name() == "sqlite" && name != "binary" {
			continue
		}
		rows, _, err := u.list(ctx, st, k, nil, 0)
		if err != nil {
			return res, fmt.Errorf("%s list %s: %w", st.Name(), name, err)
		}
		res.Lists[name] = List{Order: k, Count: len(rows), Hash: listHash(rows), Rows: rows}
	}
	return res, nil
}

// LoadProducts reads the product descriptions.
func LoadProducts(path string) ([]Product, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Products []Product `json:"products"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	if len(doc.Products) < 3 || doc.Products[0].JARevised == "" {
		return nil, fmt.Errorf("products: P-100 with a revision and two more products are required")
	}
	return doc.Products, nil
}

func attemptBy(r StoreResult, scenario string) Attempt {
	for _, a := range r.Attempts {
		if a.Scenario == scenario {
			return a
		}
	}
	return Attempt{}
}

// Stage5 runs the product description sequence on PostgreSQL and SQLite and
// compares the declared ordered lists across the two products.
func (c *Chain) Stage5(ctx context.Context, pg, lite Store) error {
	u := c.U
	products, err := LoadProducts(c.testdata("products.json"))
	if err != nil {
		return err
	}
	// Precondition: the order line has an item code, but no product
	// description attribute exists in the workspace.
	itemCode, description := false, false
	for _, ns := range c.WS2.Loaded.World.Semantics {
		for _, d := range ns.Data {
			for _, a := range d.Attributes {
				if d.ID == "sales.order_line" && a.Symbol == "itemCode" {
					itemCode = true
				}
				if a.ID == "catalog.product.description" {
					description = true
				}
			}
		}
	}
	results := map[string]StoreResult{}
	for _, st := range []Store{pg, lite} {
		r, err := c.RunStore(ctx, st, products)
		if err != nil {
			return fmt.Errorf("stage 5 %s: %w", st.Name(), err)
		}
		results[st.Name()] = r
	}
	pr, sr := results["postgresql"], results["sqlite"]
	for _, r := range []StoreResult{pr, sr} {
		name := r.Store
		c.Rec.fault("business-data-revision-history-"+name, "固有", "5", r.HistoryKept && r.StaleHidden && r.CompareOK,
			map[string]any{"historyKept": r.HistoryKept, "staleTranslationHidden": r.StaleHidden, "selectVersusView": r.Compare})
		blocked := attemptBy(r, "restricted-text-to-external")
		unauth := attemptBy(r, "external-without-authorization")
		fail := attemptBy(r, "translator-fails")
		canceled := attemptBy(r, "translator-canceled")
		unknown := attemptBy(r, "translator-outcome-unknown")
		c.Rec.fault("ptx02-"+name, "P-TX-02", "5",
			blocked.Diagnostic == "external-transfer-blocked" && blocked.Calls == 0 &&
				unauth.Diagnostic == "external-transfer-blocked" && unauth.Calls == 0 &&
				fail.Diagnostic == "translation-failed" && fail.Calls == 1 && fail.Stored == "nothing" &&
				canceled.Diagnostic == "canceled" && canceled.Calls == 1 && canceled.Stored == "nothing" &&
				unknown.Outcome == "outcome-unknown" && unknown.Calls == 1 && unknown.Completions == 1 && unknown.Stored == "nothing" &&
				r.PublishedAfter["product:P-100 en-US"] == "en-1",
			map[string]any{"attempts": []Attempt{blocked, unauth, fail, canceled, unknown}, "published": r.PublishedAfter})
		ok := attemptBy(r, "translator-succeeds")
		tgt := attemptBy(r, "target-updated-meanwhile")
		src := attemptBy(r, "source-updated-meanwhile")
		c.Rec.fault("concurrent-edit-"+name, "固有", "5",
			ok.Kind == "candidate" && ok.Stored == "candidate" && r.CandidateRows == 1 &&
				tgt.Diagnostic == "target-changed" && tgt.Stored == "nothing" && tgt.DuringError == "" &&
				src.Diagnostic == "source-changed" && src.Stored == "nothing" && src.DuringError == "" &&
				r.PublishedAfter["product:P-100 en-US"] == "en-1" && r.PublishedAfter["product:P-101 en-US"] == "en-2" &&
				r.PublishedAfter["product:P-102 ja-JP"] == "ja-3",
			map[string]any{"attempts": []Attempt{ok, tgt, src}, "candidateRows": r.CandidateRows, "published": r.PublishedAfter})
	}
	pb, sb := pr.Lists["binary"], sr.Lists["binary"]
	pi := pr.Lists["icu"]
	full := pb.Rows
	pagedBinary, err := u.Paginate(ctx, []Store{pg, lite}, []OrderKey{Binary, Binary}, 3)
	if err != nil {
		return err
	}
	icu := OrderKey{Columns: Binary.Columns, Comparison: "icu-und"}
	pagedMixed, err := u.Paginate(ctx, []Store{pg, lite}, []OrderKey{icu, Binary}, 3)
	if err != nil {
		return err
	}
	bd, bm := PageDefects(full, pagedBinary)
	md, mm := PageDefects(full, pagedMixed)
	c.Rec.fault("pdb04-declared-binary-order", "P-DB-04", "5",
		pb.Count == sb.Count && pb.Count > 0 && pb.Hash == sb.Hash && bd == 0 && bm == 0 && listHash(pagedBinary) == pb.Hash &&
			pi.Hash != sb.Hash && (md > 0 || mm > 0),
		map[string]any{
			"postgresqlBinary": map[string]any{"count": pb.Count, "hash": pb.Hash},
			"sqliteBinary":     map[string]any{"count": sb.Count, "hash": sb.Hash},
			"postgresqlICU":    map[string]any{"count": pi.Count, "hash": pi.Hash},
			"postgresqlDatabaseDefault": map[string]any{"count": pr.Lists["databaseDefault"].Count,
				"hash": pr.Lists["databaseDefault"].Hash, "equalsBinary": pr.Lists["databaseDefault"].Hash == pb.Hash},
			"alternatingPagesBinary":       map[string]any{"rows": len(pagedBinary), "duplicates": bd, "missing": bm},
			"alternatingPagesICUAndBinary": map[string]any{"rows": len(pagedMixed), "duplicates": md, "missing": mm},
			"binaryRows":                   pb.Rows, "icuRows": pi.Rows,
		})
	c.Rec.AddStage(StageRecord{Stage: 5, Name: "product descriptions in PostgreSQL and SQLite (conditional)", Conditional: true,
		Release: c.Rel2.Release, ReleaseHash: c.Rel2.Digest,
		Assets: []AssetCall{{"reference/multilingual-business-data", "Select, Validate, PrepareTranslation, testdata/schema.sql"}},
		Outputs: map[string]any{
			"precondition": map[string]any{"orderLineItemCodeAttribute": itemCode, "productDescriptionAttributeInWorkspace": description,
				"established": itemCode && description, "mapping": "product:<itemCode> (sample)"},
			"stores":                      map[string]any{"postgresql": pr.Info, "sqlite": sr.Info},
			"selectVersusView":            map[string]any{"postgresql": pr.CompareOK, "sqlite": sr.CompareOK},
			"binaryListHash":              pb.Hash,
			"targetLocalesFromStage4":     []string{"en-US"},
			"expiryShapeFromStage4":       "translation depends on the exact source revision; history is kept",
			"versionSystemIsTextRevision": true,
		}})
	return nil
}
