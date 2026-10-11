package scenariotest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// ObservationFixture contains the result of an independent synthetic read.
// It is NOT evidence of a real Command effect.
type ObservationFixture struct {
	ReadIdentity string `json:"readIdentity"`
	Nature       string `json:"nature"`
	Observed     Value  `json:"observed"`
}

func ParseObservationFixture(source []byte) (ObservationFixture, error) {
	var f ObservationFixture
	d := json.NewDecoder(bytes.NewReader(source))
	d.DisallowUnknownFields()
	if err := d.Decode(&f); err != nil {
		return ObservationFixture{}, fmt.Errorf("decode Observation Fixture: %w", err)
	}
	if d.Decode(new(any)) != io.EOF {
		return ObservationFixture{}, fmt.Errorf("trailing Observation Fixture content")
	}
	if f.ReadIdentity == "" || f.Nature != "synthetic" {
		return ObservationFixture{}, fmt.Errorf("Observation Fixture must identify a synthetic read")
	}
	if _, err := f.Observed.Decode(); err != nil {
		return ObservationFixture{}, err
	}
	return f, nil
}
