package deliverybundle

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

type ServiceReceipt struct {
	Catalog   string `json:"catalog"`
	Digest    string `json:"digest"`
	PID       int    `json:"pid"`
	Command   string `json:"command"`
	Log       string `json:"log"`
	StartedAt string `json:"startedAt"`
}

type serviceRecord struct {
	PID           int    `json:"pid"`
	Catalog       string `json:"catalog"`
	Digest        string `json:"digest"`
	StartedAt     string `json:"startedAt"`
	CommandDigest string `json:"commandDigest"`
	Log           string `json:"log"`
}

func pidFile(statePath string) string { return statePath + ".pid" }

// Start launches a trusted service command in the background and records its
// PID. It does not supervise, restart, or gracefully stop the process.
func Start(statePath, command, logPath string) (ServiceReceipt, error) {
	if command == "" {
		return ServiceReceipt{}, fmt.Errorf("service command is required")
	}
	state, err := Status(statePath)
	if err != nil {
		return ServiceReceipt{}, err
	}
	if _, err := os.Lstat(pidFile(statePath)); err == nil {
		return ServiceReceipt{}, fmt.Errorf("service is already recorded as running")
	} else if !os.IsNotExist(err) {
		return ServiceReceipt{}, err
	}
	if logPath == "" {
		logPath = statePath + ".log"
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return ServiceReceipt{}, err
	}
	cmd := exec.Command("bash", "-lc", command)
	cmd.Env = append(os.Environ(),
		"TEI_CATALOG="+state.Active.Catalog,
		"TEI_ACTIVE_DIGEST="+state.Active.Digest,
		"TEI_STATE="+statePath,
	)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		logFile.Close()
		return ServiceReceipt{}, err
	}
	pid := cmd.Process.Pid
	_ = cmd.Process.Release()
	_ = logFile.Close()
	record := serviceRecord{
		PID: pid, Catalog: state.Active.Catalog, Digest: state.Active.Digest,
		StartedAt: time.Now().UTC().Format(time.RFC3339), CommandDigest: digestBytes([]byte(command)), Log: logPath,
	}
	raw, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return ServiceReceipt{}, err
	}
	if err := os.WriteFile(pidFile(statePath), append(raw, '\n'), 0o600); err != nil {
		return ServiceReceipt{}, err
	}
	return ServiceReceipt{
		Catalog: state.Active.Catalog, Digest: state.Active.Digest, PID: pid,
		Command: command, Log: logPath, StartedAt: record.StartedAt,
	}, nil
}

// Stop sends Kill to the recorded PID. It does not drain, signal gracefully, or
// verify application-level shutdown.
func Stop(statePath string) (ServiceReceipt, error) {
	raw, err := os.ReadFile(pidFile(statePath))
	if err != nil {
		return ServiceReceipt{}, err
	}
	var record serviceRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return ServiceReceipt{}, fmt.Errorf("service record is not valid JSON: %w", err)
	}
	if record.PID <= 0 {
		return ServiceReceipt{}, fmt.Errorf("service record has no pid")
	}
	process, err := os.FindProcess(record.PID)
	if err != nil {
		return ServiceReceipt{}, err
	}
	if err := process.Kill(); err != nil {
		return ServiceReceipt{}, err
	}
	if err := os.Remove(pidFile(statePath)); err != nil {
		return ServiceReceipt{}, err
	}
	return ServiceReceipt{
		Catalog: record.Catalog, Digest: record.Digest, PID: record.PID,
		Log: record.Log, StartedAt: record.StartedAt,
	}, nil
}
