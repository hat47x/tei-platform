// Package changepacket validates a machine-readable change packet and runs the
// verification it declares. It does not grant permission, accept meaning, or
// merge anything: it only makes the declared scope and evidence checkable.
package changepacket

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const Schema = "tei.reference.change-packet/v0"

const (
	LevelA0 = "A0"
	LevelA1 = "A1"
	LevelA2 = "A2"
	LevelA3 = "A3"
	LevelA4 = "A4"
)

var (
	idPattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
	levels      = map[string]bool{LevelA0: true, LevelA1: true, LevelA2: true, LevelA3: true, LevelA4: true}
	maxOutput   = 16 * 1024
	gitTimeout  = 30 * time.Second
	defaultStep = 10 * time.Minute
)

type Command struct {
	Name           string `json:"name"`
	Run            string `json:"run"`
	TimeoutSeconds int    `json:"timeoutSeconds,omitempty"`
}

type Packet struct {
	Schema         string    `json:"schema"`
	ID             string    `json:"id"`
	Issue          int       `json:"issue"`
	Title          string    `json:"title"`
	Level          string    `json:"level"`
	Base           string    `json:"base,omitempty"`
	AllowedPaths   []string  `json:"allowedPaths"`
	ForbiddenPaths []string  `json:"forbiddenPaths,omitempty"`
	Verification   []Command `json:"verification"`
	StopConditions []string  `json:"stopConditions"`
	Evidence       []string  `json:"evidence,omitempty"`
	Notes          []string  `json:"notes,omitempty"`
}

type Violation struct {
	Path string `json:"path"`
	Rule string `json:"rule"`
}

type Step struct {
	Name      string `json:"name"`
	Command   string `json:"command"`
	ExitCode  int    `json:"exitCode"`
	Output    string `json:"output,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
	Error     string `json:"error,omitempty"`
}

type Receipt struct {
	Schema         string      `json:"schema"`
	PacketID       string      `json:"packetId"`
	Level          string      `json:"level"`
	Head           string      `json:"head"`
	Base           string      `json:"base"`
	ChangedFiles   []string    `json:"changedFiles"`
	Scope          []Violation `json:"scopeViolations,omitempty"`
	Steps          []Step      `json:"steps"`
	StopConditions []string    `json:"stopConditions,omitempty"`
	Passed         bool        `json:"passed"`
}

// Decode reads one packet and rejects unknown fields so a misspelled policy is
// visible instead of silently ignored.
func Decode(raw []byte) (Packet, error) {
	var packet Packet
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&packet); err != nil {
		return Packet{}, err
	}
	return packet, nil
}

func normalizePath(value string) (string, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" {
		return "", fmt.Errorf("path is empty")
	}
	if strings.HasPrefix(value, "/") || filepath.IsAbs(value) {
		return "", fmt.Errorf("path %q must be relative", value)
	}
	for _, part := range strings.Split(value, "/") {
		if part == ".." {
			return "", fmt.Errorf("path %q must not contain ..", value)
		}
	}
	clean := path.Clean(value)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("path %q escapes the repository", value)
	}
	if strings.Contains(strings.SplitN(clean, "/", 2)[0], ":") {
		return "", fmt.Errorf("path %q must not be an absolute drive path", value)
	}
	return clean, nil
}

func normalizePatterns(values []string, label string) ([]string, error) {
	out := make([]string, 0, len(values))
	for _, value := range values {
		normalized, err := normalizePath(value)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", label, err)
		}
		out = append(out, normalized)
	}
	return out, nil
}

func Validate(packet Packet) error {
	if packet.Schema != Schema {
		return fmt.Errorf("schema %q is not %q", packet.Schema, Schema)
	}
	if !idPattern.MatchString(packet.ID) {
		return fmt.Errorf("id %q must match %s", packet.ID, idPattern)
	}
	if packet.Issue <= 0 {
		return fmt.Errorf("issue must be positive")
	}
	if strings.TrimSpace(packet.Title) == "" {
		return fmt.Errorf("title is required")
	}
	if !levels[packet.Level] {
		return fmt.Errorf("level %q must be one of A0, A1, A2, A3, A4", packet.Level)
	}
	if len(packet.AllowedPaths) == 0 {
		return fmt.Errorf("allowedPaths must not be empty")
	}
	if _, err := normalizePatterns(packet.AllowedPaths, "allowedPaths"); err != nil {
		return err
	}
	if _, err := normalizePatterns(packet.ForbiddenPaths, "forbiddenPaths"); err != nil {
		return err
	}
	if len(packet.Verification) == 0 {
		return fmt.Errorf("verification must not be empty")
	}
	for i, command := range packet.Verification {
		if strings.TrimSpace(command.Name) == "" {
			return fmt.Errorf("verification[%d].name is required", i)
		}
		if strings.TrimSpace(command.Run) == "" {
			return fmt.Errorf("verification[%d].run is required", i)
		}
		if command.TimeoutSeconds < 0 || command.TimeoutSeconds > 3600 {
			return fmt.Errorf("verification[%d].timeoutSeconds must be between 1 and 3600", i)
		}
	}
	if len(packet.StopConditions) == 0 {
		return fmt.Errorf("stopConditions must not be empty")
	}
	if (packet.Level == LevelA2 || packet.Level == LevelA3) && strings.TrimSpace(packet.Base) == "" {
		return fmt.Errorf("base is required for %s", packet.Level)
	}
	return nil
}

func matchPattern(pattern, file string) bool {
	pattern = strings.TrimSuffix(pattern, "/")
	if pattern == "**" {
		return true
	}
	if strings.HasSuffix(pattern, "/**") {
		base := strings.TrimSuffix(pattern, "/**")
		return file == base || strings.HasPrefix(file, base+"/")
	}
	if matched, _ := path.Match(pattern, file); matched {
		return true
	}
	if !strings.ContainsAny(pattern, "*?[") {
		return strings.HasPrefix(file, pattern+"/")
	}
	return false
}

// ScopeViolations reports changed paths that are forbidden or outside the
// packet's allowed paths. An empty result means only that the path set matched
// the packet, not that the change is safe or accepted.
func ScopeViolations(changed []string, packet Packet) []Violation {
	allowed, err := normalizePatterns(packet.AllowedPaths, "allowedPaths")
	if err != nil {
		return []Violation{{Path: "", Rule: "invalid allowedPaths: " + err.Error()}}
	}
	forbidden, err := normalizePatterns(packet.ForbiddenPaths, "forbiddenPaths")
	if err != nil {
		return []Violation{{Path: "", Rule: "invalid forbiddenPaths: " + err.Error()}}
	}
	seen := map[string]bool{}
	var violations []Violation
	add := func(file, rule string) {
		key := file + "\x00" + rule
		if seen[key] {
			return
		}
		seen[key] = true
		violations = append(violations, Violation{Path: file, Rule: rule})
	}
	for _, raw := range changed {
		file, err := normalizePath(raw)
		if err != nil {
			add(raw, "invalid changed path")
			continue
		}
		forbiddenHit := false
		for _, pattern := range forbidden {
			if matchPattern(pattern, file) {
				add(file, "forbidden: "+pattern)
				forbiddenHit = true
			}
		}
		if forbiddenHit {
			continue
		}
		matched := false
		for _, pattern := range allowed {
			if matchPattern(pattern, file) {
				matched = true
				break
			}
		}
		if !matched {
			add(file, "outside allowedPaths")
		}
	}
	sort.Slice(violations, func(i, j int) bool {
		if violations[i].Path != violations[j].Path {
			return violations[i].Path < violations[j].Path
		}
		return violations[i].Rule < violations[j].Rule
	})
	return violations
}

func gitOutput(ctx context.Context, repo string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return string(out), fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(exit.Stderr)))
		}
		return string(out), err
	}
	return string(out), nil
}

// ChangedFiles returns tracked changes against base plus untracked files. It
// does not include staged-only state separately; the diff against base covers
// both committed and working-tree changes.
func ChangedFiles(ctx context.Context, repo, base string) ([]string, error) {
	if strings.TrimSpace(base) == "" {
		return nil, fmt.Errorf("base is required")
	}
	ctx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()
	out, err := gitOutput(ctx, repo, "diff", "--name-only", base, "--")
	if err != nil {
		return nil, err
	}
	untracked, err := gitOutput(ctx, repo, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var files []string
	for _, line := range strings.Split(out+"\n"+untracked, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || seen[line] {
			continue
		}
		seen[line] = true
		files = append(files, line)
	}
	sort.Strings(files)
	return files, nil
}

func headRevision(ctx context.Context, repo string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()
	out, err := gitOutput(ctx, repo, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func truncate(value string) (string, bool) {
	if len(value) <= maxOutput {
		return value, false
	}
	return value[:maxOutput], true
}

func runCommand(ctx context.Context, repo string, command Command) Step {
	step := Step{Name: command.Name, Command: command.Run}
	timeout := defaultStep
	if command.TimeoutSeconds > 0 {
		timeout = time.Duration(command.TimeoutSeconds) * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, "bash", "-lc", command.Run)
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	step.Output, step.Truncated = truncate(string(out))
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			step.ExitCode = exit.ExitCode()
		} else {
			step.ExitCode = -1
			step.Error = err.Error()
		}
		return step
	}
	step.ExitCode = 0
	return step
}

// Run checks the packet against the repository's changed paths and executes
// its verification commands. A command failure, a scope violation, or an
// unreadable git state makes the receipt fail; none of them grants authority.
func Run(ctx context.Context, packet Packet, repo string) (Receipt, error) {
	if err := Validate(packet); err != nil {
		return Receipt{}, err
	}
	head, err := headRevision(ctx, repo)
	if err != nil {
		return Receipt{}, err
	}
	receipt := Receipt{
		Schema: Schema, PacketID: packet.ID, Level: packet.Level,
		Head: head, Base: packet.Base, Steps: []Step{},
		StopConditions: append([]string(nil), packet.StopConditions...),
	}
	base := packet.Base
	if base == "" {
		base = "HEAD"
		receipt.Base = base
	}
	changed, err := ChangedFiles(ctx, repo, base)
	if err != nil {
		return Receipt{}, err
	}
	receipt.ChangedFiles = changed
	receipt.Scope = ScopeViolations(changed, packet)
	if len(receipt.Scope) > 0 {
		receipt.Passed = false
		return receipt, nil
	}
	for _, command := range packet.Verification {
		step := runCommand(ctx, repo, command)
		receipt.Steps = append(receipt.Steps, step)
		if step.ExitCode != 0 {
			receipt.Passed = false
		}
	}
	receipt.Passed = true
	for _, step := range receipt.Steps {
		if step.ExitCode != 0 {
			receipt.Passed = false
			break
		}
	}
	return receipt, nil
}

// Load reads and validates a packet file.
func Load(pathname string) (Packet, error) {
	raw, err := os.ReadFile(pathname)
	if err != nil {
		return Packet{}, err
	}
	packet, err := Decode(raw)
	if err != nil {
		return Packet{}, err
	}
	if err := Validate(packet); err != nil {
		return Packet{}, err
	}
	return packet, nil
}
