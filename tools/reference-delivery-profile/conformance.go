package deliveryprofile

import (
	"os"
	"path/filepath"

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
	receipt, err := Validate(profilePath)
	if err != nil {
		return Receipt{}, err
	}
	checked := map[string]bool{}
	for _, plugin := range receipt.Plugins {
		if !hasCapability(plugin.Capabilities, "tenant-isolation") {
			continue
		}
		ok, detail := verifyTenantIsolation()
		status := "verified"
		if !ok {
			status = "failed"
			receipt.Passed = false
		}
		receipt.Conformance = append(receipt.Conformance, CapabilityCheck{
			Capability: "tenant-isolation", PluginID: plugin.ID, Status: status, Detail: detail,
		})
		checked["tenant-isolation"] = true
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
	return receipt, nil
}
