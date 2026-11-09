package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// errRebuildDigestMismatchは、再buildで得たbytesが期待したdigestと一致しなかったことを表す。
var errRebuildDigestMismatch = errors.New("rebuilt artifact does not match the expected digest")

type artifactMemberStatus string

const (
	artifactMemberAvailable artifactMemberStatus = "available"
	artifactMemberMissing   artifactMemberStatus = "missing"
	artifactMemberCorrupt   artifactMemberStatus = "corrupt"
)

// artifactMemberRequestは、Artifact Setの一つのRoleと、そのRoleが期待するdigestを表す。
// RoleとdigestはArtifact Set（manifest）の側が持つ。storeはRoleの意味を解釈しない。
type artifactMemberRequest struct {
	Role   string
	Digest string
}

// artifactMemberは、一つのRoleについてstoreを実際に読んだ結果である。
// Bytesはstatusがavailableのときだけ設定する。
type artifactMember struct {
	Role    string
	Digest  string
	Status  artifactMemberStatus
	Bytes   []byte
	Receipt artifactStoreReceipt
}

// artifactMembersは、複数Roleを実際に読んだ結果をRole順に保持する。
type artifactMembers struct {
	Members []artifactMember
}

// Availableは、要求した全Roleがdigest一致で取得できたときだけtrueを返す。
// manifestが全Roleを列挙していることは、取得可能であることを意味しない。
func (m artifactMembers) Available() bool {
	if len(m.Members) == 0 {
		return false
	}
	for _, member := range m.Members {
		if member.Status != artifactMemberAvailable {
			return false
		}
	}
	return true
}

// getMembersは、Roleごとに期待digestのbytesをstoreから取得する。
//
// 一つでも欠落や破損があれば、その状態をRoleごとに区別して返す。欠落したRoleがあるのに
// 残りのRoleだけで「利用可能」とは報告しない。破損は欠落と区別し、破損したbytesを
// 別のbytesで黙って置き換えない。入力の不正（Roleの重複、不正なdigest）やstoreの
// 読取り障害は、状態ではなくerrorとして返す。
func (s localArtifactStore) getMembers(requests []artifactMemberRequest) (artifactMembers, error) {
	if len(requests) == 0 {
		return artifactMembers{}, fmt.Errorf("artifact member request requires at least one role")
	}
	seen := make(map[string]struct{}, len(requests))
	for _, request := range requests {
		if strings.TrimSpace(request.Role) == "" {
			return artifactMembers{}, fmt.Errorf("artifact member role is required")
		}
		if _, duplicate := seen[request.Role]; duplicate {
			return artifactMembers{}, fmt.Errorf("artifact member role %s is requested more than once", request.Role)
		}
		seen[request.Role] = struct{}{}
		if _, err := normalizeDigest(request.Digest); err != nil {
			return artifactMembers{}, fmt.Errorf("artifact member %s: %w", request.Role, err)
		}
	}

	sorted := append([]artifactMemberRequest(nil), requests...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Role < sorted[j].Role })

	result := artifactMembers{Members: make([]artifactMember, 0, len(sorted))}
	for _, request := range sorted {
		digest, _ := normalizeDigest(request.Digest)
		value, receipt, err := s.GetBytes(digest)
		member := artifactMember{Role: request.Role, Digest: digest}
		switch {
		case err == nil:
			member.Status = artifactMemberAvailable
			member.Bytes = value
			member.Receipt = receipt
		case errors.Is(err, errArtifactNotFound):
			member.Status = artifactMemberMissing
		case errors.Is(err, errArtifactCorrupt):
			member.Status = artifactMemberCorrupt
		default:
			return artifactMembers{}, fmt.Errorf("read artifact member %s: %w", request.Role, err)
		}
		result.Members = append(result.Members, member)
	}
	return result, nil
}

// adoptRebuiltは、cache missの後に再buildしたbytesを、期待digestと一致する場合だけstoreへ採用する。
//
// 再buildを許すかどうかはconsumerの方針であり、この関数は再buildを起動しない。
// 一致しないbytesは、別digestの「最新版」としてstoreにも呼出し側の期待にも反映せず拒否する。
// 拒否したbytesはstoreへ書き込まない。
func (s localArtifactStore) adoptRebuilt(expectedDigest string, rebuilt []byte) (artifactStoreReceipt, error) {
	expected, err := normalizeDigest(expectedDigest)
	if err != nil {
		return artifactStoreReceipt{}, err
	}
	if actual := digestBytes(rebuilt); actual != expected {
		return artifactStoreReceipt{}, fmt.Errorf("%w: got %s want %s", errRebuildDigestMismatch, actual, expected)
	}
	return s.PutBytes(rebuilt)
}
