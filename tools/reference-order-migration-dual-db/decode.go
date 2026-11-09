package ordermigrationdualdb

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ChangeFeed is how step 4 reads the legacy database's changes after the
// step-2 copy. The sample uses PostgreSQL logical decoding with the
// test_decoding output plugin through the SQL slot functions; the slot is
// created before the copy's snapshot is taken (see TakeSnapshot).
const (
	ChangeFeedMethod = "postgres-logical-decoding/test_decoding"
	ChangeFeedSlot   = "tei_omdd_cdc"
	// ChangeFeedExactness states how the copy and the change feed are
	// paired. The WAL position read in the copy's statement stays an
	// approximation and is not used to decide anything. The slot's
	// consistent point is a lower bound (every transaction not visible to
	// the copy commits after it), and each decoded transaction is applied
	// or skipped by testing its xid against the copy's pg_snapshot.
	ChangeFeedExactness = "exact-by-xid-visibility-against-pg-snapshot;slot-created-before-snapshot"
)

// LSN is a PostgreSQL WAL position.
type LSN uint64

// ParseLSN parses the "X/Y" text form.
func ParseLSN(s string) (LSN, error) {
	hi, lo, ok := strings.Cut(s, "/")
	if !ok {
		return 0, fmt.Errorf("lsn %q: missing '/'", s)
	}
	h, err := strconv.ParseUint(hi, 16, 32)
	if err != nil {
		return 0, fmt.Errorf("lsn %q: %w", s, err)
	}
	l, err := strconv.ParseUint(lo, 16, 32)
	if err != nil {
		return 0, fmt.Errorf("lsn %q: %w", s, err)
	}
	return LSN(h<<32 | l), nil
}

func (l LSN) String() string { return fmt.Sprintf("%X/%X", uint64(l)>>32, uint64(l)&0xffffffff) }

// PGSnapshot is the parsed text form of pg_current_snapshot():
// "xmin:xmax:xip,xip,...". Values are 64-bit (epoch-extended) xids.
type PGSnapshot struct {
	Xmin, Xmax uint64
	Xip        map[uint64]bool
}

// ParsePGSnapshot parses pg_current_snapshot()::text.
func ParsePGSnapshot(s string) (PGSnapshot, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return PGSnapshot{}, fmt.Errorf("pg_snapshot %q: want xmin:xmax:xip", s)
	}
	var snap PGSnapshot
	var err error
	if snap.Xmin, err = strconv.ParseUint(parts[0], 10, 64); err != nil {
		return PGSnapshot{}, fmt.Errorf("pg_snapshot %q: %w", s, err)
	}
	if snap.Xmax, err = strconv.ParseUint(parts[1], 10, 64); err != nil {
		return PGSnapshot{}, fmt.Errorf("pg_snapshot %q: %w", s, err)
	}
	snap.Xip = map[uint64]bool{}
	if parts[2] != "" {
		for _, x := range strings.Split(parts[2], ",") {
			v, err := strconv.ParseUint(x, 10, 64)
			if err != nil {
				return PGSnapshot{}, fmt.Errorf("pg_snapshot %q: %w", s, err)
			}
			snap.Xip[v] = true
		}
	}
	return snap, nil
}

// Extend widens a 32-bit xid, as printed by test_decoding, to the 64-bit
// form of the snapshot. Decoded transactions commit after the slot was
// created, so they lie within 2^31 of the snapshot's xmax.
func (s PGSnapshot) Extend(xid32 uint32) uint64 {
	epoch := s.Xmax &^ 0xffffffff
	x := epoch | uint64(xid32)
	switch {
	case x > s.Xmax && x-s.Xmax > 1<<31 && x >= 1<<32:
		x -= 1 << 32
	case x < s.Xmax && s.Xmax-x > 1<<31:
		x += 1 << 32
	}
	return x
}

// Visible reports whether a committed transaction's effects are in the
// snapshot (pg_visible_in_snapshot for committed xids).
func (s PGSnapshot) Visible(xid uint64) bool {
	if xid < s.Xmin {
		return true
	}
	if xid >= s.Xmax {
		return false
	}
	return !s.Xip[xid]
}

// Column is one decoded column value. Null is distinct from "".
type Column struct {
	Name  string
	Type  string
	Value string
	Null  bool
}

// RowChange is one decoded row change.
type RowChange struct {
	Table string // schema-qualified, e.g. omdd.legacy_order
	Op    string // INSERT, UPDATE, DELETE, TRUNCATE
	// Old is the old key (or old tuple) when the decoder printed one; for a
	// DELETE it holds the replica identity columns.
	Old []Column
	New []Column
}

// Get returns the named column from New, or from Old for a DELETE.
func (c RowChange) Get(name string) (Column, bool) {
	cols := c.New
	if c.Op == "DELETE" {
		cols = c.Old
	}
	for _, col := range cols {
		if col.Name == name {
			return col, true
		}
	}
	return Column{}, false
}

// DecodedTx is one committed legacy transaction from the change feed.
type DecodedTx struct {
	Xid       uint32
	CommitLSN LSN
	Changes   []RowChange
}

// DecodedRow is one row of pg_logical_slot_peek_changes.
type DecodedRow struct {
	LSN  string
	Data string
}

// GroupTransactions turns slot rows into transactions. Rows must come from
// one peek call with include-xids, in the order returned. A trailing
// transaction without COMMIT (cut by upto_nchanges) is not returned.
func GroupTransactions(rows []DecodedRow) ([]DecodedTx, error) {
	var out []DecodedTx
	var cur *DecodedTx
	for _, r := range rows {
		switch {
		case strings.HasPrefix(r.Data, "BEGIN "):
			if cur != nil {
				return nil, fmt.Errorf("BEGIN inside transaction %d", cur.Xid)
			}
			x, err := strconv.ParseUint(strings.TrimPrefix(r.Data, "BEGIN "), 10, 32)
			if err != nil {
				return nil, fmt.Errorf("BEGIN row %q: %w", r.Data, err)
			}
			cur = &DecodedTx{Xid: uint32(x)}
		case strings.HasPrefix(r.Data, "COMMIT "):
			if cur == nil {
				return nil, fmt.Errorf("COMMIT without BEGIN: %q", r.Data)
			}
			lsn, err := ParseLSN(r.LSN)
			if err != nil {
				return nil, err
			}
			cur.CommitLSN = lsn
			out = append(out, *cur)
			cur = nil
		case strings.HasPrefix(r.Data, "table "):
			if cur == nil {
				return nil, fmt.Errorf("change outside a transaction: %q", r.Data)
			}
			ch, err := ParseChange(r.Data)
			if err != nil {
				return nil, err
			}
			cur.Changes = append(cur.Changes, ch)
		default:
			// Logical decoding messages and other rows are not produced by
			// this sample; refuse rather than skip them silently.
			return nil, fmt.Errorf("unexpected decoded row %q", r.Data)
		}
	}
	return out, nil
}

// ParseChange parses one test_decoding change line, e.g.
//
//	table omdd.legacy_order: UPDATE: order_no[text]:'SO-1' status[text]:'open'
//	table omdd.legacy_order: UPDATE: old-key: order_no[text]:'A' new-tuple: order_no[text]:'B' ...
//	table omdd.legacy_order_line: DELETE: order_no[text]:'SO-1' line_no[integer]:1
//	table omdd.legacy_order: TRUNCATE: (no-flags)
func ParseChange(line string) (RowChange, error) {
	rest, ok := strings.CutPrefix(line, "table ")
	if !ok {
		return RowChange{}, fmt.Errorf("not a change line: %q", line)
	}
	table, rest, ok := strings.Cut(rest, ": ")
	if !ok {
		return RowChange{}, fmt.Errorf("change line without table: %q", line)
	}
	op, rest, _ := strings.Cut(rest, ":")
	rest = strings.TrimPrefix(rest, " ")
	ch := RowChange{Table: table, Op: op}
	switch op {
	case "TRUNCATE":
		return ch, nil
	case "INSERT", "UPDATE", "DELETE":
	default:
		return RowChange{}, fmt.Errorf("unknown operation %q in %q", op, line)
	}
	if rest == "(no-tuple-data)" {
		return RowChange{}, fmt.Errorf("change without tuple data (replica identity nothing?): %q", line)
	}
	if after, found := strings.CutPrefix(rest, "old-key: "); found {
		oldPart, newPart, ok := strings.Cut(after, " new-tuple: ")
		if !ok {
			return RowChange{}, fmt.Errorf("old-key without new-tuple: %q", line)
		}
		var err error
		if ch.Old, err = parseColumns(oldPart); err != nil {
			return RowChange{}, fmt.Errorf("%w in %q", err, line)
		}
		rest = newPart
	}
	cols, err := parseColumns(rest)
	if err != nil {
		return RowChange{}, fmt.Errorf("%w in %q", err, line)
	}
	if op == "DELETE" {
		ch.Old = cols
	} else {
		ch.New = cols
	}
	return ch, nil
}

func parseColumns(s string) ([]Column, error) {
	var cols []Column
	i := 0
	for i < len(s) {
		for i < len(s) && s[i] == ' ' {
			i++
		}
		if i >= len(s) {
			break
		}
		lb := strings.IndexByte(s[i:], '[')
		if lb < 0 {
			return nil, errors.New("column without type")
		}
		name := s[i : i+lb]
		i += lb + 1
		rb := strings.Index(s[i:], "]:")
		if rb < 0 {
			return nil, fmt.Errorf("column %q: unterminated type", name)
		}
		col := Column{Name: name, Type: s[i : i+rb]}
		i += rb + 2
		switch {
		case strings.HasPrefix(s[i:], "'"):
			var b strings.Builder
			i++
			closed := false
			for i < len(s) {
				if s[i] == '\'' {
					if i+1 < len(s) && s[i+1] == '\'' {
						b.WriteByte('\'')
						i += 2
						continue
					}
					i++
					closed = true
					break
				}
				b.WriteByte(s[i])
				i++
			}
			if !closed {
				return nil, fmt.Errorf("column %q: unterminated quoted value", name)
			}
			col.Value = b.String()
		case strings.HasPrefix(s[i:], "unchanged-toast-datum"):
			return nil, fmt.Errorf("column %q: unchanged TOAST value is not supported", name)
		default:
			end := strings.IndexByte(s[i:], ' ')
			if end < 0 {
				end = len(s) - i
			}
			v := s[i : i+end]
			i += end
			if v == "null" {
				col.Null = true
			} else {
				col.Value = v
			}
		}
		cols = append(cols, col)
	}
	return cols, nil
}
