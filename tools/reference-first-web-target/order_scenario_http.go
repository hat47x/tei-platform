//go:build !tei_repair_only

package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
)

// Link markup is a compiled presentation fragment, not a capability grant.
// Both the legacy combined diagnostic and the order-only target reuse it
// without embedding order-entry routes into repair-only server sources.
const orderScenarioIndexFooterHTML = `<footer>JSON projection: <a href="/manifest.json">/manifest.json</a> · read-only Driver slice: <a href="/scenario/read">/scenario/read</a> · submit Command: <code>POST /scenario/submit</code> · Custom UI: <a href="/custom-submit">/custom-submit</a> · health: <a href="/healthz">/healthz</a></footer>`

// Keep transport registration beside order_entry-specific handlers.
// The combined reference still mounts both route groups for compatibility;
// this file separation alone does not create app-specific Go package closure.
func registerOrderEntryScenarioRoutes(mux *http.ServeMux, s *webTargetServer) {
	mux.HandleFunc("/scenario/read", s.handleReadScenario)
	mux.HandleFunc("/scenario/submit", s.handleCommandScenario)
	mux.HandleFunc("/custom-submit", s.handleCustomSubmit)
}

type webReadScenarioResponse struct {
	HandoverBaselineID         string             `json:"handoverBaselineId"`
	BaselineRepositoryRevision string             `json:"baselineRepositoryRevision"`
	TargetBuildRevision        string             `json:"targetBuildRevision"`
	TargetAdapter              string             `json:"targetAdapter"`
	Host                       string             `json:"host"`
	Result                     readScenarioResult `json:"result"`
}

type webCommandScenarioRequest struct {
	LoginID    string `json:"loginId"`
	Credential string `json:"credential"`
	From       string `json:"from"`
	RowIndex   int    `json:"row"`
}

type webCommandScenarioResponse struct {
	HandoverBaselineID         string                `json:"handoverBaselineId"`
	BaselineRepositoryRevision string                `json:"baselineRepositoryRevision"`
	TargetBuildRevision        string                `json:"targetBuildRevision"`
	TargetAdapter              string                `json:"targetAdapter"`
	Host                       string                `json:"host"`
	Result                     commandScenarioResult `json:"result"`
}

type webCommandScenarioErrorResponse struct {
	HandoverBaselineID         string                `json:"handoverBaselineId"`
	BaselineRepositoryRevision string                `json:"baselineRepositoryRevision"`
	TargetBuildRevision        string                `json:"targetBuildRevision"`
	TargetAdapter              string                `json:"targetAdapter"`
	Host                       string                `json:"host"`
	Result                     commandScenarioResult `json:"result"`
	ExecutionError             string                `json:"executionError"`
}

func (s *webTargetServer) handleReadScenario(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.readScenario == nil {
		http.Error(w, "read scenario is not configured", http.StatusServiceUnavailable)
		return
	}

	rowIndex := 0
	if raw := r.URL.Query().Get("row"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			http.Error(w, "row must be a non-negative integer", http.StatusBadRequest)
			return
		}
		rowIndex = parsed
	}
	from := r.URL.Query().Get("from")
	if from == "" {
		from = "2026-09-01T00:00:00Z"
	}

	result, err := s.readScenario.Run(r.Context(), readScenarioInput{
		From:     from,
		RowIndex: rowIndex,
	})
	if err != nil {
		http.Error(w, "execute read scenario: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}

	response := webReadScenarioResponse{
		HandoverBaselineID:         s.projection.HandoverBaselineID,
		BaselineRepositoryRevision: s.projection.BaselineRepositoryRevision,
		TargetBuildRevision:        s.projection.TargetBuildRevision,
		TargetAdapter:              s.projection.TargetAdapter,
		Host:                       s.projection.Host,
		Result:                     result,
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(response); err != nil {
		http.Error(w, "encode read scenario result", http.StatusInternalServerError)
	}
}

func (s *webTargetServer) handleCommandScenario(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.commandScenario == nil {
		http.Error(w, "command scenario is not configured", http.StatusServiceUnavailable)
		return
	}

	var request webCommandScenarioRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "invalid command request", http.StatusBadRequest)
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		http.Error(w, "command request must contain one JSON document", http.StatusBadRequest)
		return
	}
	if request.From == "" {
		request.From = "2026-09-01T00:00:00Z"
	}

	result, err := s.commandScenario.RunCommand(r.Context(), commandScenarioInput{
		LoginID:    request.LoginID,
		Credential: request.Credential,
		From:       request.From,
		RowIndex:   request.RowIndex,
	})
	if err != nil {
		response := webCommandScenarioErrorResponse{
			HandoverBaselineID:         s.projection.HandoverBaselineID,
			BaselineRepositoryRevision: s.projection.BaselineRepositoryRevision,
			TargetBuildRevision:        s.projection.TargetBuildRevision,
			TargetAdapter:              s.projection.TargetAdapter,
			Host:                       s.projection.Host,
			Result:                     result,
			ExecutionError:             err.Error(),
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusUnprocessableEntity)
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		if encodeErr := encoder.Encode(response); encodeErr != nil {
			http.Error(w, "encode command scenario execution error", http.StatusInternalServerError)
		}
		return
	}

	response := webCommandScenarioResponse{
		HandoverBaselineID:         s.projection.HandoverBaselineID,
		BaselineRepositoryRevision: s.projection.BaselineRepositoryRevision,
		TargetBuildRevision:        s.projection.TargetBuildRevision,
		TargetAdapter:              s.projection.TargetAdapter,
		Host:                       s.projection.Host,
		Result:                     result,
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(response); err != nil {
		http.Error(w, "encode command scenario result", http.StatusInternalServerError)
	}
}
