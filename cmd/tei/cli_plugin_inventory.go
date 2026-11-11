package main

import (
	"errors"
	"fmt"
	"io"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/plugin"
	"tei/internal/plugin/management"
)

// pluginInventoryModes lists every value the response's `mode` field can take
// for plugin-inventory. The shape is selected by -plugin and -diagnose, the way
// source-index selects its shape by -subject-kind/-subject-id, so one shared
// flag set never has to hold two different defaults for -mode.
const (
	pluginInventoryModeList     = ""
	pluginInventoryModeDetail   = resultModeInventoryDetail
	pluginInventoryModeDiagnose = resultModeInventoryDiagnose
)

// pluginInventoryDetail is one plugin's row plus the report-level context a
// reader needs to judge it.
type pluginInventoryDetail struct {
	SchemaVersion   string                     `json:"schemaVersion"`
	Environment     string                     `json:"environment"`
	ScanState       string                     `json:"scanState"`
	Confirmed       bool                       `json:"confirmed"`
	SettingsVersion *string                    `json:"settingsVersion"`
	Resolver        management.ResolverOutcome `json:"resolver"`
	Row             management.Row             `json:"row"`
	Message         string                     `json:"message"`
	NotModelled     []string                   `json:"notModelled"`
	AuthorityLimits []string                   `json:"authorityLimits"`
}

// pluginInventoryJSON serves `plugin-inventory`. It derives the inventory from
// the Workspace the same way every read-only command does, and reports the
// resolution the existing resolver made: it never resolves a second time and
// never chooses a provider.
func pluginInventoryJSON(out io.Writer, workspacePath, worldPath, environmentID, pluginLockPath, pluginID string, diagnose bool, extraArgs int) (int, error) {
	if (workspacePath == "") == (worldPath == "") {
		return resultInvalidArgument(out, "plugin-inventory", "plugin-inventory requires exactly one of -workspace or -world")
	}
	if extraArgs != 0 {
		return resultInvalidArgument(out, "plugin-inventory", "plugin-inventory does not accept positional arguments")
	}
	if environmentID == "" {
		return resultInvalidArgument(out, "plugin-inventory", "plugin-inventory requires -environment")
	}
	w, identity, err := loadWorldWithIdentity(worldPath, workspacePath)
	if err != nil {
		return resultLoadFailure(out, "plugin-inventory", err)
	}

	var lock *plugin.Lock
	if pluginLockPath != "" {
		decoded, decodeErr := decodePluginLockFile(pluginLockPath)
		if decodeErr != nil {
			// A lock whose bytes are unusable is an input problem and keeps the
			// existing code, so the same bad lock is classified the same way
			// whichever command reads it (#3184).
			return emitResult(out, resultFailure("plugin-inventory", resultCodeInvalidPluginLock, decodeErr), 2)
		}
		lock = &decoded
	}

	envID, idErr := core.ParseCanonicalID(environmentID)
	if idErr != nil {
		return resultInvalidArgument(out, "plugin-inventory", "invalid -environment: "+idErr.Error())
	}

	report, err := management.Derive(management.Input{
		World:       managementWorld(w),
		Environment: envID,
		Lock:        lock,
	})
	if err != nil {
		// An unknown environment is an input problem, the same as every other
		// plugin command answers it.
		if isUnknownEnvironment(err) {
			return emitResult(out, resultFailure("plugin-inventory", resultCodeUnknownEnvironment, err), 2)
		}
		if management.IsLockError(err) {
			return emitResult(out, resultFailure("plugin-inventory", resultCodeInvalidPluginLock, err), 2)
		}
		return emitResult(out, resultFailure("plugin-inventory", resultCodePluginResolveFailed, err), 2)
	}

	if pluginID != "" {
		row, ok := management.FindRow(report, pluginID)
		if !ok {
			return emitResult(out, resultFailure("plugin-inventory", resultCodeInvalidArgument,
				fmt.Errorf("plugin %s does not appear in environment %s", pluginID, environmentID)), 2)
		}
		detail := pluginInventoryDetail{
			SchemaVersion:   report.SchemaVersion,
			Environment:     report.Environment,
			ScanState:       report.ScanState,
			Confirmed:       report.Confirmed,
			SettingsVersion: report.SettingsVersion,
			Resolver:        report.Resolver,
			Row:             row,
			Message:         management.DetailMessage(row),
			NotModelled:     report.NotModelled,
			AuthorityLimits: report.AuthorityLimits,
		}
		return emitResult(out, resultEnvelopeWithWorkspace("plugin-inventory", pluginInventoryModeDetail, detail, identity), 0)
	}
	_ = diagnose
	if diagnose {
		// The diagnose shape is the same report restricted to the rows that
		// carry a reason code, beside the display catalogue. It adds no fact
		// and grants no permission.
		diagnosis := report
		diagnosis.Rows = nil
		for _, row := range report.Rows {
			if len(row.ReasonCodes) > 0 {
				diagnosis.Rows = append(diagnosis.Rows, row)
			}
		}
		return emitResult(out, resultEnvelopeWithWorkspace("plugin-inventory", pluginInventoryModeDiagnose, diagnosis, identity), 0)
	}
	return emitResult(out, resultEnvelopeWithWorkspace("plugin-inventory", pluginInventoryModeList, report, identity), 0)
}

// managementWorld narrows ir.World to the part the inventory reads.
func managementWorld(w ir.World) management.World {
	return management.World{
		Plugins:      w.Plugins,
		Bundles:      w.Bundles,
		Environments: w.Environments,
	}
}

func isUnknownEnvironment(err error) bool {
	// Derive reports an unknown environment as a typed error, so the CLI
	// classifies it without matching message text.
	var unknown *management.UnknownEnvironmentError
	return errors.As(err, &unknown)
}
