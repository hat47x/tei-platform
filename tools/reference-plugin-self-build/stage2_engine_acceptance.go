package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"tei/internal/core"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/plugin/transform"
)

// This Reference-only stage lock describes a builder execution, not a Plugin.
// It cannot be substituted for a production RuntimeEngineLock or Host authority.
const referenceStage2LockFormatVersion = 1
const referenceStage2Stage = "stage2"

type referenceStage2EngineLock struct {
	FormatVersion int                            `json:"formatVersion"`
	Stage         string                         `json:"stage"`
	RuntimeEngine pruntime.RuntimeEngineIdentity `json:"runtimeEngine"`
}

type referenceStage2HostRegistration struct {
	FormatVersion int                            `json:"formatVersion"`
	Stage         string                         `json:"stage"`
	HostID        core.CanonicalID               `json:"hostId"`
	RuntimeEngine pruntime.RuntimeEngineIdentity `json:"runtimeEngine"`
}

type referenceStage2Acceptance struct {
	Engine                      pruntime.RuntimeEngineIdentity
	LockFingerprint             string
	HostRegistrationFingerprint string
}

func readReferenceStage2JSON(path string, target any) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("stage2 authority record must be a regular non-symlink file")
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, 1<<20+1))
	if err != nil {
		return err
	}
	if len(raw) > 1<<20 {
		return fmt.Errorf("stage2 authority record exceeds 1 MiB")
	}
	if _, err := transform.ParseJSON(raw); err != nil {
		return fmt.Errorf("stage2 authority record ambiguous or invalid JSON: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode stage2 authority record: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("stage2 authority record has trailing content")
	}
	return nil
}

func referenceStage2Fingerprint(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

// acceptReferenceStage2Engine is invoked before the external engine can run.
// A reproducible build is not an authorization: both independently supplied
// Host-owned records must match the exact binary identity.
func acceptReferenceStage2Engine(options exactEngineOptions, engine pruntime.RuntimeEngineIdentity) (referenceStage2Acceptance, error) {
	if options.Stage2Lock == "" || options.HostRegistration == "" {
		return referenceStage2Acceptance{}, fmt.Errorf("exact Runtime Engine requires an explicit stage2 lock and Host registration before invocation")
	}
	if err := engine.Validate(); err != nil {
		return referenceStage2Acceptance{}, err
	}
	var lock referenceStage2EngineLock
	if err := readReferenceStage2JSON(options.Stage2Lock, &lock); err != nil {
		return referenceStage2Acceptance{}, fmt.Errorf("stage2 engine lock: %w", err)
	}
	if lock.FormatVersion != referenceStage2LockFormatVersion || lock.Stage != referenceStage2Stage {
		return referenceStage2Acceptance{}, fmt.Errorf("stage2 engine lock has unsupported format or stage")
	}
	if err := lock.RuntimeEngine.Validate(); err != nil {
		return referenceStage2Acceptance{}, fmt.Errorf("stage2 engine lock identity: %w", err)
	}
	if lock.RuntimeEngine != engine {
		return referenceStage2Acceptance{}, fmt.Errorf("stage2 engine lock expected identity differs from verified executable")
	}
	var registration referenceStage2HostRegistration
	if err := readReferenceStage2JSON(options.HostRegistration, &registration); err != nil {
		return referenceStage2Acceptance{}, fmt.Errorf("stage2 Host registration: %w", err)
	}
	if registration.FormatVersion != referenceStage2LockFormatVersion || registration.Stage != referenceStage2Stage {
		return referenceStage2Acceptance{}, fmt.Errorf("stage2 Host registration has unsupported format or stage")
	}
	if _, err := core.ParseCanonicalID(string(registration.HostID)); err != nil {
		return referenceStage2Acceptance{}, fmt.Errorf("stage2 Host registration identity: %w", err)
	}
	if err := registration.RuntimeEngine.Validate(); err != nil {
		return referenceStage2Acceptance{}, fmt.Errorf("stage2 Host registration engine: %w", err)
	}
	if registration.RuntimeEngine != lock.RuntimeEngine {
		return referenceStage2Acceptance{}, fmt.Errorf("stage2 Host registration differs from accepted engine lock")
	}
	lockFP, err := referenceStage2Fingerprint(lock)
	if err != nil {
		return referenceStage2Acceptance{}, err
	}
	registrationFP, err := referenceStage2Fingerprint(registration)
	if err != nil {
		return referenceStage2Acceptance{}, err
	}
	return referenceStage2Acceptance{
		Engine:                      engine,
		LockFingerprint:             lockFP,
		HostRegistrationFingerprint: registrationFP,
	}, nil
}

// The execution policy binds the separately accepted Host and engine lock to
// the exact Stage2 plan. It is not a production Host registration signature.
func (acceptance referenceStage2Acceptance) PolicyFingerprint(stage2Plan transformBootstrapStagePlan) (string, error) {
	if stage2Plan.Stage != "stage2" || stage2Plan.Execution.RuntimeEngine == nil ||
		*stage2Plan.Execution.RuntimeEngine != acceptance.Engine {
		return "", fmt.Errorf("accepted Runtime Engine does not match stage2 execution plan")
	}
	planFP, err := transformBootstrapStagePlanFingerprint(stage2Plan)
	if err != nil {
		return "", err
	}
	return referenceStage2Fingerprint(struct {
		Schema                      string `json:"schema"`
		LockFingerprint             string `json:"lockFingerprint"`
		HostRegistrationFingerprint string `json:"hostRegistrationFingerprint"`
		Stage2PlanFingerprint       string `json:"stage2PlanFingerprint"`
	}{
		Schema:                      "tei.reference.stage2-runtime-policy/v1",
		LockFingerprint:             acceptance.LockFingerprint,
		HostRegistrationFingerprint: acceptance.HostRegistrationFingerprint,
		Stage2PlanFingerprint:       planFP,
	})
}
