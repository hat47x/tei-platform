package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type passState struct {
	Passed bool `json:"passed"`
}

type visualState struct {
	Status   string `json:"status"`
	Required bool   `json:"required"`
}

type sourceArtifactDigest struct {
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
}

type semanticFunctionalSummary struct {
	EvidenceNature             string                 `json:"evidenceNature"`
	RunIdentity                string                 `json:"runIdentity"`
	HandoverBaselineID         string                 `json:"handoverBaselineId"`
	BaselineRepositoryRevision string                 `json:"baselineRepositoryRevision"`
	TargetBuildRevision        string                 `json:"targetBuildRevision"`
	TargetAdapter              string                 `json:"targetAdapter"`
	Host                       string                 `json:"host"`
	SourceQualification        passState              `json:"sourceQualification"`
	SemanticConformance        passState              `json:"semanticConformance"`
	FunctionalEvidence         passState              `json:"functionalEvidence"`
	VisualEvidence             visualState            `json:"visualEvidence"`
	Artifacts                  []sourceArtifactDigest `json:"artifacts"`
	GateCReady                 bool                   `json:"gateCReady"`
}

type replayCapture struct {
	ID             string `json:"id"`
	SourceArtifact string `json:"sourceArtifact"`
	SourceSHA256   string `json:"sourceSHA256"`
	Route          string `json:"route"`
	HTML           []byte `json:"-"`
}

type replayManifest struct {
	EvidenceNature             string          `json:"evidenceNature"`
	SourceRunIdentity          string          `json:"sourceRunIdentity"`
	HandoverBaselineID         string          `json:"handoverBaselineId"`
	BaselineRepositoryRevision string          `json:"baselineRepositoryRevision"`
	TargetBuildRevision        string          `json:"targetBuildRevision"`
	TargetAdapter              string          `json:"targetAdapter"`
	Host                       string          `json:"host"`
	Captures                   []replayCapture `json:"captures"`
	AuthorityLimits            []string        `json:"authorityLimits"`
}

type replayBundle struct {
	manifest replayManifest
	byID     map[string]replayCapture
}

var requiredReplaySources = map[string]string{
	"projection-index":                       "projection-index.html",
	"custom-submit-initial":                  "custom-submit-get.html",
	"custom-submit-success":                  "custom-submit-success.html",
	"custom-submit-authorization-denied":     "custom-submit-authorization-denied.html",
	"custom-submit-validation-not-satisfied": "custom-submit-validation-not-satisfied.html",
	"custom-submit-execution-error":          "custom-submit-execution-error.html",
}

func main() {
	var (
		summaryPath = flag.String("semantic-functional-summary", "", "Semantic / Functional summary JSON")
		artifactDir = flag.String("artifact-dir", "", "Directory containing same-run raw Target HTML artifacts")
		listen      = flag.String("listen", "127.0.0.1:18181", "Loopback listen address")
	)
	flag.Parse()

	if *summaryPath == "" || *artifactDir == "" {
		fatalf("-semantic-functional-summary and -artifact-dir are required")
	}
	if !strings.HasPrefix(*listen, "127.0.0.1:") && !strings.HasPrefix(*listen, "localhost:") {
		fatalf("visual replay server must bind to loopback")
	}

	bundle, err := loadReplayBundle(*summaryPath, *artifactDir)
	if err != nil {
		fatalf("%v", err)
	}
	server := &http.Server{
		Addr:    *listen,
		Handler: newReplayHandler(bundle),
	}
	log.Printf(
		"TEI First Web Target Visual replay: http://%s/capture-manifest.json sourceRun=%s T=%s",
		*listen,
		bundle.manifest.SourceRunIdentity,
		bundle.manifest.TargetBuildRevision,
	)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func loadReplayBundle(summaryPath, artifactDir string) (replayBundle, error) {
	var summary semanticFunctionalSummary
	if err := readJSON(summaryPath, &summary); err != nil {
		return replayBundle{}, err
	}
	if summary.EvidenceNature != "first-web-target-semantic-functional-evidence" ||
		!summary.SourceQualification.Passed ||
		!summary.SemanticConformance.Passed ||
		!summary.FunctionalEvidence.Passed ||
		summary.VisualEvidence.Status != "pending" ||
		!summary.VisualEvidence.Required ||
		summary.GateCReady {
		return replayBundle{}, fmt.Errorf("Semantic / Functional summary is not a valid Visual replay source")
	}
	if summary.RunIdentity == "" ||
		summary.HandoverBaselineID == "" ||
		summary.BaselineRepositoryRevision == "" ||
		summary.TargetBuildRevision == "" ||
		summary.TargetAdapter == "" ||
		summary.Host == "" {
		return replayBundle{}, fmt.Errorf("Semantic / Functional provenance is incomplete")
	}

	digests := make(map[string]string, len(summary.Artifacts))
	for _, artifact := range summary.Artifacts {
		if artifact.File == "" || artifact.SHA256 == "" {
			return replayBundle{}, fmt.Errorf("source artifact digest is incomplete: %+v", artifact)
		}
		if _, exists := digests[artifact.File]; exists {
			return replayBundle{}, fmt.Errorf("source artifact digest %s is duplicated", artifact.File)
		}
		digests[artifact.File] = artifact.SHA256
	}

	resolvedDir, err := filepath.Abs(artifactDir)
	if err != nil {
		return replayBundle{}, fmt.Errorf("resolve artifact directory: %w", err)
	}
	resolvedDir, err = filepath.EvalSymlinks(resolvedDir)
	if err != nil {
		return replayBundle{}, fmt.Errorf("resolve artifact directory symlinks: %w", err)
	}

	ids := make([]string, 0, len(requiredReplaySources))
	for id := range requiredReplaySources {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	captures := make([]replayCapture, 0, len(ids))
	byID := make(map[string]replayCapture, len(ids))
	for _, id := range ids {
		source := requiredReplaySources[id]
		expectedDigest, ok := digests[source]
		if !ok {
			return replayBundle{}, fmt.Errorf("Semantic / Functional summary is missing replay source digest %s", source)
		}
		path := filepath.Join(resolvedDir, source)
		resolvedPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			return replayBundle{}, fmt.Errorf("resolve replay source %s: %w", source, err)
		}
		if filepath.Dir(resolvedPath) != resolvedDir {
			return replayBundle{}, fmt.Errorf("replay source %s resolves outside artifact directory", source)
		}
		info, err := os.Stat(resolvedPath)
		if err != nil {
			return replayBundle{}, fmt.Errorf("stat replay source %s: %w", source, err)
		}
		if !info.Mode().IsRegular() {
			return replayBundle{}, fmt.Errorf("replay source %s is not a regular file", source)
		}
		data, err := os.ReadFile(resolvedPath)
		if err != nil {
			return replayBundle{}, fmt.Errorf("read replay source %s: %w", source, err)
		}
		actual := sha256.Sum256(data)
		actualDigest := hex.EncodeToString(actual[:])
		if actualDigest != expectedDigest {
			return replayBundle{}, fmt.Errorf("replay source %s digest does not match Semantic / Functional summary", source)
		}
		for _, forbidden := range []string{
			"editor-secret",
			"viewer-secret",
			"locked-secret",
			"disabled-secret",
			"credential-ref://legacy/",
		} {
			if strings.Contains(string(data), forbidden) {
				return replayBundle{}, fmt.Errorf("replay source %s contains credential material %q", source, forbidden)
			}
		}

		capture := replayCapture{
			ID:             id,
			SourceArtifact: source,
			SourceSHA256:   expectedDigest,
			Route:          "/capture/" + id,
			HTML:           append([]byte(nil), data...),
		}
		captures = append(captures, capture)
		byID[id] = capture
	}

	return replayBundle{
		manifest: replayManifest{
			EvidenceNature:             "first-web-target-visual-replay",
			SourceRunIdentity:          summary.RunIdentity,
			HandoverBaselineID:         summary.HandoverBaselineID,
			BaselineRepositoryRevision: summary.BaselineRepositoryRevision,
			TargetBuildRevision:        summary.TargetBuildRevision,
			TargetAdapter:              summary.TargetAdapter,
			Host:                       summary.Host,
			Captures:                   captures,
			AuthorityLimits: []string{
				"replay-serves-exact-same-run-target-rendered-html-bytes",
				"replay-does-not-reexecute-credentials-or-business-actions",
				"replay-html-is-not-visual-fidelity-evidence",
				"screenshot-and-attributed-evaluation-remain-required",
				"replay-helper-does-not-declare-gate-c-ready",
			},
		},
		byID: byID,
	}, nil
}

func newReplayHandler(bundle replayBundle) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/capture-manifest.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(bundle.manifest); err != nil {
			http.Error(w, "encode replay manifest", http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/capture/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/capture/")
		capture, ok := bundle.byID[id]
		if !ok || id == "" || strings.Contains(id, "/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-TEI-Visual-Capture-ID", capture.ID)
		w.Header().Set("X-TEI-Source-SHA256", capture.SourceSHA256)
		w.Header().Set("X-TEI-Target-Build-Revision", bundle.manifest.TargetBuildRevision)
		_, _ = w.Write(capture.HTML)
	})
	return mux
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
