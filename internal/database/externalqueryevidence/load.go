package externalqueryevidence

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// Loadはexternal Contract Query Evidenceを厳格にJSONから読み込む。
// 未知fieldや一つ目のJSON値に続く別データを黙って無視せず、Evidence formatのずれを明示的に止める。
//
// ここでは構文だけを読み込む。Canonical World / SQL realizationとの意味整合はValidateAgainstで確認する。
func Load(data []byte) (Evidence, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var evidence Evidence
	if err := decoder.Decode(&evidence); err != nil {
		return Evidence{}, fmt.Errorf("external Contract Query Evidence JSONを読み込めません: %w", err)
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return Evidence{}, fmt.Errorf("external Contract Query Evidence JSONに複数の値があります")
		}
		return Evidence{}, fmt.Errorf("external Contract Query Evidence JSON末尾を確認できません: %w", err)
	}
	return evidence, nil
}
