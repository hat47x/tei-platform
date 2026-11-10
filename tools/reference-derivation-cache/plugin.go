package derivationcache

import (
	"fmt"
)

// PluginIdentity is the exact plugin identity under test.
type PluginIdentity struct {
	ID             string
	Version        string
	DefinitionHash string
}

// PluginInput is the cache invalidation input for plugin changes.
type PluginInput struct {
	CacheRequested bool
	// CachedPlugin is the plugin identity captured at cache time.
	CachedPlugin PluginIdentity
	// CurrentPlugin is the current plugin identity.
	CurrentPlugin PluginIdentity
	// CachedLockFingerprint is the plugin lock fingerprint at cache time.
	CachedLockFingerprint string
	// CurrentLockFingerprint is the current plugin lock fingerprint.
	CurrentLockFingerprint string
}

// PluginReport is the read-only result for plugin invalidation.
type PluginReport struct {
	Code      string
	Accepted  bool
	ValueNull bool
	Value     string
	Message   string
}

func rejectPlugin(code, message string) PluginReport {
	return PluginReport{Code: code, ValueNull: true, Message: message}
}

// ReusePlugin checks whether a cached value is still valid given plugin
// identity and lock changes.
func ReusePlugin(in PluginInput) PluginReport {
	if !in.CacheRequested {
		return PluginReport{Code: CodeDirectEvaluation, Accepted: true, Value: "",
			Message: "単純な純粋計算にPluginの鮮度を必須にしない。"}
	}
	if in.CachedPlugin.ID != in.CurrentPlugin.ID {
		return rejectPlugin(CodeStaleRule,
			fmt.Sprintf("Plugin ID %s に対するキャッシュが %s。Plugin自体が異なる。",
				in.CurrentPlugin.ID, in.CachedPlugin.ID))
	}
	if in.CachedPlugin.Version != in.CurrentPlugin.Version {
		return rejectPlugin(CodeStaleRule,
			fmt.Sprintf("Plugin版 %s に対するキャッシュが版 %s。",
				in.CurrentPlugin.Version, in.CachedPlugin.Version))
	}
	if in.CachedPlugin.DefinitionHash != in.CurrentPlugin.DefinitionHash {
		return rejectPlugin(CodeStaleRule,
			"Plugin定義の内容が変わった。実装・contract・source closureの変更を無視しない。")
	}
	if in.CachedLockFingerprint != "" && in.CurrentLockFingerprint != "" &&
		in.CachedLockFingerprint != in.CurrentLockFingerprint {
		return rejectPlugin(CodeStaleInput,
			"Plugin Lockのfingerprintが変わった。同じPluginでも受理済みlockは別物として扱う。")
	}
	return PluginReport{Code: CodeCacheReused, Accepted: true, Value: "",
		Message: "Plugin ID・版・定義・lock fingerprintが一致した再利用。"}
}
