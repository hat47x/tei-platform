package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

// This is a diagnostic read-only consumer of the actual first Web Target API.
// It does not define another TEI Action, contract resolver, or authorization.
var loopbackAPI = regexp.MustCompile(`^http://127\.0\.0\.1:([1-9][0-9]{0,4})$`)

type orderEntryContext struct {
	Schema              string `json:"schema"`
	ApplicationID       string `json:"applicationId"`
	ScenarioID          string `json:"scenarioId"`
	SelectedRowIndex    int    `json:"selectedRowIndex"`
	TargetBuildRevision string `json:"targetBuildRevision"`
}

func newOrderEntryContextHandler(origin string) (http.Handler, error) {
	m := loopbackAPI.FindStringSubmatch(origin)
	if m == nil {
		return nil, errors.New("order_entry source must be an explicit loopback HTTP origin")
	}
	port, err := strconv.Atoi(m[1])
	if err != nil || port > 65535 {
		return nil, errors.New("order_entry source port is invalid")
	}
	client := &http.Client{
		Timeout: 3 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Intentionally do not forward Browser credentials or headers to the
		// reference read endpoint. An authenticated product Action is separate.
		request, err := http.NewRequestWithContext(r.Context(), http.MethodGet,
			origin+"/scenario/read", nil)
		if err != nil {
			http.Error(w, "cannot build read request", http.StatusBadGateway)
			return
		}
		response, err := client.Do(request)
		if err != nil {
			http.Error(w, "order_entry reference is unavailable", http.StatusBadGateway)
			return
		}
		defer response.Body.Close()
		mediaType, _, parseErr := mime.ParseMediaType(response.Header.Get("Content-Type"))
		if response.StatusCode != http.StatusOK || parseErr != nil || mediaType != "application/json" {
			http.Error(w, "unexpected order_entry response", http.StatusBadGateway)
			return
		}
		var source struct {
			TargetBuildRevision string `json:"targetBuildRevision"`
			Result              struct {
				ApplicationID    string `json:"applicationId"`
				ScenarioID       string `json:"scenarioId"`
				SelectedRowIndex int    `json:"selectedRowIndex"`
			} `json:"result"`
		}
		raw, err := io.ReadAll(io.LimitReader(response.Body, (1<<16)+1))
		if err != nil || len(raw) > 1<<16 {
			http.Error(w, "order_entry read result exceeds limit", http.StatusBadGateway)
			return
		}
		decoder := json.NewDecoder(bytes.NewReader(raw))
		if err := decoder.Decode(&source); err != nil {
			http.Error(w, "invalid order_entry read result", http.StatusBadGateway)
			return
		}
		var trailing any
		if err := decoder.Decode(&trailing); err != io.EOF ||
			source.Result.ApplicationID != "order_entry" ||
			source.Result.ScenarioID == "" ||
			source.Result.SelectedRowIndex < 0 ||
			source.TargetBuildRevision == "" {
			http.Error(w, "unrecognized order_entry read result", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(orderEntryContext{
			Schema:              "tei.reference.order-entry-context/v0",
			ApplicationID:       source.Result.ApplicationID,
			ScenarioID:          source.Result.ScenarioID,
			SelectedRowIndex:    source.Result.SelectedRowIndex,
			TargetBuildRevision: source.TargetBuildRevision,
		})
	}), nil
}
