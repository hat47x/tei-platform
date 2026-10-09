// Package runtimepackaging is a read-only reference for deriving the minimum
// runtime dependency closure from already-resolved TEI plugin bindings.
// It does not resolve providers, grant permissions or produce binaries.
package runtimepackaging

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var sha256Digest = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
var buildPlanFingerprint = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Unit describes one separately deployed and updated execution boundary.
// AvailableImports is supplied by the trusted target/host profile, not by an
// untrusted plugin implementation.
type Unit struct {
	ID               string   `json:"id"`
	Host             string   `json:"host"`
	Roots            []string `json:"roots"`
	AvailableImports []string `json:"availableImports,omitempty"`
}

// RemoteCall is an explicitly bound call across execution/trust boundaries.
// Transport and TrustBinding name contracts already validated elsewhere;
// their presence is not proof of secure transport or runtime authorization.
type RemoteCall struct {
	Capability   string `json:"capability"`
	Unit         string `json:"unit"`
	Transport    string `json:"transport"`
	TrustBinding string `json:"trustBinding"`
}

// Binding is an exact selection supplied by the existing Plugin/Capability
// resolution and runtime lock. Requires only refers to capabilities in Unit.
// RemoteRequires refers to explicit, already-selected capabilities in another
// unit. DynamicOpen means reachability cannot be safely enumerated.
type Binding struct {
	Unit                 string       `json:"unit"`
	Capability           string       `json:"capability"`
	Plugin               string       `json:"plugin"`
	PluginVersion        string       `json:"pluginVersion"`
	PluginFingerprint    string       `json:"pluginFingerprint,omitempty"`
	ArtifactDigest       string       `json:"artifactDigest,omitempty"`
	BuildPlanFingerprint string       `json:"buildPlanFingerprint,omitempty"`
	ContractFingerprint  string       `json:"contractFingerprint"`
	Requires             []string     `json:"requires,omitempty"`
	RemoteRequires       []RemoteCall `json:"remoteRequires,omitempty"`
	HostImports          []string     `json:"hostImports,omitempty"`
	DynamicOpen          bool         `json:"dynamicOpen,omitempty"`
}

type Input struct {
	Units    []Unit    `json:"units"`
	Bindings []Binding `json:"bindings"`
}

type UnitPlan struct {
	Unit     string    `json:"unit"`
	Host     string    `json:"host"`
	Included []Binding `json:"included"`
}

type Plan struct {
	Units []UnitPlan `json:"units"`
}

type key struct{ unit, capability string }

func validName(s string) bool {
	return s != "" && strings.TrimSpace(s) == s && !strings.ContainsAny(s, "\t\n\r")
}

func uniqueNames(names []string) bool {
	seen := map[string]bool{}
	for _, name := range names {
		if !validName(name) || seen[name] {
			return false
		}
		seen[name] = true
	}
	return true
}

// Derive computes deterministic per-unit reachability. It uses no heuristic
// provider selection and has no access to an environment's secret values.
func Derive(in Input) (Plan, error) {
	units := map[string]Unit{}
	bindings := map[key]Binding{}
	for _, u := range in.Units {
		if !validName(u.ID) || !validName(u.Host) || len(u.Roots) == 0 ||
			!uniqueNames(u.Roots) || !uniqueNames(u.AvailableImports) {
			return Plan{}, fmt.Errorf("invalid unit %q", u.ID)
		}
		if _, ok := units[u.ID]; ok {
			return Plan{}, fmt.Errorf("duplicate unit %q", u.ID)
		}
		units[u.ID] = u
	}
	if len(units) == 0 {
		return Plan{}, fmt.Errorf("no units")
	}
	for _, b := range in.Bindings {
		if !validName(b.Capability) || !validName(b.Plugin) ||
			!validName(b.PluginVersion) || !validName(b.ContractFingerprint) ||
			(b.PluginFingerprint == "" && b.ArtifactDigest == "" && b.BuildPlanFingerprint == "") ||
			(b.PluginFingerprint != "" && !buildPlanFingerprint.MatchString(b.PluginFingerprint)) ||
			(b.ArtifactDigest != "" && !sha256Digest.MatchString(b.ArtifactDigest)) ||
			(b.BuildPlanFingerprint != "" && !buildPlanFingerprint.MatchString(b.BuildPlanFingerprint)) ||
			!uniqueNames(b.Requires) || !uniqueNames(b.HostImports) {
			return Plan{}, fmt.Errorf("invalid binding for %q/%q", b.Unit, b.Capability)
		}
		if _, ok := units[b.Unit]; !ok {
			return Plan{}, fmt.Errorf("binding %q has unknown unit %q", b.Capability, b.Unit)
		}
		k := key{b.Unit, b.Capability}
		if _, ok := bindings[k]; ok {
			return Plan{}, fmt.Errorf("ambiguous binding %q/%q", b.Unit, b.Capability)
		}
		bindings[k] = b
	}
	// Reachability, not the total number of installed plugins, determines the
	// packaging set. A graph cycle is valid and will be visited only once.
	queue := make([]key, 0)
	for _, u := range in.Units {
		for _, root := range u.Roots {
			queue = append(queue, key{u.ID, root})
		}
	}
	visited := map[key]bool{}
	for len(queue) > 0 {
		k := queue[0]
		queue = queue[1:]
		if visited[k] {
			continue
		}
		b, ok := bindings[k]
		if !ok {
			return Plan{}, fmt.Errorf("missing bound capability %q/%q", k.unit, k.capability)
		}
		if b.DynamicOpen {
			return Plan{}, fmt.Errorf("unbounded dynamic dependency at %q/%q", k.unit, k.capability)
		}
		imports := map[string]bool{}
		for _, name := range units[k.unit].AvailableImports {
			imports[name] = true
		}
		for _, name := range b.HostImports {
			if !imports[name] {
				return Plan{}, fmt.Errorf("host import %q unavailable for %q/%q", name, k.unit, k.capability)
			}
		}
		seenRemote := map[key]bool{}
		for _, remote := range b.RemoteRequires {
			target := key{remote.Unit, remote.Capability}
			if !validName(remote.Capability) || !validName(remote.Unit) || !validName(remote.Transport) ||
				!validName(remote.TrustBinding) || remote.Unit == k.unit || seenRemote[target] {
				return Plan{}, fmt.Errorf("invalid remote call from %q/%q", k.unit, k.capability)
			}
			seenRemote[target] = true
			if _, ok := units[target.unit]; !ok {
				return Plan{}, fmt.Errorf("remote target unit %q missing", target.unit)
			}
			queue = append(queue, target)
		}
		for _, dep := range b.Requires {
			queue = append(queue, key{k.unit, dep})
		}
		visited[k] = true
	}
	plan := Plan{}
	for _, u := range in.Units {
		p := UnitPlan{Unit: u.ID, Host: u.Host, Included: []Binding{}}
		for k := range visited {
			if k.unit != u.ID {
				continue
			}
			b := bindings[k]
			b.Requires = append([]string(nil), b.Requires...)
			sort.Strings(b.Requires)
			b.HostImports = append([]string(nil), b.HostImports...)
			sort.Strings(b.HostImports)
			b.RemoteRequires = append([]RemoteCall(nil), b.RemoteRequires...)
			sort.Slice(b.RemoteRequires, func(i, j int) bool {
				x, y := b.RemoteRequires[i], b.RemoteRequires[j]
				if x.Unit != y.Unit {
					return x.Unit < y.Unit
				}
				return x.Capability < y.Capability
			})
			p.Included = append(p.Included, b)
		}
		sort.Slice(p.Included, func(i, j int) bool { return p.Included[i].Capability < p.Included[j].Capability })
		plan.Units = append(plan.Units, p)
	}
	sort.Slice(plan.Units, func(i, j int) bool { return plan.Units[i].Unit < plan.Units[j].Unit })
	return plan, nil
}
