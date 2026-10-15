package main

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// publishedArtifactMember records both the no-clobber publication and a
// subsequent digest-reverified read. Neither receipt conveys trust or runtime
// authority; those decisions belong to the Host's current trust boundary.
type publishedArtifactMember struct {
	Role       string
	Digest     string
	Published  artifactStoreReceipt
	Reverified artifactStoreReceipt
}

// artifactSetPublication is emitted only when all members were successfully
// published and re-read at their expected digests. A failed publication may
// leave independently valid objects in the CAS, but never a success result.
type artifactSetPublication struct {
	Members []publishedArtifactMember
}

// publishExactMembers performs a build-output-to-store handoff. It checks the
// complete expected role/digest set and takes a private byte snapshot BEFORE
// publishing anything. Missing, extra, ambiguous, or mismatching inputs do not
// write even one object. This is not a transactional multi-object store: an I/O
// failure during publication may leave a subset of immutable CAS objects.
// In that case the caller receives an error, not "set available" evidence.
func (s localArtifactStore) publishExactMembers(expected []artifactMemberRequest, built map[string][]byte) (artifactSetPublication, error) {
	if len(expected) == 0 {
		return artifactSetPublication{}, fmt.Errorf("artifact publication requires at least one member")
	}
	if len(expected) != len(built) {
		return artifactSetPublication{}, fmt.Errorf("artifact publication role set does not match build outputs")
	}

	type checked struct {
		role   string
		digest string
		value  []byte
	}
	checkedMembers := make([]checked, 0, len(expected))
	seen := make(map[string]struct{}, len(expected))
	for _, member := range expected {
		if strings.TrimSpace(member.Role) == "" {
			return artifactSetPublication{}, fmt.Errorf("artifact publication role is required")
		}
		if _, duplicate := seen[member.Role]; duplicate {
			return artifactSetPublication{}, fmt.Errorf("artifact publication role is duplicated")
		}
		seen[member.Role] = struct{}{}
		digest, err := normalizeDigest(member.Digest)
		if err != nil {
			return artifactSetPublication{}, fmt.Errorf("artifact publication has an invalid digest: %w", err)
		}
		value, present := built[member.Role]
		if !present {
			return artifactSetPublication{}, fmt.Errorf("artifact publication output role is missing")
		}
		// A nil slice with a present map key is a valid zero-byte artifact.
		snapshot := bytes.Clone(value)
		if digestBytes(snapshot) != digest {
			return artifactSetPublication{}, fmt.Errorf("artifact publication output digest does not match its expected identity")
		}
		checkedMembers = append(checkedMembers, checked{role: member.Role, digest: digest, value: snapshot})
	}
	sort.Slice(checkedMembers, func(i, j int) bool {
		return checkedMembers[i].role < checkedMembers[j].role
	})
	published := make([]publishedArtifactMember, 0, len(checkedMembers))
	for _, member := range checkedMembers {
		receipt, err := s.PutBytes(member.value)
		if err != nil {
			return artifactSetPublication{}, fmt.Errorf("artifact publication failed for role %s: %w", member.role, err)
		}
		if receipt.Digest != member.digest {
			return artifactSetPublication{}, fmt.Errorf("artifact publication produced an unexpected digest")
		}
		published = append(published, publishedArtifactMember{
			Role: member.role, Digest: member.digest, Published: receipt,
		})
	}

	// Re-check availability by exact digest after the last Put. A put receipt
	// alone is not proof that a member remains retrievable.
	requests := make([]artifactMemberRequest, 0, len(checkedMembers))
	for _, member := range checkedMembers {
		requests = append(requests, artifactMemberRequest{Role: member.role, Digest: member.digest})
	}
	verified, err := s.getMembers(requests)
	if err != nil {
		return artifactSetPublication{}, fmt.Errorf("reverify published artifact set: %w", err)
	}
	if !verified.Available() {
		return artifactSetPublication{}, fmt.Errorf("artifact publication is incomplete after reverification")
	}
	for i, member := range verified.Members {
		if member.Role != published[i].Role || member.Digest != published[i].Digest || member.Status != artifactMemberAvailable {
			return artifactSetPublication{}, fmt.Errorf("artifact publication result is inconsistent")
		}
		published[i].Reverified = member.Receipt
	}
	return artifactSetPublication{Members: published}, nil
}
