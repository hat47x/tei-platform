package orderassistoffline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Basis records what a terminal relied on when it produced an operation:
// the price list version of its local master, its realization, and the quote
// it computed locally.
type Basis struct {
	PriceListVersion  int          `json:"priceListVersion"`
	Generation        int          `json:"generation"`
	RealizationDigest string       `json:"realizationDigest"`
	LocalResult       *QuoteResult `json:"localResult,omitempty"`
}

// Op is one terminal operation, the unit of synchronisation.
type Op struct {
	OpID         string          `json:"opId"`
	QuoteID      string          `json:"quoteId"`
	Kind         string          `json:"kind"` // create | set | put-line
	Path         string          `json:"path,omitempty"`
	Value        json.RawMessage `json:"value,omitempty"`
	BaseVersion  int             `json:"baseVersion,omitempty"`
	Line         json.RawMessage `json:"line,omitempty"`
	BaseVersions map[string]int  `json:"baseVersions,omitempty"`
	Doc          json.RawMessage `json:"doc,omitempty"`
	Basis        Basis           `json:"basis"`
}

// FieldOutcome is the result of an operation on one field path.
type FieldOutcome struct {
	Path       string `json:"path"`
	Outcome    string `json:"outcome"` // applied | unchanged | conflict | preserved-unknown | dropped-unknown
	Version    int    `json:"version"`
	ConflictID int64  `json:"conflictId,omitempty"`
}

// OpResult is the server's answer for one operation.
type OpResult struct {
	OpID     string         `json:"opId"`
	Outcome  string         `json:"outcome"` // applied | unchanged | partially-applied | conflict | rejected
	Reason   string         `json:"reason,omitempty"`
	Fields   []FieldOutcome `json:"fields,omitempty"`
	Replayed bool           `json:"replayed,omitempty"`
}

// SyncRequest is one synchronisation from a terminal.
type SyncRequest struct {
	TerminalID        string `json:"terminalId"`
	Generation        int    `json:"generation"`
	RealizationDigest string `json:"realizationDigest"`
	Ops               []Op   `json:"ops"`
}

// QuoteState is the server's current quote.
type QuoteState struct {
	QuoteID       string          `json:"quoteId"`
	Version       int64           `json:"version"`
	Status        string          `json:"status"`
	Digest        string          `json:"digest"`
	Doc           json.RawMessage `json:"doc"`
	FieldVersions map[string]int  `json:"fieldVersions"`
	Revalidation  json.RawMessage `json:"revalidation"`
}

// SyncResponse answers an accepted synchronisation.
type SyncResponse struct {
	Results []OpResult            `json:"results"`
	Quotes  map[string]QuoteState `json:"quotes"`
}

// SyncRefusal answers a synchronisation the server will not accept at all.
// It is meant to be understood by an old terminal: a fixed code and the
// accepted range, no generation-specific members.
type SyncRefusal struct {
	Code               string `json:"code"` // generation-out-of-range | unknown-realization
	TerminalGeneration int    `json:"terminalGeneration"`
	AcceptedMin        int    `json:"acceptedMin"`
	AcceptedMax        int    `json:"acceptedMax"`
}

// Revalidation is what the server found when it re-checked a quote.
type Revalidation struct {
	Seq              int64        `json:"seq"`
	PriceListVersion int          `json:"priceListVersion"`
	Judgment         Judgment     `json:"judgment"`
	StaleLines       []StaleLine  `json:"staleLines"`
	OpenConflicts    int          `json:"openConflicts"`
	TerminalChecks   []LocalCheck `json:"terminalChecks,omitempty"`
}

// StaleLine is a line priced from an older price list than the current one.
type StaleLine struct {
	SKU                  string `json:"sku"`
	QuotedUnitPriceMinor int64  `json:"quotedUnitPriceMinor"`
	BasisVersion         int64  `json:"basisPriceListVersion"`
	CurrentUnitPrice     int64  `json:"currentUnitPriceMinor"`
	CurrentVersion       int    `json:"currentPriceListVersion"`
}

// LocalCheck compares a terminal's local computation with the server's.
type LocalCheck struct {
	OpID                string `json:"opId"`
	TerminalGeneration  int    `json:"terminalGeneration"`
	LocalSubtotalMinor  int64  `json:"localSubtotalMinor"`
	ServerSubtotalMinor int64  `json:"serverSubtotalMinor"`
	Matches             bool   `json:"matches"`
}

// ServerOptions holds the negative controls.
type ServerOptions struct {
	// UnsafeDropUnknownOnPutLine removes members an old terminal did not send
	// back instead of preserving them (what a naive whole-object write does).
	UnsafeDropUnknownOnPutLine bool
}

// Server is the order side of steps 7 and 8.
type Server struct {
	Pool *pgxpool.Pool
	Real *RealizationSet
	Dist Distribution // the server-side sites used for revalidation
	Opts ServerOptions
}

// ErrQuote is a business error from the quote side.
var ErrQuote = errors.New("quote operation refused")

func nextSeq(ctx context.Context, tx pgx.Tx) (int64, error) {
	var seq int64
	err := tx.QueryRow(ctx, `UPDATE oapo.clock SET seq = seq + 1 WHERE id = 1 RETURNING seq`).Scan(&seq)
	return seq, err
}

func recordEvent(ctx context.Context, tx pgx.Tx, kind, quoteID, caseID string, detail any) (int64, error) {
	seq, err := nextSeq(ctx, tx)
	if err != nil {
		return 0, err
	}
	raw, _ := json.Marshal(detail)
	_, err = tx.Exec(ctx, `INSERT INTO oapo.event(seq, kind, quote_id, case_id, detail) VALUES ($1,$2,NULLIF($3,''),NULLIF($4,''),$5)`,
		seq, kind, quoteID, caseID, raw)
	return seq, err
}

// Init writes the clock, the generation range, and the terminal realizations
// the server recognises.
func (s *Server) Init(ctx context.Context, businessDate string, minGen, maxGen int) error {
	return pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO oapo.clock VALUES (1, 0, $1)`, businessDate); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO oapo.accepted_generation VALUES (1, $1, $2)`, minGen, maxGen); err != nil {
			return err
		}
		for _, r := range s.Real.ByName {
			if r.Runtime != RuntimeNodeJS {
				continue
			}
			if _, err := tx.Exec(ctx, `INSERT INTO oapo.terminal_realization VALUES ($1,$2)`, r.Generation, r.ArtifactDigest); err != nil {
				return err
			}
		}
		return nil
	})
}

// SetAcceptedGenerations changes the declared range of terminal generations.
func (s *Server) SetAcceptedGenerations(ctx context.Context, minGen, maxGen int) error {
	_, err := s.Pool.Exec(ctx, `UPDATE oapo.accepted_generation SET min_generation=$1, max_generation=$2 WHERE id=1`, minGen, maxGen)
	return err
}

// PublishPriceList writes a complete price list version.
func (s *Server) PublishPriceList(ctx context.Context, version int, prices map[string]int64) error {
	return pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		seq, err := recordEvent(ctx, tx, "price-list-published", "", "", map[string]any{"version": version, "prices": prices})
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO oapo.price_list VALUES ($1,$2)`, version, seq); err != nil {
			return err
		}
		for sku, p := range prices {
			if _, err := tx.Exec(ctx, `INSERT INTO oapo.price VALUES ($1,$2,$3)`, version, sku, p); err != nil {
				return err
			}
		}
		return nil
	})
}

// Master is a price list snapshot for terminals.
type Master struct {
	Version int              `json:"version"`
	Prices  map[string]int64 `json:"prices"`
}

func currentMaster(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}) (Master, error) {
	m := Master{Prices: map[string]int64{}}
	if err := q.QueryRow(ctx, `SELECT max(version) FROM oapo.price_list`).Scan(&m.Version); err != nil {
		return m, err
	}
	rows, err := q.Query(ctx, `SELECT sku, unit_price_minor FROM oapo.price WHERE version=$1`, m.Version)
	if err != nil {
		return m, err
	}
	defer rows.Close()
	for rows.Next() {
		var sku string
		var p int64
		if err := rows.Scan(&sku, &p); err != nil {
			return m, err
		}
		m.Prices[sku] = p
	}
	return m, rows.Err()
}

// Handler serves the terminal API.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /master", func(w http.ResponseWriter, r *http.Request) {
		m, err := currentMaster(r.Context(), s.Pool)
		if err != nil {
			writeJSON(w, 500, map[string]string{"code": "internal"})
			return
		}
		writeJSON(w, 200, m)
	})
	mux.HandleFunc("GET /quotes/{id}", func(w http.ResponseWriter, r *http.Request) {
		st, err := s.Quote(r.Context(), r.PathValue("id"))
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, 404, map[string]string{"code": "not-found"})
			return
		} else if err != nil {
			writeJSON(w, 500, map[string]string{"code": "internal"})
			return
		}
		writeJSON(w, 200, st)
	})
	mux.HandleFunc("POST /sync", func(w http.ResponseWriter, r *http.Request) {
		var req SyncRequest
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			writeJSON(w, 400, map[string]string{"code": "malformed"})
			return
		}
		resp, refusal, err := s.Sync(r.Context(), req)
		switch {
		case refusal != nil:
			writeJSON(w, 409, refusal)
		case err != nil:
			writeJSON(w, 500, map[string]string{"code": "internal", "detail": err.Error()})
		default:
			writeJSON(w, 200, resp)
		}
	})
	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Quote reads the current quote state.
func (s *Server) Quote(ctx context.Context, quoteID string) (QuoteState, error) {
	var st QuoteState
	var fv []byte
	err := s.Pool.QueryRow(ctx, `SELECT quote_id, version, status, digest, doc, field_versions, revalidation FROM oapo.quote WHERE quote_id=$1`,
		quoteID).Scan(&st.QuoteID, &st.Version, &st.Status, &st.Digest, &st.Doc, &fv, &st.Revalidation)
	if err != nil {
		return st, err
	}
	err = json.Unmarshal(fv, &st.FieldVersions)
	return st, err
}

type lockedQuote struct {
	doc      Doc
	fv       map[string]int
	version  int64
	digest   string
	changed  bool
	checks   []LocalCheck
	creating bool
	createOp string
}

// Sync applies a terminal's operations in order, in one transaction, and
// re-validates every touched quote before committing.
func (s *Server) Sync(ctx context.Context, req SyncRequest) (*SyncResponse, *SyncRefusal, error) {
	var refusal *SyncRefusal
	var resp *SyncResponse
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		var minGen, maxGen int
		if err := tx.QueryRow(ctx, `SELECT min_generation, max_generation FROM oapo.accepted_generation WHERE id=1 FOR SHARE`).Scan(&minGen, &maxGen); err != nil {
			return err
		}
		if req.Generation < minGen || req.Generation > maxGen {
			refusal = &SyncRefusal{Code: "generation-out-of-range", TerminalGeneration: req.Generation, AcceptedMin: minGen, AcceptedMax: maxGen}
			_, err := recordEvent(ctx, tx, "sync-refused", "", "", map[string]any{"terminal": req.TerminalID, "refusal": refusal})
			return err
		}
		var known bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM oapo.terminal_realization WHERE generation=$1 AND artifact_digest=$2)`,
			req.Generation, req.RealizationDigest).Scan(&known); err != nil {
			return err
		}
		if !known {
			refusal = &SyncRefusal{Code: "unknown-realization", TerminalGeneration: req.Generation, AcceptedMin: minGen, AcceptedMax: maxGen}
			_, err := recordEvent(ctx, tx, "sync-refused", "", "", map[string]any{"terminal": req.TerminalID, "refusal": refusal})
			return err
		}
		quotes := map[string]*lockedQuote{}
		resp = &SyncResponse{Quotes: map[string]QuoteState{}}
		for _, op := range req.Ops {
			res, err := s.applyOp(ctx, tx, req, op, quotes)
			if err != nil {
				return err
			}
			resp.Results = append(resp.Results, res)
		}
		ids := make([]string, 0, len(quotes))
		for id := range quotes {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			if err := s.saveAndRevalidate(ctx, tx, id, quotes[id], "sync:"+req.TerminalID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil || refusal != nil {
		return nil, refusal, err
	}
	for _, op := range req.Ops {
		if _, done := resp.Quotes[op.QuoteID]; done {
			continue
		}
		st, err := s.Quote(ctx, op.QuoteID)
		if err == nil {
			resp.Quotes[op.QuoteID] = st
		}
	}
	return resp, nil, nil
}

func (s *Server) lockQuote(ctx context.Context, tx pgx.Tx, id string, quotes map[string]*lockedQuote) (*lockedQuote, error) {
	if q, ok := quotes[id]; ok {
		return q, nil
	}
	var raw, fv []byte
	q := &lockedQuote{}
	err := tx.QueryRow(ctx, `SELECT doc, field_versions, version, digest FROM oapo.quote WHERE quote_id=$1 FOR UPDATE`, id).Scan(&raw, &fv, &q.version, &q.digest)
	if err != nil {
		return nil, err
	}
	if q.doc, err = ParseDoc(raw); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(fv, &q.fv); err != nil {
		return nil, err
	}
	quotes[id] = q
	return q, nil
}

func (s *Server) applyOp(ctx context.Context, tx pgx.Tx, req SyncRequest, op Op, quotes map[string]*lockedQuote) (OpResult, error) {
	res := OpResult{OpID: op.OpID}
	var prior []byte
	err := tx.QueryRow(ctx, `SELECT detail FROM oapo.sync_op WHERE op_id=$1`, op.OpID).Scan(&prior)
	if err == nil {
		if err := json.Unmarshal(prior, &res); err != nil {
			return res, err
		}
		res.Replayed = true
		return res, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return res, err
	}
	if op.OpID == "" || op.QuoteID == "" {
		return res, fmt.Errorf("%w: op without id or quote", ErrQuote)
	}
	conflicts := 0
	applied := 0
	switch op.Kind {
	case "create":
		doc, err := ParseDoc(op.Doc)
		var exists bool
		if err == nil {
			err = tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM oapo.quote WHERE quote_id=$1)`, op.QuoteID).Scan(&exists)
			if err != nil {
				return res, err
			}
		}
		var id string
		_ = json.Unmarshal(doc.Top["quoteId"], &id)
		if err != nil || exists || quotes[op.QuoteID] != nil || id != op.QuoteID {
			res.Outcome, res.Reason = "rejected", "quote exists or document malformed"
			break
		}
		q := &lockedQuote{doc: doc, fv: map[string]int{}, creating: true, createOp: op.OpID, changed: true}
		for _, p := range doc.Paths() {
			q.fv[p] = 1
			res.Fields = append(res.Fields, FieldOutcome{Path: p, Outcome: "applied", Version: 1})
		}
		quotes[op.QuoteID] = q
		res.Outcome = "applied"
	case "set":
		q, err := s.lockQuote(ctx, tx, op.QuoteID, quotes)
		if err != nil {
			return res, err
		}
		if !KnownToGeneration(op.Path, req.Generation) {
			res.Outcome, res.Reason = "rejected", "path not in the terminal's generation"
			break
		}
		fo, err := s.applyField(ctx, tx, q, op, op.Path, op.Value, op.BaseVersion)
		if err != nil {
			return res, err
		}
		res.Fields = append(res.Fields, fo)
	case "put-line":
		q, err := s.lockQuote(ctx, tx, op.QuoteID, quotes)
		if err != nil {
			return res, err
		}
		var line map[string]json.RawMessage
		if err := json.Unmarshal(op.Line, &line); err != nil || lineSKU(line) == "" {
			res.Outcome, res.Reason = "rejected", "line malformed"
			break
		}
		sku := lineSKU(line)
		cur, ok := q.doc.Line(sku)
		if !ok {
			res.Outcome, res.Reason = "rejected", "unknown line"
			break
		}
		members := make([]string, 0, len(line))
		for m := range line {
			if m != "sku" {
				members = append(members, m)
			}
		}
		sort.Strings(members)
		for _, m := range members {
			path := "lines/" + sku + "/" + m
			fo, err := s.applyField(ctx, tx, q, op, path, line[m], op.BaseVersions[m])
			if err != nil {
				return res, err
			}
			res.Fields = append(res.Fields, fo)
		}
		var absent []string
		for m := range cur {
			if _, sent := line[m]; !sent && m != "sku" {
				absent = append(absent, m)
			}
		}
		sort.Strings(absent)
		for _, m := range absent {
			path := "lines/" + sku + "/" + m
			if KnownToGeneration(path, req.Generation) {
				continue // the terminal knows the member and did not send it: no change
			}
			if s.Opts.UnsafeDropUnknownOnPutLine {
				delete(cur, m)
				q.fv[path]++
				q.changed = true
				res.Fields = append(res.Fields, FieldOutcome{Path: path, Outcome: "dropped-unknown", Version: q.fv[path]})
			} else {
				res.Fields = append(res.Fields, FieldOutcome{Path: path, Outcome: "preserved-unknown", Version: q.fv[path]})
			}
		}
	default:
		res.Outcome, res.Reason = "rejected", "unknown operation kind"
	}
	if res.Outcome == "" {
		for _, f := range res.Fields {
			switch f.Outcome {
			case "conflict":
				conflicts++
			case "applied", "dropped-unknown":
				applied++
			}
		}
		switch {
		case conflicts > 0 && applied > 0:
			res.Outcome = "partially-applied"
		case conflicts > 0:
			res.Outcome = "conflict"
		case applied > 0:
			res.Outcome = "applied"
		default:
			res.Outcome = "unchanged"
		}
	}
	if q := quotes[op.QuoteID]; q != nil && op.Basis.LocalResult != nil {
		q.checks = append(q.checks, LocalCheck{OpID: op.OpID, TerminalGeneration: op.Basis.Generation,
			LocalSubtotalMinor: op.Basis.LocalResult.SubtotalMinor})
	}
	seq, err := nextSeq(ctx, tx)
	if err != nil {
		return res, err
	}
	body, _ := json.Marshal(op)
	detail, _ := json.Marshal(res)
	if _, err := tx.Exec(ctx, `INSERT INTO oapo.sync_op VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		op.OpID, req.TerminalID, op.QuoteID, op.Kind, body, req.Generation, req.RealizationDigest,
		op.Basis.PriceListVersion, res.Outcome, detail, seq); err != nil {
		return res, err
	}
	// Conflict rows reference the op, so they are written after it.
	for i, f := range res.Fields {
		if f.Outcome != "conflict" {
			continue
		}
		q := quotes[op.QuoteID]
		serverValue, _ := q.doc.Get(f.Path)
		var opValue json.RawMessage
		base := op.BaseVersion
		if op.Kind == "put-line" {
			var line map[string]json.RawMessage
			_ = json.Unmarshal(op.Line, &line)
			member := f.Path[len("lines/"+lineSKU(line)+"/"):]
			opValue, base = line[member], op.BaseVersions[member]
		} else {
			opValue = op.Value
		}
		var id int64
		if err := tx.QueryRow(ctx, `INSERT INTO oapo.conflict(quote_id, path, op_id, base_version, op_value, server_value, server_version, status)
			VALUES ($1,$2,$3,$4,$5,$6,$7,'open') RETURNING conflict_id`,
			op.QuoteID, f.Path, op.OpID, base, nullableJSON(opValue), nullableJSON(serverValue), q.fv[f.Path]).Scan(&id); err != nil {
			return res, err
		}
		res.Fields[i].ConflictID = id
	}
	if conflicts > 0 {
		detail, _ = json.Marshal(res)
		if _, err := tx.Exec(ctx, `UPDATE oapo.sync_op SET detail=$2 WHERE op_id=$1`, op.OpID, detail); err != nil {
			return res, err
		}
	}
	return res, nil
}

func nullableJSON(v json.RawMessage) any {
	if v == nil {
		return nil
	}
	return []byte(v)
}

// applyField applies one field write with optimistic concurrency on the
// field's version. A write based on an older version than the current one
// becomes a conflict and is kept, never silently dropped or overwritten.
func (s *Server) applyField(ctx context.Context, tx pgx.Tx, q *lockedQuote, op Op, path string, value json.RawMessage, base int) (FieldOutcome, error) {
	cur, _ := q.doc.Get(path)
	ver := q.fv[path]
	if EqualJSON(cur, value) {
		return FieldOutcome{Path: path, Outcome: "unchanged", Version: ver}, nil
	}
	if base != ver {
		return FieldOutcome{Path: path, Outcome: "conflict", Version: ver}, nil
	}
	if err := q.doc.Set(path, value); err != nil {
		return FieldOutcome{}, fmt.Errorf("%w: %v", ErrQuote, err)
	}
	q.fv[path] = ver + 1
	q.changed = true
	return FieldOutcome{Path: path, Outcome: "applied", Version: ver + 1}, nil
}

// saveAndRevalidate writes the document and re-checks it: the current price
// list, the rule judgment at the server-side sites, and open conflicts. A
// changed digest invalidates approvals of open cases (P-WF-01).
func (s *Server) saveAndRevalidate(ctx context.Context, tx pgx.Tx, quoteID string, q *lockedQuote, cause string) error {
	master, err := currentMaster(ctx, tx)
	if err != nil {
		return err
	}
	seq, err := nextSeq(ctx, tx)
	if err != nil {
		return err
	}
	rv := Revalidation{Seq: seq, PriceListVersion: master.Version, StaleLines: []StaleLine{}}
	for _, l := range q.doc.Lines {
		sku := lineSKU(l)
		price, _ := intValue(l["unitPriceMinor"])
		basis, _ := intValue(l["priceListVersion"])
		current, ok := master.Prices[sku]
		if !ok || current != price {
			rv.StaleLines = append(rv.StaleLines, StaleLine{SKU: sku, QuotedUnitPriceMinor: price, BasisVersion: basis,
				CurrentUnitPrice: current, CurrentVersion: master.Version})
		}
	}
	if !q.creating {
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM oapo.conflict WHERE quote_id=$1 AND status='open'`, quoteID).Scan(&rv.OpenConflicts); err != nil {
			return err
		}
	}
	payload, err := q.doc.QuoteRequest(s.Dist.TargetGeneration)
	if err != nil {
		return err
	}
	status := "valid"
	rv.Judgment, err = s.Real.Judge(ctx, s.Dist, fmt.Sprintf("reval-%s-%d", quoteID, seq), payload, JudgeOptions{})
	if errors.Is(err, ErrDivergence) || errors.Is(err, ErrNoEligibleSite) {
		status = "divergent"
	} else if err != nil {
		return err
	}
	for i := range q.checks {
		q.checks[i].ServerSubtotalMinor = rv.Judgment.Result.SubtotalMinor
		q.checks[i].Matches = q.checks[i].LocalSubtotalMinor == rv.Judgment.Result.SubtotalMinor
	}
	rv.TerminalChecks = q.checks
	switch {
	case status == "divergent":
	case rv.OpenConflicts > 0:
		status = "conflict"
	case rv.Judgment.Rejected:
		status = "divergent"
	case len(rv.StaleLines) > 0:
		status = "stale-price"
	}
	digest := QuoteDigest(q.doc, rv.Judgment, master.Version)
	rvRaw, _ := json.Marshal(rv)
	fvRaw, _ := json.Marshal(q.fv)
	if q.creating {
		_, err = tx.Exec(ctx, `INSERT INTO oapo.quote VALUES ($1,$2,$3,1,$4,$5,$6,$7,$8)`,
			quoteID, q.doc.Marshal(), fvRaw, status, digest, rvRaw, q.createOp, seq)
	} else {
		_, err = tx.Exec(ctx, `UPDATE oapo.quote SET doc=$2, field_versions=$3, version=version+1, status=$4, digest=$5, revalidation=$6, updated_seq=$7 WHERE quote_id=$1`,
			quoteID, q.doc.Marshal(), fvRaw, status, digest, rvRaw, seq)
	}
	if err != nil {
		return err
	}
	if _, err := recordEvent(ctx, tx, "quote-revalidated", quoteID, "", map[string]any{"cause": cause, "status": status,
		"digest": digest, "previousDigest": q.digest, "staleLines": len(rv.StaleLines), "openConflicts": rv.OpenConflicts}); err != nil {
		return err
	}
	if !q.creating && digest != q.digest {
		return invalidateApprovalsForQuote(ctx, tx, quoteID, digest)
	}
	return nil
}

// ResolveConflict applies a person's choice for an open field conflict. Both
// candidate values stay in the resolution record.
func (s *Server) ResolveConflict(ctx context.Context, conflictID int64, chosen json.RawMessage, actor string) error {
	return pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		var quoteID, path, status string
		var opValue, serverValue []byte
		if err := tx.QueryRow(ctx, `SELECT quote_id, path, status, op_value, server_value FROM oapo.conflict WHERE conflict_id=$1`, conflictID).
			Scan(&quoteID, &path, &status, &opValue, &serverValue); err != nil {
			return err
		}
		if status != "open" {
			return fmt.Errorf("%w: conflict %d is %s", ErrQuote, conflictID, status)
		}
		quotes := map[string]*lockedQuote{}
		q, err := s.lockQuote(ctx, tx, quoteID, quotes)
		if err != nil {
			return err
		}
		cur, _ := q.doc.Get(path)
		if !EqualJSON(cur, chosen) {
			if err := q.doc.Set(path, chosen); err != nil {
				return err
			}
			q.fv[path]++
		}
		q.changed = true
		res, _ := json.Marshal(map[string]any{"actor": actor, "chosen": chosen,
			"candidates": map[string]json.RawMessage{"server": serverValue, "terminal": opValue}})
		if _, err := tx.Exec(ctx, `UPDATE oapo.conflict SET status='resolved', resolution=$2 WHERE conflict_id=$1`, conflictID, res); err != nil {
			return err
		}
		return s.saveAndRevalidate(ctx, tx, quoteID, q, "resolve:"+actor)
	})
}

// Reprice re-prices stale lines from the current price list. It is a server
// operation by a person; the basis of each repriced line is recorded.
func (s *Server) Reprice(ctx context.Context, quoteID, actor string) error {
	return pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		quotes := map[string]*lockedQuote{}
		q, err := s.lockQuote(ctx, tx, quoteID, quotes)
		if err != nil {
			return err
		}
		master, err := currentMaster(ctx, tx)
		if err != nil {
			return err
		}
		var repriced []string
		for _, l := range q.doc.Lines {
			sku := lineSKU(l)
			price, _ := intValue(l["unitPriceMinor"])
			if cur, ok := master.Prices[sku]; ok && cur != price {
				for path, v := range map[string]int64{"lines/" + sku + "/unitPriceMinor": cur, "lines/" + sku + "/priceListVersion": int64(master.Version)} {
					if err := q.doc.Set(path, rawInt(v)); err != nil {
						return err
					}
					q.fv[path]++
				}
				repriced = append(repriced, sku)
			}
		}
		sort.Strings(repriced)
		q.changed = true
		if _, err := recordEvent(ctx, tx, "quote-repriced", quoteID, "", map[string]any{"actor": actor, "lines": repriced, "priceListVersion": master.Version}); err != nil {
			return err
		}
		return s.saveAndRevalidate(ctx, tx, quoteID, q, "reprice:"+actor)
	})
}

// Conflicts lists the conflicts of a quote.
type Conflict struct {
	ID            int64           `json:"conflictId"`
	Path          string          `json:"path"`
	OpID          string          `json:"opId"`
	BaseVersion   int             `json:"baseVersion"`
	OpValue       json.RawMessage `json:"opValue"`
	ServerValue   json.RawMessage `json:"serverValue"`
	ServerVersion int             `json:"serverVersion"`
	Status        string          `json:"status"`
}

func (s *Server) Conflicts(ctx context.Context, quoteID string) ([]Conflict, error) {
	rows, err := s.Pool.Query(ctx, `SELECT conflict_id, path, op_id, base_version, op_value, server_value, server_version, status
		FROM oapo.conflict WHERE quote_id=$1 ORDER BY conflict_id`, quoteID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Conflict, error) {
		var c Conflict
		var ov, sv []byte
		err := r.Scan(&c.ID, &c.Path, &c.OpID, &c.BaseVersion, &ov, &sv, &c.ServerVersion, &c.Status)
		c.OpValue, c.ServerValue = ov, sv
		return c, err
	})
}

// PublishRevisedPrices copies the current list with changes as a new version.
func (s *Server) PublishRevisedPrices(ctx context.Context, changes map[string]int64) (int, error) {
	m, err := currentMaster(ctx, s.Pool)
	if err != nil {
		return 0, err
	}
	for k, v := range changes {
		m.Prices[k] = v
	}
	return m.Version + 1, s.PublishPriceList(ctx, m.Version+1, m.Prices)
}
