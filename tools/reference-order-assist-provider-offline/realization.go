// Package orderassistoffline implements steps 7 and 8 of the technical
// validation sample order-assist-provider: offline quotes on sales terminals
// with synchronisation, and discount approval with an external credit check.
// It is sample-specific code; nothing here belongs in internal/.
package orderassistoffline

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

// Runtime names the physical realization of the quote rule.
type Runtime string

const (
	RuntimeGoNative Runtime = "go-native"
	RuntimeNodeJS   Runtime = "node-js"
	RuntimeWASI     Runtime = "wasi-preview1-on-node"
)

// Generation describes one revision of the quote rule and the identity the
// existing process carrier must present for it.
type Generation struct {
	Number           int    `json:"generation"`
	RuleVersion      string `json:"ruleVersion"`
	ContractID       string `json:"contractId"`
	ImplementationID string `json:"implementationId"`
	PluginVersion    string `json:"pluginVersion"`
	contractRule     string
	requestRule      string
	resultRule       string
}

// Gen1 is the existing realization set (plugins/reference-order-quote and
// plugins/reference-order-quote-js), used unchanged. It does not report a
// rule version in its result; the version is the one its carrier identity
// accepts (it refuses any other implementation id).
var Gen1 = Generation{
	Number: 1, RuleVersion: "reference.order.quote-rule/v1",
	ContractID: "reference.order.quote-lines/v1", ImplementationID: "order-quote.v1", PluginVersion: "1.0.0",
	contractRule: "reference.order.quote-lines.contract/v1",
	requestRule:  "reference.order.quote-lines.request/v1",
	resultRule:   "reference.order.quote-lines.result/v1",
}

// Gen2 is this sample's revision (quotev2/): optional discount rate.
var Gen2 = Generation{
	Number: 2, RuleVersion: "reference.order.quote-rule/v2",
	ContractID: "reference.order.quote-lines/v2", ImplementationID: "order-quote.v2", PluginVersion: "2.0.0",
	contractRule: "reference.order.quote-lines.contract/v2",
	requestRule:  "reference.order.quote-lines.request/v2",
	resultRule:   "reference.order.quote-lines.result/v2",
}

// Generations by number.
var Generations = map[int]Generation{1: Gen1, 2: Gen2}

const (
	quotePluginID = "tei.reference.order_quote"
	carrierID     = "transport.order_quote_carrier"
	quoteTarget   = "reference/order_entry"
)

// Realization is one built, executable artifact of one generation.
type Realization struct {
	Name           string                  `json:"name"`
	Runtime        Runtime                 `json:"runtime"`
	Generation     int                     `json:"generation"`
	ArtifactPath   string                  `json:"-"`
	ArtifactDigest string                  `json:"artifactDigest"`
	HostDigest     string                  `json:"hostDigest,omitempty"` // WASI runner script
	Command        pruntime.ProcessCommand `json:"-"`
}

// QuoteResult is the union of the generation-1 and generation-2 result shapes.
type QuoteResult struct {
	Currency       string `json:"currency"`
	LineCount      int    `json:"lineCount"`
	SubtotalMinor  int64  `json:"subtotalMinor"`
	DiscountRateBp int64  `json:"discountRateBp"`
	DiscountMinor  int64  `json:"discountMinor"`
	NetMinor       int64  `json:"netMinor"`
	RuleVersion    string `json:"ruleVersion"`
}

// Outcome is what one realization returned for one request. Rejected is a
// business rejection (status=error from a process that ran); a launch failure
// is returned as an error instead and is never counted as a rejection.
type Outcome struct {
	Realization string      `json:"realization"`
	Rejected    bool        `json:"rejected"`
	Raw         []byte      `json:"-"`
	Result      QuoteResult `json:"result"`
}

func digestBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func digestFile(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return digestBytes(raw), nil
}

// RealizationSet holds the six realizations: three runtimes for each of the
// two generations.
type RealizationSet struct {
	ByName map[string]Realization
	Node   string
	Go     string
}

// Get returns the realization of a generation on a runtime.
func (s *RealizationSet) Get(gen int, rt Runtime) Realization {
	return s.ByName[realizationName(gen, rt)]
}

func realizationName(gen int, rt Runtime) string { return fmt.Sprintf("gen%d-%s", gen, rt) }

// ErrToolMissing reports that Go or Node.js is not available; callers must
// treat it as "not run", never as a pass.
var ErrToolMissing = errors.New("required tool is unavailable")

// BuildRealizations builds the six realizations into dir. Generation 1 is
// built from plugins/reference-order-quote (Go, native and WASI) and runs
// plugins/reference-order-quote-js (Node.js) unchanged; generation 2 is built
// from this package's quotev2/ the same way. Each Go build copies the source
// into a standalone module so it cannot import TEI packages.
func BuildRealizations(ctx context.Context, repoRoot, dir string) (*RealizationSet, error) {
	node, err := exec.LookPath("node")
	if err != nil {
		return nil, fmt.Errorf("%w: node: %v", ErrToolMissing, err)
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		if alt := filepath.Join(runtime.GOROOT(), "bin", "go"); fileExists(alt) {
			goBin = alt
		} else {
			return nil, fmt.Errorf("%w: go: %v", ErrToolMissing, err)
		}
	}
	runner := filepath.Join(repoRoot, "tools", "reference-capability-transport", "quote_wasip1_runner.mjs")
	runnerDigest, err := digestFile(runner)
	if err != nil {
		return nil, err
	}
	set := &RealizationSet{ByName: map[string]Realization{}, Node: node, Go: goBin}
	sources := map[int]struct{ goDir, js string }{
		1: {filepath.Join(repoRoot, "plugins", "reference-order-quote"), filepath.Join(repoRoot, "plugins", "reference-order-quote-js", "main.mjs")},
		2: {filepath.Join(repoRoot, "tools", "reference-order-assist-provider-offline", "quotev2"), filepath.Join(repoRoot, "tools", "reference-order-assist-provider-offline", "quotev2", "main.mjs")},
	}
	for gen := 1; gen <= 2; gen++ {
		src := sources[gen]
		module := filepath.Join(dir, fmt.Sprintf("gen%d-src", gen))
		if err := copyGoSources(src.goDir, module, fmt.Sprintf("example.invalid/order-quote-gen%d", gen)); err != nil {
			return nil, err
		}
		native := filepath.Join(dir, fmt.Sprintf("gen%d-native", gen))
		if err := goBuild(ctx, goBin, module, native, runtime.GOOS, runtime.GOARCH); err != nil {
			return nil, err
		}
		wasm := filepath.Join(dir, fmt.Sprintf("gen%d.wasm", gen))
		if err := goBuild(ctx, goBin, module, wasm, "wasip1", "wasm"); err != nil {
			return nil, err
		}
		for _, r := range []Realization{
			{Runtime: RuntimeGoNative, ArtifactPath: native, Command: pruntime.ProcessCommand{Executable: native}},
			{Runtime: RuntimeNodeJS, ArtifactPath: src.js, Command: pruntime.ProcessCommand{Executable: node, Args: []string{src.js}}},
			{Runtime: RuntimeWASI, ArtifactPath: wasm, HostDigest: runnerDigest, Command: pruntime.ProcessCommand{Executable: node, Args: []string{runner, wasm}}},
		} {
			r.Generation = gen
			r.Name = realizationName(gen, r.Runtime)
			if r.ArtifactDigest, err = digestFile(r.ArtifactPath); err != nil {
				return nil, err
			}
			set.ByName[r.Name] = r
		}
	}
	return set, nil
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }

func copyGoSources(src, dst, module string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	copied := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		if e.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symbolic link in provider source: %s", name)
		}
		raw, err := os.ReadFile(filepath.Join(src, name))
		if err != nil {
			return err
		}
		if bytes.Contains(raw, []byte(`"tei/`)) {
			return fmt.Errorf("provider source %s imports a TEI package", name)
		}
		if err := os.WriteFile(filepath.Join(dst, name), raw, 0o600); err != nil {
			return err
		}
		copied++
	}
	if copied == 0 {
		return fmt.Errorf("no Go source in %s", src)
	}
	return os.WriteFile(filepath.Join(dst, "go.mod"), []byte("module "+module+"\n\ngo 1.23\n"), 0o600)
}

func goBuild(ctx context.Context, goBin, dir, out, goos, goarch string) error {
	cmd := exec.CommandContext(ctx, goBin, "build", "-trimpath", "-o", out, ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOOS="+goos, "GOARCH="+goarch, "CGO_ENABLED=0", "GOENV=off", "GOWORK=off",
		"GOFLAGS=", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("build %s/%s: %v\n%s", goos, goarch, err, output)
	}
	return nil
}

type dispatchProvider struct {
	PluginID         string `json:"pluginId"`
	PluginVersion    string `json:"pluginVersion"`
	ImplementationID string `json:"implementationId"`
	ArtifactDigest   string `json:"artifactDigest"`
}

type dispatchEnvelope struct {
	Schema                  string           `json:"schema"`
	ContractID              string           `json:"contractId"`
	ContractFingerprint     string           `json:"contractFingerprint"`
	Target                  string           `json:"target"`
	Provider                dispatchProvider `json:"provider"`
	RequestShapeFingerprint string           `json:"requestShapeFingerprint"`
	ResultShapeFingerprint  string           `json:"resultShapeFingerprint"`
	PayloadCodec            string           `json:"payloadCodec"`
	Payload                 []byte           `json:"payload"`
}

// ErrArtifactChanged is returned when the artifact bytes on disk no longer
// match the digest the realization was registered with.
var ErrArtifactChanged = errors.New("realization artifact digest changed")

// Quote invokes the realization once over the existing process carrier with
// the identity of the given generation. The artifact is re-hashed first;
// there is still a window between this read and the process start (the
// same open gap as reference/wasip1-quote-realization).
func (r Realization) Quote(ctx context.Context, gen Generation, requestID string, payload []byte) (Outcome, error) {
	now, err := digestFile(r.ArtifactPath)
	if err != nil {
		return Outcome{}, err
	}
	if now != r.ArtifactDigest {
		return Outcome{}, fmt.Errorf("%w: %s", ErrArtifactChanged, r.Name)
	}
	dispatch := dispatchEnvelope{
		Schema:              "tei.reference.capability-transport-dispatch/v0",
		ContractID:          gen.ContractID,
		ContractFingerprint: digestBytes([]byte(gen.contractRule)),
		Target:              quoteTarget,
		Provider: dispatchProvider{PluginID: quotePluginID, PluginVersion: gen.PluginVersion,
			ImplementationID: gen.ImplementationID, ArtifactDigest: r.ArtifactDigest},
		RequestShapeFingerprint: digestBytes([]byte(gen.requestRule)),
		ResultShapeFingerprint:  digestBytes([]byte(gen.resultRule)),
		PayloadCodec:            "application/json",
		Payload:                 payload,
	}
	body, err := json.Marshal(dispatch)
	if err != nil {
		return Outcome{}, err
	}
	req := pruntime.Request{
		ProtocolVersion: pruntime.ProtocolVersion, RequestID: requestID,
		Plugin:         plugin.Identity{ID: core.MustCanonicalID(quotePluginID), Version: gen.PluginVersion},
		CapabilityKind: plugin.CapabilityContractQueryExecutor, CapabilityID: core.MustCanonicalID(carrierID),
		DescriptorFingerprint: digestBytes([]byte("reference.order-quote.transport-carrier/v1")),
		Operation:             "generic-dispatch-v0", Payload: body,
	}
	resp, err := (pruntime.StdioProcessLauncher{}).Invoke(ctx, r.Command, req)
	if err != nil {
		return Outcome{}, fmt.Errorf("%s: process invocation: %w", r.Name, err)
	}
	if resp.RequestID != requestID || resp.Plugin != req.Plugin || resp.CapabilityID != req.CapabilityID ||
		resp.DescriptorFingerprint != req.DescriptorFingerprint {
		return Outcome{}, fmt.Errorf("%s: carrier identity was not echoed", r.Name)
	}
	out := Outcome{Realization: r.Name}
	switch resp.Status {
	case pruntime.ResponseError:
		out.Rejected = true
		return out, nil
	case pruntime.ResponseOK:
	default:
		return Outcome{}, fmt.Errorf("%s: unexpected status %q", r.Name, resp.Status)
	}
	var result dispatchEnvelope
	dec := json.NewDecoder(bytes.NewReader(resp.Payload))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&result); err != nil {
		return Outcome{}, fmt.Errorf("%s: decode result envelope: %w", r.Name, err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return Outcome{}, fmt.Errorf("%s: trailing result data", r.Name)
	}
	if result.Schema != "tei.reference.capability-transport-result/v0" || result.ContractID != dispatch.ContractID ||
		result.ContractFingerprint != dispatch.ContractFingerprint || result.Target != dispatch.Target ||
		result.Provider != dispatch.Provider || result.RequestShapeFingerprint != dispatch.RequestShapeFingerprint ||
		result.ResultShapeFingerprint != dispatch.ResultShapeFingerprint || result.PayloadCodec != dispatch.PayloadCodec {
		return Outcome{}, fmt.Errorf("%s: result identity does not match the dispatch", r.Name)
	}
	out.Raw = append([]byte(nil), result.Payload...)
	rd := json.NewDecoder(bytes.NewReader(out.Raw))
	rd.DisallowUnknownFields()
	if err := rd.Decode(&out.Result); err != nil {
		return Outcome{}, fmt.Errorf("%s: decode quote result: %w", r.Name, err)
	}
	if gen.Number == 1 {
		// Generation 1 has no discount and no self-reported rule version.
		if out.Result.RuleVersion != "" || out.Result.DiscountRateBp != 0 || out.Result.DiscountMinor != 0 || out.Result.NetMinor != 0 {
			return Outcome{}, fmt.Errorf("%s: generation-1 result carries generation-2 members", r.Name)
		}
		out.Result.RuleVersion = gen.RuleVersion // implied by the carrier identity it accepted
		out.Result.NetMinor = out.Result.SubtotalMinor
	} else if out.Result.RuleVersion != gen.RuleVersion {
		return Outcome{}, fmt.Errorf("%s: reported rule version %q, dispatched %q", r.Name, out.Result.RuleVersion, gen.RuleVersion)
	}
	return out, nil
}
