package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const auditNature = "gate-b-readiness-artifact-audit"

var laneFiles = map[string]string{
	"deterministic-repository":           "deterministic-repository.json",
	"generated-artifact-reproducibility": "generated-artifact-reproducibility.json",
	"real-postgres-ephemeral-fixture":    "real-postgres-ephemeral-fixture.json",
}

var forbiddenEvidenceKeys = map[string]struct{}{
	"dsn": {}, "credential": {}, "credentials": {}, "password": {},
	"parametervalues": {}, "databasename": {}, "secret": {}, "secretpath": {},
	"originalsecretpath": {}, "handoffpath": {}, "dsnfile": {}, "dsnfilepath": {},
}

type auditLane struct {
	Name        string `json:"name"`
	Artifact    string `json:"artifactFile"`
	Fingerprint string `json:"fingerprint"`
	Passed      bool   `json:"passed"`
}

type auditEvidence struct {
	EvidenceNature      string      `json:"evidenceNature"`
	RepositoryRevision  string      `json:"repositoryRevision,omitempty"`
	FinalFingerprint    string      `json:"finalFingerprint,omitempty"`
	ArtifactAuditPassed bool        `json:"artifactAuditPassed"`
	Lanes               []auditLane `json:"lanes"`
}

type laneSummary struct {
	Name                        string
	ExpectedNature              string
	ObservedNature              string
	RepositoryRevision          string
	ReproducibleRevision        bool
	ArtifactFile                string
	EvidenceArtifactFingerprint string
	Passed                      bool
}

type expectedCheck struct {
	Name    string
	Command []string
}

func main() {
	finalPath := flag.String("final", "", "#387 final convergence JSON file")
	artifactDir := flag.String("artifact-dir", "", "same-run child Evidence artifact directory")
	flag.Parse()

	result, err := audit(*finalPath, *artifactDir)
	encoded, encodeErr := json.MarshalIndent(result, "", "  ")
	if encodeErr != nil {
		fmt.Fprintln(os.Stderr, encodeErr)
		os.Exit(1)
	}
	fmt.Println(string(encoded))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func audit(finalPath, artifactDir string) (auditEvidence, error) {
	result := auditEvidence{EvidenceNature: auditNature, Lanes: []auditLane{}}
	if strings.TrimSpace(finalPath) == "" || strings.TrimSpace(artifactDir) == "" {
		return result, fmt.Errorf("-finalと-artifact-dirを両方指定してください")
	}
	if err := requirePlainDirectory(artifactDir); err != nil {
		return result, fmt.Errorf("child artifact directoryを監査できません: %w", err)
	}

	finalBytes, err := readPlainRegularFile(finalPath)
	if err != nil {
		return result, fmt.Errorf("final convergence artifactを監査できません: %w", err)
	}
	result.FinalFingerprint = fingerprint(finalBytes)
	finalObject, err := decodeObject(finalBytes)
	if err != nil {
		return result, fmt.Errorf("final convergence JSONを解析できません: %w", err)
	}
	if err := rejectSensitiveKeys(finalObject, "final"); err != nil {
		return result, err
	}

	revision, broad, lanes, err := verifyFinalSummary(finalObject)
	if err != nil {
		return result, err
	}
	result.RepositoryRevision = revision

	for _, lane := range lanes {
		artifactPath := filepath.Join(artifactDir, lane.ArtifactFile)
		content, err := readPlainRegularFile(artifactPath)
		if err != nil {
			return result, fmt.Errorf("%s child artifactを監査できません: %w", lane.Name, err)
		}
		gotFingerprint := fingerprint(content)
		if gotFingerprint != lane.EvidenceArtifactFingerprint {
			return result, fmt.Errorf("%s child artifact SHA-256=%s want=%s", lane.Name, gotFingerprint, lane.EvidenceArtifactFingerprint)
		}
		child, err := decodeObject(content)
		if err != nil {
			return result, fmt.Errorf("%s child JSONを解析できません: %w", lane.Name, err)
		}
		if err := rejectSensitiveKeys(child, lane.Name); err != nil {
			return result, err
		}
		if err := verifyChild(lane.Name, child, revision, broad); err != nil {
			return result, err
		}
		result.Lanes = append(result.Lanes, auditLane{
			Name: lane.Name, Artifact: lane.ArtifactFile, Fingerprint: gotFingerprint, Passed: true,
		})
	}

	result.ArtifactAuditPassed = true
	return result, nil
}

func verifyFinalSummary(obj map[string]any) (string, bool, []laneSummary, error) {
	if err := requireStringEquals(obj, "evidenceNature", "gate-b-three-lane-final-convergence"); err != nil {
		return "", false, nil, err
	}
	revision, err := requireNonEmptyString(obj, "repositoryRevision")
	if err != nil {
		return "", false, nil, err
	}
	for _, key := range []string{"workingTreeClean", "reproducibleRevision", "allLanesSameRevision", "childArtifactsRetained"} {
		if err := requireBoolEquals(obj, key, true); err != nil {
			return "", false, nil, err
		}
	}
	if err := verifyPostRun(obj, revision); err != nil {
		return "", false, nil, err
	}
	goEnv, err := requireObject(obj, "goEnvironment")
	if err != nil {
		return "", false, nil, err
	}
	if err := requireBoolEquals(goEnv, "externalWorkspaceActive", false); err != nil {
		return "", false, nil, err
	}
	if err := requireBoolEquals(goEnv, "goFlagsEmpty", true); err != nil {
		return "", false, nil, err
	}
	if err := requireBoolEquals(goEnv, "goEnvDisabled", true); err != nil {
		return "", false, nil, err
	}
	broad, err := requireBool(obj, "broadRequested")
	if err != nil {
		return "", false, nil, err
	}

	items, err := requireArray(obj, "lanes")
	if err != nil {
		return "", false, nil, err
	}
	if len(items) != 3 {
		return "", false, nil, fmt.Errorf("final lanes=%d want=3", len(items))
	}
	byName := map[string]laneSummary{}
	for i, raw := range items {
		laneObj, ok := raw.(map[string]any)
		if !ok {
			return "", false, nil, fmt.Errorf("final lanes[%d]がobjectではありません", i)
		}
		name, err := requireNonEmptyString(laneObj, "name")
		if err != nil {
			return "", false, nil, err
		}
		expectedFile, ok := laneFiles[name]
		if !ok {
			return "", false, nil, fmt.Errorf("未知のfinal lane=%qです", name)
		}
		if _, dup := byName[name]; dup {
			return "", false, nil, fmt.Errorf("final lane=%qが重複しています", name)
		}
		expectedNature, err := requireNonEmptyString(laneObj, "expectedNature")
		if err != nil {
			return "", false, nil, err
		}
		observedNature, err := requireNonEmptyString(laneObj, "observedNature")
		if err != nil {
			return "", false, nil, err
		}
		if expectedNature != name || observedNature != name {
			return "", false, nil, fmt.Errorf("%s Evidence natureが一致しません: expected=%q observed=%q", name, expectedNature, observedNature)
		}
		laneRevision, err := requireNonEmptyString(laneObj, "repositoryRevision")
		if err != nil {
			return "", false, nil, err
		}
		if laneRevision != revision {
			return "", false, nil, fmt.Errorf("%s repositoryRevision=%q want=%q", name, laneRevision, revision)
		}
		if err := requireBoolEquals(laneObj, "reproducibleRevision", true); err != nil {
			return "", false, nil, err
		}
		if err := requireBoolEquals(laneObj, "passed", true); err != nil {
			return "", false, nil, err
		}
		artifactFile, err := requireNonEmptyString(laneObj, "artifactFile")
		if err != nil {
			return "", false, nil, err
		}
		if artifactFile != expectedFile {
			return "", false, nil, fmt.Errorf("%s artifactFile=%q want=%q", name, artifactFile, expectedFile)
		}
		fp, err := requireNonEmptyString(laneObj, "evidenceArtifactFingerprint")
		if err != nil {
			return "", false, nil, err
		}
		if !strings.HasPrefix(fp, "sha256:") || len(fp) != len("sha256:")+64 {
			return "", false, nil, fmt.Errorf("%s fingerprint形式が不正です", name)
		}
		byName[name] = laneSummary{Name: name, ExpectedNature: expectedNature, ObservedNature: observedNature, RepositoryRevision: laneRevision, ReproducibleRevision: true, ArtifactFile: artifactFile, EvidenceArtifactFingerprint: fp, Passed: true}
	}
	order := []string{"deterministic-repository", "generated-artifact-reproducibility", "real-postgres-ephemeral-fixture"}
	out := make([]laneSummary, 0, 3)
	for _, name := range order {
		lane, ok := byName[name]
		if !ok {
			return "", false, nil, fmt.Errorf("final lane=%qがありません", name)
		}
		out = append(out, lane)
	}
	return revision, broad, out, nil
}

func verifyChild(name string, obj map[string]any, revision string, broad bool) error {
	if err := requireStringEquals(obj, "evidenceNature", name); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	childRevision, err := requireNonEmptyString(obj, "repositoryRevision")
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if childRevision != revision {
		return fmt.Errorf("%s child repositoryRevision=%q want=%q", name, childRevision, revision)
	}
	if err := requireBoolEquals(obj, "workingTreeClean", true); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if err := requireBoolEquals(obj, "reproducibleRevision", true); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if err := verifyPostRun(obj, revision); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	switch name {
	case "deterministic-repository":
		return verifyRepositoryChild(obj, broad)
	case "generated-artifact-reproducibility":
		return verifyGeneratedChild(obj)
	case "real-postgres-ephemeral-fixture":
		return verifyPostgresChild(obj)
	default:
		return fmt.Errorf("未知のchild Evidence=%qです", name)
	}
}

func verifyRepositoryChild(obj map[string]any, broad bool) error {
	if err := requireStringEquals(obj, "workspace", "reference/workspace"); err != nil {
		return err
	}
	validation, err := requireObject(obj, "workspaceValidation")
	if err != nil {
		return err
	}
	if err := requireBoolEquals(validation, "passed", true); err != nil {
		return err
	}
	if errors, ok := validation["errors"]; ok {
		arr, ok := errors.([]any)
		if !ok || len(arr) != 0 {
			return fmt.Errorf("workspaceValidation.errorsが空ではありません")
		}
	}
	childBroad, err := requireBool(obj, "broadRequested")
	if err != nil {
		return err
	}
	if childBroad != broad {
		return fmt.Errorf("#310 broadRequested=%v want final=%v", childBroad, broad)
	}
	required := []expectedCheck{
		{
			Name:    "final-ui-less-scenario",
			Command: []string{"go", "test", "./internal/driver", "-run", "^TestOrderEntryGateBScenarioConnectsMajorBoundariesWithoutUI$", "-count=1"},
		},
		{
			Name: "external-evidence-intake",
			Command: []string{
				"go", "test",
				"./internal/database/externalqueryevidence",
				"./tools/reference-external-query-evidence",
				"-count=1",
			},
		},
		{
			Name:    "repository-evidence-source-boundary",
			Command: []string{"go", "test", "./tools/internal/repositoryevidence", "-count=1"},
		},
	}
	if broad {
		required = append(required, expectedCheck{
			Name:    "repository-wide-go-test",
			Command: []string{"go", "test", "./..."},
		})
	}
	return requireExactPassedChecks(obj, required)
}

func verifyGeneratedChild(obj map[string]any) error {
	return requireExactPassedChecks(obj, []expectedCheck{
		{Name: "world-plugin-lock-golden", Command: []string{"go", "run", "./tools/reference-golden"}},
		{Name: "durable-run-golden", Command: []string{"go", "run", "./tools/reference-durable-run-golden"}},
	})
}

func verifyPostgresChild(obj map[string]any) error {
	for _, identity := range []struct {
		key  string
		want string
	}{
		{key: "pattern", want: "order_entry.order_lines"},
		{key: "relation", want: "sales.order.lines"},
		{key: "environment", want: "development"},
		{key: "databaseInstance", want: "sales_core.dev"},
		{key: "executorCapability", want: "database.query.postgres"},
		{key: "contractQueryTarget", want: "sales_core/sales_core.order_line/find_by_order"},
	} {
		if err := requireStringEquals(obj, identity.key, identity.want); err != nil {
			return err
		}
	}
	for _, key := range []string{"preexistingFixtureProtected", "fixtureOwnedByTool", "executionErrorObserved", "semanticMappingApplied"} {
		if err := requireBoolEquals(obj, key, true); err != nil {
			return err
		}
	}
	if err := requireBoolEquals(obj, "parameterValuesPersisted", false); err != nil {
		return err
	}
	if err := requireBoolEquals(obj, "semanticOrderLineParentAttributeAdded", false); err != nil {
		return err
	}
	if err := requireStringEquals(obj, "fixtureObject", "public.order_lines"); err != nil {
		return err
	}
	if err := requireStringEquals(obj, "relationStorageColumn", "order_id"); err != nil {
		return err
	}
	if err := requireIntEquals(obj, "rows", 2); err != nil {
		return err
	}
	if err := requireIntEquals(obj, "zeroRows", 0); err != nil {
		return err
	}

	before, err := requireObject(obj, "fixtureDatabaseSafety")
	if err != nil {
		return err
	}
	for _, key := range []string{"guardPassed", "databaseIdentityVerified", "nonDefaultDatabaseVerified", "fixtureObjectAbsentBeforeCreate"} {
		if err := requireBoolEquals(before, key, true); err != nil {
			return err
		}
	}
	if err := requireBoolEquals(before, "databaseIdentityPersisted", false); err != nil {
		return err
	}
	if err := requireIntEquals(before, "userRelationsBeforeFixture", 0); err != nil {
		return err
	}
	if err := requireIntEquals(before, "eventTriggersBeforeFixture", 0); err != nil {
		return err
	}

	after, err := requireObject(obj, "fixtureCleanupSafety")
	if err != nil {
		return err
	}
	if err := requireBoolEquals(after, "verified", true); err != nil {
		return err
	}
	if err := requireIntEquals(after, "userRelationsAfterCleanup", 0); err != nil {
		return err
	}
	if err := requireIntEquals(after, "eventTriggersAfterCleanup", 0); err != nil {
		return err
	}
	if err := requireBoolEquals(after, "fixtureObjectAbsentAfterCleanup", true); err != nil {
		return err
	}

	if err := requireStringArrayEquals(obj, "parameterNames", []string{"order_id"}); err != nil {
		return err
	}
	if err := requireStringArrayEquals(obj, "semanticAttributes", []string{
		"sales.order_line.item_code",
		"sales.order_line.line_number",
		"sales.order_line.order_line_id",
		"sales.order_line.quantity",
		"sales.order_line.unit_price",
	}); err != nil {
		return err
	}
	return nil
}

func requireExactPassedChecks(obj map[string]any, required []expectedCheck) error {
	items, err := requireArray(obj, "checks")
	if err != nil {
		return err
	}
	if len(items) != len(required) {
		return fmt.Errorf("checks=%d want=%d", len(items), len(required))
	}
	expected := make(map[string]expectedCheck, len(required))
	for _, check := range required {
		if _, duplicate := expected[check.Name]; duplicate {
			return fmt.Errorf("required check=%qが重複しています", check.Name)
		}
		expected[check.Name] = check
	}
	seen := make(map[string]struct{}, len(required))
	for i, raw := range items {
		check, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("checks[%d]がobjectではありません", i)
		}
		name, err := requireNonEmptyString(check, "name")
		if err != nil {
			return err
		}
		want, ok := expected[name]
		if !ok {
			return fmt.Errorf("unexpected check=%qがあります", name)
		}
		if _, duplicate := seen[name]; duplicate {
			return fmt.Errorf("check=%qが重複しています", name)
		}
		if err := requireStringArrayEquals(check, "command", want.Command); err != nil {
			return fmt.Errorf("check=%q command: %w", name, err)
		}
		okPassed, err := requireBool(check, "passed")
		if err != nil {
			return err
		}
		if !okPassed {
			return fmt.Errorf("required check=%qが成功していません", name)
		}
		seen[name] = struct{}{}
	}
	for _, check := range required {
		if _, exists := seen[check.Name]; !exists {
			return fmt.Errorf("required check=%qがありません", check.Name)
		}
	}
	return nil
}

func verifyPostRun(obj map[string]any, revision string) error {
	post, err := requireObject(obj, "postRunRepository")
	if err != nil {
		return err
	}
	if err := requireBoolEquals(post, "checked", true); err != nil {
		return err
	}
	if err := requireBoolEquals(post, "revisionStable", true); err != nil {
		return err
	}
	if err := requireBoolEquals(post, "workingTreeClean", true); err != nil {
		return err
	}
	postRevision, err := requireNonEmptyString(post, "repositoryRevision")
	if err != nil {
		return err
	}
	if postRevision != revision {
		return fmt.Errorf("postRunRepository.repositoryRevision=%q want=%q", postRevision, revision)
	}
	return nil
}

func rejectSensitiveKeys(value any, path string) error {
	switch v := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			lower := strings.ToLower(key)
			if _, forbidden := forbiddenEvidenceKeys[lower]; forbidden {
				return fmt.Errorf("%s.%sは通常Evidenceへ保存してはいけないsecret/physical identity fieldです", path, key)
			}
			if err := rejectSensitiveKeys(v[key], path+"."+key); err != nil {
				return err
			}
		}
	case []any:
		for i, item := range v {
			if err := rejectSensitiveKeys(item, fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	}
	return nil
}

func decodeObject(content []byte) (map[string]any, error) {
	decoder := json.NewDecoder(strings.NewReader(string(content)))
	decoder.UseNumber()
	var obj map[string]any
	if err := decoder.Decode(&obj); err != nil {
		return nil, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("JSON末尾に追加dataがあります")
	}
	if obj == nil {
		return nil, fmt.Errorf("JSON rootがobjectではありません")
	}
	return obj, nil
}

func readPlainRegularFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("regular non-symlink fileではありません")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	opened, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		_ = file.Close()
		return nil, fmt.Errorf("open中にfile identityが変化しました")
	}
	content, err := io.ReadAll(file)
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	after, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if after.Mode()&os.ModeSymlink != 0 || !after.Mode().IsRegular() || !os.SameFile(opened, after) {
		return nil, fmt.Errorf("read中にfile identityが変化しました")
	}
	return content, nil
}

func requirePlainDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("regular directoryではありません")
	}
	return nil
}

func fingerprint(content []byte) string {
	sum := sha256.Sum256(content)
	return fmt.Sprintf("sha256:%x", sum[:])
}

func requireObject(obj map[string]any, key string) (map[string]any, error) {
	value, ok := obj[key]
	if !ok {
		return nil, fmt.Errorf("%sがありません", key)
	}
	out, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%sがobjectではありません", key)
	}
	return out, nil
}

func requireArray(obj map[string]any, key string) ([]any, error) {
	value, ok := obj[key]
	if !ok {
		return nil, fmt.Errorf("%sがありません", key)
	}
	out, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("%sがarrayではありません", key)
	}
	return out, nil
}

func requireBool(obj map[string]any, key string) (bool, error) {
	value, ok := obj[key]
	if !ok {
		return false, fmt.Errorf("%sがありません", key)
	}
	out, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("%sがboolではありません", key)
	}
	return out, nil
}

func requireBoolEquals(obj map[string]any, key string, want bool) error {
	got, err := requireBool(obj, key)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("%s=%v want=%v", key, got, want)
	}
	return nil
}

func requireNonEmptyString(obj map[string]any, key string) (string, error) {
	value, ok := obj[key]
	if !ok {
		return "", fmt.Errorf("%sがありません", key)
	}
	out, ok := value.(string)
	if !ok || strings.TrimSpace(out) == "" {
		return "", fmt.Errorf("%sがnon-empty stringではありません", key)
	}
	return out, nil
}

func requireStringEquals(obj map[string]any, key, want string) error {
	got, err := requireNonEmptyString(obj, key)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("%s=%q want=%q", key, got, want)
	}
	return nil
}

func requireIntEquals(obj map[string]any, key string, want int64) error {
	value, ok := obj[key]
	if !ok {
		return fmt.Errorf("%sがありません", key)
	}
	n, ok := value.(json.Number)
	if !ok {
		return fmt.Errorf("%sがnumberではありません", key)
	}
	got, err := n.Int64()
	if err != nil {
		return fmt.Errorf("%sがintegerではありません", key)
	}
	if got != want {
		return fmt.Errorf("%s=%d want=%d", key, got, want)
	}
	return nil
}

func requireStringArray(obj map[string]any, key string) ([]string, error) {
	items, err := requireArray(obj, key)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(items))
	for i, item := range items {
		s, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("%s[%d]がstringではありません", key, i)
		}
		out = append(out, s)
	}
	return out, nil
}

func requireStringArrayEquals(obj map[string]any, key string, want []string) error {
	got, err := requireStringArray(obj, key)
	if err != nil {
		return err
	}
	if len(got) != len(want) {
		return fmt.Errorf("%s=%v want=%v", key, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			return fmt.Errorf("%s=%v want=%v", key, got, want)
		}
	}
	return nil
}
