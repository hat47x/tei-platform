// Manifest capture observes a live First Web Target projection without
// changing the Target or promoting a runtime HTTP response to accepted source.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	runtimesnapshot "tei/tools/reference-runtime-snapshot"
)

const maxCaptureBytes = 8 << 20

type captureOptions struct {
	URL      string
	Identity runtimesnapshot.Identity
}

func parseCaptureOptions(args []string) (captureOptions, error) {
	var o captureOptions
	fs := flag.NewFlagSet("reference-first-web-target-manifest-capture", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&o.URL, "url", "", "literal 127.0.0.1 First Web Target /manifest.json URL")
	fs.StringVar(&o.Identity.ApplicationID, "application", "", "expected Application ID")
	fs.StringVar(&o.Identity.HandoverBaselineID, "baseline-id", "", "expected baseline ID")
	fs.StringVar(&o.Identity.BaselineRepositoryRevision, "baseline-revision", "", "expected handover revision")
	fs.StringVar(&o.Identity.TargetBuildRevision, "target-revision", "", "expected Target build revision")
	if err := fs.Parse(args); err != nil {
		return captureOptions{}, err
	}
	if fs.NArg() != 0 {
		return captureOptions{}, fmt.Errorf("unexpected positional arguments")
	}
	if strings.TrimSpace(o.Identity.ApplicationID) == "" ||
		strings.TrimSpace(o.Identity.HandoverBaselineID) == "" ||
		strings.TrimSpace(o.Identity.BaselineRepositoryRevision) == "" ||
		strings.TrimSpace(o.Identity.TargetBuildRevision) == "" {
		return captureOptions{}, fmt.Errorf("explicit Application and Handover B/R/T are required")
	}
	if err := validateCaptureURL(o.URL); err != nil {
		return captureOptions{}, err
	}
	return o, nil
}

func validateCaptureURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "http" || u.User != nil || u.Hostname() != "127.0.0.1" ||
		u.Path != "/manifest.json" || u.RawQuery != "" || u.Fragment != "" || u.RawFragment != "" ||
		u.Opaque != "" || u.RawPath != "" {
		return fmt.Errorf("capture URL must be literal http://127.0.0.1:<port>/manifest.json")
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1 || port > 65535 ||
		rawURL != "http://"+net.JoinHostPort("127.0.0.1", strconv.Itoa(port))+"/manifest.json" {
		return fmt.Errorf("capture URL must have a literal loopback host and valid port")
	}
	return nil
}

// capture fetches an unaccepted observation from a local reference Target.
// Its computed digest is an observation, NOT an independently approved lock.
func capture(ctx context.Context, client *http.Client, opts captureOptions) ([]byte, string, error) {
	if err := validateCaptureURL(opts.URL); err != nil {
		return nil, "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, opts.URL, nil)
	if err != nil {
		return nil, "", err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, "", fmt.Errorf("observe first Web Target: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("first Web Target returned HTTP %d", response.StatusCode)
	}
	contentType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		return nil, "", fmt.Errorf("first Web Target did not return application/json")
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxCaptureBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("read First Web Target manifest: %w", err)
	}
	if len(raw) == 0 || len(raw) > maxCaptureBytes {
		return nil, "", fmt.Errorf("First Web Target manifest is empty or too large")
	}
	sum := sha256.Sum256(raw)
	digest := hex.EncodeToString(sum[:])
	expected := opts.Identity
	expected.ManifestSHA256 = digest
	if _, err := runtimesnapshot.Accept(raw, expected); err != nil {
		return nil, "", fmt.Errorf("observed manifest identity is inconsistent: %w", err)
	}
	return raw, digest, nil
}

func runCapture(args []string, out, notices io.Writer) error {
	opts, err := parseCaptureOptions(args)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{Proxy: nil},
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	raw, digest, err := capture(ctx, client, opts)
	if err != nil {
		return err
	}
	if n, err := out.Write(raw); err != nil {
		return fmt.Errorf("write manifest observation: %w", err)
	} else if n != len(raw) {
		return io.ErrShortWrite
	}
	_, _ = fmt.Fprintf(notices, "OBSERVED SHA-256: %s (not an approved Build Artifact lock)\n", digest)
	return nil
}

func main() {
	if err := runCapture(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
