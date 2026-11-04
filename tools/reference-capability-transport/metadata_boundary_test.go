package capabilitytransport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenericMetadataRejectsAmbiguityWithoutParsingOpaquePayload(t *testing.T) {
	const secret = "UNTRUSTED_FIELD_SECRET"
	original := genericDispatch([]byte(`{"quantity":1,"quantity":2}`))
	raw, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var decoded GenericDispatch
	if err := decodeGenericTransportMetadata(raw, &decoded); err != nil || !bytes.Equal(decoded.Payload, original.Payload) {
		t.Fatalf("valid opaque payload changed: %+v, %v", decoded, err)
	}
	for _, tc := range []struct{ name, anchor, replacement string }{
		{"duplicate", `"contractId":`, `"contractId":"forged","contractId":`},
		{"escaped duplicate", `"contractId":`, `"contract\u0049d":"forged","contractId":`},
		{"case alias", `"contractId":`, `"ContractId":"forged","contractId":`},
		{"nested duplicate", `"pluginId":`, `"pluginId":"forged","pluginId":`},
		{"nested case alias", `"pluginId":`, `"PluginId":"forged","pluginId":`},
		{"untrusted unknown", `"contractId":`, `"` + secret + `":"x","contractId":`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !bytes.Contains(raw, []byte(tc.anchor)) {
				t.Fatal("missing test anchor")
			}
			malformed := bytes.Replace(raw, []byte(tc.anchor), []byte(tc.replacement), 1)
			var out GenericDispatch
			if err := decodeGenericTransportMetadata(malformed, &out); err == nil || strings.Contains(err.Error(), secret) {
				t.Fatalf("ambiguous metadata accepted or leaked field name: %v", err)
			}
		})
	}
}

func TestHTTPMetadataRejectsAmbiguityBeforeProviderButKeepsOpaqueBytes(t *testing.T) {
	payload := []byte(`{"quantity":1,"quantity":2}`)
	dispatch := genericDispatch(payload)
	called := 0
	handler, err := NewHTTPTransportHandler(genericDispatchIdentity(dispatch), func(_ context.Context, got GenericDispatch) (GenericTransportResult, error) {
		called++
		if !bytes.Equal(got.Payload, payload) {
			t.Errorf("opaque payload changed: %s", got.Payload)
		}
		return echoHTTPTransportImplementation(context.Background(), got)
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(dispatch)
	if err != nil {
		t.Fatal(err)
	}
	request := func(body []byte) int {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		req.Header.Set("Content-Type", httpTransportContentType)
		req.Header.Set(httpTransportRequestID, "metadata-boundary")
		result := httptest.NewRecorder()
		handler.ServeHTTP(result, req)
		return result.Code
	}
	if status := request(raw); status != http.StatusOK || called != 1 {
		t.Fatalf("normal request: status=%d calls=%d", status, called)
	}
	bad := bytes.Replace(raw, []byte(`"contractId":`), []byte(`"contractId":"forged","contractId":`), 1)
	if status := request(bad); status != http.StatusBadRequest || called != 1 {
		t.Fatalf("ambiguous request reached provider: status=%d calls=%d", status, called)
	}
}
