# intentBindingをentryの導出値基準へ移し、実プロセスで四つの観測を同時に得る

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0で、`intentBinding`が**childの申告に依存していた**点を改善した記録です。I-1a（`internal/buildintent`）でentryが自分で宣言を読めるようになり、さらに[#3649](https://github.com/hat47x/tei-platform/pull/3649)でbuilder childが宣言を読むようになったため、**実プロセスで四つの観測が同時に成立**しました。

**L2-bは依然として成立しません。**

## 何が問題だったか

`intentBinding`のbyte検査は、**childが「読んだバイトのdigest」を申告する**ことを前提にしています。childが申告しなければ`intentByteCheckPerformed=false`のままです。つまり**照合の成立が相手の協力に依存**していました。

## 足したもの

`intentBinding`に4欄を追加しました。

| 欄 | 意味 |
| --- | --- |
| `entryIntentFingerprint` | entryが保持バイトから**自分で導いた**宣言の照合値（I-1a） |
| `intentFingerprintCheckPerformed` | 両側にfingerprintが揃ったか |
| `intentFingerprintMatched` | childの申告値とentryの導出値が一致したか |
| `fingerprintCheckNotPerformedReason` | 実施しなかった理由 |

### 二つの検査は別の問いに答える

| 検査 | 問い | 何に依存するか |
| --- | --- | --- |
| byte検査（従来） | childは**entryが保持したのと同じバイト**を読んだか | childがバイトdigestを申告すること |
| fingerprint検査（新） | childは**entryが読んだのと同じ宣言**を読んだか | childがfingerprintを申告すること＋entry自身の導出 |

**fingerprint検査はbyte検査の結果を引き継ぎません。** childがfingerprintだけ申告してバイトを申告しない場合でも比較します。逆に、バイトが一致していてもfingerprintが違えば**別の宣言**を読んだことになります（正規化後に同じ宣言になる場合を除く）。不一致は`failureCause`に載り、`ok=false`になります。

## 実測（実プロセス、WSL2 / go1.27.3）

宣言は`plugin tei.plugin.bf_probe v1.0.0`（`outputKind: plugin-package`）。解決childで計画を得て、builder childには`-plan`・`-plugin`・`-intent`を渡しました。

| 欄 | 値 |
| --- | --- |
| `ok` / `qualified` | `true` / **`false`** |
| `intentBytesDigest` | `sha256:fdbb6b2a…` |
| `childIntentBytesDigest` | `sha256:fdbb6b2a…`（**一致**） |
| `intentByteCheckPerformed` / `Matched` | `true` / **`true`** |
| `entryIntentFingerprint` | `e2768f56…` |
| `childIntentFingerprint` | `e2768f56…`（**一致**） |
| `intentFingerprintCheckPerformed` / `Matched` | `true` / **`true`** |
| `l2bClaimed` | **`false`** |
| `planCrossCheck.state` | `matched` |
| `planArtifactBinding.state` | `bound` |
| `planArtifactBinding.childLockMatched` | `true` |

**四つの観測が同じ1回の試行で同時に成立しました。**

| 観測 | 何を言うか |
| --- | --- |
| byte検査 | childはentryが保持したのと同じ**バイト**を読んだ |
| fingerprint検査 | childはentryが読んだのと同じ**宣言**を読んだ |
| `planCrossCheck` | 解決childとbuilder childが同じ**計画**を名乗った |
| `planArtifactBinding` | その計画と**artifact**が一つの検証済みlockとして束ねられた |

## 既存試験を直した点

`TestSelfBuildOuterBindsTheDeclarationBytesItHeld`のchildは`fp.intent`という**架空のfingerprint**を申告していました。fingerprint検査を入れたことで、この架空値が**正しく不一致として検出**され、試験が落ちました。childの報告値を**entryが導出した実際の値**へ合わせています。

**これは検査が効いていることの証拠**であり、試験を緩めたのではありません。

## 何が増えて、何が増えないか

| 増える | 増えない |
| --- | --- |
| childの協力に依存しない照合の基準 | **L2-b**。`l2bClaimed`はすべての欄で`false` |
| バイト一致でも捕れない食い違いの検出 | 「その宣言がその計画を導く」こと |
| 実施しなかった理由の記録 | 意味の受理・Runtime Authority・供給のTrust |
| 実プロセスでの四観測の同時成立 | (c)の完全な証明（lockの二値はchildの自己申告） |

**`entryIntentFingerprint`はentryの導出値ですが、childの申告値と比べている以上、childが嘘をつかない保証にはなりません。** 減ったのは「childがバイトを申告しないと何も言えない」という依存です。

## 検証

- `gofmt -l cmd/tei`：差分なし
- `go vet ./cmd/tei`：PASS
- `go test ./cmd/tei -run TestFingerprintBinding`：3件PASS
- `go test ./cmd/tei -count=1 -timeout 900s`：PASS（311秒）
- `go test ./tools/repository-plan-check -count=1`：PASS

## 再現手順

`outputKind: plugin-package`の宣言が要る。カタログは`tools/reference-build-authoring-intent/testdata/catalog.json`。

```json
{
  "formatVersion": 1,
  "subject": {"kind": "plugin", "id": "tei.plugin.bf_probe", "version": "1.0.0", "definitionSource": "definition"},
  "sources": [{"kind": "plugin-definition", "id": "definition", "ref": "plugin.tei"}],
  "builder": {"contract": "tei.plugin-build", "compatibleMajor": 1},
  "recipe": {"kind": "go-package", "entrypoint": "plugin.tei"},
  "outputKind": "plugin-package",
  "target": "host/linux/amd64",
  "profile": "release"
}
```

```text
plugin tei.plugin.bf_probe v1.0.0 {
    runtime builtin
    provides deployment-executor runtime.bf_probe {
        action application-start
    }
}
```

1. 解決childで計画を得て`plan.json`へ書く
2. `tei self-build`を`-intent-resolver`付きで起動し、builder childへ`-child-arg`で`-plan`・`-plugin`・`-intent`を渡す
3. `intentBinding`と`planCrossCheck`と`planArtifactBinding`を読む

## 限界

- **L2-bは未成立。** 四観測が同時に成立しても、`l2bClaimed`は`false`である。「宣言が計画を導く」ことと「その計画がartifactを生む」ことは、まだ別の観測を要する。
- 計画の導出は依然`Resolve`（参照実装）にあり、entryは計画を導かない（I-1b未着手）。
- 供給のTrust・署名、Seed Trust、`go.sum`のTrust根拠は未検証。
- G4は`UNQUALIFIED`のまま。
