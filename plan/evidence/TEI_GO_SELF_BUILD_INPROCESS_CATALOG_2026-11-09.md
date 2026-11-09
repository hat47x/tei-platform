# entryがin-processで計画を導く経路（§4.22 段2 / SC0-25）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.22 段2を実装した記録です。**entry が宣言を読み、呼び手が名指しした catalog を引き、自分で計画を導く**経路ができました。

## 何を作ったか

| flag | 意味 |
| --- | --- |
| `-build-catalog` | 呼び手が名指しする catalog のパス |
| `-build-catalog-digest` | その catalog の digest（**必須の対**） |
| `-build-source-root` | 宣言の source 参照を解決するディレクトリ（既定は catalog のディレクトリ） |

`-build-catalog` と `-build-catalog-digest` の**両方**が要ります。片方だけなら起動しません。**「どの catalog で導いたか」を後から答えられるようにする**ためで、これは `planReceipt.resolverDigest` が resolver child について果たしているのと同じ役割です。

経路は既存の receipt 形状へ変換されるので、**下流（`planCrossCheck`・`planArtifactBinding`・`planRecompute`）は経路を意識しません**。

## 何が変わって、何が変わらないか

| 変わる | 変わらない |
| --- | --- |
| 計画の導出に別の実行ファイルが要らない | **Authority は作らない**。catalog は呼び手が名指しする |
| entry が `buildintent.Resolve` を自分で呼ぶ | entry は builder を**発見しない** |
| 呼び手の catalog で導いたことが記録に残る | **その catalog が正しいとは主張しない** |
| — | `qualified` は `false` のまま |

## 実測（新規5件）

| 検証 | 結果 |
| --- | --- |
| catalog を名指しして導出 | **成立**。`state=resolved`、catalog の digest を**申告値と実測値の両方**記録、entry の `intentFingerprint` と `planFingerprint` が入り、`planRecompute=matched` |
| `-build-catalog` のみ（digest 無し） | exit 2、`build-catalog-digest` を名指し、**artifact を作らない** |
| `-build-catalog-digest` のみ | exit 2、`build-catalog` を名指し |
| digest が実測と違う | exit 1、「does not match」、**artifact を作らない** |
| `-expected-plan-fingerprint` を満たさない | exit 1 |

1行目で **`l2bClaimed=false`・`runtimeAuthorityClaimed=false`・`qualified=false`** を明示的に確認しています。経路が増えても主張は増えません。

## 実装中に捕まえた2つの不具合

### 1. source root を catalog のディレクトリに固定していた

最初は `filepath.Dir(catalogPath)` を source root にしていました。fixture では宣言の source が `testdata/src/plugin.tei` にあり、catalog は `testdata/catalog.json` なので、**`source path component "plugin.tei" does not exist`** で落ちました。

**catalog とその source が同じディレクトリにあるとは限らない**ので、`-build-source-root` を足し、既定を catalog のディレクトリにしました。テストが捕まえました。

### 2. 対の拒否が JSON で返っていなかった

`-build-catalog` だけを渡したとき、`INVALID_ARGUMENT` を Go の error として返していたため、**他の拒否と違って report が出ませんでした**。呼び手が理由を2箇所から読むことになるので、同じ envelope で返すよう統一しました。

## 何が言えるようになったか

**「この entry が宣言を解釈してその計画を導いた」と言えます。** 段2の前は、計画は resolver child が作っていました。

## 何が言えないか（変わらない）

| 言えないこと | 理由 |
| --- | --- |
| 宣言の**意味の受理** | 導出は解釈であり、受理ではない |
| Runtime Authority | `qualified=false` |
| **catalog が正しいこと** | K-3 は「呼び手が名指しした catalog を使った」ところまで |
| 供給の Trust・Seed Trust | 別の層 |
| **既定経路の変更** | 段3。`-intent-resolver` が既定のまま |

## 検証

- `go build ./...`：PASS
- `gofmt -l cmd/tei`：差分なし
- `go vet ./cmd/tei`：PASS
- `go test ./cmd/tei -run TestInProcess`：5件PASS
- `go test ./cmd/tei -count=1 -timeout 3600s`：実行中（結果はPR本文に記録）
- `go test ./tools/repository-plan-check -count=1`：PASS

**NOT RUN**：Windows、別Go版、実プロセスでの段2観測（単体試験のみ）。

## 限界

- **既定は resolver child のまま**（段3は未決）。段2は「経路がある」ことまでです。
- `-build-source-root` を省略すると catalog のディレクトリを使います。**catalog と source が離れている場合は明示が要ります。**
- catalog の中身を誰が作るかは未決（§4.22末尾）。
- G4は`UNQUALIFIED`のまま。
