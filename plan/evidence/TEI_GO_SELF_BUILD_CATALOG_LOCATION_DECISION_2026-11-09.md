# catalogを書く場所は製品の外に置く（§4.32）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.29は、catalogの7欄のうち**5欄に宣言面が無い**ことを示しました。本記録は「ではどこに書くか」を、**既存readerへの影響**という軸で測り、決めたことを記録します。

## 測ったこと：三つの案は、どれも既存readerを壊しうる

| 案 | 何を変えるか | 既存readerへの影響 |
| --- | --- | --- |
| **K-1** Workspaceに`builders:`を足す | `tei.workspace.yaml` | **壊れる。** manifest parserは未知のentryで**fail closed**（`internal/workspace/imports.go`の`unsupported workspace manifest entry`）。`formatVersion`も1と2だけを受理し、未知の版はfail closed（`internal/workspace/loader.go`） |
| **K-2** plugin定義に構文を足す | `.tei`の構文 | **壊れる。** parserは構文ごとのfield集合を明示し、未知のfieldでエラー（`internal/syntax/parser.go`の`unknown lease-policy field`と同じ形） |
| **外部成果物** | **何も足さない** | **壊れない。** 既存のどのファイルにも新しいentryが増えない |

**Workspaceもplugin定義も、未知の入力を拒否する設計です。** これは「分からないことを成功として扱わない」という本リポジトリの原則そのもので、正しい。だからこそ、**既存ファイルへ欄を足す案は古いbinaryとの互換を壊します**。

具体的な行:

| 場所 | 内容 |
| --- | --- |
| `internal/workspace/imports.go:115`・`:173` | `unsupported workspace manifest entry` |
| `internal/workspace/loader.go:207` | `unsupported workspace formatVersion %d (supported: 1, 2)` |
| `internal/syntax/parser.go:658` | `unknown lease-policy field` |

## 決めたこと

**catalogは製品の外で作り、呼び手がdigestで名指しする。** §4.22のK-3が既にこの形を実装しているので、**製品側の変更は要りません**。

| 項目 | 決定 |
| --- | --- |
| catalogの置き場所 | **製品のファイル形式の外**（独立したJSON） |
| 誰が作るか | **ツール**（能力宣言を入力に、catalogを組み立ててdigestを付ける） |
| entryの読み方 | 変わらない（K-3） |
| Workspaceの`formatVersion` | 変えない（1と2のまま） |
| plugin構文 | 変えない |

## なぜこれで足りるのか

catalogは**データ**であり、entryはそれを**名指しされたものとして読む**（§4.22が「entryはbuilderを自分で探さない」と決めた）。**誰がそのデータを作ったかは、entryの観測の対象ではありません。**

能力宣言（どのcontractを、どのrecipe kindで、どのtarget/profileに向けて提供するか）は、**catalogを作る者の入力**であって、Workspaceの意味でもpluginの意味でもありません。だから製品の構文に入れません。

## この決定が言っていないこと

- **能力宣言の形式**（ツールの入力の形）。本記録は決めていない
- **catalogが正しいこと**（K-3、SC0-26のまま）
- **誰が能力を宣言してよいか**（権限）。catalogは「buildできると名乗るデータ」で、**名乗りの妥当性は範囲外**
- 既定の試行でのClaim（§4.30のまま。catalogを渡す一手が要る）

## 覆す条件

- 能力宣言を**Workspaceの意味として扱う必要が出た場合**（例：Workspaceの妥当性検査がbuilderの有無に依存する）。そのときは`formatVersion`の改版と一緒にK-1を採る
- 能力宣言を**pluginの意味として扱う必要が出た場合**（例：pluginの起動時に自分のbuild能力を検証する）。同じくK-2を版付けで採る

いずれも**後から足せます**。逆（製品に足してから外す）は互換を壊すので、順序はこちらが正しい。

## 検証

- 三つの案の影響を、実装の該当行で確認（上の表）
- `go test ./tools/repository-plan-check -count=1`：PASS
- `doc-reference-probe` 未解決参照0件／文字化け検査 該当なし

**NOT RUN**：catalogを作るツールの実装（能力宣言の形式が未決のため）、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.32
- 欄の宣言面が無いこと：[`TEI_GO_SELF_BUILD_CATALOG_FIELD_PROVENANCE_2026-11-09.md`](TEI_GO_SELF_BUILD_CATALOG_FIELD_PROVENANCE_2026-11-09.md)
- 出所の決定（K-3）：[同 §4.22](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
