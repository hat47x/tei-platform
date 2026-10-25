package monthendclose

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"sort"

	"github.com/jackc/pgx/v5"
)

// TaxRule declares the rounding of consumption tax. Only one rule is
// implemented: tax is computed once per invoice (customer and posting
// month) and per rate, from the sum of the line amounts kept in hundredths
// of a yen, and truncated toward zero to whole yen. Truncation toward zero
// makes the tax of a red slip the exact negation of the tax of the slip it
// reverses. The invoice tax is allocated to its lines in proportion,
// truncated toward zero, and the remainder goes to the line with the
// largest absolute amount (the smallest slip id among equals).
type TaxRule struct {
	ID         string `json:"id"`
	Unit       string `json:"unit"`
	InvoiceKey string `json:"invoiceKey"`
	Method     string `json:"method"`
	LineScale  int    `json:"lineAmountScale"`
	TaxScale   int    `json:"taxScale"`
	Allocation string `json:"allocation"`
	Remainder  string `json:"remainderTo"`
}

// DefaultTaxRule is the only supported rule. The close uses it.
var DefaultTaxRule = TaxRule{ID: "invoice-per-rate-truncate-v1", Unit: "invoice-per-rate", InvoiceKey: "customer+posting-month", Method: "truncate-toward-zero",
	LineScale: 2, TaxScale: 0, Allocation: "proportional-truncate-toward-zero", Remainder: "largest-absolute-amount-then-smallest-slip-id"}

// ErrTaxPathsDisagree is a difference between the computation paths.
var ErrTaxPathsDisagree = errors.New("tax computation paths disagree")

// ReadTaxRule reads a rule and accepts it only if it declares exactly the
// implemented rule.
func ReadTaxRule(path string) (TaxRule, error) {
	var r TaxRule
	if err := readJSON(path, &r); err != nil {
		return r, err
	}
	if r != DefaultTaxRule {
		return r, fmt.Errorf("tax rule %+v is not the implemented rule %+v", r, DefaultTaxRule)
	}
	return r, nil
}

// TaxLine is one slip as the tax computation sees it.
type TaxLine struct {
	SlipID       int64  `json:"slipId"`
	Customer     string `json:"customer"`
	PostingMonth string `json:"postingMonth"`
	Rate         int    `json:"rate"`
	AmountSen    int64  `json:"amountSen"`
}

// InvoiceTax is the tax of one invoice and rate.
type InvoiceTax struct {
	Customer     string `json:"customer"`
	PostingMonth string `json:"postingMonth"`
	Rate         int    `json:"rate"`
	Lines        int    `json:"lines"`
	BaseSen      int64  `json:"baseSen"`
	TaxYen       int64  `json:"taxYen"`
}

// TaxResult is the result of one computation path.
type TaxResult struct {
	Invoices   []InvoiceTax    `json:"invoices"`
	Allocation map[int64]int64 `json:"-"`
}

func sortInvoices(v []InvoiceTax) {
	sort.Slice(v, func(i, j int) bool {
		a, b := v[i], v[j]
		if a.Customer != b.Customer {
			return a.Customer < b.Customer
		}
		if a.PostingMonth != b.PostingMonth {
			return a.PostingMonth < b.PostingMonth
		}
		return a.Rate < b.Rate
	})
}

type invKey struct {
	customer, month string
	rate            int
}

// GoTax is the application path: integer hundredths, Go's integer division
// (which truncates toward zero).
func GoTax(lines []TaxLine) TaxResult {
	inv := map[invKey]*InvoiceTax{}
	members := map[invKey][]TaxLine{}
	for _, l := range lines {
		k := invKey{l.Customer, l.PostingMonth, l.Rate}
		if inv[k] == nil {
			inv[k] = &InvoiceTax{Customer: l.Customer, PostingMonth: l.PostingMonth, Rate: l.Rate}
		}
		inv[k].Lines++
		inv[k].BaseSen += l.AmountSen
		members[k] = append(members[k], l)
	}
	res := TaxResult{Allocation: map[int64]int64{}}
	for k, v := range inv {
		v.TaxYen = v.BaseSen * int64(v.Rate) / 10000
		res.Invoices = append(res.Invoices, *v)
		ms := members[k]
		sort.Slice(ms, func(i, j int) bool {
			ai, aj := abs64(ms[i].AmountSen), abs64(ms[j].AmountSen)
			if ai != aj {
				return ai > aj
			}
			return ms[i].SlipID < ms[j].SlipID
		})
		var sum int64
		for _, m := range ms {
			share := int64(0)
			if v.BaseSen != 0 {
				share = v.TaxYen * m.AmountSen / v.BaseSen
			}
			res.Allocation[m.SlipID] = share
			sum += share
		}
		res.Allocation[ms[0].SlipID] += v.TaxYen - sum
	}
	sortInvoices(res.Invoices)
	return res
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// LoadTaxLines reads the slips posted to the months.
func LoadTaxLines(ctx context.Context, q pgx.Tx, months []string) ([]TaxLine, error) {
	rows, err := q.Query(ctx, `SELECT slip_id, customer, posting_month, tax_rate, (amount*100)::bigint FROM mecc.slip WHERE posting_month = ANY($1) ORDER BY slip_id`, months)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TaxLine
	for rows.Next() {
		var l TaxLine
		if err := rows.Scan(&l.SlipID, &l.Customer, &l.PostingMonth, &l.Rate, &l.AmountSen); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

const pgInvoiceSQL = `
WITH s AS (SELECT slip_id, customer, posting_month, tax_rate, amount FROM mecc.slip WHERE posting_month = ANY($1)),
inv AS (SELECT customer, posting_month, tax_rate, count(*) AS n, sum(amount) AS base, div(sum(amount) * tax_rate, 100) AS tax
        FROM s GROUP BY customer, posting_month, tax_rate)
SELECT customer, posting_month, tax_rate, n, (base * 100)::bigint, tax::bigint FROM inv`

const pgAllocationSQL = `
WITH s AS (SELECT slip_id, customer, posting_month, tax_rate, amount FROM mecc.slip WHERE posting_month = ANY($1)),
inv AS (SELECT customer, posting_month, tax_rate, sum(amount) AS base, div(sum(amount) * tax_rate, 100) AS tax
        FROM s GROUP BY customer, posting_month, tax_rate),
sh AS (SELECT s.slip_id, s.customer, s.posting_month, s.tax_rate, i.tax,
              CASE WHEN i.base = 0 THEN 0 ELSE div(i.tax * s.amount, i.base) END AS share,
              row_number() OVER (PARTITION BY s.customer, s.posting_month, s.tax_rate ORDER BY abs(s.amount) DESC, s.slip_id) AS rk
       FROM s JOIN inv i USING (customer, posting_month, tax_rate))
SELECT slip_id, (share + CASE WHEN rk = 1 THEN tax - sum(share) OVER (PARTITION BY customer, posting_month, tax_rate) ELSE 0 END)::bigint FROM sh`

// PgTax is the PostgreSQL path: numeric amounts in yen with two places and
// div(), which truncates toward zero.
func PgTax(ctx context.Context, q pgx.Tx, months []string) (TaxResult, error) {
	res := TaxResult{Allocation: map[int64]int64{}}
	rows, err := q.Query(ctx, pgInvoiceSQL, months)
	if err != nil {
		return res, err
	}
	for rows.Next() {
		var v InvoiceTax
		if err := rows.Scan(&v.Customer, &v.PostingMonth, &v.Rate, &v.Lines, &v.BaseSen, &v.TaxYen); err != nil {
			rows.Close()
			return res, err
		}
		res.Invoices = append(res.Invoices, v)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return res, err
	}
	rows, err = q.Query(ctx, pgAllocationSQL, months)
	if err != nil {
		return res, err
	}
	for rows.Next() {
		var id, yen int64
		if err := rows.Scan(&id, &yen); err != nil {
			rows.Close()
			return res, err
		}
		res.Allocation[id] = yen
	}
	rows.Close()
	sortInvoices(res.Invoices)
	return res, rows.Err()
}

// SQLite has no decimal type: the SQLite path declares amounts as INTEGER
// hundredths in a STRICT table and uses integer division, which truncates
// toward zero. amount_real is the naive REAL column, used only to show the
// floating-point trap.
var sqliteSetup = []string{
	`CREATE TABLE slip (slip_id INTEGER PRIMARY KEY, customer TEXT NOT NULL, posting_month TEXT NOT NULL, tax_rate INTEGER NOT NULL, amount_sen INTEGER NOT NULL, amount_real REAL NOT NULL) STRICT`,
}

const sqliteInsert = `INSERT INTO slip VALUES (?,?,?,?,?,?)`

var sqliteQueries = map[string]string{
	"invoices": `SELECT customer, posting_month, tax_rate, count(*), sum(amount_sen), (sum(amount_sen) * tax_rate) / 10000 FROM slip GROUP BY customer, posting_month, tax_rate`,
	"allocation": `WITH inv AS (SELECT customer, posting_month, tax_rate, sum(amount_sen) AS base, (sum(amount_sen) * tax_rate) / 10000 AS tax FROM slip GROUP BY customer, posting_month, tax_rate),
sh AS (SELECT s.slip_id, s.customer, s.posting_month, s.tax_rate, i.tax,
              CASE WHEN i.base = 0 THEN 0 ELSE (i.tax * s.amount_sen) / i.base END AS share,
              row_number() OVER (PARTITION BY s.customer, s.posting_month, s.tax_rate ORDER BY abs(s.amount_sen) DESC, s.slip_id) AS rk
       FROM slip s JOIN inv i ON i.customer = s.customer AND i.posting_month = s.posting_month AND i.tax_rate = s.tax_rate)
SELECT slip_id, share + CASE WHEN rk = 1 THEN tax - sum(share) OVER (PARTITION BY customer, posting_month, tax_rate) ELSE 0 END FROM sh`,
	"naiveReal": `SELECT customer, posting_month, tax_rate, CAST(sum(amount_real) * tax_rate / 100 AS INTEGER) FROM slip GROUP BY customer, posting_month, tax_rate`,
}

// sqliteRunner runs SQL handed over on stdin in an in-memory SQLite
// database through Python's standard sqlite3 module. All SQL text is the
// Go side's; the script only executes it.
const sqliteRunner = `import json, sqlite3, sys
req = json.load(sys.stdin)
db = sqlite3.connect(":memory:")
for s in req["setup"]:
    db.execute(s)
db.executemany(req["insert"], req["rows"])
out = {"sqliteVersion": sqlite3.sqlite_version, "results": {}}
for name, sql in req["queries"].items():
    out["results"][name] = [list(r) for r in db.execute(sql)]
json.dump(out, sys.stdout)
`

// PythonEnv overrides the Python interpreter that runs the SQLite path.
const PythonEnv = "TEI_MECC_PYTHON"

// SQLiteResult is the SQLite path with its engine version and the naive
// REAL computation.
type SQLiteResult struct {
	TaxResult
	Version   string       `json:"sqliteVersion"`
	NaiveReal []InvoiceTax `json:"naiveReal"`
}

// SQLiteTax computes the tax in SQLite. SQLite runs in a separate process
// (python3 -I with the standard sqlite3 module) because the module has no
// SQLite driver and adding one is a dependency decision.
func SQLiteTax(ctx context.Context, lines []TaxLine) (SQLiteResult, error) {
	py := os.Getenv(PythonEnv)
	if py == "" {
		var err error
		if py, err = exec.LookPath("python3"); err != nil {
			return SQLiteResult{}, fmt.Errorf("SQLite path needs python3 with the sqlite3 module (or %s): %w", PythonEnv, err)
		}
	}
	rows := make([][]any, 0, len(lines))
	for _, l := range lines {
		rows = append(rows, []any{l.SlipID, l.Customer, l.PostingMonth, l.Rate, l.AmountSen, float64(l.AmountSen) / 100})
	}
	req, _ := json.Marshal(map[string]any{"setup": sqliteSetup, "insert": sqliteInsert, "rows": rows, "queries": sqliteQueries})
	cmd := exec.CommandContext(ctx, py, "-I", "-c", sqliteRunner)
	cmd.Stdin = bytes.NewReader(req)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return SQLiteResult{}, fmt.Errorf("SQLite path: %v: %s", err, stderr.String())
	}
	var resp struct {
		Version string             `json:"sqliteVersion"`
		Results map[string][][]any `json:"results"`
	}
	dec := json.NewDecoder(bytes.NewReader(out))
	dec.UseNumber()
	if err := dec.Decode(&resp); err != nil {
		return SQLiteResult{}, err
	}
	res := SQLiteResult{Version: resp.Version, TaxResult: TaxResult{Allocation: map[int64]int64{}}}
	num := func(v any) int64 {
		n, _ := v.(json.Number).Int64()
		return n
	}
	for _, r := range resp.Results["invoices"] {
		res.Invoices = append(res.Invoices, InvoiceTax{Customer: r[0].(string), PostingMonth: r[1].(string), Rate: int(num(r[2])), Lines: int(num(r[3])), BaseSen: num(r[4]), TaxYen: num(r[5])})
	}
	for _, r := range resp.Results["allocation"] {
		res.Allocation[num(r[0])] = num(r[1])
	}
	for _, r := range resp.Results["naiveReal"] {
		res.NaiveReal = append(res.NaiveReal, InvoiceTax{Customer: r[0].(string), PostingMonth: r[1].(string), Rate: int(num(r[2])), TaxYen: num(r[3])})
	}
	sortInvoices(res.Invoices)
	sortInvoices(res.NaiveReal)
	return res, nil
}

// TaxCheck compares the paths over the slips of the months.
type TaxCheck struct {
	Rule          string       `json:"rule"`
	Months        []string     `json:"months"`
	Lines         int          `json:"lines"`
	Invoices      []InvoiceTax `json:"invoices"`
	TaxTotal      int64        `json:"taxTotal"`
	SQLiteVersion string       `json:"sqliteVersion"`
	Agree         bool         `json:"agree"`
	Differences   []string     `json:"differences,omitempty"`
	// Naive designs, observed only.
	PerLineRoundingDiffers int `json:"perLineRoundingDiffers"`
	FloorDiffers           int `json:"floorDiffers"`
	SQLiteRealDiffers      int `json:"sqliteRealDiffers"`
	GoFloatDiffers         int `json:"goFloatDiffers"`
}

// CheckTax computes the tax by PostgreSQL, Go and SQLite and compares the
// invoices and the allocation to lines.
func CheckTax(ctx context.Context, conn *pgx.Conn, rule TaxRule, months []string) (TaxCheck, error) {
	c := TaxCheck{Rule: rule.ID, Months: months}
	if rule != DefaultTaxRule {
		return c, fmt.Errorf("tax rule %s is not implemented", rule.ID)
	}
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return c, err
	}
	defer tx.Rollback(ctx)
	lines, err := LoadTaxLines(ctx, tx, months)
	if err != nil {
		return c, err
	}
	pg, err := PgTax(ctx, tx, months)
	if err != nil {
		return c, err
	}
	gr := GoTax(lines)
	sq, err := SQLiteTax(ctx, lines)
	if err != nil {
		return c, err
	}
	c.Lines, c.Invoices, c.SQLiteVersion = len(lines), pg.Invoices, sq.Version
	for _, v := range pg.Invoices {
		c.TaxTotal += v.TaxYen
	}
	cmpInv := func(name string, other []InvoiceTax) {
		if len(other) != len(pg.Invoices) {
			c.Differences = append(c.Differences, fmt.Sprintf("%s: %d invoices, PostgreSQL %d", name, len(other), len(pg.Invoices)))
			return
		}
		for i := range other {
			if other[i] != pg.Invoices[i] {
				c.Differences = append(c.Differences, fmt.Sprintf("%s: %+v, PostgreSQL %+v", name, other[i], pg.Invoices[i]))
			}
		}
	}
	cmpAlloc := func(name string, other map[int64]int64) {
		if len(other) != len(pg.Allocation) {
			c.Differences = append(c.Differences, fmt.Sprintf("%s: %d allocations, PostgreSQL %d", name, len(other), len(pg.Allocation)))
		}
		for id, v := range pg.Allocation {
			if other[id] != v {
				c.Differences = append(c.Differences, fmt.Sprintf("%s: slip %d allocated %d, PostgreSQL %d", name, id, other[id], v))
			}
		}
	}
	cmpInv("Go", gr.Invoices)
	cmpInv("SQLite", sq.Invoices)
	cmpAlloc("Go", gr.Allocation)
	cmpAlloc("SQLite", sq.Allocation)
	c.Agree = len(c.Differences) == 0
	// Naive designs, for observation.
	perLine := map[invKey]int64{}
	for _, l := range lines {
		perLine[invKey{l.Customer, l.PostingMonth, l.Rate}] += l.AmountSen * int64(l.Rate) / 10000
	}
	for i, v := range pg.Invoices {
		k := invKey{v.Customer, v.PostingMonth, v.Rate}
		if perLine[k] != v.TaxYen {
			c.PerLineRoundingDiffers++
		}
		if floor := int64(math.Floor(float64(v.BaseSen*int64(v.Rate)) / 10000)); floor != v.TaxYen {
			c.FloorDiffers++
		}
		if i < len(sq.NaiveReal) && sq.NaiveReal[i].TaxYen != v.TaxYen {
			c.SQLiteRealDiffers++
		}
		if f := int64(float64(v.BaseSen) / 100 * (float64(v.Rate) / 100)); f != v.TaxYen {
			c.GoFloatDiffers++
		}
	}
	if !c.Agree {
		return c, fmt.Errorf("%w: %v", ErrTaxPathsDisagree, c.Differences)
	}
	return c, nil
}
