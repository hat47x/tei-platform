//go:build tei_api_only && tei_order_only && !tei_repair_only

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// This server type intentionally contains no html/template imports, template
// fields, index page or custom-submit rendering. The existing Driver-backed
// runners and authorization boundary are unchanged.
const webTargetServerHasPresentation = false

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
	// A diagnostic observation of the exact running Linux process image.
	// This is NOT signed provenance, a credential, or a Host deployment permit.
	mux.HandleFunc("/diagnostic-executable.json", handleDiagnosticExecutable)
	registerCombinedScenarioRoutes(mux, s)
	// API-only diagnostic route guards the same business Runner at dispatch.
	mux.HandleFunc(apiBoundCommandPath, s.handleBoundCommandScenario)
	mux.HandleFunc(apiBoundReadPath, s.handleBoundReadScenario)
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

// handleDiagnosticExecutable lets the first-Web diagnostic Browser Host compare
// the actual API process image to the *existing* E1 artifact measurement.
// Linux /proc/self/exe also resolves the sealed memfd snapshot used by the
// synthetic Host. An API process can lie about its own bytes; this endpoint is
// an accidental-misbinding check, NOT remote attestation or authentication.
func handleDiagnosticExecutable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	f, err := os.Open("/proc/self/exe")
	if err != nil {
		http.Error(w, "diagnostic executable unavailable", http.StatusServiceUnavailable)
		return
	}
	defer f.Close()
	const maxExecutableBytes = 256 << 20
	sha := sha256.New()
	size, err := io.Copy(sha, io.LimitReader(f, maxExecutableBytes+1))
	if err != nil || size < 1 || size > maxExecutableBytes {
		http.Error(w, "diagnostic executable observation unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(struct {
		SHA256 string `json:"sha256"`
		Bytes  int64  `json:"bytes"`
	}{SHA256: hex.EncodeToString(sha.Sum(nil)), Bytes: size}); err != nil {
		// A broken transport cannot constitute successful provenance evidence.
		return
	}
}
