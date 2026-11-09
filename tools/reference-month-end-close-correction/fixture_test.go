package monthendclose

import (
	"bytes"
	"fmt"
	"path/filepath"
	"testing"

	"golang.org/x/text/encoding/japanese"
)

func testSpec(t *testing.T) *FileSpec {
	t.Helper()
	s, err := ReadFileSpec(filepath.Join("testdata", "file-spec.json"))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func sjis(t *testing.T, s string) []byte {
	t.Helper()
	b, err := japanese.ShiftJIS.NewEncoder().Bytes([]byte(s))
	if err != nil {
		t.Fatalf("%q: %v", s, err)
	}
	return b
}

func padBytes(t *testing.T, b []byte, n int) []byte {
	t.Helper()
	if len(b) > n {
		t.Fatalf("%d bytes exceed %d", len(b), n)
	}
	return append(append([]byte(nil), b...), bytes.Repeat([]byte(" "), n-len(b))...)
}

// shipmentRow is one synthetic data record. Item and Note are the exact
// bytes written to the 20-byte and 9-byte text fields; ItemText is what a
// sender on the message route writes for the same item.
type shipmentRow struct {
	ShipmentID, OrderNo, ShipDate string
	Qty, Amount                   int64
	Item, Note                    []byte
	ItemText                      string
	ExpectHold                    string
	// RawItemNote replaces the two text fields (29 bytes) as one range, so a
	// double-byte character can straddle their boundary.
	RawItemNote []byte
}

func dataRecordBytes(t *testing.T, seq int, r shipmentRow) []byte {
	t.Helper()
	b := []byte(fmt.Sprintf("D%06d%-10s%-10s%s%06d%010d", seq, r.ShipmentID, r.OrderNo, r.ShipDate, r.Qty, r.Amount))
	if r.RawItemNote != nil {
		b = append(b, padBytes(t, r.RawItemNote, 29)...)
	} else {
		b = append(b, padBytes(t, r.Item, 20)...)
		b = append(b, padBytes(t, r.Note, 9)...)
	}
	if len(b) != 80 {
		t.Fatalf("data record %d is %d bytes", seq, len(b))
	}
	return b
}

type fileOpts struct {
	sender       string
	seq          int
	month        string
	countDelta   int
	sumDelta     int
	truncateTail int // bytes cut from the end
}

func buildFile(t *testing.T, rows []shipmentRow, o fileOpts) []byte {
	t.Helper()
	if o.sender == "" {
		o.sender = "PARTNER01"
	}
	if o.month == "" {
		o.month = "202609"
	}
	var out []byte
	line := func(b []byte) { out = append(append(out, b...), '\r', '\n') }
	line([]byte(fmt.Sprintf("H%-10s%06d%s%s%43s", o.sender, o.seq, "20260930163000", o.month, "")))
	var sum int64
	for i, r := range rows {
		line(dataRecordBytes(t, i+1, r))
		sum += r.Amount
	}
	line([]byte(fmt.Sprintf("T%06d%012d%61s", len(rows)+o.countDelta, sum+int64(o.sumDelta), "")))
	return out[:len(out)-o.truncateTail]
}

// syntheticShipments returns the September shipments of the partner file.
// The first seven rows carry the encoding cases; the rest are plain.
func syntheticShipments(t *testing.T) []shipmentRow {
	t.Helper()
	tenWide := sjis(t, "東西南北上下左右前後")                           // exactly 20 bytes: the field is full to its last byte
	straddle := append(sjis(t, "品名九文字の商品名X"), sjis(t, "あ")...) // 19 bytes, then a double-byte character over bytes 20-21
	rows := []shipmentRow{
		{ShipmentID: "SH-0001", OrderNo: "SO-0001", ShipDate: "20260903", Qty: 3, Amount: 3600,
			Item: sjis(t, "波～ダッシュ品"), Note: []byte("A"), ItemText: "波〜ダッシュ品"}, // CP932 0x8160; the message sender maps it to U+301C
		{ShipmentID: "SH-0002", OrderNo: "SO-0002", ShipDate: "20260904", Qty: 1, Amount: 12000,
			Item: tenWide, Note: sjis(t, "全角始"), ItemText: "東西南北上下左右前後"},
		{ShipmentID: "SH-0003", OrderNo: "SO-0003", ShipDate: "20260907", Qty: 5, Amount: 550,
			Item: sjis(t, "ｱｲｳｴｵ半角ｶﾅ"), Note: []byte("B"), ItemText: "ｱｲｳｴｵ半角ｶﾅ"},
		{ShipmentID: "SH-0004", OrderNo: "SO-0004", ShipDate: "20260908", Qty: 2, Amount: 2400,
			Item: []byte{0x87, 0x40, 0x81, 0x61, 0x81, 0x7C, 0x81, 0xCA, 0x81, 0x5F}, Note: []byte("C"), ItemText: "①‖−¬＼"}, // ①∥－￢＼, message side in JIS mapping
		{ShipmentID: "SH-0005", OrderNo: "SO-0005", ShipDate: "20260909", Qty: 1, Amount: 800,
			Item: append(sjis(t, "IBM拡張"), 0xFA, 0x5C), Note: []byte("D"), ItemText: "IBM拡張纊", ExpectHold: HoldNonRoundtrip}, // 0xFA5C re-encodes as 0xED40
		{ShipmentID: "SH-0006", OrderNo: "SO-0006", ShipDate: "20260910", Qty: 4, Amount: 4000,
			Item: append(sjis(t, "外字"), 0xF0, 0x40), Note: []byte("E"), ItemText: "外字", ExpectHold: HoldUndecodable}, // user-defined area
		{ShipmentID: "SH-0007", OrderNo: "SO-0007", ShipDate: "20260911", Qty: 1, Amount: 1500,
			RawItemNote: straddle, ItemText: "品名九文字の商品名Xあ", ExpectHold: HoldUndecodable}, // "あ" straddles the item/note boundary
	}
	for i := 8; i <= 30; i++ {
		rows = append(rows, shipmentRow{ShipmentID: fmt.Sprintf("SH-%04d", i), OrderNo: fmt.Sprintf("SO-%04d", i),
			ShipDate: fmt.Sprintf("202609%02d", 1+i%29), Qty: int64(1 + i%5), Amount: int64(1000 + 37*i),
			Item: sjis(t, fmt.Sprintf("通常品%02d", i)), Note: []byte("N"), ItemText: fmt.Sprintf("通常品%02d", i)})
	}
	return rows
}
