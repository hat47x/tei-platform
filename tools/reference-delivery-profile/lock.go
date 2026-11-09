package deliveryprofile

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

const LockSchema = "tei.reference.delivery-lock/v0"

type LockPlugin struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Digest  string `json:"digest"`
}

type Lock struct {
	Schema         string       `json:"schema"`
	ProfileID      string       `json:"profileId"`
	ProfileVersion string       `json:"profileVersion"`
	Plugins        []LockPlugin `json:"plugins"`
	Order          []string     `json:"order"`
	Digest         string       `json:"digest"`
}

func lockDigest(lock Lock) string {
	lock.Digest = ""
	raw, _ := json.Marshal(lock)
	return digestBytes(raw)
}

func BuildLock(receipt Receipt) Lock {
	plugins := make([]LockPlugin, 0, len(receipt.Plugins))
	for _, plugin := range receipt.Plugins {
		plugins = append(plugins, LockPlugin{ID: plugin.ID, Version: plugin.Version, Digest: plugin.Digest})
	}
	sort.Slice(plugins, func(i, j int) bool { return plugins[i].ID < plugins[j].ID })
	lock := Lock{
		Schema: LockSchema, ProfileID: receipt.ProfileID, ProfileVersion: receipt.ProfileVersion,
		Plugins: plugins, Order: append([]string(nil), receipt.Order...),
	}
	lock.Digest = lockDigest(lock)
	return lock
}

func WriteLock(receipt Receipt, path string) (Lock, error) {
	lock := BuildLock(receipt)
	raw, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return Lock{}, err
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		return Lock{}, err
	}
	return lock, nil
}

func LoadLock(path string) (Lock, error) {
	var lock Lock
	if err := readJSON(path, &lock); err != nil {
		return Lock{}, err
	}
	if lock.Schema != LockSchema {
		return Lock{}, fmt.Errorf("lock schema %q is not %q", lock.Schema, LockSchema)
	}
	if lock.Digest == "" || lock.Digest != lockDigest(lock) {
		return Lock{}, fmt.Errorf("lock digest is missing or does not match its content")
	}
	return lock, nil
}

func VerifyLock(lock Lock, receipt Receipt) error {
	if lock.ProfileID != receipt.ProfileID || lock.ProfileVersion != receipt.ProfileVersion {
		return fmt.Errorf("lock targets %s@%s, profile is %s@%s", lock.ProfileID, lock.ProfileVersion, receipt.ProfileID, receipt.ProfileVersion)
	}
	if len(lock.Plugins) != len(receipt.Plugins) {
		return fmt.Errorf("lock has %d plugins, resolved profile has %d", len(lock.Plugins), len(receipt.Plugins))
	}
	expected := BuildLock(receipt)
	for i := range expected.Plugins {
		if lock.Plugins[i] != expected.Plugins[i] {
			return fmt.Errorf("lock plugin %d is %+v, expected %+v", i, lock.Plugins[i], expected.Plugins[i])
		}
	}
	if len(lock.Order) != len(expected.Order) {
		return fmt.Errorf("lock order length does not match the resolved dependency order")
	}
	for i := range expected.Order {
		if lock.Order[i] != expected.Order[i] {
			return fmt.Errorf("lock order does not match the resolved dependency order")
		}
	}
	return nil
}

func ValidateWithLock(profilePath, catalogDir, lockPath string) (Receipt, error) {
	receipt, err := ValidateWithCatalog(profilePath, catalogDir)
	if err != nil {
		return Receipt{}, err
	}
	if lockPath == "" {
		return receipt, nil
	}
	lock, err := LoadLock(lockPath)
	if err != nil {
		return Receipt{}, err
	}
	if err := VerifyLock(lock, receipt); err != nil {
		return Receipt{}, err
	}
	receipt.LockVerified = true
	receipt.LockDigest = lock.Digest
	return receipt, nil
}

func ResolveLock(profilePath, catalogDir, lockPath string) (Lock, error) {
	receipt, err := ValidateWithCatalog(profilePath, catalogDir)
	if err != nil {
		return Lock{}, err
	}
	if !receipt.Passed {
		return Lock{}, fmt.Errorf("profile did not pass validation; lock not written")
	}
	return WriteLock(receipt, lockPath)
}
