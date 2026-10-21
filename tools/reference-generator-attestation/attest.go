// Package generatorattestation signs and verifies a canonical statement about
// one generator candidate. It proves that a signer key signed the statement; it
// does not establish the signer's authority or the candidate's correctness.
package generatorattestation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	generatorselection "tei/tools/reference-generator-selection"
)

const (
	StatementSchema = "tei.reference.generator-candidate-statement/v0"
	ReceiptSchema   = "tei.reference.generator-attestation-receipt/v0"
	Namespace       = "tei-generator"
)

type Statement struct {
	Schema            string `json:"schema"`
	CandidateID       string `json:"candidateId"`
	Language          string `json:"language"`
	Origin            string `json:"origin"`
	SourceDigest      string `json:"sourceDigest"`
	BinaryDigest      string `json:"binaryDigest,omitempty"`
	ContractID        string `json:"contractId"`
	ContractVersion   string `json:"contractVersion"`
	ContractDigest    string `json:"contractDigest,omitempty"`
	WorkspaceRevision string `json:"workspaceRevision"`
	FixtureDigest     string `json:"fixtureDigest"`
	SuiteVersion      string `json:"suiteVersion"`
	BuildInputDigest  string `json:"buildInputDigest,omitempty"`
	ReceiptDigest     string `json:"receiptDigest"`
}

type Receipt struct {
	Schema            string   `json:"schema"`
	StatementDigest   string   `json:"statementDigest"`
	SignatureDigest   string   `json:"signatureDigest,omitempty"`
	Principal         string   `json:"principal,omitempty"`
	Verified          bool     `json:"verified"`
	SignaturePath     string   `json:"signaturePath,omitempty"`
	Reason            string   `json:"reason,omitempty"`
	GrantsAuthority   bool     `json:"grantsAuthority"`
	VerifiesTrustRoot bool     `json:"verifiesTrustRoot"`
	Notes             []string `json:"notes"`
}

func digest(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func MarshalStatement(statement Statement) ([]byte, error) {
	raw, err := json.MarshalIndent(statement, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

// BuildStatementFromSelection reads one selection receipt and creates a
// canonical statement for the named candidate. It does not sign or verify.
func BuildStatementFromSelection(receiptPath, candidateID string) (Statement, []byte, error) {
	raw, err := os.ReadFile(receiptPath)
	if err != nil {
		return Statement{}, nil, err
	}
	var selection generatorselection.Receipt
	if err := json.Unmarshal(raw, &selection); err != nil {
		return Statement{}, nil, fmt.Errorf("%s: %w", receiptPath, err)
	}
	var candidate *generatorselection.CandidateReceipt
	for i := range selection.Candidates {
		if selection.Candidates[i].ID == candidateID {
			candidate = &selection.Candidates[i]
			break
		}
	}
	if candidate == nil {
		return Statement{}, nil, fmt.Errorf("candidate %q is not in the selection receipt", candidateID)
	}
	contractDigest, buildInputDigest := "", ""
	if candidate.Generator != nil {
		contractDigest = candidate.Generator.ContractDigest
	}
	if candidate.Generation != nil && contractDigest == "" {
		contractDigest = candidate.Generation.ContractDigest
	}
	if candidate.BuildEvidence != nil {
		buildInputDigest = candidate.BuildEvidence.BuildInputDigest
	}
	statement := Statement{
		Schema: StatementSchema, CandidateID: candidate.ID, Language: candidate.Language,
		Origin: candidate.Origin, SourceDigest: candidate.SourceDigest, BinaryDigest: candidate.BinaryDigest,
		ContractID: selection.ContractID, ContractVersion: selection.ContractVersion,
		ContractDigest: contractDigest, WorkspaceRevision: selection.WorkspaceRevision,
		FixtureDigest: selection.FixtureDigest, SuiteVersion: selection.SuiteVersion,
		BuildInputDigest: buildInputDigest, ReceiptDigest: digest(raw),
	}
	encoded, err := MarshalStatement(statement)
	if err != nil {
		return Statement{}, nil, err
	}
	return statement, encoded, nil
}

// Sign signs a statement file with ssh-keygen. The key must be usable without
// an interactive passphrase in this reference. The default output is
// "<statement>.sig".
func Sign(ctx context.Context, statementPath, keyPath, outputPath, namespace string) (Receipt, error) {
	statement, err := os.ReadFile(statementPath)
	if err != nil {
		return Receipt{}, err
	}
	if namespace == "" {
		namespace = Namespace
	}
	cmd := exec.CommandContext(ctx, "ssh-keygen", "-Y", "sign", "-f", keyPath, "-n", namespace, statementPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return Receipt{}, fmt.Errorf("ssh-keygen sign failed: %w: %s", err, output)
	}
	defaultSignature := statementPath + ".sig"
	signaturePath := defaultSignature
	if outputPath != "" && filepath.Clean(outputPath) != filepath.Clean(defaultSignature) {
		signature, err := os.ReadFile(defaultSignature)
		if err != nil {
			return Receipt{}, err
		}
		if err := os.WriteFile(outputPath, signature, 0o644); err != nil {
			return Receipt{}, err
		}
		_ = os.Remove(defaultSignature)
		signaturePath = outputPath
	}
	signature, err := os.ReadFile(signaturePath)
	if err != nil {
		return Receipt{}, err
	}
	return Receipt{
		Schema: ReceiptSchema, StatementDigest: digest(statement), SignatureDigest: digest(signature),
		SignaturePath: signaturePath, Verified: false,
		GrantsAuthority: false, VerifiesTrustRoot: false,
		Notes: []string{
			"statementをssh-keygenで署名した。署名はstatementのbyte列に束縛される。",
			"署名者のAuthority・trust root・候補の正しさは証明しない。",
		},
	}, nil
}

// Verify checks a detached signature against caller-provided allowed signers.
// The allowed signers file is the caller's trust input; this package does not
// decide whether that trust is appropriate.
func Verify(ctx context.Context, statementPath, signaturePath, allowedSignersPath, identity, namespace string) (Receipt, error) {
	statement, err := os.ReadFile(statementPath)
	if err != nil {
		return Receipt{}, err
	}
	if namespace == "" {
		namespace = Namespace
	}
	cmd := exec.CommandContext(ctx, "ssh-keygen", "-Y", "verify",
		"-f", allowedSignersPath, "-I", identity, "-n", namespace, "-s", signaturePath)
	cmd.Stdin = bytes.NewReader(statement)
	output, err := cmd.CombinedOutput()
	receipt := Receipt{
		Schema: ReceiptSchema, StatementDigest: digest(statement), Principal: identity,
		SignaturePath: signaturePath, GrantsAuthority: false, VerifiesTrustRoot: false,
		Notes: []string{
			"allowed signers fileとidentityは呼出し側が指定したtrust入力である。",
			"署名の検証は、署名者のAuthority・意味の受理・実行許可を証明しない。",
		},
	}
	if signature, readErr := os.ReadFile(signaturePath); readErr == nil {
		receipt.SignatureDigest = digest(signature)
	}
	if err != nil {
		receipt.Reason = strings.TrimSpace(string(output))
		if receipt.Reason == "" {
			receipt.Reason = err.Error()
		}
		return receipt, nil
	}
	receipt.Verified = true
	return receipt, nil
}
