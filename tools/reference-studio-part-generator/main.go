// reference-studio-part-generator realizes one small TEI Studio part from a
// TEI-owned contract Source with the existing deterministic transform Runtime
// Engine. It is a child for `tei self-build`: it writes one artifact and
// reports what it read. It adds no transform operation, resolver, or Host.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"

	"tei/internal/plugin/transform"
)

const (
	reportSchema  = "tei.reference.studio-part-generator/v0"
	contractInput = "contract"
	claimBoundary = "one deterministic transform evaluation over one contract Source; " +
		"not acceptance, not Runtime Authority, not proof that a Desktop used the artifact"
)

// The transform IR joins contract strings into program text and has no
// escaping operation. The Host therefore admits only strings that cannot leave
// a quoted literal or a property name. This is input admission for this
// generator, not a general Source language.
var codeSafeScalar = regexp.MustCompile(`^[A-Za-z0-9_.:-]+$`)

type engineIdentity struct {
	Package   string `json:"package"`
	Profile   string `json:"profile"`
	GoVersion string `json:"goVersion"`
	Target    string `json:"target"`
}

type report struct {
	SchemaVersion string `json:"schemaVersion"`
	// ArtifactDigest is the digest of the bytes this process wrote.
	ArtifactDigest string `json:"artifactDigest"`
	// TransformBytesDigest is what was read; TransformSourceFingerprint is the
	// normalized Generator Source identity. They answer different questions.
	TransformBytesDigest       string `json:"transformBytesDigest"`
	TransformSourceFingerprint string `json:"transformSourceFingerprint"`
	TransformID                string `json:"transformId"`
	TransformVersion           string `json:"transformVersion"`
	// ContractBytesDigest is what was read; ContractCanonicalDigest is the
	// value the generated part carries as its own provenance.
	ContractBytesDigest     string         `json:"contractBytesDigest"`
	ContractCanonicalDigest string         `json:"contractCanonicalDigest"`
	Engine                  engineIdentity `json:"engine"`
	Steps                   int            `json:"steps"`
	StateDir                string         `json:"stateDir,omitempty"`
	ClaimBoundary           string         `json:"claimBoundary"`
}

func digest(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func admitContract(value transform.Value, path string) error {
	switch value.Kind {
	case transform.ValueString:
		if !codeSafeScalar.MatchString(value.String) {
			return fmt.Errorf("contract %s is not a code-safe scalar", path)
		}
	case transform.ValueObject:
		for _, field := range value.Object {
			if !codeSafeScalar.MatchString(field.Name) {
				return fmt.Errorf("contract %s has a field name that is not a code-safe scalar", path)
			}
			if err := admitContract(field.Value, path+"."+field.Name); err != nil {
				return err
			}
		}
	case transform.ValueNumber:
	default:
		return fmt.Errorf("contract %s has unsupported kind %s", path, value.Kind)
	}
	return nil
}

// ContractCanonicalDigest is the identity a generated part carries. It is
// computed here independently of the transform so a consumer can check the
// part against the Source without trusting the part's own text.
func contractCanonicalDigest(contractRaw []byte) (string, error) {
	value, err := transform.ParseJSON(contractRaw)
	if err != nil {
		return "", fmt.Errorf("contract Source JSON: %w", err)
	}
	canonical, err := value.JSON()
	if err != nil {
		return "", err
	}
	return digest(canonical), nil
}

// generate evaluates the Generator Source over the contract Source. It reads
// nothing else: no clock, environment, file system, or network.
func generate(transformRaw, contractRaw []byte) ([]byte, report, error) {
	source, err := transform.ParseSourceJSON(transformRaw)
	if err != nil {
		return nil, report{}, err
	}
	if len(source.Inputs) != 1 || source.Inputs[0].Name != contractInput ||
		source.Inputs[0].Kind != transform.InputObject {
		return nil, report{}, fmt.Errorf("generator Source must declare exactly one object input named %q", contractInput)
	}
	fingerprint, err := transform.SourceFingerprint(source)
	if err != nil {
		return nil, report{}, err
	}
	contract, err := transform.ParseJSON(contractRaw)
	if err != nil {
		return nil, report{}, fmt.Errorf("contract Source JSON: %w", err)
	}
	if contract.Kind != transform.ValueObject {
		return nil, report{}, errors.New("contract Source must be one JSON object")
	}
	if err := admitContract(contract, contractInput); err != nil {
		return nil, report{}, err
	}
	canonicalDigest, err := contractCanonicalDigest(contractRaw)
	if err != nil {
		return nil, report{}, err
	}
	result, err := transform.Evaluate(source, map[string]transform.Value{contractInput: contract})
	if err != nil {
		return nil, report{}, err
	}
	if result.Output.Kind != transform.ValueString {
		return nil, report{}, fmt.Errorf("generator Source must produce a string, got %s", result.Output.Kind)
	}
	artifact := []byte(result.Output.String)
	return artifact, report{
		SchemaVersion:              reportSchema,
		ArtifactDigest:             digest(artifact),
		TransformBytesDigest:       digest(transformRaw),
		TransformSourceFingerprint: fingerprint,
		TransformID:                string(source.Identity.ID),
		TransformVersion:           source.Identity.Version,
		ContractBytesDigest:        digest(contractRaw),
		ContractCanonicalDigest:    canonicalDigest,
		Engine: engineIdentity{
			Package:   "tei/internal/plugin/transform",
			Profile:   source.Profile,
			GoVersion: runtime.Version(),
			Target:    runtime.GOOS + "/" + runtime.GOARCH,
		},
		Steps:         result.Steps,
		ClaimBoundary: claimBoundary,
	}, nil
}

func run(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("reference-studio-part-generator", flag.ContinueOnError)
	transformPath := fs.String("transform", "", "exact deterministic transform Source (the Generator Source)")
	contractPath := fs.String("contract", "", "exact contract Source of the Studio part")
	artifactOut := fs.String("artifact-out", "", "path to write the generated part to; never overwritten")
	stateDir := fs.String("state", "", "state directory named by the entry; reported back, not used")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 || *transformPath == "" || *contractPath == "" || *artifactOut == "" {
		return errors.New("-transform, -contract and -artifact-out are required; no positional arguments")
	}
	transformRaw, err := os.ReadFile(*transformPath)
	if err != nil {
		return fmt.Errorf("read generator Source: %w", err)
	}
	contractRaw, err := os.ReadFile(*contractPath)
	if err != nil {
		return fmt.Errorf("read contract Source: %w", err)
	}
	artifact, result, err := generate(transformRaw, contractRaw)
	if err != nil {
		return err
	}
	// no-clobber: an existing file would let a stale part pass as this run's.
	file, err := os.OpenFile(*artifactOut, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("write generated part: %w", err)
	}
	if _, err := file.Write(artifact); err != nil {
		file.Close()
		return fmt.Errorf("write generated part: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("write generated part: %w", err)
	}
	result.StateDir = *stateDir
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
