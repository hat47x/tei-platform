// Package generatorrevocationreview checks that a registry's revoked generator
// entries have a separate, syntactically complete approval record whose target
// identity matches. It does not verify the approver's authority or trust.
package generatorrevocationreview

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	generatorselection "tei/tools/reference-generator-selection"
)

const (
	ApprovalSchema = "tei.reference.generator-revocation-approvals/v0"
	ReviewSchema   = "tei.reference.generator-revocation-review/v0"
)

var digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

type Approval struct {
	ID               string `json:"id"`
	Version          string `json:"version"`
	SourceDigest     string `json:"sourceDigest"`
	Authority        string `json:"authority"`
	Role             string `json:"role"`
	EvidenceDigest   string `json:"evidenceDigest"`
	ApprovedRevision string `json:"approvedRevision"`
	Reason           string `json:"reason,omitempty"`
}

type Approvals struct {
	Schema          string     `json:"schema"`
	ContractID      string     `json:"contractId"`
	ContractVersion string     `json:"contractVersion"`
	ContractDigest  string     `json:"contractDigest"`
	Approvals       []Approval `json:"approvals"`
}

type ReviewEntry struct {
	ID           string `json:"id"`
	Version      string `json:"version"`
	Status       string `json:"status"`
	SourceDigest string `json:"sourceDigest"`
	State        string `json:"state"`
	Authority    string `json:"authority,omitempty"`
	Role         string `json:"role,omitempty"`
	Revision     string `json:"approvedRevision,omitempty"`
	Reason       string `json:"reason,omitempty"`
}

type Receipt struct {
	Schema              string        `json:"schema"`
	ContractID          string        `json:"contractId"`
	ContractVersion     string        `json:"contractVersion"`
	RegistryDigest      string        `json:"registryDigest"`
	ApprovalsDigest     string        `json:"approvalsDigest"`
	Entries             []ReviewEntry `json:"entries"`
	Approved            []string      `json:"approved"`
	Unapproved          []string      `json:"unapproved"`
	Invalid             []string      `json:"invalid"`
	Passed              bool          `json:"passed"`
	GrantsAuthority     bool          `json:"grantsAuthority"`
	VerifiesIssuerTrust bool          `json:"verifiesIssuerTrust"`
	Notes               []string      `json:"notes"`
}

func readJSON(path string, into any) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(into); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if decoder.More() {
		return nil, fmt.Errorf("%s: trailing JSON value", path)
	}
	return raw, nil
}

// Review compares revoked registry entries with a separate approval file. It
// only checks schema, target identity, required fields, and duplicate/unknown
// approvals. It does not authenticate the authority, verify signatures, or
// grant execution authority.
func Review(registryPath, contractPath, approvalsPath string) (Receipt, error) {
	registry, err := generatorselection.LoadRegistry(registryPath, contractPath)
	if err != nil {
		return Receipt{}, err
	}
	registryRaw, err := os.ReadFile(registryPath)
	if err != nil {
		return Receipt{}, err
	}
	contractRaw, err := os.ReadFile(contractPath)
	if err != nil {
		return Receipt{}, err
	}
	var approvals Approvals
	approvalsRaw, err := readJSON(approvalsPath, &approvals)
	if err != nil {
		return Receipt{}, err
	}
	if approvals.Schema != ApprovalSchema {
		return Receipt{}, fmt.Errorf("approvals schema %q is not %q", approvals.Schema, ApprovalSchema)
	}
	if approvals.ContractID != registry.ContractID || approvals.ContractVersion != registry.ContractVersion {
		return Receipt{}, fmt.Errorf("approvals target %s/%s, registry targets %s/%s",
			approvals.ContractID, approvals.ContractVersion, registry.ContractID, registry.ContractVersion)
	}
	if approvals.ContractDigest != digestBytes(contractRaw) {
		return Receipt{}, fmt.Errorf("approvals contract digest does not match %s", contractPath)
	}
	byKey := map[string]Approval{}
	for _, approval := range approvals.Approvals {
		if err := validateApproval(approval); err != nil {
			return Receipt{}, err
		}
		key := approval.ID + "@" + approval.Version
		if _, exists := byKey[key]; exists {
			return Receipt{}, fmt.Errorf("duplicate approval for %s", key)
		}
		byKey[key] = approval
	}
	receipt := Receipt{
		Schema: ReviewSchema, ContractID: registry.ContractID, ContractVersion: registry.ContractVersion,
		RegistryDigest: digestBytes(registryRaw), ApprovalsDigest: digestBytes(approvalsRaw),
		Entries: []ReviewEntry{}, Approved: []string{}, Unapproved: []string{}, Invalid: []string{},
		GrantsAuthority: false, VerifiesIssuerTrust: false,
		Notes: []string{
			"registryのrevoked entryと、registry編集とは別の承認recordの対象一致を検査する。",
			"承認者のAuthority・issuer trust・署名・失効判断の妥当性は検証しない。",
			"passed=trueは承認recordが揃っていることだけを示し、実行権限や失効の受理を与えない。",
		},
	}
	registryKeys := map[string]bool{}
	for _, entry := range registry.Entries {
		key := entry.ID + "@" + entry.Version
		registryKeys[key] = true
		review := ReviewEntry{ID: entry.ID, Version: entry.Version, Status: entry.Status, SourceDigest: entry.SourceDigest}
		approval, hasApproval := byKey[key]
		if entry.Status == generatorselection.RegistryRevoked {
			if !hasApproval {
				review.State = "unapproved"
				receipt.Unapproved = append(receipt.Unapproved, key)
			} else if approval.SourceDigest != entry.SourceDigest {
				review.State = "unapproved"
				review.Reason = "approval source digest does not match the revoked entry"
				receipt.Unapproved = append(receipt.Unapproved, key)
			} else {
				review.State = "approved-syntax-only"
				review.Authority = approval.Authority
				review.Role = approval.Role
				review.Revision = approval.ApprovedRevision
				receipt.Approved = append(receipt.Approved, key)
			}
		} else if hasApproval {
			review.State = "invalid"
			review.Reason = "approval exists for a non-revoked entry"
			receipt.Invalid = append(receipt.Invalid, key)
		}
		receipt.Entries = append(receipt.Entries, review)
	}
	for key, approval := range byKey {
		if !registryKeys[key] {
			receipt.Invalid = append(receipt.Invalid, key)
			receipt.Entries = append(receipt.Entries, ReviewEntry{
				ID: approval.ID, Version: approval.Version, Status: "missing",
				SourceDigest: approval.SourceDigest, State: "invalid", Reason: "approval target is absent from the registry",
			})
		}
	}
	receipt.Passed = len(receipt.Unapproved) == 0 && len(receipt.Invalid) == 0
	return receipt, nil
}

func validateApproval(approval Approval) error {
	for name, value := range map[string]string{
		"id": approval.ID, "version": approval.Version, "authority": approval.Authority,
		"role": approval.Role, "approvedRevision": approval.ApprovedRevision,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("approval %s is required", name)
		}
	}
	for name, value := range map[string]string{
		"sourceDigest": approval.SourceDigest, "evidenceDigest": approval.EvidenceDigest,
	} {
		if !digestPattern.MatchString(value) {
			return fmt.Errorf("approval %s is not a sha256 digest", name)
		}
	}
	return nil
}

func digestBytes(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}
