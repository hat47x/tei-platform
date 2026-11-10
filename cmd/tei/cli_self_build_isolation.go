package main

import (
	"fmt"
	"strings"
)

// §4.44 left one thing unsaid: the entry recorded the builder's stdout but did not
// read the part where the builder says it built from a staged tree. This turns
// that statement into a readable record.
//
// It stays a statement. The entry cannot see the tree the builder used; what it
// can do is report what the builder said and whether the tree was made only of
// planned sources (no module context taken from the working tree).

const childIsolationBoundary = "the builder reported that it built from a tree staged from the plan; the entry did not see that tree, and this record adds nothing to the L2-b conditions"

const (
	childIsolationIsolated    = "isolated"
	childIsolationWithContext = "isolated-with-external-context"
	childIsolationNotReported = "not-reported"
)

// selfBuildChildIsolation is what the builder said about its build tree.
type selfBuildChildIsolation struct {
	// Root is the staged tree the builder named. The entry records it but takes no
	// meaning from the path beyond telling attempts apart.
	Root        string `json:"root,omitempty"`
	MemberCount int    `json:"memberCount"`
	// ModuleFiles are the module context files the builder took from the working
	// tree instead of from the plan. An empty list is what makes "the tree is only
	// planned sources" true.
	ModuleFiles []string `json:"moduleFiles,omitempty"`
	Boundary    string   `json:"boundary"`
}

// selfBuildChildIsolationRecord is the entry-level reading, with the state a
// reader needs to see at a glance.
type selfBuildChildIsolationRecord struct {
	State       string   `json:"state"`
	Root        string   `json:"root,omitempty"`
	MemberCount int      `json:"memberCount,omitempty"`
	ModuleFiles []string `json:"moduleFiles,omitempty"`
	Reason      string   `json:"reason,omitempty"`
	Boundary    string   `json:"boundary"`
}

// readChildIsolation turns the builder's statement into a record with a state.
func readChildIsolation(child *selfBuildChildIsolation) *selfBuildChildIsolationRecord {
	if child == nil {
		return nil
	}
	record := &selfBuildChildIsolationRecord{
		Root:        child.Root,
		MemberCount: child.MemberCount,
		ModuleFiles: child.ModuleFiles,
		Boundary:    childIsolationBoundary,
	}
	if len(child.ModuleFiles) == 0 {
		record.State = childIsolationIsolated
		return record
	}
	record.State = childIsolationWithContext
	record.Reason = fmt.Sprintf("the staged tree also took %s from the working tree, so it is not made only of planned sources", strings.Join(child.ModuleFiles, ", "))
	return record
}
