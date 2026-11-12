package jsonl_test

import (
	"strings"
	"testing"

	"tei/internal/jsonl"
)

func TestReadBytesPreservesRecordsAndBlankLines(t *testing.T) {
	type record struct {
		N int `json:"n"`
	}
	records, err := jsonl.ReadBytes([]byte("{\"n\":1}\n\n{\"n\":2}\n"), "test witness", jsonl.DecodeJSON[record])
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 || records[0].N != 1 || records[1].N != 2 {
		t.Fatalf("decoded records=%+v", records)
	}
}

func TestReadBytesDamagedLineReturnsQualifiedPrefix(t *testing.T) {
	type record struct {
		N int `json:"n"`
	}
	records, err := jsonl.ReadBytes([]byte("{\"n\":1}\n{\"n\":2}\n{invalid}\n"), "test witness", jsonl.DecodeJSON[record])
	if err == nil || !strings.Contains(err.Error(), "line 3") {
		t.Fatalf("missing line-position error: %v", err)
	}
	if len(records) != 2 || records[0].N != 1 || records[1].N != 2 {
		t.Fatalf("damaged line discarded qualified prefix: %+v", records)
	}
}

func TestReadBytesOversizedLineDoesNotCountAsComplete(t *testing.T) {
	type record struct {
		N int `json:"n"`
	}
	raw := []byte("{\"n\":1}\n" + strings.Repeat("x", 1024*1024+1))
	records, err := jsonl.ReadBytes(raw, "test witness", jsonl.DecodeJSON[record])
	if err == nil {
		t.Fatal("an oversized line must fail")
	}
	if len(records) != 1 || records[0].N != 1 {
		t.Fatalf("prefix lost or invalid line accepted: %+v", records)
	}
}
