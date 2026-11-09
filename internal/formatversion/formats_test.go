package formatversion_test

import (
	"strings"
	"testing"

	"tei/internal/formatversion"
)

// passthroughCodec keeps the central document unchanged. It is enough to test
// format-level dispatch and duplicate handling.
type passthroughCodec struct{ version formatversion.Version }

func (c passthroughCodec) Version() formatversion.Version { return c.version }
func (c passthroughCodec) Decode([]byte) (formatversion.Document, []formatversion.Diagnostic, error) {
	return formatversion.Document{Declared: c.version}, nil, nil
}
func (c passthroughCodec) Encode(formatversion.Document) ([]byte, []formatversion.Diagnostic, error) {
	return []byte("v"), nil, nil
}

func TestFormatsDispatchByFormat(t *testing.T) {
	formats := formatversion.NewFormats()
	if err := formats.Register("tei.semantic", passthroughCodec{version: 1}); err != nil {
		t.Fatal(err)
	}
	result, err := formats.Convert("tei.semantic", []byte("source"), 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != formatversion.Success {
		t.Fatalf("status=%v reason=%q", result.Status, result.Reason)
	}
	if got := formats.Supported(); len(got) != 1 || got[0] != "tei.semantic" {
		t.Fatalf("supported=%v", got)
	}
}

func TestFormatsRejectUnknownFormatAndNameTheRegisteredOnes(t *testing.T) {
	formats := formatversion.NewFormats()
	if err := formats.Register("tei.semantic", passthroughCodec{version: 1}); err != nil {
		t.Fatal(err)
	}
	result, err := formats.Convert("unknown.format", []byte("source"), 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != formatversion.Failure {
		t.Fatalf("unknown format was accepted: %+v", result)
	}
	if !strings.Contains(result.Reason, "tei.semantic") {
		t.Fatalf("registered formats are not named: %q", result.Reason)
	}
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "UNKNOWN_FORMAT" {
		t.Fatalf("diagnostics=%+v", result.Diagnostics)
	}
}

func TestFormatsRejectDuplicateFormatVersion(t *testing.T) {
	formats := formatversion.NewFormats()
	if err := formats.Register("tei.semantic", passthroughCodec{version: 1}); err != nil {
		t.Fatal(err)
	}
	if err := formats.Register("tei.semantic", passthroughCodec{version: 1}); err == nil {
		t.Fatal("duplicate format/version was accepted")
	}
}

func TestFormatsKeepVersionsSeparatePerFormat(t *testing.T) {
	formats := formatversion.NewFormats()
	if err := formats.Register("format.a", passthroughCodec{version: 1}); err != nil {
		t.Fatal(err)
	}
	if err := formats.Register("format.b", passthroughCodec{version: 1}); err != nil {
		t.Fatalf("the same version in another format collided: %v", err)
	}
	if err := formats.Register("format.a", passthroughCodec{version: 2}); err != nil {
		t.Fatal(err)
	}
	if got := formats.Registry("format.a").Supported(); len(got) != 2 {
		t.Fatalf("format.a versions=%v", got)
	}
}

func TestFormatsRejectEmptyFormatIdentifier(t *testing.T) {
	formats := formatversion.NewFormats()
	if err := formats.Register("  ", passthroughCodec{version: 1}); err == nil {
		t.Fatal("empty format identifier was accepted")
	}
}
