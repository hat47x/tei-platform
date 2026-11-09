package studioplugindatalifecycle

import (
	"strings"
	"testing"
)

const protectedOriginal = `<root data-id="1"><name>old</name><x-data q="z"/><ext:note>keep</ext:note></root>`

func editRequest() EditRequest {
	edited := strings.Replace(protectedOriginal, "<name>old</name>", "<name>new</name>", 1)
	return EditRequest{
		Path:         "applications/order.tei",
		BaseRevision: "r7", CurrentRevision: "r7",
		BaseFingerprint: Fingerprint(protectedOriginal), CurrentFingerprint: Fingerprint(protectedOriginal),
		Original: protectedOriginal, Edited: edited,
		Protected:            []string{`<x-data q="z"/>`, `<ext:note>keep</ext:note>`},
		PluginAvailability:   PluginMissing,
		SyntaxValid:          false,
		UnresolvedReferences: []string{"semantic.unknown"},
		Accepted:             false, Executable: false,
	}
}

func TestSaveAllowsUnsupportedPluginAndInvalidSyntax(t *testing.T) {
	result, err := Save(editRequest())
	if err != nil {
		t.Fatal(err)
	}
	if !result.Saved || result.Bytes != editRequest().Edited {
		t.Fatalf("save=%+v", result)
	}
	if !contains(result.Diagnosis, "plugin-unavailable:missing") {
		t.Fatalf("diagnosis=%v", result.Diagnosis)
	}
	if !contains(result.Diagnosis, "syntax-invalid") || !contains(result.Diagnosis, "unknown-reference:semantic.unknown") {
		t.Fatalf("diagnosis=%v", result.Diagnosis)
	}
	if !contains(result.Diagnosis, "execution-denied") {
		t.Fatalf("diagnosis=%v", result.Diagnosis)
	}
	if result.AcceptanceChanged || result.Accepted || result.Executable {
		t.Fatalf("save changed acceptance or execution: %+v", result)
	}
	if !result.SourcePreserved {
		t.Fatal("source preservation was not reported")
	}
}

func TestSaveStillWorksForDisabledAndVersionUnsupportedPlugins(t *testing.T) {
	for _, availability := range []string{PluginDisabled, PluginVersionUnsupported} {
		req := editRequest()
		req.PluginAvailability = availability
		result, err := Save(req)
		if err != nil {
			t.Fatal(err)
		}
		if !result.Saved {
			t.Fatalf("%s: save refused: %v", availability, result.Reasons)
		}
		if !contains(result.Diagnosis, "plugin-unavailable:"+availability) {
			t.Fatalf("%s: diagnosis=%v", availability, result.Diagnosis)
		}
	}
}

func TestSaveRefusesStaleRevisionAndExternalBytes(t *testing.T) {
	t.Run("revision", func(t *testing.T) {
		req := editRequest()
		req.CurrentRevision = "r8"
		result, err := Save(req)
		if err != nil {
			t.Fatal(err)
		}
		if result.Saved || !contains(result.Reasons, "stale-revision") {
			t.Fatalf("result=%+v", result)
		}
		if result.Bytes != "" {
			t.Fatal("a refused save returned bytes")
		}
	})
	t.Run("bytes", func(t *testing.T) {
		req := editRequest()
		req.CurrentFingerprint = Fingerprint("externally edited")
		result, err := Save(req)
		if err != nil {
			t.Fatal(err)
		}
		if result.Saved || !contains(result.Reasons, "source-bytes-mismatch") {
			t.Fatalf("result=%+v", result)
		}
	})
}

func TestSaveRefusesLostProtectedContent(t *testing.T) {
	req := editRequest()
	req.Edited = `<root data-id="1"><name>new</name></root>`
	result, err := Save(req)
	if err != nil {
		t.Fatal(err)
	}
	if result.Saved || !contains(result.Reasons, "protected-content-lost") {
		t.Fatalf("result=%+v", result)
	}
	if len(result.ProtectedMissing) != 2 {
		t.Fatalf("missing=%v", result.ProtectedMissing)
	}
	if result.Bytes != "" {
		t.Fatal("a refused save returned bytes")
	}
}

func TestSaveDoesNotGrantAcceptanceOrExecution(t *testing.T) {
	req := editRequest()
	req.Accepted = true
	req.Executable = true
	result, err := Save(req)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Saved {
		t.Fatalf("save refused: %v", result.Reasons)
	}
	if !result.Accepted || result.AcceptanceChanged {
		t.Fatalf("acceptance=%v changed=%v", result.Accepted, result.AcceptanceChanged)
	}
	if result.Executable {
		t.Fatal("an invalid source was reported as executable")
	}
}

func TestSaveWhenAvailableAndValid(t *testing.T) {
	req := editRequest()
	req.PluginAvailability = PluginAvailable
	req.SyntaxValid = true
	req.UnresolvedReferences = nil
	req.Executable = true
	result, err := Save(req)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Saved || !result.Executable {
		t.Fatalf("result=%+v", result)
	}
	if contains(result.Diagnosis, "execution-denied") {
		t.Fatalf("diagnosis=%v", result.Diagnosis)
	}
}

func TestSavePreservesBytesOutsideTheEdit(t *testing.T) {
	result, err := Save(editRequest())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Bytes, `<x-data q="z"/>`) ||
		!strings.Contains(result.Bytes, `<ext:note>keep</ext:note>`) {
		t.Fatalf("protected bytes were dropped: %s", result.Bytes)
	}
}
