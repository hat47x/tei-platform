package orderassistoffline

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"sort"
	"strings"

	offlinestorage "tei/tools/reference-offline-storage"
)

// Terminal is a simulated sales terminal. Its state lives in a local store
// (tools/reference-offline-storage, one root per terminal, single-user
// lock): the price master it downloaded, the quotes it holds as raw JSON
// with the field versions it last saw, and an outbox of operations not yet
// acknowledged by the server. It computes quotes locally with its own
// JavaScript realization.
type Terminal struct {
	ID          string
	Generation  int
	Realization Realization
	Store       *offlinestorage.Store
	ServerURL   string
	HTTP        *http.Client
	// NaiveDecode makes the terminal decode documents into the members its
	// generation knows and drop the rest (negative control for P-MIX-03).
	NaiveDecode bool
}

type terminalMeta struct {
	NextSeq int `json:"nextSeq"`
}

// ErrSyncRefused wraps a server refusal the terminal understood.
type ErrSyncRefused struct{ Refusal SyncRefusal }

func (e *ErrSyncRefused) Error() string {
	return fmt.Sprintf("sync refused: %s (terminal generation %d, accepted %d..%d)",
		e.Refusal.Code, e.Refusal.TerminalGeneration, e.Refusal.AcceptedMin, e.Refusal.AcceptedMax)
}

func (t *Terminal) getJSON(key string, v any) error {
	raw, err := t.Store.Get(key)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, v)
}

func (t *Terminal) putJSON(key string, v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return t.Store.Put(key, raw)
}

func (t *Terminal) call(ctx context.Context, method, path string, body any, out any) (int, error) {
	var rd io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		rd = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, t.ServerURL+path, rd)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := t.HTTP.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, err
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return resp.StatusCode, fmt.Errorf("decode %s %s (%d): %w", method, path, resp.StatusCode, err)
		}
	}
	return resp.StatusCode, nil
}

// DownloadMaster stores the server's current price list (online).
func (t *Terminal) DownloadMaster(ctx context.Context) (Master, error) {
	var m Master
	if code, err := t.call(ctx, http.MethodGet, "/master", nil, &m); err != nil || code != 200 {
		return m, fmt.Errorf("download master: %d %v", code, err)
	}
	return m, t.putJSON("master.json", m)
}

// Master reads the local price master.
func (t *Terminal) Master() (Master, error) {
	var m Master
	return m, t.getJSON("master.json", &m)
}

func quoteKey(id, part string) string { return "quotes/" + id + "/" + part + ".json" }

// storeDoc keeps a document. A naive terminal re-encodes it through the
// members of its generation and loses the others.
func (t *Terminal) storeDoc(id string, raw []byte) error {
	if t.NaiveDecode {
		d, err := ParseDoc(raw)
		if err != nil {
			return err
		}
		raw = naiveDoc(d, t.Generation).Marshal()
	}
	return t.Store.Put(quoteKey(id, "doc"), raw)
}

func naiveDoc(d Doc, gen int) Doc {
	out := Doc{Top: map[string]json.RawMessage{}}
	for k, v := range d.Top {
		if g, ok := topMembers[k]; ok && g <= gen {
			out.Top[k] = v
		}
	}
	for _, l := range d.Lines {
		nl := map[string]json.RawMessage{}
		for k, v := range l {
			if g, ok := lineMembers[k]; ok && g <= gen {
				nl[k] = v
			}
		}
		out.Lines = append(out.Lines, nl)
	}
	return out
}

// DownloadQuote stores the server's current quote and field versions (online).
func (t *Terminal) DownloadQuote(ctx context.Context, id string) (QuoteState, error) {
	var st QuoteState
	if code, err := t.call(ctx, http.MethodGet, "/quotes/"+id, nil, &st); err != nil || code != 200 {
		return st, fmt.Errorf("download quote %s: %d %v", id, code, err)
	}
	if err := t.storeDoc(id, st.Doc); err != nil {
		return st, err
	}
	return st, t.putJSON(quoteKey(id, "versions"), st.FieldVersions)
}

// LocalDoc reads the terminal's copy of a quote.
func (t *Terminal) LocalDoc(id string) (Doc, error) {
	raw, err := t.Store.Get(quoteKey(id, "doc"))
	if err != nil {
		return Doc{}, err
	}
	return ParseDoc(raw)
}

func (t *Terminal) versions(id string) map[string]int {
	v := map[string]int{}
	_ = t.getJSON(quoteKey(id, "versions"), &v)
	return v
}

// compute runs the terminal's realization on the local document.
func (t *Terminal) compute(ctx context.Context, d Doc, opID string) (*QuoteResult, error) {
	payload, err := d.QuoteRequest(t.Generation)
	if err != nil {
		return nil, err
	}
	out, err := t.Realization.Quote(ctx, Generations[t.Generation], "terminal-"+opID, payload)
	if err != nil {
		return nil, err
	}
	if out.Rejected {
		return nil, fmt.Errorf("terminal realization rejected the quote")
	}
	return &out.Result, nil
}

func (t *Terminal) enqueue(ctx context.Context, op Op, d Doc) (Op, error) {
	var meta terminalMeta
	if err := t.getJSON("meta.json", &meta); err != nil && !isNotExist(err) {
		return op, err
	}
	meta.NextSeq++
	op.OpID = fmt.Sprintf("%s-%06d", t.ID, meta.NextSeq)
	m, err := t.Master()
	if err != nil {
		return op, fmt.Errorf("terminal has no price master: %w", err)
	}
	local, err := t.compute(ctx, d, op.OpID)
	if err != nil {
		return op, err
	}
	op.Basis = Basis{PriceListVersion: m.Version, Generation: t.Generation, RealizationDigest: t.Realization.ArtifactDigest, LocalResult: local}
	if err := t.storeDoc(op.QuoteID, d.Marshal()); err != nil {
		return op, err
	}
	if err := t.putJSON(fmt.Sprintf("outbox/%06d.json", meta.NextSeq), op); err != nil {
		return op, err
	}
	return op, t.putJSON("meta.json", meta)
}

func isNotExist(err error) bool { return errors.Is(err, fs.ErrNotExist) }

// NewLine is a line entered on the terminal; the price comes from the local master.
type NewLine struct {
	SKU       string
	Quantity  int64
	Packaging string // generation 2 only
}

// CreateQuote creates a quote offline from the local master.
func (t *Terminal) CreateQuote(ctx context.Context, id, customer string, discountBp int64, lines []NewLine) (Op, error) {
	m, err := t.Master()
	if err != nil {
		return Op{}, err
	}
	d := Doc{Top: map[string]json.RawMessage{}}
	d.Top["quoteId"], _ = json.Marshal(id)
	d.Top["customerCode"], _ = json.Marshal(customer)
	d.Top["currency"], _ = json.Marshal("JPY")
	if t.Generation >= 2 {
		d.Top["discountRateBp"] = rawInt(discountBp)
	}
	for _, nl := range lines {
		price, ok := m.Prices[nl.SKU]
		if !ok {
			return Op{}, fmt.Errorf("SKU %s not in the local master", nl.SKU)
		}
		l := map[string]json.RawMessage{"quantity": rawInt(nl.Quantity), "unitPriceMinor": rawInt(price), "priceListVersion": rawInt(int64(m.Version))}
		l["sku"], _ = json.Marshal(nl.SKU)
		if nl.Packaging != "" && t.Generation >= 2 {
			l["packaging"], _ = json.Marshal(nl.Packaging)
		}
		d.Lines = append(d.Lines, l)
	}
	op := Op{QuoteID: id, Kind: "create", Doc: d.Marshal()}
	return t.enqueue(ctx, op, d)
}

// SetField edits one field offline (generation 2 terminals send field
// operations).
func (t *Terminal) SetField(ctx context.Context, id, path string, value json.RawMessage) (Op, error) {
	d, err := t.LocalDoc(id)
	if err != nil {
		return Op{}, err
	}
	if err := d.Set(path, value); err != nil {
		return Op{}, err
	}
	op := Op{QuoteID: id, Kind: "set", Path: path, Value: value, BaseVersion: t.versions(id)[path]}
	return t.enqueue(ctx, op, d)
}

// EditLineQuantity edits a line's quantity offline the way a generation-1
// terminal does: it writes the whole line object back. A preserving terminal
// sends the line with every member it holds, known or not.
func (t *Terminal) EditLineQuantity(ctx context.Context, id, sku string, qty int64) (Op, error) {
	d, err := t.LocalDoc(id)
	if err != nil {
		return Op{}, err
	}
	l, ok := d.Line(sku)
	if !ok {
		return Op{}, fmt.Errorf("no line %s", sku)
	}
	l["quantity"] = rawInt(qty)
	line, _ := json.Marshal(l)
	vers := t.versions(id)
	base := map[string]int{}
	for m := range l {
		if m != "sku" {
			base[m] = vers["lines/"+sku+"/"+m]
		}
	}
	op := Op{QuoteID: id, Kind: "put-line", Line: line, BaseVersions: base}
	return t.enqueue(ctx, op, d)
}

// Outbox lists unacknowledged operations in order.
func (t *Terminal) Outbox() ([]Op, error) {
	keys, err := t.Store.List()
	if err != nil {
		return nil, err
	}
	var ops []Op
	sort.Strings(keys)
	for _, k := range keys {
		if !strings.HasPrefix(k, "outbox/") {
			continue
		}
		var op Op
		if err := t.getJSON(k, &op); err != nil {
			return nil, err
		}
		ops = append(ops, op)
	}
	return ops, nil
}

// Sync sends the outbox (online). Acknowledged operations leave the outbox
// and their results are kept; a refusal leaves the outbox untouched.
func (t *Terminal) Sync(ctx context.Context) (*SyncResponse, error) {
	ops, err := t.Outbox()
	if err != nil {
		return nil, err
	}
	req := SyncRequest{TerminalID: t.ID, Generation: t.Generation, RealizationDigest: t.Realization.ArtifactDigest, Ops: ops}
	raw, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, t.ServerURL+"/sync", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	resp, err := t.HTTP.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case 200:
	case 409:
		// Decode only the members every generation understands.
		var r SyncRefusal
		if err := json.Unmarshal(body, &r); err != nil || r.Code == "" {
			return nil, fmt.Errorf("sync: unreadable refusal: %s", body)
		}
		return nil, &ErrSyncRefused{Refusal: r}
	default:
		return nil, fmt.Errorf("sync: status %d: %s", resp.StatusCode, body)
	}
	var out SyncResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	acked := map[string]bool{}
	for _, r := range out.Results {
		acked[r.OpID] = true
		if err := t.putJSON("results/"+r.OpID+".json", r); err != nil {
			return nil, err
		}
	}
	keys, _ := t.Store.List()
	for _, k := range keys {
		if !strings.HasPrefix(k, "outbox/") {
			continue
		}
		var op Op
		if err := t.getJSON(k, &op); err == nil && acked[op.OpID] {
			if err := t.Store.Delete(k); err != nil {
				return nil, err
			}
		}
	}
	for id, st := range out.Quotes {
		if err := t.storeDoc(id, st.Doc); err != nil {
			return nil, err
		}
		if err := t.putJSON(quoteKey(id, "versions"), st.FieldVersions); err != nil {
			return nil, err
		}
	}
	return &out, nil
}
