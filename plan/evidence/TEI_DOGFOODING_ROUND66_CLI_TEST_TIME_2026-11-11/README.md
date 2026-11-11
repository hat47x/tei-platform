# cmd/tei のテストが 10 分上限に当たる問題を、ビルド 1 回化で解消（2026-11-11、第66ラウンド）

- 対象: 第65ラウンドで記録した「`cmd/tei` が `go test` の 600 秒上限に近い」という課題。
- 前段: [第65ラウンド](../TEI_DOGFOODING_ROUND65_INTERACTION_FORM_2026-11-11/README.md)。
- 実行日: 2026-11-11（Asia/Tokyo）。
- 状態: **`cmd/tei` の 1 テストがビルドを 11 回繰り返していたのを 1 回にした。同テストは 62.6s → 5.3s。**

## 1. 目的

第65ラウンドで「`cmd/tei` は単独で 564.8s、同時実行で 600.1s の上限に達して FAIL」と記録した。**機能の失敗ではなく実行時間の問題**なので、原因を測って直す。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `e91b46c0e`（`origin/main`） |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |
| 測定方法 | `go test -json` の `Elapsed` を集計 |

## 3. 測定: 何が時間を使っていたか

`cmd/tei` は 353 テスト・合計 676.2s（**サブテストを含むと 582 テスト・535.8s**）。上位は:

```text
202.72s  TestSelfBuildFullChainOnTheRealSubject
 61.48s  TestMissingRequiredFlagIsAlwaysInvalidArgument
 47.50s  TestSelfBuildEstablishesL2BFromARealGoBuild
 13.05s  TestUnusableSubjectValueStaysInvalidSubject
 12.53s  TestDiscoveryArgumentValuesMatchImplementation
```

**`TestMissingRequiredFlagIsAlwaysInvalidArgument` の 61.5s が異常**だった。サブテストは 12 個あり、合計 74.5s（親 37.3s＋サブの和）。つまり**親が 37s をサブテスト以外で使っている**。

### 3.1 原因

```go
// cmd/tei/cli_failure_classification_test.go
func runCLIForClassification(t *testing.T, args ...string) (int, bool, string) {
	binary := filepath.Join(t.TempDir(), "tei")
	if out, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
```

**各ケースが `go build` を 1 回ずつ実行していた。** 12 ケース＋他の 2 テストで、**同じバイナリを十数回ビルド**していた。テストの目的は「実プロセスの終了コードと JSON を見る」ことで、バイナリは 1 回作れば足りる。

## 4. 修正

`sync.Once` でテストバイナリごとに 1 回だけビルドし、パスを共有する。

```go
var (
	cliBinaryOnce sync.Once
	cliBinaryPath string
	cliBinaryErr  error
)

func sharedCLIBinary(t *testing.T) string {
	cliBinaryOnce.Do(func() { ... exec.Command("go", "build", "-o", path, ".") ... })
	...
}
```

`runCLIForClassification` はこれを呼ぶだけにした。

### 4.1 実測（同一テスト）

| | 時間 |
| --- | --- |
| 修正前 | **62.6s**（wall 93.7s） |
| 修正後 | **5.3s** |

### 4.2 実測（パッケージ全体）

| 実行 | 修正前 | 修正後 |
| --- | --- | --- |
| `./cmd/tei`（通常） | 564.8s / 600.1s FAIL（同時実行） | **232.9s ok** |
| `./cmd/tei -short` | 434.8s | **306.3s ok** |

**注意して書く**: `-short` の 306.3s は**通常実行の 232.9s より遅い**。これは -short を先に実行し（キャッシュ cold）、通常実行を後に実行した（キャッシュ warm）ためで、**順序の効果であって -short が遅いのではない**。同じ条件で比べていないので、この2つの大小関係は主張しない。

### 4.3 残った上位

修正後の上位は自己ビルド系（17.6s・12.2s）で、これらは `-short` で意図的にスキップされる設計になっている。**上限に対する余裕は 600s → 233s で 2.5 倍以上になった。**

## 5. 試験

**この変更はテストコードのみ**で、製品コードは触っていない。したがって:

- 振る舞いの試験は増やしていない（テストの実行時間を測る試験は、CI の時間に依存するため不安定になる）
- 代わりに **`cmd/tei` 全体を実行して、全 603 テストが ok であること**を確認した
- `go vet ./cmd/tei` に指摘なし

### 5.1 変異試験をしない理由

この変更は「テストの実行方法」であり、製品の振る舞いに影響しない。**変異で確かめられる対象がない。** 代わりに、**ビルドが 1 回だけ行われたこと**を測った:

- 修正前の当該テスト 62.6s → 修正後 5.3s（ビルド 1 回分のコストに収束）
- テストの合計時間 676.2s → 190.3s

（`-count=1` でキャッシュを無効化した状態で測った。）

## 6. この変更が直さないこと

- **自己ビルド系のテスト（202.7s の `TestSelfBuildFullChainOnTheRealSubject` など）は残る。** これらは実際にツリー全体をビルドするので、時間がかかるのは設計意図どおり。`-short` でスキップされる。
- **CI が `cmd/tei` をどう実行しているかは変えていない。** CI の `tei-studio-authoring-ci.yml` は一部のテストだけを `-run` で指定しており、全体実行はしていない（`timeout-minutes: 45`）。
- 600 秒上限の設定自体は変えていない。

## 7. 読み取れないこと（NOT RUN）

- 他のパッケージに同じ「毎回ビルド」パターンがあるか（`cmd/tei` 内には他にも 10 箇所の `go build` があるが、**それらが 1 テスト 1 ビルドかは未測定**）。
- CI 上の実行時間（ローカルのみ測定）。
- `-short` と通常実行の同条件比較。
- 実DB・実Provider・実Host、HTTP・Desktop。

## 8. 位置付け

**「失敗ではなく時間の問題」を、測ってから直した回**である。第65ラウンドで FAIL を記録したとき、私は「機能の失敗ではない」と書き留めた。その判断が正しかったことを、原因の特定と修正で裏づけた。

**同じ `go build` が他に 10 箇所あることを記録した。** 今回は 1 箇所だけ直した。残りは踏んだときに測る。
