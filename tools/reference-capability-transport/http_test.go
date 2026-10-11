package capabilitytransport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func echoHTTPTransportImplementation(
	_ context.Context,
	dispatch GenericDispatch,
) (GenericTransportResult, error) {
	return GenericTransportResult{
		Schema:                  resultSchema,
		ContractID:              dispatch.ContractID,
		ContractFingerprint:     dispatch.ContractFingerprint,
		Target:                  dispatch.Target,
		Provider:                dispatch.Provider,
		RequestShapeFingerprint: dispatch.RequestShapeFingerprint,
		ResultShapeFingerprint:  dispatch.ResultShapeFingerprint,
		PayloadCodec:            dispatch.PayloadCodec,
		Payload:                 append([]byte(nil), dispatch.Payload...),
	}, nil
}

func newHTTPTransportServer(t *testing.T) *httptest.Server {
	t.Helper()
	handler, err := NewHTTPTransportHandler(
		genericDispatchIdentity(genericDispatch(nil)),
		echoHTTPTransportImplementation,
	)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

func TestHTTPTransportRoundTripPreservesGenericIdentityAndOpaquePayload(t *testing.T) {
	server := newHTTPTransportServer(t)
	payload := []byte{0x00, 0x01, 0x7f, 0x80, 0xff, 'W', 'E', 'B'}
	dispatch := genericDispatch(payload)

	execution, err := invokeOverHTTPTransport(
		context.Background(),
		server.Client(),
		server.URL,
		"http-roundtrip-1",
		dispatch,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(execution.ResultPayload, payload) {
		t.Fatalf("HTTP result payload=%v want=%v", execution.ResultPayload, payload)
	}
	receipt := execution.Receipt
	if receipt.Transport != "http-json-loopback-v0" ||
		receipt.Carrier.Kind != "http-reference" ||
		receipt.Carrier.Plugin != nil ||
		receipt.Carrier.LegacyCapabilityKind != "" ||
		receipt.Carrier.LegacyCapabilityID != "" {
		t.Fatalf("HTTP carrier leaked legacy process identity: %+v", receipt.Carrier)
	}
	if receipt.InvocationIdentity != genericDispatchIdentity(dispatch) ||
		receipt.RequestDigest != digestBytes(payload) ||
		receipt.ResultDigest != digestBytes(payload) {
		t.Fatalf("HTTP generic identity drifted: %+v", receipt)
	}
}

func TestProcessAndHTTPReceiptsPreserveSameGenericIdentity(t *testing.T) {
	dispatch := genericDispatch([]byte{0x00, 0xfe, 0xff, 'T', 'E', 'I'})

	processLauncher, processCommand := helperLauncher("echo")
	processExecution, err := invokeOverProcessTransport(
		context.Background(),
		processLauncher,
		processCommand,
		transportCarrier(),
		"cross-transport-process",
		dispatch,
	)
	if err != nil {
		t.Fatal(err)
	}

	server := newHTTPTransportServer(t)
	httpExecution, err := invokeOverHTTPTransport(
		context.Background(),
		server.Client(),
		server.URL,
		"cross-transport-http",
		dispatch,
	)
	if err != nil {
		t.Fatal(err)
	}

	left := processExecution.Receipt
	right := httpExecution.Receipt
	if left.InvocationIdentity != right.InvocationIdentity ||
		left.RequestDigest != right.RequestDigest ||
		left.ResultDigest != right.ResultDigest {
		t.Fatalf(
			"generic receipt identity differs by transport: process=%+v http=%+v",
			left,
			right,
		)
	}
	if left.Transport == right.Transport || left.Carrier.Kind == right.Carrier.Kind {
		t.Fatalf(
			"physical transport identity collapsed: process=%+v http=%+v",
			left.Carrier,
			right.Carrier,
		)
	}
}

func TestHTTPTransportRejectsImplementationIdentityDriftBeforeResponse(t *testing.T) {
	handler, err := NewHTTPTransportHandler(
		genericDispatchIdentity(genericDispatch(nil)),
		func(
			_ context.Context,
			dispatch GenericDispatch,
		) (GenericTransportResult, error) {
			result, err := echoHTTPTransportImplementation(context.Background(), dispatch)
			if err != nil {
				return GenericTransportResult{}, err
			}
			result.Provider.ImplementationID = "tampered.implementation"
			return result, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	_, err = invokeOverHTTPTransport(
		context.Background(),
		server.Client(),
		server.URL,
		"http-implementation-drift",
		genericDispatch([]byte("payload")),
	)
	if err == nil || !strings.Contains(err.Error(), "status 502") {
		t.Fatalf("HTTP implementation identity drift was accepted: %v", err)
	}
}

func TestHTTPTransportHostRejectsClientSelectedProviderBeforeImplementation(t *testing.T) {
	calls := 0
	expected := genericDispatchIdentity(genericDispatch(nil))
	handler, err := NewHTTPTransportHandler(
		expected,
		func(
			_ context.Context,
			dispatch GenericDispatch,
		) (GenericTransportResult, error) {
			calls++
			return echoHTTPTransportImplementation(context.Background(), dispatch)
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	dispatch := genericDispatch([]byte("payload"))
	dispatch.Provider.PluginID = "tei.reference.client-selected"
	_, err = invokeOverHTTPTransport(
		context.Background(),
		server.Client(),
		server.URL,
		"http-client-selected-provider",
		dispatch,
	)
	if err == nil || !strings.Contains(err.Error(), "status 409") {
		t.Fatalf("client-selected Provider identity was accepted: %v", err)
	}
	if calls != 0 {
		t.Fatalf("implementation ran before Host identity gate: calls=%d", calls)
	}
}

func TestHTTPTransportClientRejectsUntrustedResponseIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var dispatch GenericDispatch
		if err := json.NewDecoder(r.Body).Decode(&dispatch); err != nil {
			t.Errorf("decode dispatch: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		result, err := echoHTTPTransportImplementation(r.Context(), dispatch)
		if err != nil {
			t.Errorf("echo result: %v", err)
			http.Error(w, "bad gateway", http.StatusBadGateway)
			return
		}
		result.Target = "tampered-target"
		payload, err := json.Marshal(result)
		if err != nil {
			t.Errorf("encode result: %v", err)
			http.Error(w, "encode", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", httpTransportContentType)
		w.Header().Set(httpTransportRequestID, r.Header.Get(httpTransportRequestID))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	_, err := invokeOverHTTPTransport(
		context.Background(),
		server.Client(),
		server.URL,
		"http-untrusted-response",
		genericDispatch([]byte("payload")),
	)
	if err == nil || !strings.Contains(err.Error(), "identity drifted") {
		t.Fatalf("untrusted HTTP response identity was accepted: %v", err)
	}
}

type countingRoundTripper struct {
	calls int
}

func (r *countingRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	r.calls++
	return nil, fmt.Errorf("HTTP transport should not have been called")
}

func TestInvalidGenericDispatchStopsBeforeHTTPTransport(t *testing.T) {
	roundTripper := &countingRoundTripper{}
	client := &http.Client{Transport: roundTripper}
	dispatch := genericDispatch([]byte("payload"))
	dispatch.ResultShapeFingerprint = "invalid"

	_, err := invokeOverHTTPTransport(
		context.Background(),
		client,
		"http://example.invalid/capability",
		"http-invalid-dispatch",
		dispatch,
	)
	if err == nil || !strings.Contains(err.Error(), "shape identity") {
		t.Fatalf("invalid dispatch was not rejected before HTTP transport: %v", err)
	}
	if roundTripper.calls != 0 {
		t.Fatalf("invalid dispatch reached HTTP transport: calls=%d", roundTripper.calls)
	}
}

func TestHTTPTransportDoesNotClaimAuthenticationOrRuntimeAuthority(t *testing.T) {
	server := newHTTPTransportServer(t)
	execution, err := invokeOverHTTPTransport(
		context.Background(),
		server.Client(),
		server.URL,
		"http-authority-limits",
		genericDispatch([]byte("payload")),
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"transport-success-is-not-capability-conformance",
		"transport-success-is-not-runtime-authority",
		"transport-success-is-not-canonical-acceptance",
		"http-reference-does-not-provide-authentication-or-authorization",
		"http-json-is-one-realization-not-universal-capability-abi",
	} {
		found := false
		for _, got := range execution.Receipt.AuthorityLimits {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("HTTP authority limit %q missing: %+v", want, execution.Receipt.AuthorityLimits)
		}
	}
}

func TestHTTPTransportHandlerRequiresHostOwnedValidIdentity(t *testing.T) {
	if _, err := NewHTTPTransportHandler(
		GenericInvocationIdentity{},
		echoHTTPTransportImplementation,
	); err == nil || !strings.Contains(err.Error(), "expected identity") {
		t.Fatalf("invalid Host-owned identity was accepted: %v", err)
	}
}

func TestHTTPTransportAcceptsEquivalentJSONMediaTypeParameters(t *testing.T) {
	expected := genericDispatchIdentity(genericDispatch(nil))
	handler, err := NewHTTPTransportHandler(expected, echoHTTPTransportImplementation)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	dispatch := genericDispatch([]byte("payload"))
	payload, err := json.Marshal(dispatch)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(
		http.MethodPost,
		server.URL,
		bytes.NewReader(payload),
	)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(httpTransportRequestID, "media-type-params")
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("equivalent JSON media type status=%d", resp.StatusCode)
	}
}

func TestHTTPTransportHandlerRejectsMalformedBoundaryInputs(t *testing.T) {
	handler, err := NewHTTPTransportHandler(
		genericDispatchIdentity(genericDispatch(nil)),
		echoHTTPTransportImplementation,
	)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	for _, tc := range []struct {
		name        string
		method      string
		requestID   string
		contentType string
		body        string
		wantStatus  int
	}{
		{
			name:        "wrong-method",
			method:      http.MethodGet,
			requestID:   "request-1",
			contentType: httpTransportContentType,
			body:        "{}",
			wantStatus:  http.StatusMethodNotAllowed,
		},
		{
			name:        "missing-request-id",
			method:      http.MethodPost,
			contentType: httpTransportContentType,
			body:        "{}",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "wrong-content-type",
			method:      http.MethodPost,
			requestID:   "request-1",
			contentType: "text/plain",
			body:        "{}",
			wantStatus:  http.StatusUnsupportedMediaType,
		},
		{
			name:        "unknown-field",
			method:      http.MethodPost,
			requestID:   "request-1",
			contentType: httpTransportContentType,
			body:        `{"schema":"tei.reference.capability-transport-dispatch/v0","unknown":true}`,
			wantStatus:  http.StatusBadRequest,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, server.URL, strings.NewReader(tc.body))
			if err != nil {
				t.Fatal(err)
			}
			if tc.requestID != "" {
				req.Header.Set(httpTransportRequestID, tc.requestID)
			}
			if tc.contentType != "" {
				req.Header.Set("Content-Type", tc.contentType)
			}
			resp, err := server.Client().Do(req)
			if err != nil {
				t.Fatal(err)
			}
			_ = resp.Body.Close()
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("status=%d want=%d", resp.StatusCode, tc.wantStatus)
			}
		})
	}
}
