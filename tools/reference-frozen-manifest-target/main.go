// Frozen-manifest-only is an intentionally read-only Go executable entry.
// It cannot execute business operations or select TEI Providers at runtime.
package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"

	runtimesnapshot "tei/tools/reference-runtime-snapshot"
)

const maxManifestBytes = 8 << 20

type options struct {
	manifest string
	identity runtimesnapshot.Identity
	port     int
}

func parseOptions(args []string) (options, error) {
	var cfg options
	flags := flag.NewFlagSet("reference-frozen-manifest-target", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&cfg.manifest, "manifest", "", "path to an independently pinned JSON manifest")
	flags.StringVar(&cfg.identity.ApplicationID, "application", "", "expected Application ID")
	flags.StringVar(&cfg.identity.HandoverBaselineID, "baseline-id", "", "expected Handover baseline ID")
	flags.StringVar(&cfg.identity.BaselineRepositoryRevision, "baseline-revision", "", "expected baseline Git revision")
	flags.StringVar(&cfg.identity.TargetBuildRevision, "target-revision", "", "expected target build revision")
	flags.StringVar(&cfg.identity.ManifestSHA256, "sha256", "", "independently pinned manifest SHA-256")
	flags.IntVar(&cfg.port, "port", 8091, "local-only HTTP port")
	if err := flags.Parse(args); err != nil {
		return options{}, err
	}
	if flags.NArg() != 0 {
		return options{}, fmt.Errorf("positional arguments are not supported")
	}
	if cfg.manifest == "" {
		return options{}, fmt.Errorf("-manifest is required")
	}
	if cfg.port < 1 || cfg.port > 65535 {
		return options{}, fmt.Errorf("-port must be between 1 and 65535")
	}
	return cfg, nil
}

func loadSnapshot(file string, expected runtimesnapshot.Identity) (runtimesnapshot.Snapshot, error) {
	input, err := os.Open(file)
	if err != nil {
		return runtimesnapshot.Snapshot{}, fmt.Errorf("open frozen manifest: %w", err)
	}
	defer input.Close()
	// Even a pinned input must not permit unbounded allocation on the host.
	raw, err := io.ReadAll(io.LimitReader(input, maxManifestBytes+1))
	if err != nil {
		return runtimesnapshot.Snapshot{}, fmt.Errorf("read frozen manifest: %w", err)
	}
	if len(raw) > maxManifestBytes {
		return runtimesnapshot.Snapshot{}, fmt.Errorf("frozen manifest exceeds %d bytes", maxManifestBytes)
	}
	return runtimesnapshot.Accept(raw, expected)
}

func serve(args []string, output io.Writer) error {
	cfg, err := parseOptions(args)
	if err != nil {
		return err
	}
	snapshot, err := loadSnapshot(cfg.manifest, cfg.identity)
	if err != nil {
		return err
	}
	// Intentionally do not offer a public-interface bind option. This is a
	// non-production, local-only Reference without authentication or TLS.
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(cfg.port))
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("listen on loopback: %w", err)
	}
	defer listener.Close()
	fmt.Fprintf(output, "Read-only frozen manifest Reference on http://%s (no business endpoints)\n", listener.Addr())
	return http.Serve(listener, snapshot.Handler())
}

func main() {
	if err := serve(os.Args[1:], os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
