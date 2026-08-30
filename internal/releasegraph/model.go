package releasegraph

import (
	"fmt"
	"sort"

	apprelease "tei/internal/application/release"
	"tei/internal/core"
	dbrelease "tei/internal/database/release"
)

type NodeKind string

const (
	NodeDatabaseState      NodeKind = "database-state"
	NodeDatabaseReadiness  NodeKind = "database-readiness"
	NodeApplicationRelease NodeKind = "application-release"
)

type Node struct {
	ID          string           `json:"id"`
	Kind        NodeKind         `json:"kind"`
	Application core.CanonicalID `json:"application,omitempty"`
	Database    core.CanonicalID `json:"database,omitempty"`
	Version     string           `json:"version,omitempty"`
	Source      string           `json:"source,omitempty"`
	Readiness   string           `json:"readiness,omitempty"`
}

type Edge struct {
	From     string           `json:"from"`
	To       string           `json:"to"`
	Relation string           `json:"relation"`
	Contract core.CanonicalID `json:"contract,omitempty"`
	Alias    core.LocalName   `json:"alias,omitempty"`
}

type Graph struct {
	Application core.CanonicalID `json:"application"`
	Version     string           `json:"version"`
	Nodes       []Node           `json:"nodes"`
	Edges       []Edge           `json:"edges"`
	Deployable  bool             `json:"deployable"`
	Reasons     []string         `json:"reasons,omitempty"`
}

// Build derives release ordering from an immutable Application release and a
// set of independently versioned database states. It does not invent a global
// application/database version or distributed transaction boundary.
func Build(app apprelease.Definition, states []dbrelease.StateSnapshot) Graph {
	app = app.Normalize()
	g := Graph{Application: app.Application, Version: app.Version, Deployable: true}
	byDB := map[core.CanonicalID]dbrelease.StateSnapshot{}
	for _, s := range states {
		byDB[s.Database] = s
	}
	appNode := fmt.Sprintf("application-release:%s@%s", app.Application, app.Version)
	g.Nodes = append(g.Nodes, Node{ID: appNode, Kind: NodeApplicationRelease, Application: app.Application, Version: app.Version})

	seenNode := map[string]struct{}{appNode: {}}
	for _, req := range app.Databases {
		s, ok := byDB[req.TargetDatabase]
		if !ok {
			g.Deployable = false
			g.Reasons = append(g.Reasons, fmt.Sprintf("database state unavailable: %s", req.TargetDatabase))
			continue
		}
		stateVersion := s.Release
		if stateVersion == "" {
			stateVersion = "observed"
		}
		stateNode := fmt.Sprintf("database-state:%s@%s", s.Database, stateVersion)
		readyNode := fmt.Sprintf("database-readiness:%s@%s", s.Database, stateVersion)
		if _, exists := seenNode[stateNode]; !exists {
			g.Nodes = append(g.Nodes, Node{ID: stateNode, Kind: NodeDatabaseState, Database: s.Database, Version: s.Release, Source: string(s.Source), Readiness: string(s.DataReadiness)})
			seenNode[stateNode] = struct{}{}
			g.Nodes = append(g.Nodes, Node{ID: readyNode, Kind: NodeDatabaseReadiness, Database: s.Database, Version: s.Release, Source: string(s.Source), Readiness: string(s.DataReadiness)})
			seenNode[readyNode] = struct{}{}
			g.Edges = append(g.Edges, Edge{From: stateNode, To: readyNode, Relation: "establishes-readiness"})
		}
		if s.DataReadiness != dbrelease.DataReady {
			g.Deployable = false
			g.Reasons = append(g.Reasons, fmt.Sprintf("database %s is not ready for consumers: %s", s.Database, s.DataReadiness))
		}
		provided := map[core.CanonicalID]dbrelease.ProvidedContract{}
		for _, p := range s.Provides {
			provided[p.ID] = p
		}
		for _, cr := range req.RequiredContracts {
			p, exists := provided[cr.ContractID]
			if !exists || p.Version.Major != cr.CompatibleMajor {
				g.Deployable = false
				g.Reasons = append(g.Reasons, fmt.Sprintf("contract requirement unsatisfied: %s needs %s compatible major = %d", req.Alias, cr.ContractID, cr.CompatibleMajor))
				continue
			}
			g.Edges = append(g.Edges, Edge{From: readyNode, To: appNode, Relation: "requires-contract", Contract: cr.ContractID, Alias: req.Alias})
		}
	}
	sort.Slice(g.Nodes, func(i, j int) bool { return g.Nodes[i].ID < g.Nodes[j].ID })
	sort.Slice(g.Edges, func(i, j int) bool {
		if g.Edges[i].From == g.Edges[j].From {
			if g.Edges[i].To == g.Edges[j].To {
				return g.Edges[i].Contract < g.Edges[j].Contract
			}
			return g.Edges[i].To < g.Edges[j].To
		}
		return g.Edges[i].From < g.Edges[j].From
	})
	sort.Strings(g.Reasons)
	return g
}
