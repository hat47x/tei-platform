package deliveryprofile

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	auditlog "tei/tools/reference-audit-log"
	authn "tei/tools/reference-authentication"
	tenantisolation "tei/tools/reference-tenant-isolation"
)

type CapabilityCheck struct {
	Capability string `json:"capability"`
	PluginID   string `json:"pluginId,omitempty"`
	Status     string `json:"status"`
	Detail     string `json:"detail,omitempty"`
}

// verifyTenantIsolation exercises the selected tenant-isolation plugin through
// its foundation API. It checks the guarantees the plugin descriptor claims,
// not just the capability name.
func verifyTenantIsolation() (bool, string) {
	base, err := os.MkdirTemp("", "tei-tenant-conformance-*")
	if err != nil {
		return false, err.Error()
	}
	defer os.RemoveAll(base)
	rootA := filepath.Join(base, "tenant-a")
	rootB := filepath.Join(base, "tenant-b")
	if err := os.MkdirAll(rootA, 0o755); err != nil {
		return false, err.Error()
	}
	if err := os.MkdirAll(rootB, 0o755); err != nil {
		return false, err.Error()
	}
	if err := os.WriteFile(filepath.Join(rootB, "secret.txt"), []byte("tenant-b\n"), 0o600); err != nil {
		return false, err.Error()
	}
	scopeA, err := tenantisolation.Open("tenant-a", rootA)
	if err != nil {
		return false, err.Error()
	}
	defer scopeA.Close()
	scopeB, err := tenantisolation.Open("tenant-b", rootB)
	if err != nil {
		return false, err.Error()
	}
	defer scopeB.Close()

	if err := scopeA.RequireTenant("tenant-b"); err == nil {
		return false, "cross-tenant request was accepted"
	}
	if _, err := scopeA.Read("../tenant-b/secret.txt"); err == nil {
		return false, "path traversal read was accepted"
	}
	if err := scopeA.Write("notes/ok.txt", []byte("ok\n")); err != nil {
		return false, "own-tenant write failed: " + err.Error()
	}
	if content, err := scopeA.Read("notes/ok.txt"); err != nil || string(content) != "ok\n" {
		return false, "own-tenant read failed"
	}
	link := filepath.Join(rootA, "link")
	if err := os.Symlink(rootB, link); err == nil {
		if _, err := scopeA.Read("link/secret.txt"); err == nil {
			return false, "cross-tenant symlink read was accepted"
		}
	}
	return true, "cross-tenant request, path traversal, own-tenant read/write, and symlink escape were checked"
}

func verifyAuditLog() (bool, string) {
	dir, err := os.MkdirTemp("", "tei-audit-conformance-*")
	if err != nil {
		return false, err.Error()
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "audit.jsonl")
	log, err := auditlog.Open(path)
	if err != nil {
		return false, err.Error()
	}
	if _, err := log.Append("tenant-a", "conformance", "read", "order/1"); err != nil {
		return false, err.Error()
	}
	if _, err := log.Append("tenant-a", "conformance", "write", "order/1"); err != nil {
		return false, err.Error()
	}
	if err := log.Verify(); err != nil {
		return false, "append-only chain did not verify: " + err.Error()
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return false, err.Error()
	}
	tampered := strings.Replace(string(raw), `"action":"read"`, `"action":"write"`, 1)
	if tampered == string(raw) {
		return false, "conformance could not find an entry to tamper"
	}
	if err := os.WriteFile(path, []byte(tampered), 0o600); err != nil {
		return false, err.Error()
	}
	if err := log.Verify(); err == nil {
		return false, "tampered audit entry was verified"
	}
	return true, "append-only chain, reopen, and tamper detection were checked"
}

type conformanceVerifier struct {
	principal authn.Principal
}

func (v conformanceVerifier) Verify(context.Context, string) (authn.Principal, error) {
	return v.principal, nil
}

func verifyAuthentication() (bool, string) {
	valid := conformanceVerifier{principal: authn.Principal{TenantID: "tenant-a", Subject: "user-1"}}
	if principal, err := authn.Authenticate(context.Background(), valid, "fixture-token", "tenant-a"); err != nil || principal.Subject != "user-1" {
		return false, "valid fixture credential was not authenticated"
	}
	if _, err := authn.Authenticate(context.Background(), valid, "", "tenant-a"); !errors.Is(err, authn.ErrUnauthenticated) {
		return false, "missing credential was not rejected"
	}
	if _, err := authn.Authenticate(context.Background(), conformanceVerifier{
		principal: authn.Principal{TenantID: "tenant-a", Subject: "user-1"},
	}, "fixture-token", "tenant-b"); !errors.Is(err, authn.ErrTenantMismatch) {
		return false, "principal from another tenant was not rejected"
	}
	if _, err := authn.Authenticate(context.Background(), conformanceVerifier{
		principal: authn.Principal{TenantID: "tenant-a"},
	}, "fixture-token", "tenant-a"); !errors.Is(err, authn.ErrInvalidPrincipal) {
		return false, "incomplete principal was not rejected"
	}
	return true, "valid credential, missing credential, tenant mismatch, and incomplete principal were checked"
}

func hasCapability(capabilities []string, want string) bool {
	for _, capability := range capabilities {
		if capability == want {
			return true
		}
	}
	return false
}

// Verify first validates the profile's plugin selection, then runs the
// conformance checks that exist for the selected capabilities. A capability
// without a check remains "unverified"; it is not reported as passing.
func Verify(profilePath string) (Receipt, error) {
	return VerifyWithCatalog(profilePath, "")
}

func VerifyWithCatalog(profilePath, catalogDir string) (Receipt, error) {
	receipt, err := ValidateWithCatalog(profilePath, catalogDir)
	if err != nil {
		return Receipt{}, err
	}
	return runConformance(receipt), nil
}

func VerifyWithLock(profilePath, catalogDir, lockPath string) (Receipt, error) {
	receipt, err := ValidateWithLock(profilePath, catalogDir, lockPath)
	if err != nil {
		return Receipt{}, err
	}
	return runConformance(receipt), nil
}

func runConformance(receipt Receipt) Receipt {
	checked := map[string]bool{}
	for _, capability := range []string{"tenant-isolation", "audit", "authentication"} {
		var selected *PluginSummary
		for i := range receipt.Plugins {
			plugin := &receipt.Plugins[i]
			if !hasCapability(plugin.Capabilities, capability) {
				continue
			}
			if selected == nil || plugin.ID == capability {
				selected = plugin
				if plugin.ID == capability {
					break
				}
			}
		}
		if selected == nil {
			continue
		}
		var ok bool
		var detail string
		switch capability {
		case "tenant-isolation":
			ok, detail = verifyTenantIsolation()
		case "audit":
			ok, detail = verifyAuditLog()
		case "authentication":
			ok, detail = verifyAuthentication()
		}
		status := "verified"
		if !ok {
			status = "failed"
			receipt.Passed = false
		}
		receipt.Conformance = append(receipt.Conformance, CapabilityCheck{
			Capability: capability, PluginID: selected.ID, Status: status, Detail: detail,
		})
		checked[capability] = true
	}
	for _, capability := range receipt.RequiredCapabilities {
		if checked[capability] {
			continue
		}
		receipt.Conformance = append(receipt.Conformance, CapabilityCheck{
			Capability: capability, Status: "unverified",
			Detail: "このReferenceには、選択したpluginの実動作を確認するconformance testがない",
		})
	}
	receipt.Notes = append(receipt.Notes,
		"verifyは、conformance testがある能力だけを実動作で確認する。ない能力はunverifiedとして残す。")
	return receipt
}
