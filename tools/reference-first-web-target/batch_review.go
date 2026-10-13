//go:build !tei_repair_only

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"unicode/utf8"
)

type batchReviewItem struct {
	OrderID       int64  `json:"orderId"`
	Recorded      string `json:"recorded"`
	RecordedStage string `json:"recordedStage,omitempty"`
}
type batchReview struct {
	JournalDigest   string                   `json:"journalDigest"`
	Ended           bool                     `json:"ended"`
	IncompleteTail  bool                     `json:"incompleteTail"`
	Items           []batchReviewItem        `json:"items"`
	Current         *batchObservationReceipt `json:"current,omitempty"`
	AuthorityLimits []string                 `json:"authorityLimits"`
}

// Reject duplicate keys before typed decoding; neither a later key nor a
// partial final line may silently replace a recorded identity or result.
func checkJournalJSON(dec *json.Decoder, depth int) error {
	if depth > 64 {
		return fmt.Errorf("journal JSON nesting is too deep")
	}
	token, err := dec.Token()
	if err != nil {
		return err
	}
	if delimiter, ok := token.(json.Delim); ok {
		keys := map[string]bool{}
		for dec.More() {
			if delimiter == '{' {
				key, err := dec.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok || keys[name] {
					return fmt.Errorf("journal contains duplicate JSON keys")
				}
				keys[name] = true
			}
			if err := checkJournalJSON(dec, depth+1); err != nil {
				return err
			}
		}
		_, err = dec.Token()
	}
	return err
}

func readBatchReview(path string) (batchReview, []int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return batchReview{}, nil, fmt.Errorf("cannot open batch journal")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return batchReview{}, nil, fmt.Errorf("batch journal must be a regular file")
	}
	const limit = 16 << 20
	raw, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || len(raw) > limit || !utf8.Valid(raw) {
		return batchReview{}, nil, fmt.Errorf("invalid or oversized batch journal")
	}
	return decodeBatchReview(raw)
}

func decodeBatchReview(raw []byte) (batchReview, []int64, error) {
	result := batchReview{JournalDigest: fmt.Sprintf("sha256:%x", sha256.Sum256(raw)), Items: []batchReviewItem{}, AuthorityLimits: []string{
		"caller-supplied journal is not trusted authority or a database transaction receipt",
		"recorded result and current state do not prove which operation committed",
		"no automatic retry, resume authorization or rollback inference",
	}}
	var ids []int64
	recorded := []batchItem{}
	pending := int64(0)
	sequence := uint64(0)
	lines := bytes.Split(raw, []byte{'\n'})
	result.IncompleteTail = len(lines[len(lines)-1]) != 0
	for _, line := range lines[:len(lines)-1] {
		if result.Ended || len(line) == 0 {
			return batchReview{}, nil, fmt.Errorf("invalid journal event order")
		}
		keys := json.NewDecoder(bytes.NewReader(line))
		if err := checkJournalJSON(keys, 0); err != nil {
			return batchReview{}, nil, fmt.Errorf("invalid journal JSON")
		}
		dec := json.NewDecoder(bytes.NewReader(line))
		dec.DisallowUnknownFields()
		var event batchJournalEvent
		if err := dec.Decode(&event); err != nil {
			return batchReview{}, nil, fmt.Errorf("invalid journal event")
		}
		if err := dec.Decode(new(any)); err != io.EOF {
			return batchReview{}, nil, fmt.Errorf("trailing journal JSON")
		}
		sequence++
		if event.Sequence != sequence || event.Schema != "tei.reference.batch-progress/v1" {
			return batchReview{}, nil, fmt.Errorf("journal schema or sequence mismatch")
		}
		switch event.Kind {
		case "planned":
			if sequence != 1 || validateObservationIDs(event.Planned) != nil || event.Item != nil || event.Receipt != nil || event.OrderID != 0 {
				return batchReview{}, nil, fmt.Errorf("invalid journal plan")
			}
			ids = event.Planned
			for _, id := range ids {
				result.Items = append(result.Items, batchReviewItem{OrderID: id, Recorded: "not-recorded-as-started"})
			}
		case "starting":
			if pending != 0 || len(recorded) >= len(ids) || event.OrderID != ids[len(recorded)] || event.Item != nil || event.Receipt != nil || len(event.Planned) != 0 {
				return batchReview{}, nil, fmt.Errorf("invalid journal start")
			}
			pending = event.OrderID
			result.Items[len(recorded)].Recorded = "started-without-result"
		case "result":
			if pending == 0 || event.Item == nil || event.Item.OrderID != pending || event.Receipt != nil || event.OrderID != 0 || len(event.Planned) != 0 {
				return batchReview{}, nil, fmt.Errorf("invalid journal result")
			}
			result.Items[len(recorded)].Recorded = "result-recorded"
			result.Items[len(recorded)].RecordedStage = event.Item.Result.Stage
			recorded = append(recorded, *event.Item)
			pending = 0
		case "ended":
			if len(ids) == 0 || pending != 0 || event.Receipt == nil || event.Item != nil || event.OrderID != 0 || len(event.Planned) != 0 ||
				!reflect.DeepEqual(event.Receipt.Planned, ids) || !reflect.DeepEqual(event.Receipt.Items, recorded) {
				return batchReview{}, nil, fmt.Errorf("journal ending does not match recorded results")
			}
			result.Ended = true
		default:
			return batchReview{}, nil, fmt.Errorf("unknown journal event")
		}
	}
	if len(ids) == 0 || (result.Ended && result.IncompleteTail) {
		return batchReview{}, nil, fmt.Errorf("journal has no complete plan or trails its ending")
	}
	return result, ids, nil
}
