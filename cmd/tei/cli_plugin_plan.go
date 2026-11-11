package main

import (
	"errors"
	"io"

	"tei/internal/core"
	"tei/internal/plugin"
	"tei/internal/plugin/management"
)

// pluginPlanMode is the single response shape of `plugin-plan`. The action is
// carried inside the plan, not in `mode`, because the plan type is the same for
// every action.
const pluginPlanMode = "plan"

// pluginPlanJSON serves `plugin-plan`. It derives the read-only plan from the
// same World, environment and lock the inventory reads, and it never applies
// anything: the response is a proposal whose digest an applying operation must
// match.
func pluginPlanJSON(out io.Writer, workspacePath, worldPath, environmentID, pluginLockPath, targetLockPath, action, pluginID string, extraArgs int) (int, error) {
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
	w, identity, err := loadWorldWithIdentity(worldPath, workspacePath)
	if err != nil {
		return resultLoadFailure(out, "plugin-plan", err)
	}

	lock, lockErr := optionalLock(pluginLockPath)
	if lockErr != nil {
		return emitResult(out, resultFailure("plugin-plan", resultCodeInvalidPluginLock, lockErr), 2)
	}
	targetLock, targetErr := optionalLock(targetLockPath)
	if targetErr != nil {
		return emitResult(out, resultFailure("plugin-plan", resultCodeInvalidPluginLock, targetErr), 2)
	}

	envID, idErr := core.ParseCanonicalID(environmentID)
	if idErr != nil {
		return resultInvalidArgument(out, "plugin-plan", "invalid -environment: "+idErr.Error())
	}

	plan, err := management.DerivePlan(management.PlanInput{
		World:       managementWorld(w),
		Environment: envID,
		Lock:        lock,
		TargetLock:  targetLock,
		Request:     management.PlanRequest{Action: action, PluginID: pluginID},
	})
	if err != nil {
		var unsupported *management.UnsupportedPlanError
		if errors.As(err, &unsupported) {
			return resultInvalidArgument(out, "plugin-plan", unsupported.Error())
		}
		var unknown *management.UnknownEnvironmentError
		if errors.As(err, &unknown) {
			return emitResult(out, resultFailure("plugin-plan", resultCodeUnknownEnvironment, err), 2)
		}
		var refusal *management.RefusalError
		if errors.As(err, &refusal) {
			return resultInvalidArgument(out, "plugin-plan", refusal.Error())
		}
		if management.IsLockError(err) {
			return emitResult(out, resultFailure("plugin-plan", resultCodeInvalidPluginLock, err), 2)
		}
		return emitResult(out, resultFailure("plugin-plan", resultCodePluginResolveFailed, err), 2)
	}
	return emitResult(out, resultEnvelopeWithWorkspace("plugin-plan", pluginPlanMode, plan, identity), 0)
}

// optionalLock decodes a lock when a path was given. A refusal to decode is an
// input problem and keeps the existing INVALID_PLUGIN_LOCK code.
func optionalLock(path string) (*plugin.Lock, error) {
	if path == "" {
		return nil, nil
	}
	decoded, err := decodePluginLockFile(path)
	if err != nil {
		return nil, err
	}
	return &decoded, nil
}
