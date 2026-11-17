package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"

	"tei/internal/core"
	"tei/internal/operations"
	"tei/internal/plugin"
)

// pluginPlanJSON serves `plugin-plan` (#4377). It derives a read-only change
// plan from the Workspace the same way every read-only command does, and
// returns the plan the existing resolver and declarations support. It applies
// nothing: no Source edit, no setting change, no data deletion, no Artifact
// change. Applying a plan is a separate, qualified operation (#2550/#2553).
func pluginPlanJSON(out io.Writer, workspacePath, worldPath, environmentID, action, pluginID, pluginLockPath, targetLockPath, sourceIndexPath string, extraArgs int) (int, error) {
	if (workspacePath == "") == (worldPath == "") {
		return resultInvalidArgument(out, "plugin-plan", "plugin-plan requires exactly one of -workspace or -world")
	}
	if extraArgs != 0 {
		return resultInvalidArgument(out, "plugin-plan", "plugin-plan does not accept positional arguments")
	}
	if environmentID == "" {
		return resultInvalidArgument(out, "plugin-plan", "plugin-plan requires -environment")
	}
	if action == "" {
		return resultInvalidArgument(out, "plugin-plan", "plugin-plan requires -action")
	}
	if pluginID == "" {
		return resultInvalidArgument(out, "plugin-plan", "plugin-plan requires -plugin")
	}

	envID, envErr := core.ParseCanonicalID(environmentID)
	if envErr != nil {
		return resultInvalidArgument(out, "plugin-plan", "invalid -environment: "+envErr.Error())
	}
	targetID, pluginErr := core.ParseCanonicalID(pluginID)
	if pluginErr != nil {
		return resultInvalidArgument(out, "plugin-plan", "invalid -plugin: "+pluginErr.Error())
	}

	w, identity, err := loadWorldWithIdentity(worldPath, workspacePath)
	if err != nil {
		return resultLoadFailure(out, "plugin-plan", err)
	}

	var lock *plugin.Lock
	if pluginLockPath != "" {
		decoded, decodeErr := decodePluginLockFile(pluginLockPath)
		if decodeErr != nil {
			// A lock whose bytes are unusable is an input problem and keeps the
			// existing code, so the same bad lock is classified the same way
			// whichever command reads it (#3184).
			return emitResult(out, resultFailure("plugin-plan", resultCodeInvalidPluginLock, decodeErr), 2)
		}
		lock = &decoded
	}
	var targetLock *plugin.Lock
	if targetLockPath != "" {
		decoded, decodeErr := decodePluginLockFile(targetLockPath)
		if decodeErr != nil {
			return emitResult(out, resultFailure("plugin-plan", resultCodeInvalidPluginLock, decodeErr), 2)
		}
		targetLock = &decoded
	}

	// The plan snapshot records the Source it was read at. An explicit
	// -source-index file binds it to that index; otherwise the Workspace
	// manifest identity is the fingerprint.
	sourceFingerprint := ""
	if sourceIndexPath != "" {
		fp, fpErr := sourceIndexFingerprint(sourceIndexPath)
		if fpErr != nil {
			return resultInvalidArgument(out, "plugin-plan", "invalid -source-index: "+fpErr.Error())
		}
		sourceFingerprint = fp
	} else if identity != nil {
		sourceFingerprint = identity.Manifest.SHA256
	}

	plan, planErr := operations.PluginPlan(operations.PluginPlanInput{
		Plugins:           w.Plugins,
		Bundles:           w.Bundles,
		Environments:      w.Environments,
		Environment:       envID,
		Action:            action,
		Plugin:            targetID,
		Lock:              lock,
		TargetLock:        targetLock,
		SourceFingerprint: sourceFingerprint,
	})
	if planErr != nil {
		var badAction *operations.InvalidPlanActionError
		var unknownEnv *operations.UnknownEnvironmentError
		switch {
		case errors.As(planErr, &badAction):
			return resultInvalidArgument(out, "plugin-plan", "invalid -action: "+planErr.Error())
		case errors.As(planErr, &unknownEnv):
			return emitResult(out, resultFailure("plugin-plan", resultCodeUnknownEnvironment, planErr), 2)
		default:
			return emitResult(out, resultFailure("plugin-plan", resultCodePluginResolveFailed, planErr), 2)
		}
	}

	return emitResult(out, resultEnvelopeWithWorkspace("plugin-plan", "", plan, identity), 0)
}

// sourceIndexFingerprint is the byte照合 of a source-index file, so a plan can
// name the exact index it was read against without carrying the index itself.
func sourceIndexFingerprint(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
