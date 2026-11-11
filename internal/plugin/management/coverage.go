package management

import (
	"fmt"
	"sort"

	"tei/internal/environment"
	"tei/internal/permission"
	"tei/internal/plugin"
	"tei/internal/plugin/runtime"
)

// runtimeSummary is the runtime fact of one resolved plugin. It is filled from
// pluginruntime.BuildPlan, which requires a lock: without a lock the inventory
// reports runtime-unverified instead of guessing a boundary.
func runtimeSummary(plan runtime.EnvironmentPlan) map[string]runtimeFact {
	out := map[string]runtimeFact{}
	for _, pr := range plan.Plugins {
		out[string(pr.Plugin.ID)] = runtimeFact{
			Boundary: string(pr.Boundary),
			Trust:    string(pr.Trust),
			Command:  pr.Command,
		}
	}
	return out
}

type runtimeFact struct {
	Boundary string
	Trust    string
	Command  string
}

// grantGaps reports the capability permissions the environment grants do not
// cover. The comparison reuses pluginruntime's own grouping: a grant counts for
// a capability only when its CapabilityKind/CapabilityID match, exactly as
// runtime.grantsFor does. A lock is required to reach this, because the runtime
// plan is what carries the resolved capabilities.
func grantGaps(plan runtime.EnvironmentPlan) map[string][]permission.Spec {
	byPlugin := map[string][]permission.Spec{}
	for _, cr := range plan.Capabilities {
		if missing := permission.Missing(cr.Permissions, cr.Capability.Permissions); len(missing) > 0 {
			byPlugin[string(cr.Plugin.ID)] = append(byPlugin[string(cr.Plugin.ID)], missing...)
		}
	}
	return byPlugin
}

// deriveLevels reports the data levels the environment declares. A database
// instance names a provider; a feature names an editor capability. The
// inventory reports how many declared capabilities can serve the level.
func deriveLevels(env environment.Environment, defs []plugin.Definition) []Level {
	var levels []Level

	for _, instance := range env.Instances {
		level := Level{
			Kind:     "database-instance",
			ID:       string(instance.ID),
			Provider: string(instance.Provider),
		}
		var serving []string
		for _, d := range defs {
			for _, c := range d.Provides {
				if !hasDatabaseProvider(c, instance.Provider) {
					continue
				}
				serving = append(serving, string(d.ID))
				level.Requires++
				level.Covered++
				break
			}
		}
		sort.Strings(serving)
		level.Providers = serving
		switch len(serving) {
		case 0:
			level.Status = LevelNoProvider
			level.ReasonCodes = append(level.ReasonCodes, "no-provider")
		case 1:
			level.Status = LevelProvided
		default:
			level.Status = LevelMultipleProviders
			level.ReasonCodes = append(level.ReasonCodes, "multiple-providers")
		}
		levels = append(levels, level)
	}

	for _, feature := range env.Features {
		level := Level{Kind: "feature", ID: string(feature)}
		var serving []string
		for _, d := range defs {
			for _, c := range d.Provides {
				if !hasFeature(c, feature) {
					continue
				}
				serving = append(serving, string(d.ID))
				level.Requires++
				level.Covered++
				break
			}
		}
		sort.Strings(serving)
		level.Providers = serving
		switch len(serving) {
		case 0:
			level.Status = LevelUnsupported
			level.ReasonCodes = append(level.ReasonCodes, "feature-not-provided")
		case 1:
			level.Status = LevelProvided
		default:
			level.Status = LevelMultipleProviders
			level.ReasonCodes = append(level.ReasonCodes, "multiple-providers")
		}
		levels = append(levels, level)
	}

	sort.Slice(levels, func(i, j int) bool {
		if levels[i].Kind != levels[j].Kind {
			return levels[i].Kind < levels[j].Kind
		}
		return levels[i].ID < levels[j].ID
	})
	return levels
}

func hasDatabaseProvider(c plugin.Capability, provider environment.DatabaseProvider) bool {
	for _, p := range c.DatabaseProviders {
		if p == provider {
			return true
		}
	}
	return false
}

func hasFeature(c plugin.Capability, feature environment.Feature) bool {
	for _, f := range c.Features {
		if f == feature {
			return true
		}
	}
	return false
}

func allLevelsCovered(levels []Level) bool {
	for _, level := range levels {
		if level.Status != LevelProvided {
			return false
		}
	}
	return true
}

// messageCodes lists every reason code the report mentions, so the display
// catalogue covers the codes actually emitted.
func messageCodes(report Report) map[string]bool {
	codes := map[string]bool{}
	add := func(values []string) {
		for _, value := range values {
			codes[value] = true
		}
	}
	add(report.ScanReasons)
	add(report.Resolver.ReasonCodes)
	for _, row := range report.Rows {
		add(row.ReasonCodes)
	}
	for _, level := range report.Levels {
		add(level.ReasonCodes)
	}
	return codes
}

// detailMessage is used by Detail to explain a row in one sentence without
// turning a reason code into a permission claim.
func detailMessage(row Row) string {
	if len(row.ReasonCodes) == 0 {
		return fmt.Sprintf("%s@%s は宣言・環境選択・解決のいずれも揃っています。", row.PluginID, row.Version)
	}
	return fmt.Sprintf("%s には %s があります。", row.PluginID, joinCodes(row.ReasonCodes))
}

func joinCodes(codes []string) string {
	out := ""
	for i, code := range codes {
		if i > 0 {
			out += ", "
		}
		out += code
	}
	return out
}
