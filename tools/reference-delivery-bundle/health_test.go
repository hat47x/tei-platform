package deliverybundle

import (
	"path/filepath"
	"testing"
	"time"
)

func activeState(t *testing.T) string {
	t.Helper()
	statePath := filepath.Join(t.TempDir(), "state.json")
	if _, err := Activate(catalogWith(t, "audit"), statePath); err != nil {
		t.Fatal(err)
	}
	return statePath
}

// healthTestBudgetは、成功経路の試験が子プロセスの起動待ちを含めて完了できる
// 負荷耐性のある予算である。Healthの期限はbashの起動（sh -lc相当）から
// コマンド完了までを一体化して測るため、リポジトリ全体のgo testと並列に
// 走ると起動待ちが数百msから数秒まで伸びる（#2380）。期限超過の報告そのものは
// TestHealthTimeoutIsReportedが意図的に短い期限で別に検証するので、成功経路の
// 予算を延ばしても超過の検出確認は弱まらない。30秒はHealthが未指定時に使う
// 既定値と同じである。
const healthTestBudget = 30 * time.Second

func TestHealthUsesActiveDeployment(t *testing.T) {
	statePath := activeState(t)
	receipt, err := Health(statePath, `test -n "$TEI_CATALOG" && test -n "$TEI_ACTIVE_DIGEST" && test -n "$TEI_STATE"`, healthTestBudget)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Healthy || receipt.ExitCode != 0 || receipt.OutputDigest == "" {
		t.Fatalf("unexpected health receipt: %+v", receipt)
	}
}

func TestHealthFailureIsReported(t *testing.T) {
	statePath := activeState(t)
	receipt, err := Health(statePath, "exit 3", healthTestBudget)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Healthy || receipt.ExitCode != 3 {
		t.Fatalf("unexpected health receipt: %+v", receipt)
	}
}

func TestHealthTimeoutIsReported(t *testing.T) {
	statePath := activeState(t)
	receipt, err := Health(statePath, "sleep 2", 100*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Healthy || receipt.Error != "timeout" {
		t.Fatalf("unexpected health receipt: %+v", receipt)
	}
}

func TestHealthRequiresValidState(t *testing.T) {
	if _, err := Health(filepath.Join(t.TempDir(), "missing.json"), "exit 0", time.Second); err == nil {
		t.Fatal("missing state was accepted")
	}
}
