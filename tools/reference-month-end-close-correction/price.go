package monthendclose

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Errors of the price history.
var (
	ErrPriceOverlap           = errors.New("price overlaps a row known at the same time")
	ErrPriceRecordedBackwards = errors.New("price change recorded before the item's latest knowledge")
	ErrNoPrice                = errors.New("no price is valid on the date at the record time")
	ErrNothingToCancel        = errors.New("no revision starts on the date")
)

// Price operations recorded in item_price.op and price_change.op.
const (
	PriceInitial     = "initial"
	PriceRetroactive = "retroactive" // valid from a date before the record time's date
	PriceRevise      = "revise"      // valid from the record time's date
	PriceReserve     = "reserve"     // valid from a date after the record time's date
	PriceCancel      = "cancel"
	PriceRaw         = "raw-register"
)

// ItemDef declares an item, its tax rate and its initial price.
type ItemDef struct {
	ItemCode   string `json:"itemCode"`
	Name       string `json:"name"`
	TaxRate    int    `json:"taxRate"`
	UnitPrice  string `json:"unitPrice"` // decimal, at most 2 places
	ValidFrom  string `json:"validFrom"`
	RecordedAt string `json:"recordedAt"`
}

// Price is one row of the price history.
type Price struct {
	PriceID      int64  `json:"priceId"`
	ItemCode     string `json:"itemCode"`
	UnitPrice    string `json:"unitPrice"`
	UnitPriceSen int64  `json:"-"`
	ValidFrom    string `json:"validFrom"`
	ValidTo      string `json:"validTo"`
	RecordedFrom string `json:"recordedFrom"`
	RecordedTo   string `json:"recordedTo"`
	Op           string `json:"op"`
	TaxRate      int    `json:"taxRate"`
}

// PriceChange is the record of one requested change of the price history.
type PriceChange struct {
	ChangeID   int64     `json:"changeId"`
	ItemCode   string    `json:"itemCode"`
	Op         string    `json:"op"`
	ValidFrom  string    `json:"validFrom"`
	ValidTo    string    `json:"validTo,omitempty"`
	UnitPrice  string    `json:"unitPrice,omitempty"`
	RecordedAt string    `json:"recordedAt"`
	Closed     []int64   `json:"closedPriceIds"`
	Inserted   []int64   `json:"insertedPriceIds"`
	Repriced   []SlipRef `json:"repriced,omitempty"`
}

// ParseSen parses a non-negative or negative decimal with at most two
// places into hundredths, without floating point.
func ParseSen(s string) (int64, error) {
	s = strings.TrimSpace(s)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	whole, frac, _ := strings.Cut(s, ".")
	if whole == "" || len(frac) > 2 || strings.ContainsAny(whole+frac, "+-") {
		return 0, fmt.Errorf("decimal %q: want digits with at most 2 places", s)
	}
	frac += strings.Repeat("0", 2-len(frac))
	w, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("decimal %q: %w", s, err)
	}
	f, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("decimal %q: %w", s, err)
	}
	v := w*100 + f
	if neg {
		v = -v
	}
	return v, nil
}

// FormatSen formats hundredths as a decimal with exactly two places.
func FormatSen(v int64) string {
	sign := ""
	if v < 0 {
		sign, v = "-", -v
	}
	return fmt.Sprintf("%s%d.%02d", sign, v/100, v%100)
}

func lockItem(ctx context.Context, tx pgx.Tx, item string) (int, error) {
	var rate int
	if err := tx.QueryRow(ctx, `SELECT tax_rate FROM mecc.item WHERE item_code=$1 FOR UPDATE`, item).Scan(&rate); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("item %s is not registered", item)
		}
		return 0, err
	}
	return rate, nil
}

func recordPriceChange(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, c *PriceChange, outcome, msg string) error {
	var validTo, price any
	if c.ValidTo != "" {
		validTo = c.ValidTo
	}
	if c.UnitPrice != "" {
		price = c.UnitPrice
	}
	return q.QueryRow(ctx, `INSERT INTO mecc.price_change(item_code,op,valid_from,valid_to,unit_price,recorded_at,outcome,message)
		VALUES ($1,$2,$3::text::date,$4::text::date,$5::text::numeric,$6,$7,$8) RETURNING change_id`,
		c.ItemCode, c.Op, c.ValidFrom, validTo, price, c.RecordedAt, outcome, msg).Scan(&c.ChangeID)
}

// currentRows returns the rows of the item that are current in record time
// (recorded_to is open), in valid order.
func currentRows(ctx context.Context, tx pgx.Tx, item string) ([]Price, error) {
	rows, err := tx.Query(ctx, `SELECT price_id, unit_price::text, (unit_price*100)::bigint, valid_from::text, valid_to::text, recorded_from, op
		FROM mecc.item_price WHERE item_code=$1 AND recorded_to='infinity' ORDER BY valid_from`, item)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Price
	for rows.Next() {
		var p Price
		var rf time.Time
		if err := rows.Scan(&p.PriceID, &p.UnitPrice, &p.UnitPriceSen, &p.ValidFrom, &p.ValidTo, &rf, &p.Op); err != nil {
			return nil, err
		}
		p.ItemCode, p.RecordedFrom = item, rf.UTC().Format(time.RFC3339)
		out = append(out, p)
	}
	return out, rows.Err()
}

// assertPrice makes [from, to) carry priceSen from record time at on. The
// current rows overlapping the range are closed in record time at at, their
// parts outside the range are re-inserted unchanged, and the new row is
// inserted. Nothing known before at changes.
func assertPrice(ctx context.Context, tx pgx.Tx, c *PriceChange, item, from, to string, priceSen int64, op string, at time.Time) error {
	var latest *time.Time
	if err := tx.QueryRow(ctx, `SELECT max(recorded_from) FROM mecc.item_price WHERE item_code=$1`, item).Scan(&latest); err != nil {
		return err
	}
	if latest != nil && !at.After(*latest) {
		return fmt.Errorf("%w: %s recorded at %s, latest %s", ErrPriceRecordedBackwards, item, at.Format(time.RFC3339), latest.Format(time.RFC3339))
	}
	cur, err := currentRows(ctx, tx, item)
	if err != nil {
		return err
	}
	type part struct {
		from, to string
		sen      int64
		op       string
	}
	var keep []part
	for _, p := range cur {
		if !(p.ValidFrom < to && from < p.ValidTo) {
			continue
		}
		if _, err := tx.Exec(ctx, `UPDATE mecc.item_price SET recorded_to=$2 WHERE price_id=$1`, p.PriceID, at); err != nil {
			return err
		}
		c.Closed = append(c.Closed, p.PriceID)
		if p.ValidFrom < from {
			keep = append(keep, part{p.ValidFrom, from, p.UnitPriceSen, p.Op})
		}
		if to < p.ValidTo {
			keep = append(keep, part{to, p.ValidTo, p.UnitPriceSen, p.Op})
		}
	}
	keep = append(keep, part{from, to, priceSen, op})
	for _, k := range keep {
		var id int64
		if err := tx.QueryRow(ctx, `INSERT INTO mecc.item_price(item_code,unit_price,valid_from,valid_to,recorded_from,op,change_id)
			VALUES ($1,$2::text::numeric,$3::text::date,$4::text::date,$5,$6,$7) RETURNING price_id`,
			item, FormatSen(k.sen), k.from, k.to, at, k.op, c.ChangeID).Scan(&id); err != nil {
			return err
		}
		c.Inserted = append(c.Inserted, id)
	}
	return nil
}

const openDate = "infinity"

// LoadItems registers items and their initial prices.
func LoadItems(ctx context.Context, conn *pgx.Conn, items []ItemDef) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, it := range items {
		sen, err := ParseSen(it.UnitPrice)
		if err != nil {
			return err
		}
		at, err := time.Parse(time.RFC3339, it.RecordedAt)
		if err != nil {
			return fmt.Errorf("item %s recordedAt: %w", it.ItemCode, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO mecc.item VALUES ($1,$2,$3)`, it.ItemCode, it.Name, it.TaxRate); err != nil {
			return err
		}
		c := &PriceChange{ItemCode: it.ItemCode, Op: PriceInitial, ValidFrom: it.ValidFrom, UnitPrice: FormatSen(sen), RecordedAt: it.RecordedAt}
		if err := recordPriceChange(ctx, tx, c, "applied", ""); err != nil {
			return err
		}
		if err := assertPrice(ctx, tx, c, it.ItemCode, it.ValidFrom, openDate, sen, PriceInitial, at); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func classifyRevision(validFrom string, at time.Time, loc *time.Location) string {
	today := at.In(loc).Format(dateLayout)
	switch {
	case validFrom < today:
		return PriceRetroactive
	case validFrom > today:
		return PriceReserve
	}
	return PriceRevise
}

// revisePriceTx changes the price from validFrom up to the next change
// already known (the end of the current row containing validFrom).
func revisePriceTx(ctx context.Context, tx pgx.Tx, item, validFrom, price string, at time.Time, loc *time.Location) (PriceChange, error) {
	c := PriceChange{ItemCode: item, ValidFrom: validFrom, UnitPrice: price, RecordedAt: at.Format(time.RFC3339), Op: classifyRevision(validFrom, at, loc)}
	sen, err := ParseSen(price)
	if err != nil {
		return c, err
	}
	c.UnitPrice = FormatSen(sen)
	if _, err := lockItem(ctx, tx, item); err != nil {
		return c, err
	}
	cur, err := currentRows(ctx, tx, item)
	if err != nil {
		return c, err
	}
	to := ""
	for _, p := range cur {
		if p.ValidFrom <= validFrom && validFrom < p.ValidTo {
			to = p.ValidTo
		}
	}
	if to == "" {
		return c, fmt.Errorf("%w: %s has no price on %s to revise", ErrNoPrice, item, validFrom)
	}
	c.ValidTo = to
	if err := recordPriceChange(ctx, tx, &c, "applied", ""); err != nil {
		return c, err
	}
	return c, assertPrice(ctx, tx, &c, item, validFrom, to, sen, c.Op, at)
}

// cancelRevisionTx withdraws the current revision that starts on validFrom:
// its range carries the price of the row just before it again.
func cancelRevisionTx(ctx context.Context, tx pgx.Tx, item, validFrom string, at time.Time) (PriceChange, error) {
	c := PriceChange{ItemCode: item, ValidFrom: validFrom, RecordedAt: at.Format(time.RFC3339), Op: PriceCancel}
	if _, err := lockItem(ctx, tx, item); err != nil {
		return c, err
	}
	cur, err := currentRows(ctx, tx, item)
	if err != nil {
		return c, err
	}
	var target, prev *Price
	for i := range cur {
		if cur[i].ValidFrom == validFrom && cur[i].Op != PriceInitial && cur[i].Op != PriceCancel {
			target = &cur[i]
		}
		if cur[i].ValidTo == validFrom {
			prev = &cur[i]
		}
	}
	if target == nil || prev == nil {
		return c, fmt.Errorf("%w: %s on %s", ErrNothingToCancel, item, validFrom)
	}
	c.ValidFrom, c.ValidTo, c.UnitPrice = prev.ValidFrom, target.ValidTo, prev.UnitPrice
	if err := recordPriceChange(ctx, tx, &c, "applied", fmt.Sprintf("withdraws the revision from %s", validFrom)); err != nil {
		return c, err
	}
	return c, assertPrice(ctx, tx, &c, item, prev.ValidFrom, target.ValidTo, prev.UnitPriceSen, PriceCancel, at)
}

// RegisterPriceRaw inserts a price row without closing what is known: the
// naive registration. It is refused when it overlaps a current row; the
// refusal is recorded.
func RegisterPriceRaw(ctx context.Context, conn *pgx.Conn, item, from, to, price string, at time.Time) error {
	c := PriceChange{ItemCode: item, ValidFrom: from, ValidTo: to, UnitPrice: price, RecordedAt: at.Format(time.RFC3339), Op: PriceRaw}
	sen, err := ParseSen(price)
	if err != nil {
		return err
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := lockItem(ctx, tx, item); err != nil {
		return err
	}
	var overlapping int64
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM mecc.item_price WHERE item_code=$1 AND valid_from < $3::text::date AND $2::text::date < valid_to
		AND recorded_from < 'infinity' AND $4 < recorded_to`, item, from, to, at).Scan(&overlapping); err != nil {
		return err
	}
	if overlapping > 0 {
		tx.Rollback(ctx)
		_ = recordPriceChange(ctx, conn, &c, "refused-overlap", fmt.Sprintf("%d current rows overlap", overlapping))
		return fmt.Errorf("%w: %s [%s,%s) at %s overlaps %d rows", ErrPriceOverlap, item, from, to, c.RecordedAt, overlapping)
	}
	if err := recordPriceChange(ctx, tx, &c, "applied", ""); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO mecc.item_price(item_code,unit_price,valid_from,valid_to,recorded_from,op,change_id)
		VALUES ($1,$2::text::numeric,$3::text::date,$4::text::date,$5,$6,$7)`, item, FormatSen(sen), from, to, at, PriceRaw, c.ChangeID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// PriceAt returns the price valid on date as known at knownAt. Exactly one
// row must answer; a gap is an error, never zero.
func PriceAt(ctx context.Context, q queryer, item, date string, knownAt time.Time) (Price, error) {
	p := Price{ItemCode: item}
	var rf time.Time
	var rt *time.Time
	err := q.QueryRow(ctx, `SELECT p.price_id, p.unit_price::text, (p.unit_price*100)::bigint, p.valid_from::text, p.valid_to::text, p.recorded_from, CASE WHEN p.recorded_to = 'infinity' THEN NULL ELSE p.recorded_to END, p.op, i.tax_rate
		FROM mecc.item_price p JOIN mecc.item i USING (item_code)
		WHERE p.item_code=$1 AND p.valid_from <= $2::text::date AND $2::text::date < p.valid_to AND p.recorded_from <= $3 AND $3 < p.recorded_to`,
		item, date, knownAt).Scan(&p.PriceID, &p.UnitPrice, &p.UnitPriceSen, &p.ValidFrom, &p.ValidTo, &rf, &rt, &p.Op, &p.TaxRate)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, fmt.Errorf("%w: %s on %s known at %s", ErrNoPrice, item, date, knownAt.Format(time.RFC3339))
	}
	if err != nil {
		return p, err
	}
	p.RecordedFrom = rf.UTC().Format(time.RFC3339)
	p.RecordedTo = "infinity"
	if rt != nil {
		p.RecordedTo = rt.UTC().Format(time.RFC3339)
	}
	return p, nil
}
