package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	httpTransportContentType = "application/json"
	httpTransportRequestID    = "X-TEI-Request-ID"
	httpTransportMaxBytes     = 8 << 20
)

type HTTPTransportImplementation func(
	context.Context,
	GenericDispatch,
) (GenericTransportResult, error)

func NewHTTPTransportHandler(
	expectedIdentity GenericInvocationIdentity,
	implementation HTTPTransportImplementation,
) (http.Handler, error) {
	if err := validateGenericInvocationIdentity(expectedIdentity); err != nil {
		return nil, fmt.Errorf("HTTP capability transport expected identity: %w", err)
	}
	if implementation == nil {
		return nil, fmt.Errorf("HTTP capability transport requires implementation")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		requestID := strings.TrimSpace(r.Header.Get(httpTransportRequestID))
		if requestID == "" {
			http.Error(w, "missing request id", http.StatusBadRequest)
			return
		}
		contentType := r.Header.Get("Content-Type")
		if contentType != httpTransportContentType {
			http.Error(w, "unsupported content type", http.StatusUnsupportedMediaType)
			return
		}

		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, httpTransportMaxBytes))
		if err != nil {
			http.Error(w, "read transport request", http.StatusBadRequest)
			return
		}
		var dispatch GenericDispatch
		if err := decodeJSONExact(body, &dispatch); err != nil {
			http.Error(w, "invalid transport dispatch", http.StatusBadRequest)
			return
		}
		if err := validateGenericDispatch(dispatch); err != nil {
			http.Error(w, "invalid transport dispatch", http.StatusBadRequest)
			return
		}
		if genericDispatchIdentity(dispatch) != expectedIdentity {
			http.Error(w, "transport dispatch identity is not accepted by host", http.StatusConflict)
			return
		}

		result, err := implementation(r.Context(), dispatch)
		if err != nil {
			http.Error(w, "transport implementation failed", http.StatusBadGateway)
			return
		}
		if err := validateGenericTransportResult(dispatch, result); err != nil {
			http.Error(w, "transport implementation returned invalid identity", http.StatusBadGateway)
			return
		}
		payload, err := json.Marshal(result)
		if err != nil {
			http.Error(w, "encode transport result", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", httpTransportContentType)
		w.Header().Set(httpTransportRequestID, requestID)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}), nil
}

func invokeOverHTTPTransport(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	requestID string,
	dispatch GenericDispatch,
) (TransportExecution, error) {
	if ctx == nil {
		return TransportExecution{}, fmt.Errorf("HTTP capability transport requires context")
	}
	if client == nil {
		return TransportExecution{}, fmt.Errorf("HTTP capability transport requires client")
	}
	if strings.TrimSpace(endpoint) == "" {
		return TransportExecution{}, fmt.Errorf("HTTP capability transport requires endpoint")
	}
	if strings.TrimSpace(requestID) == "" {
		return TransportExecution{}, fmt.Errorf("HTTP capability transport requires request id")
	}
	if err := validateGenericDispatch(dispatch); err != nil {
		return TransportExecution{}, err
	}

	payload, err := json.Marshal(dispatch)
	if err != nil {
		return TransportExecution{}, fmt.Errorf("encode HTTP transport dispatch: %w", err)
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		bytes.NewReader(payload),
	)
	if err != nil {
		return TransportExecution{}, fmt.Errorf("build HTTP transport request: %w", err)
	}
	req.Header.Set("Content-Type", httpTransportContentType)
	req.Header.Set(httpTransportRequestID, requestID)

	resp, err := client.Do(req)
	if err != nil {
		return TransportExecution{}, fmt.Errorf("invoke HTTP capability transport: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return TransportExecution{}, fmt.Errorf(
			"HTTP capability transport returned status %d",
			resp.StatusCode,
		)
	}
	if got := strings.TrimSpace(resp.Header.Get(httpTransportRequestID)); got != requestID {
		return TransportExecution{}, fmt.Errorf(
			"HTTP capability transport request id %q does not match %q",
			got,
			requestID,
		)
	}
	if resp.Header.Get("Content-Type") != httpTransportContentType {
		return TransportExecution{}, fmt.Errorf(
			"HTTP capability transport content type %q is not %q",
			resp.Header.Get("Content-Type"),
			httpTransportContentType,
		)
	}
	responsePayload, err := io.ReadAll(io.LimitReader(resp.Body, httpTransportMaxBytes+1))
	if err != nil {
		return TransportExecution{}, fmt.Errorf("read HTTP transport result: %w", err)
	}
	if len(responsePayload) > httpTransportMaxBytes {
		return TransportExecution{}, fmt.Errorf(
			"HTTP capability transport response exceeds %d bytes",
			httpTransportMaxBytes,
		)
	}

	var result GenericTransportResult
	if err := decodeJSONExact(responsePayload, &result); err != nil {
		return TransportExecution{}, fmt.Errorf("decode HTTP transport result: %w", err)
	}
	if err := validateGenericTransportResult(dispatch, result); err != nil {
		return TransportExecution{}, err
	}

	return TransportExecution{
		ResultPayload: append([]byte(nil), result.Payload...),
		Receipt: newTransportReceipt(
			"http-json-loopback-v0",
			TransportCarrierReceipt{Kind: "http-reference"},
			requestID,
			dispatch,
			result,
			[]string{
				"transport-success-is-not-capability-conformance",
				"transport-success-is-not-runtime-authority",
				"transport-success-is-not-canonical-acceptance",
				"http-reference-does-not-provide-authentication-or-authorization",
				"http-json-is-one-realization-not-universal-capability-abi",
			},
		),
	}, nil
}
