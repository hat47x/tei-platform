// Package main verifies an offline Go module supply (#2831 SB-0b).
//
// The check binds the physical module files in a named supply directory to the
// hashes the repository's go.sum commits to, then re-resolves the target using
// that supply alone and a fresh isolated module cache. It does not run a build,
// create an artifact, or attest a supply origin, publisher, signature or
// authority. A verified supply is a precondition for the next step, not an
// acceptance.
package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const supplySchema = "tei.reference.go-module-supply-verification/v0"

const verificationBoundary = "exact go.sum hashes for each declared module@version, re-resolved offline from the named supply only; no supply origin, publisher, signature, toolchain trust or build authority"

// ModuleRequirement is one module@version the repository declares and the
// supply is expected to contain.
type ModuleRequirement struct {
	Path     string `json:"path"`
	Version  string `json:"version"`
	Declared bool   `json:"declared"`
}

// FileCheck is the verdict for one physical module file.
type FileCheck struct {
	Name          string `json:"name"`
	Present       bool   `json:"present"`
	SizeBytes     int64  `json:"sizeBytes,omitempty"`
	SHA256        string `json:"sha256,omitempty"`
	GoSumLine     string `json:"goSumLine,omitempty"`
	GoSumHash     string `json:"goSumHash,omitempty"`
	ComputedHash  string `json:"computedHash,omitempty"`
	MatchesGoSum  bool   `json:"matchesGoSum"`
	FailureReason string `json:"failureReason,omitempty"`
}

// ModuleCheck is the per-module verdict.
type ModuleCheck struct {
	Path         string    `json:"path"`
	Version      string    `json:"version"`
	SupplyDir    string    `json:"supplyDir,omitempty"`
	Mod          FileCheck `json:"mod"`
	Zip          FileCheck `json:"zip"`
	Verified     bool      `json:"verified"`
	FailureCause string    `json:"failureCause,omitempty"`
}

// SupplyVerification is the evidence record for one verification run.
type SupplyVerification struct {
	SchemaVersion      string        `json:"schemaVersion"`
	Nature             string        `json:"nature"`
	Repository         string        `json:"repository"`
	SupplyRoot         string        `json:"supplyRoot"`
	GoSumPresent       bool          `json:"goSumPresent"`
	GoSumDigest        string        `json:"goSumDigest,omitempty"`
	DeclaredModules    []string      `json:"declaredModules"`
	UnselectedDeclared []string      `json:"unselectedDeclaredModules,omitempty"`
	Modules            []ModuleCheck `json:"modules"`
	Verified           bool          `json:"verified"`
	ResolvedModules    []string      `json:"resolvedModules,omitempty"`
	UnverifiedResolved []string      `json:"unverifiedResolvedModules,omitempty"`
	UnresolvedPackages []string      `json:"unresolvedPackages,omitempty"`
	ReresolveExitCode  int           `json:"reresolveExitCode"`
	ReresolveError     string        `json:"reresolveError,omitempty"`
	FailureCause       string        `json:"failureCause,omitempty"`
	Boundary           string        `json:"verificationBoundary"`
}

// VerifySupply re-resolves the target from the named supply alone, then binds
// every module the resolver actually selected to the repository's go.sum.
//
// Selection is resolved before byte verification on purpose: the supply only
// has to satisfy the modules the target really consumes, not every module the
// go.mod declares. The Go toolchain independently rejects a supply whose bytes
// disagree with go.sum, so this reference recomputes the same hashes separately
// rather than trusting the resolver's success.
func VerifySupply(repo, supplyRoot, packagePath, goPath, stateDir string, explicit []ModuleRequirement) (SupplyVerification, error) {
	root, err := cleanPath(repo)
	if err != nil {
		return SupplyVerification{}, err
	}
	supply, err := cleanPath(supplyRoot)
	if err != nil {
		return SupplyVerification{}, fmt.Errorf("supply root is not a readable directory: %w", err)
	}
	verification := SupplyVerification{
		SchemaVersion: supplySchema,
		Nature:        "offline module supply verification; not a build attempt",
		Repository:    root,
		SupplyRoot:    supply,
		Boundary:      verificationBoundary,
	}

	goSum, present, err := readGoSum(root)
	if err != nil {
		return SupplyVerification{}, err
	}
	verification.GoSumPresent = present
	if !present {
		verification.FailureCause = "go.sum is absent; module bytes cannot be bound"
		return verification, nil
	}
	verification.GoSumDigest = digestBytes(goSum)
	sums, err := parseGoSum(goSum)
	if err != nil {
		return SupplyVerification{}, err
	}

	declared, err := declaredRequirements(root)
	if err != nil {
		return SupplyVerification{}, err
	}
	for _, requirement := range declared {
		verification.DeclaredModules = append(verification.DeclaredModules, requirement.Path+"@"+requirement.Version)
	}
	if len(declared) == 0 {
		verification.FailureCause = "go.mod declares no module requirements"
		return verification, nil
	}
	if len(explicit) > 0 {
		declared = explicit
		verification.DeclaredModules = nil
		for _, requirement := range declared {
			verification.DeclaredModules = append(verification.DeclaredModules, requirement.Path+"@"+requirement.Version)
		}
	}

	resolved, unresolved, exitCode, commandError, err := reresolveFromSupply(root, supply, packagePath, goPath, stateDir)
	if err != nil {
		return SupplyVerification{}, err
	}
	verification.ResolvedModules = resolved
	verification.UnresolvedPackages = unresolved
	verification.ReresolveExitCode = exitCode
	verification.ReresolveError = commandError
	if exitCode != 0 || len(unresolved) > 0 {
		// The failure is not attributable by module identity alone. Check the
		// bytes of every declared module the supply claims to carry, so the
		// record says whether a tampered file or a missing file produced the
		// stop. Both outcomes remain `verified: false`.
		checks, tampered := byteCheckModules(supply, declared, sums)
		verification.Modules = checks
		verification.FailureCause = "the supply alone cannot resolve the target offline"
		if tampered != "" {
			verification.FailureCause = "supply bytes do not match go.sum for " + tampered
		}
		return verification, nil
	}

	declaredSet := map[string]struct{}{}
	for _, requirement := range declared {
		declaredSet[requirement.Path+"@"+requirement.Version] = struct{}{}
	}
	for _, module := range resolved {
		if _, ok := declaredSet[module]; !ok {
			verification.UnverifiedResolved = append(verification.UnverifiedResolved, module)
		}
	}
	for _, requirement := range declared {
		if !slicesContains(resolved, requirement.Path+"@"+requirement.Version) {
			verification.UnselectedDeclared = append(verification.UnselectedDeclared, requirement.Path+"@"+requirement.Version)
		}
	}
	if len(verification.UnverifiedResolved) > 0 {
		verification.FailureCause = "re-resolution selected modules that go.mod never declared"
		return verification, nil
	}

	// A declared module that the target does not select is outside this
	// target's supply requirement; record it instead of failing on it. An
	// explicitly named module is always verified, so a caller can demand a
	// module the pure resolver would not choose.
	mustVerify := resolved
	if len(explicit) > 0 {
		mustVerify = append([]string(nil), resolved...)
		for _, requirement := range explicit {
			identifier := requirement.Path + "@" + requirement.Version
			if !slicesContains(mustVerify, identifier) {
				mustVerify = append(mustVerify, identifier)
			}
		}
	}
	sort.Strings(mustVerify)
	if len(mustVerify) == 0 {
		// Nothing was checked, so nothing is verified. A target that selects
		// no external module does not exercise the supply at all; reporting
		// `verified` here would be a vacuous claim.
		verification.FailureCause = "the target selects no external module; the supply was never exercised"
		return verification, nil
	}

	verified := true
	for _, module := range mustVerify {
		requirement, ok := splitModuleVersion(module)
		if !ok {
			return SupplyVerification{}, fmt.Errorf("resolver reported a module without a version: %s", module)
		}
		checks, cause := byteCheckOne(supply, requirement, sums)
		for _, check := range checks {
			if !check.Verified {
				verified = false
			}
		}
		if cause != "" && verification.FailureCause == "" {
			verification.FailureCause = cause
		}
		verification.Modules = append(verification.Modules, checks...)
	}
	verification.Verified = verified
	if !verified {
		if verification.FailureCause == "" {
			verification.FailureCause = "a selected module's bytes do not match go.sum"
		}
	}
	return verification, nil
}

// byteCheckModules checks every supplied directory for the named modules and
// returns the checks plus the first module whose bytes disagree with go.sum.
func byteCheckModules(supply string, requirements []ModuleRequirement, sums map[string]map[string]string) ([]ModuleCheck, string) {
	var checks []ModuleCheck
	tampered := ""
	for _, requirement := range requirements {
		moduleChecks, _ := byteCheckOne(supply, requirement, sums)
		for _, check := range moduleChecks {
			checks = append(checks, check)
			if !check.Verified && tampered == "" && check.Mod.Present && check.Zip.Present {
				tampered = check.Path
			}
		}
	}
	return checks, tampered
}

// byteCheckOne verifies the go.mod and zip of one module and returns the
// per-file verdicts with a stable failure cause.
func byteCheckOne(supply string, requirement ModuleRequirement, sums map[string]map[string]string) ([]ModuleCheck, string) {
	check := ModuleCheck{Path: requirement.Path, Version: requirement.Version}
	dir, err := supplyModuleDir(supply, requirement)
	if err != nil {
		check.FailureCause = err.Error()
		return []ModuleCheck{check}, check.FailureCause
	}
	check.SupplyDir = dir
	check.Mod = checkFile(dir, requirement, sums, false)
	check.Zip = checkFile(dir, requirement, sums, true)
	check.Verified = check.Mod.Present && check.Mod.MatchesGoSum && check.Zip.Present && check.Zip.MatchesGoSum
	if !check.Verified {
		check.FailureCause = firstFailure(check.Mod, check.Zip)
	}
	return []ModuleCheck{check}, check.FailureCause
}

func slicesContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// splitModuleVersion splits a "path@version" identifier at the LAST '@', so a
// module path containing an '@' cannot be misread.
func splitModuleVersion(module string) (ModuleRequirement, bool) {
	index := strings.LastIndex(module, "@")
	if index <= 0 || index == len(module)-1 {
		return ModuleRequirement{}, false
	}
	return ModuleRequirement{Path: module[:index], Version: module[index+1:], Declared: true}, true
}

func cleanPath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if evaluated, err := filepath.EvalSymlinks(absolute); err == nil {
		return evaluated, nil
	}
	return filepath.Clean(absolute), nil
}

func readGoSum(root string) ([]byte, bool, error) {
	path := filepath.Join(root, "go.sum")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return raw, true, nil
}

// declaredRequirements reads the root module's go.mod require directives. It
// deliberately reads the declaration, not a resolver cache, so that an
// unverified module never enters by accident. Only single-line require
// directives are read; a parenthesized block is expanded line by line.
func declaredRequirements(root string) ([]ModuleRequirement, error) {
	raw, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return nil, fmt.Errorf("read go.mod: %w", err)
	}
	var out []ModuleRequirement
	inBlock := false
	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if index := strings.Index(trimmed, "//"); index >= 0 {
			trimmed = strings.TrimSpace(trimmed[:index])
		}
		switch {
		case strings.HasPrefix(trimmed, "require ("):
			inBlock = true
			continue
		case inBlock && trimmed == ")":
			inBlock = false
			continue
		case strings.HasPrefix(trimmed, "require "):
			trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "require "))
		case !inBlock:
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) != 2 {
			return nil, fmt.Errorf("go.mod require line is not path and version: %q", trimmed)
		}
		out = append(out, ModuleRequirement{Path: fields[0], Version: fields[1], Declared: true})
	}
	return out, nil
}

func supplyModuleDir(supply string, requirement ModuleRequirement) (string, error) {
	path := filepath.FromSlash(requirement.Path)
	if filepath.IsAbs(path) || strings.Contains(path, "..") {
		return "", fmt.Errorf("module path is not a relative Go module path: %s", requirement.Path)
	}
	dir := filepath.Join(supply, path, "@v")
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("supply entry is not a directory: %s", dir)
	}
	return dir, nil
}

func checkFile(dir string, requirement ModuleRequirement, sums map[string]map[string]string, isZip bool) FileCheck {
	kind := "go.mod"
	fileName := requirement.Version + ".mod"
	if isZip {
		kind = "zip"
		fileName = requirement.Version + ".zip"
	}
	check := FileCheck{Name: kind}
	line, hash, ok := goSumEntry(sums, requirement.Path, requirement.Version, isZip)
	check.GoSumLine, check.GoSumHash = line, hash
	if !ok {
		check.FailureReason = "go.sum has no entry for this module@version"
		return check
	}
	if dir == "" {
		check.FailureReason = "supply has no directory for this module"
		return check
	}
	path := filepath.Join(dir, fileName)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			check.FailureReason = fileName + " is absent from the supply"
			return check
		}
		check.FailureReason = err.Error()
		return check
	}
	if !info.Mode().IsRegular() {
		check.FailureReason = fileName + " is not a regular file"
		return check
	}
	check.Present = true
	check.SizeBytes = info.Size()
	raw, err := os.ReadFile(path)
	if err != nil {
		check.FailureReason = err.Error()
		return check
	}
	check.SHA256 = digestBytes(raw)
	computed, err := moduleHash(raw, isZip)
	if err != nil {
		check.FailureReason = err.Error()
		return check
	}
	check.ComputedHash = computed
	check.MatchesGoSum = computed == hash
	if !check.MatchesGoSum {
		check.FailureReason = "computed module hash differs from go.sum"
	}
	return check
}

func firstFailure(checks ...FileCheck) string {
	for _, check := range checks {
		if !check.Present || !check.MatchesGoSum {
			if check.FailureReason != "" {
				return check.FailureReason
			}
			return check.Name + " does not match go.sum"
		}
	}
	return ""
}

// moduleHash reproduces the h1 module hash go.sum records. For a zip it hashes
// each file's SHA-256 in sorted-name order; for go.mod the same construction
// uses the single name "go.mod". The encoding is "<hex>  <name>\n" per member.
func moduleHash(raw []byte, isZip bool) (string, error) {
	if !isZip {
		return h1Of([]member{{name: "go.mod", raw: raw}}), nil
	}
	reader, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return "", fmt.Errorf("supply zip is unreadable: %w", err)
	}
	members := make([]member, 0, len(reader.File))
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		handle, err := file.Open()
		if err != nil {
			return "", err
		}
		sum := sha256.New()
		if _, err := io.Copy(sum, handle); err != nil {
			handle.Close()
			return "", err
		}
		handle.Close()
		members = append(members, member{name: file.Name, sum: sum.Sum(nil)})
	}
	sort.Slice(members, func(i, j int) bool { return members[i].name < members[j].name })
	return h1Of(members), nil
}

type member struct {
	name string
	raw  []byte
	sum  []byte
}

func h1Of(members []member) string {
	var buf bytes.Buffer
	for _, item := range members {
		digest := item.sum
		if digest == nil {
			sum := sha256.Sum256(item.raw)
			digest = sum[:]
		}
		buf.WriteString(hex.EncodeToString(digest))
		buf.WriteString("  ")
		buf.WriteString(item.name)
		buf.WriteString("\n")
	}
	sum := sha256.Sum256(buf.Bytes())
	return "h1:" + base64.StdEncoding.EncodeToString(sum[:])
}

func digestBytes(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func parseGoSum(raw []byte) (map[string]map[string]string, error) {
	out := map[string]map[string]string{}
	for number, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) != 3 {
			return nil, fmt.Errorf("go.sum line %d is not three fields", number+1)
		}
		key := fields[0] + " " + fields[1]
		out[key] = map[string]string{"line": trimmed, "hash": fields[2]}
	}
	return out, nil
}

func goSumEntry(sums map[string]map[string]string, modulePath, version string, isZip bool) (string, string, bool) {
	key := modulePath + " " + version
	if !isZip {
		key += "/go.mod"
	}
	entry, ok := sums[key]
	if !ok {
		return "", "", false
	}
	return entry["line"], entry["hash"], true
}

// reresolveFromSupply runs the resolver with GOPROXY pointed at the supply and
// a fresh isolated module cache, then reports which module versions the
// resolver actually selected.
func reresolveFromSupply(root, supply, packagePath, goPath, stateDir string,
) ([]string, []string, int, string, error) {
	cmd := exec.Command(goPath, "list", "-deps", "-json", "-e", "-mod=readonly", packagePath)
	cmd.Dir = root
	env, err := supplyEnvironment(os.Environ(), stateDir, supply)
	if err != nil {
		return nil, nil, 0, "", err
	}
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	exitCode := 0
	commandError := ""
	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
		commandError = strings.TrimSpace(stderr.String())
		if commandError == "" {
			commandError = runErr.Error()
		}
	}

	modules := map[string]struct{}{}
	var unresolved []string
	decoder := json.NewDecoder(&stdout)
	for {
		var pkg goResolvedPackage
		if err := decoder.Decode(&pkg); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, nil, exitCode, commandError, fmt.Errorf("decode re-resolution output: %w", err)
		}
		if pkg.Standard {
			continue
		}
		if pkg.Error != nil {
			unresolved = append(unresolved, pkg.ImportPath)
			continue
		}
		if pkg.Module == nil || pkg.Module.Path == "" || pkg.Module.Dir == root {
			continue
		}
		modules[pkg.Module.Path+"@"+pkg.Module.Version] = struct{}{}
	}
	resolved := make([]string, 0, len(modules))
	for module := range modules {
		resolved = append(resolved, module)
	}
	sort.Strings(resolved)
	sort.Strings(unresolved)
	return resolved, unresolved, exitCode, commandError, nil
}

type goResolvedPackage struct {
	ImportPath string `json:"ImportPath"`
	Standard   bool   `json:"Standard"`
	Module     *struct {
		Path    string `json:"Path"`
		Version string `json:"Version"`
		Dir     string `json:"Dir"`
	} `json:"Module"`
	Error *struct {
		Err string `json:"Err"`
	} `json:"Error"`
}
