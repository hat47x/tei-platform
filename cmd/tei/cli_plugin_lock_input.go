package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"tei/internal/plugin"
)

// decodePluginLockFile reads a Plugin Lock, accepting both the bare lock object
// (`{"formatVersion": 2, ...}`) and the `tei.cli.plugin-lock/v0` envelope that
// `tei plugin-lock` writes (`{"schemaVersion": ..., "ok": true, "result": {...}}`).
//
// The CLI writes the envelope, so a pipeline must be able to feed that file
// straight back into -plugin-lock without an external `jq`/`python3` step
// (#3183). The shape is selected by the presence of `result`, not by guessing
// a format version: a bare lock has no `result`, and an envelope carries its
// lock there. A file with neither is decoded as a bare lock, so the existing
// "unsupported plugin lock format" message keeps naming the value it saw.
func decodePluginLockFile(path string) (plugin.Lock, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return plugin.Lock{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var top map[string]json.RawMessage
	if err := decoder.Decode(&top); err != nil {
		return plugin.Lock{}, fmt.Errorf("%s: %w", path, err)
	}
	if result, ok := top["result"]; ok {
		var lock plugin.Lock
		if err := json.Unmarshal(result, &lock); err != nil {
			return plugin.Lock{}, fmt.Errorf("%s: decode lock in result: %w", path, err)
		}
		return lock, nil
	}
	var lock plugin.Lock
	if err := json.Unmarshal(raw, &lock); err != nil {
		return plugin.Lock{}, fmt.Errorf("%s: %w", path, err)
	}
	return lock, nil
}
