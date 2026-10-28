package monthendclose

import (
	"bytes"
	"strings"
	"testing"

	"golang.org/x/text/encoding/japanese"
)

func TestParseAndReemitByteForByte(t *testing.T) {
	spec := testSpec(t)
	rows := syntheticShipments(t)
	b := buildFile(t, rows, fileOpts{seq: 1})
	pf, err := spec.ParseFile(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(pf.Data) != len(rows) {
		t.Fatalf("%d data records", len(pf.Data))
	}
	for i, r := range rows {
		got := pf.Data[i]
		if got.HeldReason != r.ExpectHold {
			t.Fatalf("%s: held %q (%s), want %q", r.ShipmentID, got.HeldReason, got.HeldField, r.ExpectHold)
		}
	}
	if v := pf.Data[0].Values["itemName"]; v != "波～ダッシュ品" {
		t.Fatalf("0x8160 decoded as %q, want the CP932 mapping U+FF5E", v)
	}
	if pf.Data[1].Values["itemName"] != "東西南北上下左右前後" || pf.Data[1].Values["note"] != "全角始" {
		t.Fatalf("byte fields: %q %q", pf.Data[1].Values["itemName"], pf.Data[1].Values["note"])
	}
	if pf.Data[2].Values["itemName"] != "ｱｲｳｴｵ半角ｶﾅ" {
		t.Fatalf("half-width kana %q", pf.Data[2].Values["itemName"])
	}
	all := append(append([]Record{pf.Header}, pf.Data...), pf.Trailer)
	out, err := spec.EmitFile(all)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, b) {
		t.Fatal("re-emitted file differs from the original")
	}
	// Without the raw bytes of held records the duplicate code would come
	// back as another code: the hold is what keeps the bytes.
	r5 := pf.Data[4]
	r5.HeldReason = ""
	if enc, err := spec.EncodeRecord(r5); err == nil && bytes.Equal(enc, r5.Raw) {
		t.Fatal("0xFA5C unexpectedly round-trips; the hold case is not exercised")
	}
}

// A character-based reader of the same record shifts every field after a
// double-byte character; the byte-based parser does not.
func TestCharacterSlicingWouldShiftFields(t *testing.T) {
	rows := syntheticShipments(t)
	rec := dataRecordBytes(t, 2, rows[1])
	decoded, err := japanese.ShiftJIS.NewDecoder().Bytes(rec)
	if err != nil {
		t.Fatal(err)
	}
	runes := []rune(string(decoded))
	naiveItem := strings.TrimRight(string(runes[51:71]), " ")
	if naiveItem == "東西南北上下左右前後" {
		t.Fatal("character slicing unexpectedly matched; the case does not exercise byte offsets")
	}
	pf, err := testSpec(t).ParseFile(buildFile(t, rows[1:2], fileOpts{seq: 1}))
	if err != nil {
		t.Fatal(err)
	}
	if pf.Data[0].Values["itemName"] != "東西南北上下左右前後" {
		t.Fatalf("byte parser item %q", pf.Data[0].Values["itemName"])
	}
}

func TestWholeFileRejections(t *testing.T) {
	spec := testSpec(t)
	rows := syntheticShipments(t)
	cases := []struct {
		name string
		b    []byte
		want string
	}{
		{"trailer count off by one", buildFile(t, rows, fileOpts{seq: 1, countDelta: 1}), RejectTrailerCount},
		{"cut in the middle of a record", buildFile(t, rows, fileOpts{seq: 1, truncateTail: 120}), RejectTruncated},
		{"cut at a record boundary, no trailer", buildFile(t, rows, fileOpts{seq: 1, truncateTail: 82}), RejectTruncated},
	}
	for _, c := range cases {
		_, err := spec.ParseFile(c.b)
		if reason, ok := IsReject(err); !ok || reason != c.want {
			t.Fatalf("%s: %v, want %s", c.name, err, c.want)
		}
	}
	bad := buildFile(t, rows, fileOpts{seq: 1})
	bad[82+41] = 'X' // a non-digit in the first amount
	if reason, ok := IsReject(func() error { _, err := spec.ParseFile(bad); return err }()); !ok || reason != RejectMalformed {
		t.Fatalf("malformed amount: %v", reason)
	}
}

func TestCompareEquivalents(t *testing.T) {
	spec := testSpec(t)
	rows := syntheticShipments(t)
	for _, i := range []int{0, 3} {
		pf, err := spec.ParseFile(buildFile(t, rows[i:i+1], fileOpts{seq: 1}))
		if err != nil {
			t.Fatal(err)
		}
		fileEv, err := spec.eventFromRecord(pf.Data[0])
		if err != nil {
			t.Fatal(err)
		}
		msgEv := fileEv
		msgEv.ItemName = rows[i].ItemText
		if fileEv.ItemName == msgEv.ItemName {
			t.Fatalf("%s: the message text is not in the other mapping", rows[i].ShipmentID)
		}
		if fileEv.ContentDigest(spec) != msgEv.ContentDigest(spec) {
			t.Fatalf("%s: %q and %q are not the same event", rows[i].ShipmentID, fileEv.ItemName, msgEv.ItemName)
		}
	}
}

func TestSpecMustTileRecords(t *testing.T) {
	spec := testSpec(t)
	s := *spec
	s.Data.Fields = append([]Field(nil), spec.Data.Fields...)
	s.Data.Fields[7].Length = 19
	if err := ValidateSpec(&s); err == nil {
		t.Fatal("a gap in the record was accepted")
	}
}
