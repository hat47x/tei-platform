package generatorartifactstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

const (
	ListSchema  = "tei.reference.generator-artifact-list/v0"
	PruneSchema = "tei.reference.generator-artifact-prune/v0"
)

type StoredObject struct {
	SourceDigest  string `json:"sourceDigest"`
	BinaryDigest  string `json:"binaryDigest"`
	CandidateID   string `json:"candidateId"`
	Path          string `json:"path"`
	SourceBytes   int64  `json:"sourceBytes"`
	ArtifactBytes int64  `json:"artifactBytes"`
	Verified      bool   `json:"verified"`
	Reason        string `json:"reason,omitempty"`
}

type ListReceipt struct {
	Schema   string         `json:"schema"`
	StoreDir string         `json:"storeDir"`
	Objects  []StoredObject `json:"objects"`
	Passed   bool           `json:"passed"`
	Reason   string         `json:"reason,omitempty"`
}

type PruneOptions struct {
	StoreDir string
	Keep     []string
	DryRun   bool
}

type PruneReceipt struct {
	Schema           string   `json:"schema"`
	StoreDir         string   `json:"storeDir"`
	DryRun           bool     `json:"dryRun"`
	Keep             []string `json:"keep"`
	Kept             []string `json:"kept"`
	Deleted          []string `json:"deleted,omitempty"`
	WouldDelete      []string `json:"wouldDelete,omitempty"`
	MissingKeep      []string `json:"missingKeep,omitempty"`
	Passed           bool     `json:"passed"`
	Reason           string   `json:"reason,omitempty"`
	GrantsAuthority  bool     `json:"grantsAuthority"`
	ProvesRealTarget bool     `json:"provesRealTarget"`
	Notes            []string `json:"notes"`
}

// List reads every object directory and re-verifies its source and artifact
// digests. A malformed or tampered object makes the list fail closed.
func List(storeDir string) (ListReceipt, error) {
	receipt := ListReceipt{
		Schema: ListSchema, StoreDir: storeDir, Objects: []StoredObject{}, Passed: true,
	}
	objectsDir := filepath.Join(storeDir, "objects")
	entries, err := os.ReadDir(objectsDir)
	if os.IsNotExist(err) {
		return receipt, nil
	}
	if err != nil {
		return receipt, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		objectPath := filepath.Join(objectsDir, entry.Name())
		object := StoredObject{Path: objectPath}
		rawObject, err := os.ReadFile(filepath.Join(objectPath, "object.json"))
		if err != nil {
			object.Reason = "object metadata is unreadable: " + err.Error()
			receipt.Passed = false
			receipt.Objects = append(receipt.Objects, object)
			continue
		}
		var metadata Object
		if err := json.Unmarshal(rawObject, &metadata); err != nil {
			object.Reason = "object metadata is invalid: " + err.Error()
			receipt.Passed = false
			receipt.Objects = append(receipt.Objects, object)
			continue
		}
		object.SourceDigest = metadata.SourceDigest
		object.BinaryDigest = metadata.BinaryDigest
		object.CandidateID = metadata.CandidateID
		source, sourceErr := os.ReadFile(filepath.Join(objectPath, "source"))
		artifact, artifactErr := os.ReadFile(filepath.Join(objectPath, "artifact"))
		switch {
		case sourceErr != nil:
			object.Reason = "source is unreadable: " + sourceErr.Error()
		case artifactErr != nil:
			object.Reason = "artifact is unreadable: " + artifactErr.Error()
		case digest(source) != metadata.SourceDigest:
			object.Reason = "stored source digest does not match"
		case digest(artifact) != metadata.BinaryDigest:
			object.Reason = "stored artifact digest does not match"
		case filepath.Base(objectPath) != trimDigest(metadata.SourceDigest):
			object.Reason = "object path does not match the source digest"
		default:
			object.Verified = true
			object.SourceBytes = int64(len(source))
			object.ArtifactBytes = int64(len(artifact))
		}
		if !object.Verified {
			receipt.Passed = false
		}
		receipt.Objects = append(receipt.Objects, object)
	}
	sort.Slice(receipt.Objects, func(i, j int) bool {
		return receipt.Objects[i].SourceDigest < receipt.Objects[j].SourceDigest
	})
	if !receipt.Passed {
		receipt.Reason = "one or more objects failed verification"
	}
	return receipt, nil
}

// Prune deletes objects that are not in an explicit keep set. It refuses to
// delete anything when the keep set is empty, has an unknown digest, or the
// store contains an unverified object. It does not choose a retention period.
func Prune(opts PruneOptions) (PruneReceipt, error) {
	receipt := PruneReceipt{
		Schema: PruneSchema, StoreDir: opts.StoreDir, DryRun: opts.DryRun,
		Keep: []string{}, Kept: []string{}, Deleted: []string{}, WouldDelete: []string{}, MissingKeep: []string{},
		GrantsAuthority: false, ProvesRealTarget: false,
		Notes: []string{
			"明示したkeep setだけを残し、それ以外のobjectを削除する参照。",
			"保持期間・監査アクセス権・削除承認は決めない。",
			"空のkeep set、未知のkeep digest、検証失敗objectがある場合は削除しない。",
		},
	}
	if len(opts.Keep) == 0 {
		receipt.Reason = "keep set must not be empty"
		return receipt, nil
	}
	keep := map[string]bool{}
	for _, sourceDigest := range opts.Keep {
		if _, err := digestHex(sourceDigest); err != nil {
			receipt.Reason = err.Error()
			return receipt, nil
		}
		if !keep[sourceDigest] {
			keep[sourceDigest] = true
			receipt.Keep = append(receipt.Keep, sourceDigest)
		}
	}
	sort.Strings(receipt.Keep)
	list, err := List(opts.StoreDir)
	if err != nil {
		return receipt, err
	}
	if !list.Passed {
		receipt.Reason = "store contains an unverified object; refusing to prune"
		return receipt, nil
	}
	found := map[string]bool{}
	for _, object := range list.Objects {
		if keep[object.SourceDigest] {
			found[object.SourceDigest] = true
			receipt.Kept = append(receipt.Kept, object.SourceDigest)
			continue
		}
		if opts.DryRun {
			receipt.WouldDelete = append(receipt.WouldDelete, object.SourceDigest)
			continue
		}
		if err := os.RemoveAll(object.Path); err != nil {
			receipt.Reason = "deleting " + object.SourceDigest + ": " + err.Error()
			return receipt, nil
		}
		receipt.Deleted = append(receipt.Deleted, object.SourceDigest)
	}
	for _, sourceDigest := range receipt.Keep {
		if !found[sourceDigest] {
			receipt.MissingKeep = append(receipt.MissingKeep, sourceDigest)
		}
	}
	if len(receipt.MissingKeep) > 0 {
		receipt.Reason = "keep set contains a digest that is not in the store"
		return receipt, nil
	}
	sort.Strings(receipt.Kept)
	sort.Strings(receipt.Deleted)
	sort.Strings(receipt.WouldDelete)
	receipt.Passed = true
	return receipt, nil
}

func trimDigest(value string) string {
	if len(value) > len("sha256:") {
		return value[len("sha256:"):]
	}
	return value
}
