// Diagnostic Go Host for a project-owned advisory and a Lit web component.
// It does not replace TEI's canonical validation, authorization or Action.
package main

import (
	"embed"
	"encoding/json"
	"flag"
	"io"
	"log"
	"mime"
	"net/http"
	"time"

	customeradvice "tei/plugins/reference-project-customer-advice"
)

//go:embed web/index.html web/advice-element.mjs web/advice-action.mjs web/order-entry-context.mjs web/lit-core.min.js web/LIT-LICENSE
var webAssets embed.FS

var staticFiles = map[string]struct {
	path        string
	contentType string
}{
	"/":                        {"web/index.html", "text/html; charset=utf-8"},
	"/advice-element.mjs":      {"web/advice-element.mjs", "text/javascript; charset=utf-8"},
	"/advice-action.mjs":       {"web/advice-action.mjs", "text/javascript; charset=utf-8"},
	"/order-entry-context.mjs": {"web/order-entry-context.mjs", "text/javascript; charset=utf-8"},
	"/lit-core.min.js":         {"web/lit-core.min.js", "text/javascript; charset=utf-8"},
	"/lit-license.txt":         {"web/LIT-LICENSE", "text/plain; charset=utf-8"},
}

func handler() http.Handler { return handlerWithContext(nil) }

func handlerWithContext(contextHandler http.Handler) http.Handler {
	bridge, err := newProjectAdvisoryBridge()
	if err != nil {
		panic(err)
	}
	mux := http.NewServeMux()
	if contextHandler != nil {
		mux.Handle("/project/order-entry-context", contextHandler)
	}
	mux.HandleFunc("/project/advice", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// This diagnostic server is loopback-only. Never accept cross-origin
		// writes or treat an advisory request as an authenticated TEI Action.
		if origin := r.Header.Get("Origin"); origin != "" &&
			origin != "http://"+r.Host {
			http.Error(w, "origin not accepted", http.StatusForbidden)
			return
		}
		mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || mediaType != "application/json" {
			http.Error(w, "JSON required", http.StatusUnsupportedMediaType)
			return
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
		decoder.DisallowUnknownFields()
		var input customeradvice.Input
		if err := decoder.Decode(&input); err != nil {
			http.Error(w, "invalid advisory request", http.StatusBadRequest)
			return
		}
		var trailing any
		if err := decoder.Decode(&trailing); err != io.EOF {
			http.Error(w, "trailing advisory data", http.StatusBadRequest)
			return
		}
		result, err := bridge.Evaluate(input)
		if err != nil {
			http.Error(w, "invalid advisory input", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(result)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		asset, ok := staticFiles[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := webAssets.ReadFile(asset.path)
		if err != nil {
			http.Error(w, "missing reference asset", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", asset.contentType)
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; connect-src 'self'; style-src 'unsafe-inline'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		if r.Method == http.MethodGet {
			_, _ = w.Write(body)
		}
	})
	return mux
}

func main() {
	upstream := flag.String("order-entry-api", "", "explicit loopback origin for read-only order_entry reference")
	flag.Parse()
	var contextHandler http.Handler
	if *upstream != "" {
		var err error
		contextHandler, err = newOrderEntryContextHandler(*upstream)
		if err != nil {
			log.Fatal(err)
		}
	}
	// Restrict this diagnostic endpoint to the local machine.
	server := &http.Server{
		Addr:              "127.0.0.1:8787",
		Handler:           handlerWithContext(contextHandler),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Print("reference project advice: http://127.0.0.1:8787")
	log.Fatal(server.ListenAndServe())
}
