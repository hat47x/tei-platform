//go:build tei_api_only && tei_order_only && !tei_repair_only

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// This server type intentionally contains no html/template imports, template
// fields, index page or custom-submit rendering. The existing Driver-backed
// runners and authorization boundary are unchanged.
type webTargetServer struct {
	projection                  webTargetProjection
	readScenario                readScenarioRunner
	commandScenario             commandScenarioRunner
	protectedSessionConfig      *protectedSessionCommandConfig
	protectedSessionHandler     http.Handler
	protectedSessionOptionCount int
	customerLookup              customerLookupRunner
}

type applicationBound interface {
	ApplicationID() string
}

type webTargetOption func(*webTargetServer)

func withCustomerLookup(runner customerLookupRunner) webTargetOption {
	return func(s *webTargetServer) { s.customerLookup = runner }
}

func requireApplicationMatch(kind string, runner any, want string) error {
	bound, ok := runner.(applicationBound)
	if !ok {
		return fmt.Errorf("%s runner does not declare its Application; refusing to serve Application %s", kind, want)
	}
	if got := bound.ApplicationID(); got != want {
		return fmt.Errorf("%s runner executes Application %s but the projection is for Application %s", kind, got, want)
	}
	return nil
}

func newWebTargetServer(
	projection webTargetProjection,
	readScenario readScenarioRunner,
	commandScenario commandScenarioRunner,
	options ...webTargetOption,
) (*webTargetServer, error) {
	server := &webTargetServer{
		projection:      projection,
		readScenario:    readScenario,
		commandScenario: commandScenario,
	}
	for _, option := range options {
		option(server)
	}
	if server.readScenario != nil {
		if err := requireApplicationMatch("read scenario", server.readScenario, projection.ApplicationID); err != nil {
			return nil, err
		}
	}
	if server.commandScenario != nil {
		if err := requireApplicationMatch("command scenario", server.commandScenario, projection.ApplicationID); err != nil {
			return nil, err
		}
	}
	if server.customerLookup != nil {
		if err := requireApplicationMatch("customer lookup", server.customerLookup, projection.ApplicationID); err != nil {
			return nil, err
		}
	}
	if err := server.configureProtectedSessionCommand(); err != nil {
		return nil, err
	}
	return server, nil
}

// API-only intentionally has no HTML route; clients receive a real 404.
func (s *webTargetServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/manifest.json", s.handleManifest)
	mux.HandleFunc("/healthz", s.handleHealth)
	registerCombinedScenarioRoutes(mux, s)
	return mux
}

func (s *webTargetServer) handleManifest(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(s.projection); err != nil {
		http.Error(w, "encode target manifest", http.StatusInternalServerError)
	}
}

func (s *webTargetServer) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}
