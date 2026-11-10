# TEIドッグフーディング：生成AIがCLIだけでTEI自身の設計を扱えるかの実測と課題

- **状態**: 2026-11-07 実測と課題の起票を完了。本書と実測記録は `main` に統合済み（PR #3162、merge commit `6fa253312`）。D-1・D-2・D-4〜D-8 は #3180〜#3186 として起票し、**すべて main へ統合済み**（2026-11-07〜08）。D-3・D-9・D-10 は #3345〜#3347 として起票済み。各課題の対応PRは §3.12 にまとめた。
- **D-1〜D-8 の修正**（main で確認済み）: D-1 `trace` の未対応kind→`UNSUPPORTED_SUBJECT_KIND`（#3180、`48163352a`）、D-2 応答の`workspace`欄（#3181、`dae251693`）、D-4 `writeFlags`（#3182、`2c8f083f2`系）、D-5 `-plugin-lock`のenvelope受理（#3183、`76f5da576`）、D-6 入力不正の終了コード統一（#3184、`c55498903`）、D-7 `-h`成功化（#3185、`575518c0f`）、D-8 要件JSONの例（#3186、`50b9eaa27`・`90d1fe518`）。D-3「版の名前と置き場所」は #3345、D-9 は #3346、D-10 は #3347。**いずれも未着手**。
- **担当課題**: [#3149](https://github.com/hat47x/tei-platform/issues/3149)（対応方針の正本）。
- **検証記録**: [TEI自身の設計をCLIで扱う実測](../evidence/TEI_DOGFOODING_CLI_SELF_DESIGN_2026-11-07/README.md)（生出力50ファイルを含む）。
- **関連**: [#1155](https://github.com/hat47x/tei-platform/issues/1155)（CLI主入口）、[#2605](https://github.com/hat47x/tei-platform/issues/2605)（Product Skill）、[#2354](https://github.com/hat47x/tei-platform/issues/2354)（CLI JSON契約、Closed）、[#2824](https://github.com/hat47x/tei-platform/issues/2824)（discoveryの広告、Open）。

## 1. この作業で確かめたこと

「生成AIがTEIを使ってTEI自身の設計を効果的に行えるか」を、次の3段に分けて実測した。

1. **発見**: CLIだけで、対象Workspaceにある設計要素の種類とIDを列挙できるか。
2. **読取りと検証**: 選んだ要素の意味、宣言済みの参照、Source上の位置、検証診断を、同じ版の範囲で機械可読に取得できるか。
3. **戻りと引継ぎ**: 得た結果から元の定義へ戻り、別のコマンド（または別セッション）の結果と同一版かを照合できるか。

「設計を行う」は、意味の受理・変更権限・本番配備を含まない。既存の設計記述を読み、整合を確かめ、根拠から元定義へ戻る範囲に限定した（[#3149](https://github.com/hat47x/tei-platform/issues/3149)）。

実測は、`origin/main` の祖先 `0b9c14a836eac0746339c7133f494452cfe56b33` から作った単一の実行物（SHA-256 `bf6930a047cf94f276fb63b082bf71599986d71f1a5c53804105c04aa0021cfd`）で行った。ビルドと実行はWSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64` の環境である。

## 2. 結論の要約

- **発見・読取り・検証の主要な経路は成立した。** `commands` → `model catalog` → `inspect` → `trace` → `validate -json` を同じ実行物で連続実行し、各段の終了コードと`ok`、`diagnostics`、照合値を別々に取得できた。`validate`・`model catalog`・`inspect`・`trace`の`worldFingerprint`は同一Workspaceで一致した。
- **「設計を扱う」ための不足は、主に4点に集まる。** 対象の種類の非対称、Workspace同一性の欠落、版の値の名前と置き場所の不統一、設計文書（Markdown）が意味として読まれないこと。
- **副作用の宣言と実挙動が一致しない箇所がある。** `commands`は`view`を`effect: read-only`と広告するが、`-execute`に`-operation-log`や`-lifecycle-state`を付けると作業ディレクトリへファイルを書く。
- 検出した課題は10件。うち D-1・D-2・D-4〜D-8 は #3180〜#3186 として起票済み。D-3・D-10 は [#2354](https://github.com/hat47x/tei-platform/issues/2354)（Closed）・[#2824](https://github.com/hat47x/tei-platform/issues/2824)・[#2822](https://github.com/hat47x/tei-platform/issues/2822) の範囲、D-9 は [#2619](https://github.com/hat47x/tei-platform/issues/2619)・[#2621](https://github.com/hat47x/tei-platform/issues/2621) の設計範囲として、新規Issueを作らない。重複確認と分割先は§4、次の作業は§6にある。

## 3. 検出課題（一覧）

| ID | 課題 | 区分 | 重複確認 |
| --- | --- | --- | --- |
| D-1 | `trace`が`semantic-candidate-key`を扱えず、未知kindと同じ符号で失敗する | CLI契約 / 種類の非対称 | 該当Issueなし（`#1165`はClosed） |
| D-2 | Workspaceの名前とmanifestの照合値が、どのCLI応答にも出ない | CLI契約 / 同一性 | 該当Issueなし（`#2823`はsources診断で別） |
| D-3 | 「版」を表す値の名前と置き場所がコマンドごとに違う | CLI契約 / 版の統一 | `#2354`（Closed）の残範囲。`#2824`と関連 |
| D-4 | `view`が`effect: read-only`を広告しつつ、`-execute`の記録系flagでファイルを書く | CLI契約 / 作用の宣言 | 該当Issueなし（`#2998`は`committed`の意味で別） |
| D-5 | `plugin-lock`のenvelopeをそのまま`-plugin-lock`へ渡すと失敗する（round-trip不成立） | CLI契約 / 引数の受理 | 該当Issueなし |
| D-6 | 同じ入力不正でも、`plugin-runtime-plan`は1、`plugin-resolve`は2を返す | CLI契約 / 終了コード | 該当Issueなし |
| D-7 | `-h`が3コマンドで`INVALID_ARGUMENT`（終了コード2）になる | CLI契約 / 発見性 | `#2824`と近いが対象が異なる |
| D-8 | `knowledge-qualify`の要件JSONの形がGoの構造体にしかなく、例も文書も無い | 文書 / 入力契約 | 該当Issueなし |
| D-9 | 設計文書（`docs/`・`plan/`）をWorkspaceにしても、CLIは管理ファイルとしてしか扱わない | 設計支援の不足 | `#2619`・`#2621`（Markdown設計）と関連、別課題 |
| D-10 | 出力の形の不統一（`diagnostics[]`の二重構造、`result`の配列、`model catalog`の標準定義混在） | CLI契約 / 形の統一 | `#2822`（`declarationCounts`）・`#2824`と関連 |

---

### D-1 `trace`が扱えない種類を、未知kindと同じ符号で返す

**再現手順**

```bash
tei model catalog -workspace examples/xtea-lowcode           # semantic-candidate-key: sales.order.order_identity が indexed
tei inspect -workspace examples/xtea-lowcode \
  -subject-kind semantic-candidate-key -subject-id sales.order.order_identity    # 終了コード0
tei trace -workspace examples/xtea-lowcode -scope dogfood \
  -subject-kind semantic-candidate-key -subject-id sales.order.order_identity -direction both -mode summary
```

**観測**

- `inspect`は`ok:true`で候補キーの定義を返す（生データ`cli-inspect-candidate-key.json`）。
- `trace`は終了コード1、`error.code=SUBJECT_NOT_FOUND`、`message="subject does not exist in typed graph"`（`cli-trace-candidate-key-not-found.json`）。
- 存在しない種類`not-a-kind`でも、`trace`の応答は**バイト単位で同一**だった（`cli-trace-unknown-kind.json`。`cmp`で一致）。
- 一方`inspect`は未知kindに対して`UNSUPPORTED_SUBJECT_KIND`・終了コード2を返す（`cli-failure-inspect-unsupported-kind.json`）。
- 種類ごとの突合（`cli-kind-matrix.json`）では、19種類のうち`examples/xtea-lowcode`の`semantic-candidate-key`だけが`inspect=ok`・`trace=SUBJECT_NOT_FOUND`になった。

**期待との差**

CLI利用者が「この対象には宣言済みの参照が無い」と「この種類は参照探索の対象外」を区別できない。前者は結果の範囲、後者は機能の非対応であり、次の行動が変わる（前者は Source を読む、後者は別の入口を探すか未対応として報告する）。

**影響**

生成AIが「参照が無い」と結論づけると、実際には未対応なだけの関係を「無関係」と報告する危険がある。`#2605`のSkill手順も`trace`の結果範囲を「宣言済みの直接参照」に限定しているが、種類ごとの非対応は明記していない。

**候補の修正**

1. `trace`でも未知kindを`UNSUPPORTED_SUBJECT_KIND`（終了コード2）として返し、`SUBJECT_NOT_FOUND`（1）と区別する。
2. `derivedbridge`が投影しない種類（`semantic-candidate-key`・`semantic-functional-dependency`）を、応答に`unprojectedKind`などの欄で明示する。第二の正本を作らず、既存の`authorityLimits`と同じ形の注記で足りるかを含めて判断する。

**対応（2026-11-07）**

候補1を [PR #3208](https://github.com/hat47x/tei-platform/pull/3208) で実装した。`trace`は世界を読む前に投影範囲を確かめ、対象外なら`UNSUPPORTED_SUBJECT_KIND`（終了コード2）を返す。投影する種類で識別子が存在しない場合は`SUBJECT_NOT_FOUND`（終了コード1）のままである。続けて、投影範囲の一覧を`derivedbridge.SupportsSubjectKind`へ寄せ、`derivedbridge`の投影結果と一致することを検査する試験を加えた。候補2の`unprojectedKind`欄は、失敗時に符号で示せるため追加していない。追跡は[#3180](https://github.com/hat47x/tei-platform/issues/3180)。

---

### D-2 Workspaceの名前とmanifestの照合値が応答に出ない

**再現手順**

```bash
tei model catalog -workspace reference/workspace      # workspace: tei_reference
tei model catalog -workspace examples/xtea-lowcode    # workspace: xtea_lowcode
tei inspect -workspace reference/workspace -subject-kind semantic-data -subject-id sales.order
tei inspect -workspace examples/xtea-lowcode -subject-kind semantic-data -subject-id sales.order
```

**観測**

- 2つのWorkspaceはどちらも`sales.order`を宣言するが、意味は異なる（`cli-inspect-sales-order.json`と`cli-inspect-sales-order-xtea-lowcode.json`の`worldFingerprint`は`sha256:c92de09f…`と`sha256:8930145c…`）。
- どちらの応答にも`workspace:`の名前（`tei_reference`／`xtea_lowcode`）は現れない。`tei.workspace.yaml`は`workspace-files`の`managedFiles`にも載らない（`cli-workspace-files-reference.json`の54件は`.tei`51・`.sql`2・`.md`1で、manifestは含まない）。
- `internal/workspace`の`Result`は`ManifestIdentity()`（パスとSHA-256。`loader.go`）を持つが、CLI応答には出ない。`workspace-files`の`result`は`managedFiles`と`sourceRootFindings`だけで、`source-index`の`result`は`artifacts`と`references`だけである（`cli-workspace-files-reference.json`・`cli-source-index-first-workspace.json`）。

**期待との差**

同じCanonical IDを複数のWorkspaceで扱うとき、応答だけではどのWorkspaceの結果か示せない。`worldFingerprint`は内容の同一性を示すが、Workspaceの識別子ではない。Skillの手順（`#2605`）は「対象Workspace/Source改訂」を作業結果に含めるよう求めているが、CLIの応答からは取れない。

**影響**

生成AIが複数のWorkspaceを1セッションで扱う場合、結果の混在を防ぐには呼出し側がパスを別に記録する必要がある。パスの表記揺れ（相対／絶対、シンボリックリンク）を吸収した「作業領域の識別子」はCLIから得られない。

**候補の修正**

`model catalog`・`inspect`・`trace`・`validate`の応答に、manifestの`workspace`名と`formatVersion`、manifest自体のSHA-256を載せる。既存の「新しい欄を足すのは互換な変更」の規則（`CLI_JSON_OUTPUT_CONTRACT_2026-10.md`§5）の範囲で足せるか、`v1`を要するかを判断する。

**対応（2026-11-07）**

互換な変更として足した。`workspace-files`も含む5コマンドの成功応答が`workspace`（`name`・`formatVersion`・`manifest.sha256`）を持つ。`internal/workspace`に`WorkspaceIdentity`と`Result.Identity()`を置き、manifestの`workspace:`を`name`として読む（`tei.workspace.tei`の旧形式も同じ）。`workspace:`が無いmanifestは従来どおり読み込み、`name`は空にする（パスから名前を作らない）。重複した`workspace:`行は拒否する。追跡は[#3181](https://github.com/hat47x/tei-platform/issues/3181)。

---

### D-3 「版」を表す値の名前と置き場所がコマンドごとに違う

**観測**

| コマンド | 版を表す欄 | 置き場所 | 値の意味 |
| --- | --- | --- | --- |
| `validate -json` | `worldFingerprint`・`sourceIndexFingerprint`・`declarationCounts` | 応答の最上位 | 正規化モデルと入力元索引 |
| `model catalog` | `worldFingerprint`・`sourceIndexFingerprint`・`modelRevision`・`declarationCounts` | `result`の下 | 同上＋管理対象Sourceの版 |
| `inspect` | `worldFingerprint`・`sourceIndexFingerprint` | `result`の下 | 同上 |
| `trace` | `worldFingerprint`・`sourceIndexFingerprint`・`snapshotFingerprint` | `result`の下（`snapshotFingerprint`はさらに`trace`の下にも同名で存在） | 同上＋継続用トークン |
| `view -format json` | `semanticFingerprint`のみ | `result`の下 | **使用する名前空間だけ**の意味 |
| `workspace-files`・`source-index` | なし | — | ファイル／索引の内容のみ |

**観測の補足**

- `view`の`semanticFingerprint`は`sha256:f6c4cf29…`、同じWorkspaceの`worldFingerprint`は`sha256:c92de09f…`で**異なる**（`cli-view-application.json`と`cli-model-catalog-reference.json`）。
- `trace`は`result.snapshotFingerprint`と`result.trace.snapshotFingerprint`の2つを持ち、片方は継続トークン、片方は関係図の識別子である（`getting-started.md`はこの区別を説明している）。

- 2026-11-09（#3625）: 内側の`result.trace.snapshotFingerprint`を`result.trace.graphFingerprint`へ改名し、旧名は1 cycleのaliasとして同値を残す。継続ページには`nextOffset`の隣へ`result.trace.continuation.fingerprint`（外側と同値）と`field`を足し、どの値を`-expected-fingerprint`へ渡すかを応答の中だけで判別できるようにした。- 意味の版を`semanticFingerprint`、世界全体を`worldFingerprint`と呼び分けているが、名前だけでは「どちらが正規化モデルか」は分からない。

**期待との差**

同じ「版」でも、比べてよい相手が異なる。`view`の値と`model catalog`の値をそのまま比較すると不一致になるが、これは内容の変化ではなく**範囲の違い**である。名前が似ているため、生成AIは同一の版と誤認しやすい。

**影響**

`#2605`のSkillは「結果ごとの`worldFingerprint`と`sourceIndexFingerprint`を比較する」と書いているが、`view`にはそのどちらも無い。`view`を経由した結果は版の照合ができない。

**候補の修正**

1. `view`にも`worldFingerprint`・`sourceIndexFingerprint`を足し、`semanticFingerprint`は範囲を名前で示す（例：`semanticUsesFingerprint`）。
2. 置き場所を最上位へ揃えるか、`result`の下へ揃えるかを決め、`CLI_JSON_OUTPUT_CONTRACT_2026-10.md`§5の「版を上げる変更」に該当するか判断する。

関連: [#2354](https://github.com/hat47x/tei-platform/issues/2354)（Closed。schemaVersionと失敗時形式）、`#2824`（Open。discovery広告と文書）。

---

### D-4 `view`が`effect: read-only`を広告しつつ、`-execute`でファイルを書く

**再現手順**

```bash
tei view -workspace reference/workspace -application order_entry \
  -pattern order_entry.submit_order -command sales.order.submit \
  -input order=sales.order -execute -operation-log /tmp/tei-dogfood-out/op.jsonl
ls -l /tmp/tei-dogfood-out/op.jsonl

tei view -workspace reference/workspace -application order_entry \
  -pattern order_entry.submit_order -command sales.order.submit \
  -input order=sales.order -execute \
  -lifecycle-state /tmp/tei-dogfood-out/state.db -lifecycle-command submit \
  -lifecycle-owner owner1 -lifecycle-generation 1
ls -l /tmp/tei-dogfood-out/state.db
```

**観測**

- `commands`は`view`を`"effect": "read-only"`と広告する（`cli-commands.json`）。
- `-execute`の応答は`committed: true`・`persisted: false`・`persistence: "not-applicable"`・`durability: "none"`、終了コード0（`cli-view-execute.json`）。
- 同じ呼出しに`-operation-log`を付けると`op.jsonl`（215バイト）が作られる。`-lifecycle-state`を付けると`state.db`（12,288バイト）が作られる（`cli-view-execute-operation-log.txt`・`cli-view-execute-lifecycle-state.txt`）。
- CLIの`optional`欄は`-input and -execute with -command`、`-operation-log with -execute`などと記し、`-execute`が特有の作用を持つことを示している。

**期待との差**

`effect: read-only`は「正本を書き換えない」ことを示すが、作業ディレクトリへのファイル書込みは「外部から見える作用」である。`commands`の`effect`欄には「正本を書き換えない」と「どこにも書かない」のどちらかを選ぶ基準が無く、読み手は`read-only`を後者と受け取りやすい。

**影響**

生成AIが`commands`の一覧だけから安全な操作を選ぶと、`-execute`付きの呼出しで意図しないファイルを作る。実際の書込み先はflagで指定するため無制限ではなく、指定しなければ書かない。それでも、宣言と作用の対応は呼出し側が引数を読んで判断する必要がある。

**候補の修正**

1. `commands`の`effect`を、`read-only`／`read-only-with-named-output`（出力先を明示しないと書かない）／`write`のように分け、`view`を2番目にする。既存の`#2824`の`jsonContract`と同じく、entryに小さな欄を足す方向。
2. または、`-execute`を含む呼出しを別entry（例：`view execute`）として広告し、`effect`を分ける。

`AGENTS.md`のA2（隔離ブランチでの変更）の範囲で実装候補を出せるが、`effect`の語彙はCLIの契約なので、決めるのはOwner判断とする。

関連: [#2998](https://github.com/hat47x/tei-platform/issues/2998)（`committed`の意味）、`#2824`（discoveryの広告）。

---

### D-5 `plugin-lock`のenvelopeをそのまま`-plugin-lock`へ渡せない

**再現手順**

```bash
tei plugin-lock -workspace reference/workspace > /tmp/lock-envelope.json
tei plugin-runtime-plan -workspace reference/workspace -environment development \
  -plugin-lock /tmp/lock-envelope.json          # 終了コード1
# 内側の result を保存して渡す
python3 -c 'import json,sys; r=json.load(sys.stdin); json.dump(r["result"],open("/tmp/lock-inner.json","w"))' < /tmp/lock-envelope.json
tei plugin-runtime-plan -workspace reference/workspace -environment development \
  -plugin-lock /tmp/lock-inner.json             # 終了コード0
```

**観測**

- envelopeを渡すと`PLUGIN_RUNTIME_PLAN_FAILED`／`unsupported plugin lock format 0`（`cli-plugin-runtime-plan-envelope-error.json`）。
- 内側の`result`を渡すと成功（`cli-plugin-runtime-plan-inner-lock.json`）。
- `plugin-resolve`でも同じ差（envelopeは`PLUGIN_RESOLVE_FAILED`・終了コード2、内側は成功）。
- `getting-started.md`§5は`python3 -c`で`result`を取り出す手順を書いており、**実装の挙動を説明している**。ただしCLI自身の出力をCLI自身が読めない点は残る。

**期待との差**

同じツールの出力を次の呼出しの入力に使えない。生成AIが「lockを作る→計画を立てる」を連続実行するとき、外部の`python3`に依存する。`getting-started.md`はこの依存を明記しているが、`-plugin-lock`がenvelopeも受理できるか、専用の`-plugin-lock-out`を持つかは未決である。

**影響**

パイプラインの途中に別言語の処理が入る。`jq`や`python3`が無い環境では、`plugin-lock`の出力をそのまま使えない。

**候補の修正**

`-plugin-lock`がenvelopeと内側の両方を受け付ける（`formatVersion`の有無で判別する）。互換な変更として足せるか、`plugin-lock`の出力を内側だけにする（envelopeをやめる）かのどちらかを選ぶ。後者は`#2354`の決定を覆すためOwner判断が要る。

---

### D-6 同じ入力不正で、`plugin-runtime-plan`は1、`plugin-resolve`は2を返す

**観測**

同じ`unsupported plugin lock format 0`に対して、`plugin-runtime-plan`は終了コード1、`plugin-resolve`は終了コード2（`cli-plugin-runtime-plan-envelope-error.json`・`cli-plugin-resolve-envelope-error.json`）。

**期待との差**

`getting-started.md`は「0が成功、1が検証不成立、2が入力・読込みなどの実行失敗」と定める。同じ「入力の不正」が両方に当たり得るが、コマンドごとに解釈が違う。

**影響**

生成AIが終了コードだけで再試行の可否を決めると、同じ原因で違う判断をする。`error.code`を読めば区別できるが、終了コードはパイプラインの分岐に使われやすい。

**候補の修正**

入力の不正（lock形式の非対応）を、両コマンドで同じ終了コードに揃える。どちらへ揃えるかは`CLI_JSON_OUTPUT_CONTRACT_2026-10.md`§7の決定案（D案）と合わせて決める。

---

### D-7 `-h`が3コマンドで`INVALID_ARGUMENT`（終了コード2）になる

**観測**

`inspect -h`・`view -h`・`knowledge-qualify -h`はいずれも`{"ok":false,"error":{"code":"INVALID_ARGUMENT","message":"flag: help requested"}}`、終了コード2（`cli-help-inspect.json`・`cli-help-view.json`・`cli-help-knowledge-qualify.json`）。

**期待との差**

利用者が引数を調べる最も自然な操作が「エラー」として返る。`commands`の`required`／`optional`は文字列の配列で、型・既定値・許容値は載っていない。`inspect`の`-subject-kind`が12種類のどれか、といった情報は`getting-started.md`の本文にしかない。

**影響**

生成AIは`getting-started.md`を読むか、誤った引数を試して符号を得るしかない。前者は文書と実装のずれ（`#2824`）に弱く、後者はムダな呼出しを増やす。

**候補の修正**

1. `-h`を成功（終了コード0、標準出力へusage）にする。
2. `commands`のentryに、引数の型・許容値（`subjectKind`の列挙など）を足す。`#2824`の`schemaVersion`追加と同じ方向。

---

### D-8 `knowledge-qualify`の要件JSONの形が、コードにしかない

**観測**

- `{}`を渡すと`KNOWLEDGE_BUILD_FAILED`／`qualification expectedKind is required`、終了コード2（`cli-knowledge-qualify-empty-requirements.json`）。
- `{"requirements":[…]}`を渡すと`json: unknown field "requirements"`、終了コード2（`cli-knowledge-qualify-unknown-field.json`）。
- 正しい形は`internal/knowledgecontext/qualification.go`の`QualificationRequirements`構造体にある（`expectedKind`必須、`application`は`expectedKind=application-pattern`のときだけ、`dependencies`は`relation`・`kind`・`id`が必要）。
- リポジトリ内に要件JSONの例ファイルは無い。`docs/business-system-design-reading-guide.md`は`-qualification-requirements <requirements.json>`とだけ書く。

**期待との差**

生成AIが`knowledge-qualify`・`knowledge-review-handoff`を使うには、Goのソースを読むか、失敗のメッセージから形を推測するしかない。`unknown field`の拒否は厳密で良いが、正解の形を示す経路が無い。

**影響**

未知の入力形式に対する試行錯誤が増える。`#2605`のSkillは`knowledge-*`を扱っていないため、この経路の手順も未整備である。

**候補の修正**

1. 要件JSONの最小例を`docs/`または`plan/design/CLI_JSON_OUTPUT_CONTRACT_2026-10.md`へ置く。
2. `-h`（D-7）か`commands`のentryに、要件ファイルの必須欄を示す。

---

### D-9 設計文書（`docs/`・`plan/`）をWorkspaceにしても、CLIは管理ファイルとしてしか扱わない

**再現手順**

```bash
mkdir -p /tmp/md-ws/docs /tmp/md-ws/plan
printf 'workspace: docs_probe\nformatVersion: 1\nsources:\n  - docs\n  - plan\n' > /tmp/md-ws/tei.workspace.yaml
cp docs/overview.md /tmp/md-ws/docs/
cp plan/design/CLI_FIRST_INTERFACE_DESIGN.md /tmp/md-ws/plan/
tei validate -workspace /tmp/md-ws -json       # ok:true、WORKSPACE_DECLARES_NO_MODEL_ELEMENTS（warning）
tei workspace-files -workspace /tmp/md-ws      # 全件 readByLoader:false、sourceRootFindings warning
tei world -workspace /tmp/md-ws                # 標準定義だけ
```

**観測**

- `validate`は`ok:true`・終了コード0。`diagnostics`に`WORKSPACE_DECLARES_NO_MODEL_ELEMENTS`（warning）を1件返し、`sourceRootFindings`に`docs`と`plan`の2件のwarning（「Loaderが解釈しないroot」）を返す（`cli-validate-markdown-workspace.json`）。
- `workspace-files`は`readByLoader:false`と`sourceRootFindings`のwarningを返す（`cli-workspace-files-markdown-workspace.json`）。
- `world`は標準定義（`std`）だけを返す（`cli-world-markdown-workspace.json`）。

**期待との差**

TEI自身の設計は、`docs/`と`plan/`のMarkdownで書かれている。この実測の題材B（`cmd/tei/cli_contract.go`・`docs/getting-started.md`・実装の三者照合）は、まさにこの本文をCLIで扱えないことを示した。CLIから見えるのは「管理対象だが読まれないファイル」である。

**影響**

「TEIがTEI自身の設計を扱えるか」という問いに対して、現状の答えは**Markdownは対象外**である。生成AIは`grep`と`read`で本文を読み、CLIは`.tei`の意味だけを扱う。この分担自体は設計判断として妥当だが、CLIの応答からは「何が読まれていないか」しか分からない。

**候補の修正**

- 即時の修正候補ではなく、設計判断の対象。`#2619`（Everything is Data：Markdown設計の横断統合）と`#2621`（Markdown設計・表形式テスト・Authoring契約）が扱う範囲である。本記録では、**現状の到達点と不足を明示する**に留める。
- 小さな改善として、`validate`が`readByLoader:false`のファイル数（ここでは2件）を警告に含める案がある。現状はroot単位のwarning（`docs`・`plan`）だけでは、管理対象のうち何件が読まれていないかを応答だけから数えられない。ただし`#2823`（sources診断）の範囲と重なる。

---

### D-10 出力の形の不統一

**観測**

| 箇所 | 観測 | 生データ |
| --- | --- | --- |
| `validate`の`diagnostics[]` | 要素が`{"diagnostic":{…}}`の二重構造 | `cli-validate-xtea-lowcode.json` |
| `validate`の`proposals[]` | 要素が平坦（`code`・`severity`・`diagnostic`・`accepted`・`revision`） | 同上 |
| `requirements`の`result` | 配列（`result: []`） | `cli-requirements-reference.json` |
| `model catalog`の`entries` | 標準定義の型12件を含む（`declarationCounts.types=0`のWorkspaceでも`semantic-type`のentryがある） | `cli-model-catalog-xtea-lowcode.json` |
| `view`の`result.schemaVersion`と`result.plan.schemaVersion` | 同じ値`tei.applicationview/v0`が2箇所 | `cli-view-execute.json` |
| `commands`のentry | 9件が`modes`を広告するが、`identity`を持つentryは0件（main相当の再確認） | `cli-main-recheck-commands.json` |

**期待との差**

同じ「診断の一覧」が2つの形で出る。`diagnostics`は機械処理が前提で、`proposals`は受理候補の表示用と役割が違うが、名前からは分からない。`model catalog`の`entries`は「宣言の一覧」と「読める対象の一覧」を兼ねており、`declarationCounts`と件数が一致しない（`#2822`が扱う）。

**影響**

生成AIが`entries`を「このWorkspaceの宣言」と数えると過大になる。`sourceStatus: not-indexed`で判別できるが、`declarationCounts`と突き合わせる必要がある。

**候補の修正**

- `diagnostics`の二重構造は`v0`の既存形であり、変えるなら`v1`が要る（`CLI_JSON_OUTPUT_CONTRACT_2026-10.md`§5）。`#2354`の決定を待つ。
- `model catalog`の`entries`に`origin: declared｜prelude`のような欄を足し、標準定義を機械的に除外できるようにする。`#2822`と重なるため、同Issueへ追記するのが適切。

**対応（2026-11-08）**

- `model catalog`の`entries`は`prelude: true`で標準意味型（`std`）を印付けする形にした（PR #3404、merge `6a51aeedd`）。`declarationCounts.total`と併用し、`entries`の側からも宣言と標準定義を区別できる。
- `requirements`の`result`が配列であることは、`tei commands`のentryに`resultShape: "array"`を足して発見可能にした（PR #3370、merge `a46e421e3`）。省略した他のコマンドは「object保証」ではなく未宣言と契約文書に明記した。
- `validate`の`diagnostics[]`は、要素が`diagnostic`を持つ入れ子の形を`v0`の契約として維持する。`proposals[]`が平坦なのは、位置情報を持つ`diagnostics[]`と役割が違うためで、平坦化は既存の読み手の解釈を変える`v1`の判断に残した（`CLI_JSON_OUTPUT_CONTRACT_2026-10.md`§4・`cmd/tei/cli_validation_test.go`）。
- 追跡は[#3347](https://github.com/hat47x/tei-platform/issues/3347)（Closed）。形の統一をv0のまま進めるかv1を新設するかは、未決の設計判断として残る。

### 3.11 マージ後main相当での再確認（2026-11-07 追記）

`origin/main`（`70089048e`）を取り込んだ作業ツリーから別の実行物（SHA-256 `fe52894ab9a092c896125b834aa101b81cc0c2097c1e97f0037b805626a99c70`、`vcsRevision: 6d215d328…`、`vcsModified: true`）を作り、D-1〜D-5・D-7〜D-9を再確認した。詳細は[実測記録](../evidence/TEI_DOGFOODING_CLI_SELF_DESIGN_2026-11-07/README.md)の§2.10にある。

| 課題 | 再確認の結果 |
| --- | --- |
| D-1 | 同じ。候補キーと未知kindの`trace`応答は`SUBJECT_NOT_FOUND`・終了コード1で、バイト単位でも同一 |
| D-2 | 同じ。`model catalog`・`inspect`・`validate`・`workspace-files`・`source-index`に`workspace`／`manifest`の欄が無い |
| D-3 | 同じ。`view -format json`の`result`に`worldFingerprint`・`sourceIndexFingerprint`が無く、`semanticFingerprint`は`sha256:f6c4cf29…`のまま |
| D-4 | 同じ。`view -execute -operation-log`は`op.jsonl`（215バイト）を書く。テキスト出力に`authorization: reference-allow-all`の行が加わった |
| D-5 | 同じ。envelopeは終了コード1、内側の`result`は終了コード0 |
| D-6 | 未確認（再確認では`plugin-resolve`との終了コード差を測っていない） |
| D-7 | 同じ。`inspect -h`は`INVALID_ARGUMENT`・終了コード2 |
| D-8 | 同じ。要件JSONの`{}`は`qualification expectedKind is required` |
| D-9 | 同じ。Markdown Workspaceは`ok:true`・`WORKSPACE_DECLARES_NO_MODEL_ELEMENTS`・`sourceRootFindings`2件 |

`commands`は19件のままで、`world`だけが`schemaVersion`を持たない。9件が`modes`を広告するようになっている（`#2824`の追加）。**D-3・D-10の一部（`commands`の広告）は`main`で進んでいる**が、`view`の版の欄と`entries`の標準定義混在は変わっていない。

この再確認は`vcsModified: true`の作業ツリーから作った実行物であり、§1の固定実行物の実測と数値を直接比較しない。CI・実DB・実Providerは引き続きNOT RUN。

### 3.12 課題の対応状況（2026-11-08 追記）

D-1・D-2・D-4〜D-8 の対応が `main` に入り、対応する Issue はすべて Closed になった。D-3・D-9・D-10 は 2026-11-08 に #3345〜#3347 として起票され、**いずれも未着手**である。

| 課題 | Issue | 対応（`main` のPR・merge） | 何が変わったか |
| --- | --- | --- | --- |
| D-1 | #3180 Closed | #3208（`55cb15e93`）、#3215（`0d4d41d82`） | `trace` は投影しない種類を `UNSUPPORTED_SUBJECT_KIND`（終了コード2）で返し、`SUBJECT_NOT_FOUND`（1）と区別する。投影範囲は `derivedbridge.SupportsSubjectKind` に一本化し、投影結果との一致を試験で固定した。 |
| D-2 | #3181 Closed | #3247（`1a66959cd`） | `validate -json`・`model catalog`・`inspect`・`trace`・`workspace-files` の成功応答が `workspace`（`name`・`formatVersion`・`manifest.sha256`）を持つ。`tei.workspace.tei` の旧形式も同じ。 |
| D-4 | #3182 Closed | #3333（`b83c8309c`） | `view` の entry に `writeFlags`（`-operation-log`・`-lifecycle-state`）を載せ、実ファイル書込みと照合する。`effect` の語彙は変えていない。 |
| D-5 | #3183 Closed | #3214（`3774f9945`） | `-plugin-lock` が `plugin-lock` の envelope をそのまま受理する。 |
| D-6 | #3184 Closed | #3233（`163b02be9`） | 使えない plugin lock を両コマンドで終了コード2の `INVALID_PLUGIN_LOCK` に揃えた。 |
| D-7 期待1 | #3185 Closed | #3227（`575518c0f`） | `-h` は成功（終了コード0、標準出力へusage）。 |
| D-7 期待2 / D-10の一部 | #3242 Closed | #3307（`6a956e92d`） | `tei commands` の entry が `argumentValues`・`argumentType`・`argumentDefaults` を持ち、実装の受理値と同じ変数を参照する。 |
| D-8 | #3186 Closed | #3323（`03fb618bb`） | `docs/business-system-design-reading-guide.md` に要件JSONの形と既存例を記載し、その例を実際に読む回帰試験を追加した。 |

この表のPR・mergeは、いずれもCI（Repository Healthを含む）の成功と、変更範囲の `go test` を確認して統合した。**意味の受理・実DB・実Provider・本番配備の証拠ではない**。§1の固定実行物による再実測（同じバイナリ digest での比較）はまだ行っていない。次の再実測では、この表の後の `main` から1つの実行物を作り、D-1〜D-8の解消とD-3・D-9・D-10の残存を同じ手順で確かめる。

---

## 4. 重複の確認と分割先

| 課題 | 既存Issueとの関係 | 分割先（2026-11-07 起票済み） |
| --- | --- | --- |
| D-1 | `#1165`（Closed）はtraceの初期実装。種類の非対応は未起票 | [#3180](https://github.com/hat47x/tei-platform/issues/3180)（CLI: traceの未対応kindをSUBJECT_NOT_FOUNDと区別する） |
| D-2 | `#2823`（Open）はsources診断、`#2605`はSkill側の記録 | [#3181](https://github.com/hat47x/tei-platform/issues/3181)（CLI: 応答にWorkspace同一性を含める） |
| D-3 | `#2354`（Closed）の残範囲。`#2824`（Closed）は広告の一致 | [#3345](https://github.com/hat47x/tei-platform/issues/3345) として起票（2026-11-08） |
| D-4 | `#2998`（Open）は`committed`の意味。作用の宣言は未起票 | [#3182](https://github.com/hat47x/tei-platform/issues/3182)（CLI: effectを作用の範囲で分ける） |
| D-5 | 該当なし（`getting-started.md`は回避策を記載済み） | [#3183](https://github.com/hat47x/tei-platform/issues/3183)（CLI: plugin-lockのround-trip） |
| D-6 | 該当なし | [#3184](https://github.com/hat47x/tei-platform/issues/3184)（CLI: 同じlock不正で終了コードが分かれる） |
| D-7 | `#2824`（Open）と近いが対象が異なる | [#3185](https://github.com/hat47x/tei-platform/issues/3185)（CLI: -hと引数契約） |
| D-8 | 該当なし | [#3186](https://github.com/hat47x/tei-platform/issues/3186)（文書: knowledge-qualifyの要件例） |
| D-9 | `#2619`・`#2621`（Open）の設計範囲 | [#3346](https://github.com/hat47x/tei-platform/issues/3346) として起票（2026-11-08） |
| D-10 | `#2822`（Closed）と重なる | [#3347](https://github.com/hat47x/tei-platform/issues/3347) として起票（2026-11-08） |

**分割は 2026-11-07 に完了した。** 本書を `main` へ統合した後（PR #3162、merge commit `6fa253312`）、重複の最終確認をして D-1・D-2・D-4〜D-8 を #3180〜#3186 として起票した。D-3・D-10 は `#2824`・`#2822` の範囲、D-9 は `#2619`・`#2621` の設計範囲として、新規Issueを作らない。各Issueは、本書 §3 の再現手順と `plan/evidence/TEI_DOGFOODING_CLI_SELF_DESIGN_2026-11-07/` の生データのパスを引用している。

`#3149` の定めどおり、本記録は横断の実測と転記を持ち、個別の修正は分割したIssueが持つ。

## 5. 設計支援の観点で残った問い

- **発見の粒度**: `commands`はコマンド名と引数の一部を返すが、対象の種類（`subject-kind`の許容値）は返さない。生成AIが「何を inspect できるか」を知るには`model catalog`の結果と文書を突き合わせる必要がある。これはD-7の一部である。
- **「効果的」の測り方**: 本実測は、成功・失敗・未対応の区別が付くことまでを確かめた。設計判断の質（どの要素を読むべきか、どの順で確かめるか）は測っていない。`#2605`のS1a（同一checkout・単一実行物での実測）と同じ範囲に留まる。
- **意味の受理との距離**: 本実測の全呼出しはread-onlyで、`view -execute`だけがローカルファイルを書いた。設計の変更を正本へ戻す経路（Proposal・Shared Acceptance）は試していない。`#3149`の完了条件にも含めていない。

## 6. 次に必要な作業

1. ~~本書と検証記録をブランチへコミットし、`#3149`へ結果を転記する~~（完了。PR #3162、merge commit `6fa253312`。`#3149` への転記は 2026-11-07 のコメント2件）。
2. ~~課題の分割~~（完了。D-1・D-2・D-4〜D-8 を #3180〜#3186 として起票。D-3・D-9・D-10 は #3345〜#3347 として起票）。
3. 修正の採否はOwner判断。本記録は候補と根拠を提示するに留める。#3180〜#3186 のいずれかを着手する場合は、本書 §3 の再現手順と生データから始める。
4. 再実測は、`0b9c14a8…`ではなく修正後の`main`で行い、同じ実行物の識別（SHA-256）を記録する。改行の異なるcheckoutでの照合（`#2399`）と、大規模Workspaceでの完全性（`#1177`）は別途。

## 7. 第2ラウンド（2026-11-08）: 修正後の再実測と MCP の位置づけ

第1ラウンドの起票から修正が進んだため、現行 `main`（`4fd88c2b1`）の単一実行物（SHA-256 `16c9090d…`）で同じ手順を再実行した。生出力は[第2ラウンドの実測記録](../evidence/TEI_DOGFOODING_ROUND2_2026-11-08/README.md)にある。

| 課題 | 現行main | 次に必要なこと |
| --- | --- | --- |
| D-1 `trace` の未対応kind | 解消（`UNSUPPORTED_SUBJECT_KIND`・終了コード2） | なし |
| D-2 Workspace同一性 | 解消（`workspace` 欄） | なし |
| D-3 版の欄の不統一 | 実測時点（`4fd88c2b1`）は残る（`view` は `semanticFingerprint` のみ）。**実測後に #3366（`9bc015876`）で解消**し、現在の main は応答の最上位に `worldFingerprint`・`sourceIndexFingerprint` を返す（#3345 は Closed） | なし（残るのは `semanticFingerprint` の範囲を名前で示すかの設計判断のみ） |
| D-3 版の欄の不統一 | **残る**（`view` は `semanticFingerprint` のみ、`worldFingerprint` を持たない） | `view` の版の欄をどう揃えるかは未決。`#2824`・`#2822` の範囲で扱う |
| D-4 `view` の作用の広告 | 解消（`writeFlags` に `-operation-log`・`-lifecycle-state`） | `effect` の語彙（`read-only-with-named-output` など）の統一は未着手 |
| D-5 `plugin-lock` のround-trip | 解消 | なし |
| D-6 不正lockの終了コード | 解消（間接確認。個別再実行は未実施） | 必要なら個別に再実行する |
| D-7 `-h` | 解消（終了コード0・usage） | なし |
| D-8 要件JSONの例 | 解消（読み案内の表と実例ファイル） | 実例ファイルがパーサを通る回帰試験は未整備（下記） |
| D-9 設計文書が意味として読まれない | **残る**（`readByLoader` 0件、`WORKSPACE_DECLARES_NO_MODEL_ELEMENTS`） | `#2619`・`#2621` の設計判断 |
| D-10 出力の形の不統一 | **残る**（`diagnostics[]` の二重構造） | `v1` を要するか、`#2354` の決定に沿うか |

第2ラウンドで新たに分かったこと:

- **MCP 入口は存在しない。** `commands` 19件にMCP関連の名前は無く、GoソースにもMCPサーバー実装は無い。`docs/architecture.md` は「TEI独自の開発用MCPサーバーは必須にしない」「将来追加しても別の正本を作らない」と位置づけている。依頼の範囲が CLI/MCP に広がったが、**実測できたのは CLI 経路だけ**であり、MCP経路は未実装として `NOT RUN` に置く。
- **discovery が引数の許容値と出力flagを広告するようになった。** `commands` の出力は 6,957 バイトから 9,513 バイトへ増え、`argumentValues`・`argumentType`・`argumentDefaults`・`writeFlags` を持つ。生成AIが「何を渡せるか」を文書なしで知る経路ができた。
- **D-3 は実測後に解消した。** #3366 が `view -format json` の応答の最上位へ `worldFingerprint`・`sourceIndexFingerprint` を足し、`result.semanticFingerprint` と並べて3つの値が別物として読めるようになった（#3345 は Closed、`9bc015876`）。実測時点の観測は変えず、追補を[実測記録](../evidence/TEI_DOGFOODING_ROUND2_2026-11-08/README.md) §9に置いた。
- **残るのは D-9 と D-10 の2件。** D-9 はMarkdownを意味として扱うかの設計判断（#3346）、D-10 は `v0` の互換性判断（#3347）を含み、いずれも単一の欄の追加では閉じない。
- **残る3件はいずれも単一の欄の追加では閉じない。** D-3 は版の語彙と置き場所の設計、D-9 はMarkdownを意味として扱うかの設計判断、D-10 は `v0` の互換性判断を含む。

次に必要な作業:

1. D-3・D-9・D-10 は、それぞれの設計判断のIssueで扱う。本記録は現況の観測を渡すだけに留める。
2. `docs/business-system-design-reading-guide.md` が参照する実例ファイルが `QualificationRequirements` として復号できることを、回帰試験で固定する余地がある（第2ラウンドでは未実施）。
3. MCP入口が実装された時点で、同じ手順をMCP経由でも実行し、CLIとの結果一致を確認する。入口が無い現時点では `NOT RUN`。

## 8. MCP経路（2026-11-08）: 参照AdapterでのCLI/MCP一致

依頼の範囲が `CLI/MCP` に広がったが、#3400 のとおり **TEIにMCP入口は無い**。そこで「入口を作る場合に、CLIと同じ結果を返せるか」だけを、参照実装で確かめた。生出力は[MCP経路の実測記録](../evidence/TEI_DOGFOODING_MCP_ROUND_2026-11-08/README.md)にある。

- **参照実装**: [tools/reference-mcp-readonly-adapter](../../tools/reference-mcp-readonly-adapter/README.md)。`internal/operations` の `CatalogWorkspace`・`InspectSubject`・`TraceWorkspace`・`ValidateWorkspace` を呼ぶだけで、第二の正本解釈・知識の保存先・検証規則・権限モデルを持たない。
- **結果**: 同じ Workspace に対して、`model catalog`・`validate`・`inspect`・`trace` の `worldFingerprint`・`sourceIndexFingerprint`・`snapshotFingerprint`・`sourceStatus`・`declarationCounts`・`schemaVersion` が **CLIと一致**した（不一致0件）。未対応の対象種類は `isError: true` と `UNSUPPORTED_SUBJECT_KIND` で返り、成功の空結果にしなかった。
- **境界**: Workspace は起動時に固定し、tool の入力に path を取らない。書込み・受理・配備・付与・実行の tool は無い。認可・開示範囲は Adapter では判定しない。
- **未確認**: 製品のMCP契約、MCPの版ごとの差、実Host・実Harnessからの接続、認可、外部送出、期限・再接続・同時実行。

### 8.1 次の判断

1. **MCP入口を作るか**（#3400）。作る場合の範囲は、この参照実装が扱った4操作（`model catalog`・`inspect`・`trace`・`validate`）に限る案を推す。`commands` の discovery と同じ範囲で、第二の正本を作らない制約をそのまま維持できる。
2. 作らない場合は、外部AIハーネスからはホスト提供のMCP経由で `tei` CLIを起動する経路に限られる（#2509 の実測と同じ形）。その場合は本節の参照実装を製品へは残さない。
3. どちらを選んでも、`docs/architecture.md` の「別の正本を作らない」制約を維持する。

## 9. 第3ラウンド（2026-11-08）: 設計文書の扱いと応答の同一性

第2ラウンドの後に main が進み、読み取りコマンドの応答を横断して調べた。生出力は[応答同一性の実測記録](../evidence/TEI_DOGFOODING_ROUND3_IDENTITY_2026-11-08/README.md)にある。

| 観点 | 実測前（`3b4f5519d`） | 対応 |
| --- | --- | --- |
| D-9 設計文書が意味として読まれない | `docs`／`plan`を`sources`に置くと`readByLoader:false`、`validate`の`diagnostics`は0件、`source-index`の`references`は`null` | **残る**（#3346は設計判断を#2621へ送ってクローズ済み） |
| Workspace同一性の有無 | `view`・`source-index`・`knowledge-context`・`requirements`に`workspace`欄が無い | **#3455として起票し、修正した** |
| 空の一覧の直列化 | `source-index`の`references`が`null` | **#3456として起票し、修正した** |

#3455 と #3456 の修正:

- 読み取りコマンドの応答すべてに`workspace`（`name`・`formatVersion`・`manifest`）を付けた。`-world`ファイルだけを入力した場合は、manifestを読んでいないため値を推測せず省く。
- `internal/sourceindex.Index`の`MarshalJSON`が、空の`artifacts`・`references`を`[]`にする。
- 公開契約の互換性: 既存欄は変えず、欄の追加と「0件を`null`から`[]`へ」の変更だけ。`null`を「欄が無い」と解釈していた読み手には告知が必要（[CLI JSON出力契約](../design/CLI_JSON_OUTPUT_CONTRACT_2026-10.md)）。

D-9は残る。CLIは設計文書を索引に載せるが、意味の定義として読まない。変更するなら、Markdownを意味として扱う設計判断（#2621）が前提になる。

## 10. 第4ラウンド（2026-11-09）: 修正後の版での再実測

#3455・#3456 の修正が `main`（`00ead5a98`、PR #3467）に統合された後、同じCLI操作で再測定した。生出力は[第4ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND4_2026-11-09/README.md)にある。

- 読み取りを行う11コマンドの応答すべてが`workspace`（`name`・`formatVersion`・`manifest`）を返した。同一性の無い応答は0件。
- 設計文書Workspaceの`source-index`は、参照0件を`[]`で返した（`artifacts`は2件）。
- `validate`の診断と`sourceRootFindings`は変わらず、D-9（設計文書が意味として読まれない）は残る。

この記録はCLIのread-only観測である。HTTP・MCP・Desktopの各入口の同値性は #3419〜#3421 の担当で、未実行（NOT RUN）のままである。

## 11. 第5ラウンド（2026-11-09）: CLIとMCP参照Adapterの一致と厳密入力

`main` `4f97c86cf` の CLI と `tools/reference-mcp-readonly-adapter` を同じWorkspaceへ向けて実行し、値・失敗・入力検証を比較した。生出力は[第5ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND5_CLI_MCP_2026-11-09/README.md)にある。

- 4操作（`validate`・`model catalog`・`inspect`・`trace`）の成功応答は、キー集合と型の差が0件だった。`SUBJECT_NOT_FOUND`・`UNSUPPORTED_SUBJECT_KIND` もCLIと同じ文字列符号で、MCPは `isError:true` と `ok:false` を併せて返した。
- 基準の版では、**宣言していない引数はMCPだけが無視して成功した**。CLIは `INVALID_ARGUMENT` で拒否する。これを #3575 として起票した。
- 起票のあと、`main` の `b88759643`（#3420 CAPI-2b）が同じ問題を直し、`rejectUnknownArguments` で不正値を `-32602` にすることを確認した。現行 `main` の実行物では未知欄・`workspace`欄・`null`・型不正がすべて拒否され、既存4操作の成功は変わらない。#3575 は重複として閉じる。
- 必須欠落の符号はCLIが `INVALID_SUBJECT`、MCPが `-32602` で、意味層とプロトコル層の対応は #3420 の未決である。値と代表的な失敗は揃ったが、失敗分類の統一は未達。
- `commands` の19件に操作ID・入口の欄は無く、入口ごとの対応はIssueと文書でしか辿れない。#3419 の対応表で扱う。

## 12. 第6ラウンド（2026-11-09）: discoveryの広告と実行の突き合わせ

ADR-0048 §2.6 の「各入口の広告が独立した正本にならないこと」を確かめるため、`tei commands` の19件と MCP `tools/list` の4件を、実際の受理と突き合わせた。生出力は[第6ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND6_DISCOVERY_2026-11-09/README.md)にある。

- 広告した subject-kind 12種（`inspect`）・10種（`trace`）のすべてで、実在しないIDに対する結果が `SUBJECT_NOT_FOUND`（exit 1）、実在するIDでは成功だった。広告と実行の食い違いは検出されなかった。
- フラグの条件（`-offset and -limit (full only)`・`-json (workspace only)`・`-input and -execute with -command`・`at most one of -pattern, -package, -command` など）も、実行時の受理と一致した。`effect` は19件すべて `read-only`、`writeFlags` は `view` のみ。
- 一方で、**CLI の discovery と MCP の `tools/list` を突き合わせる試験は無い**。MCP の tool 宣言は `inputSchema` と `description` だけで、`effect`・`schemaVersion`・`annotations` を持たない。応答版は `dispatch` の文字列リテラルが持つ。
- 実プロセス比較が固定されているのは `validate` の1操作だけ（`validate_parity_test.go`）。`catalog`・`inspect`・`trace` は `operations` の戻り値との比較にとどまる。

新しい欠陥は検出しなかったため起票はしない。広告の突き合わせと4操作の実プロセス比較は、それぞれ #3419（対応表）と #3420 CAPI-2a が既に持つ範囲である。

## 13. 第7ラウンド（2026-11-09）: discoveryの突き合わせ試験を置く

第6ラウンドで「CLI discovery と MCP `tools/list` を比べる試験が無い」と記録した。ADR-0048 §2.6 の「生成または一致試験でずれを検出する」に沿って、最小の回帰試験を追加した。生出力は[第7ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND7_CROSS_SURFACE_2026-11-09/README.md)にある。

- `discovery_parity_test.go` が、実CLIをビルドして `commands` を実行し、4 tool の対応コマンドの存在・`effect` が `read-only`・`writeFlags` が無い・`schemaVersion`・`jsonContract` の5点を突き合わせる。対応付けは文字列変換ではなく表で宣言する。
- 現行 `main` では PASS した。加えて、故意のずれ6種（adapter 側3・CLI 側3）を入れると6件すべて FAIL し、戻すと PASS に戻ることを確認した。両面のどちらを変えても検出できる。
- 突き合わせるのは discovery の一致だけである。`catalog` の namespace 失敗と `validate` の byte 一致は既存の試験が持ち、`inspect`・`trace` の実プロセス比較は #3420 CAPI-2a が持つ。

## 14. 第8ラウンド（2026-11-09）: 空文字列の引数と既定値

ADR-0048 §2.4 の「既定値の適用は共通契約で定める」を、CLI と MCP で確かめた。生出力は[第8ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND8_EMPTY_ARGUMENT_2026-11-09/README.md)にある。

- `direction` と `mode` の値集合は、CLI の `argumentValues` と MCP の `inputSchema.enum` が同じ値を持つ。省略時の既定も両方 `direction=both`・`mode=summary` で一致した。
- 一方、**空文字列を渡すと CLI だけが「未指定」として扱い、既定とは別の値へ落ちた**。`-mode ""` は結果が `mode=full`（rows=4）になり、省略時の `summary`（rows=0）と異なる。MCP は `-32602` で拒否する。
- `-expected-fingerprint ""` は照合を省略するため、呼出し元が「空なら照合しない」つもりでいると、CLI では別Workspaceの結果を継続として使う経路が開く。
- `trace` の3引数（`-direction`・`-mode`・`-expected-fingerprint`）を、フラグを書いたうえで値が空なら `INVALID_ARGUMENT`（exit 2）で拒否するよう修正した。省略時は従来どおり既定を使う。回帰試験は `cmd/tei/cli_trace_empty_argument_test.go`。
- `model catalog -namespace ""` と `inspect -subject-kind ""` は変更していない。前者は「絞り込み無し」の意味があり、後者は「拒否する」点で MCP と同じで符号だけが違う（#3420）。

起票は [#3605](https://github.com/hat47x/tei-platform/issues/3605)。

## 15. 第9ラウンド（2026-11-09）: 空文字列の修正を統合後に再実測

第8ラウンドの修正が `main`（`0067d6a1e`、PR #3607）に統合された後、CLI と MCP の両方から再測定した。生出力は[第9ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND9_EMPTY_ARGUMENT_REMEASURE_2026-11-09/README.md)にある。

- `-mode ""`・`-direction ""`・`-expected-fingerprint ""` は、CLI が `INVALID_ARGUMENT`（exit 2）、MCP が `-32602` で、両方とも拒否する。
- 3引数を省略すると `ok:true`・`mode=summary`。`-mode full` は `mode=full`。省略時の既定は保たれている。
- 残る差は符号だけ（`INVALID_ARGUMENT` と `-32602`）。失敗分類の統一は #3420 の未決で、本ラウンドでは変更していない。

#3605 は解消として扱える。Issue を閉じ、残る分類統一は #3420 に残す。

## 16. 第10ラウンド（2026-11-09）: inspect・trace の実プロセス同値と失敗分類

#3420 CAPI-2a の範囲を実測した。生出力は[第10ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND10_CAPI2A_PARITY_2026-11-09/README.md)にある。

- 8ケースのうち7ケースで、CLI と MCP の成否と `error.code` が一致した（`SUBJECT_NOT_FOUND`・`UNSUPPORTED_SUBJECT_KIND`・`STALE_SNAPSHOT`・`INVALID_ARGUMENT`・`NAMESPACE_NOT_FOUND`）。
- `inspect -workspace /no/such/dir` は MCP では tool 引数で表現できない（起動時に固定する設計）。起動時の `-workspace` が無い／manifest が壊れている場合、MCP は `WORKSPACE_LOAD_FAILED` を返し、CLI と同じ符号だった。差は「いつ分かるか」だけ。
- **必須引数の欠落だけが層の違い**である。CLI は `INVALID_SUBJECT`（exit 2）、MCP は `-32602`。両方拒否するが文字列の符号が揃わない。ADR-0048 §2.3「失敗」の未決。
- `inspect_trace_parity_test.go` を追加し、成功2件・失敗5件の値と符号、成功時の `result` の byte 一致を実プロセスで固定した。符号をわざと変える2種の変異はどちらも検出した。

## 17. 第11・12ラウンド（2026-11-09）: traceの継続トークン

第10ラウンドで「`trace` のページ継続は未検査」と記録した。T06 を実測し、**同じ名前で値の違う `snapshotFingerprint` が2つあり、継続に使えるのは外側だけ**であることを見つけた。生出力は[第11ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND11_TRACE_CONTINUATION_2026-11-09/README.md)と[第12ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND12_CONTINUATION_MAIN_2026-11-09/README.md)にある。

- ページング自体は動く。`-mode full` は全4行、`-limit 2` は2行と `nextOffset: 2`、`-offset 2 -limit 2` は残り2行。`-offset 99` は0行で成功。MCP の `tei_trace` も `limit` を受け同じ結果。
- 同じ入力なら `result.snapshotFingerprint` は安定し、ページを変えても変わらない。
- 修正前は `result.snapshotFingerprint`（作業領域全体）と `result.trace.snapshotFingerprint`（グラフ・規則・対象・方向のみ）が同名で併存し、`nextOffset` は内側にあった。**`nextOffset` の隣の値をそのまま `-expected-fingerprint` に渡すと `STALE_SNAPSHOT` になった。**
- `main` の `e75d599ba`（PR #3633）が同じ問題を解消した。内側を **`graphFingerprint`** に改名して旧名を別名として残し、`result.trace.continuation`（`fingerprint` と `field`）を `nextOffset` の隣に置く。
- 統合後の実測（`main` `5d5d5de6b`）では、`continuation.fingerprint` で継続すると成功し、`graphFingerprint` を渡すと `STALE_SNAPSHOT`。応答の名前と `field` だけで、どちらを使うか分かる。
- 私が書いた重複実装（平坦な `continuationFingerprint`）は採用しなかった。`main` の形が上位互換である。
- `-scope`・`-direction` を変えた場合の拒否、MCP が `continuation` を返すかは未確認。

## 18. 第13ラウンド（2026-11-09）: 設計変更の影響範囲をCLIだけで答えられるか

ここまでは「CLIの応答が正しいか」を測った。このラウンドは**実際の設計作業の問い**を試した。題材は実際の改名（`snapshotFingerprint` → `graphFingerprint`、PR #3633）。生出力は[第13ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND13_DESIGN_IMPACT_2026-11-09/README.md)にある。

| 問い | 結果 |
| --- | --- |
| モデル要素の定義箇所は分かるか | **分かる**。`inspect semantic-data:sales.order` が `semantic/sales/order.tei` を返す |
| リポジトリ全体をWorkspaceにできるか | **できない**。ルートに `tei.workspace.yaml` が無く `-workspace .` は `WORKSPACE_LOAD_FAILED` |
| 設計文書だけをWorkspaceにできるか | **索引には載る**。`source-index` は `artifacts: 3, references: 0` で成功 |
| 設計文書の中の名前を問えるか | **問えない**。`inspect … -subject-id snapshotFingerprint` は `SUBJECT_NOT_FOUND` |
| 改名の影響範囲を挙げられるか | **挙げられない**。どの文書のどの行がその名前を含むかは `grep` が要る |

読み取れること:

- 意味モデル（`.tei` の宣言）については、定義箇所も参照関係もCLIで答えられる。**モデルに対する設計作業はCLIで完結できる。**
- 設計文書（Markdown）については、CLIはファイルとして管理するところまでで、**中の名前を対象にした問いには答えられない**。D-9 の具体的な帰結である。
- リポジトリ全体を1つのWorkspaceにする経路が無いため、横断の問いは入力を作れない。

新しいIssueは作らない。#2619・#2621 の設計範囲であり、D-9 として第1ラウンドから記録している。この実測を D-9 の具体的な証拠として残す。
- 一方、**宣言していない引数はMCPだけが無視して成功する**。CLIは `INVALID_ARGUMENT` で拒否する。入力契約の共通化は未達で、#3467（起票）として扱う。
- 必須欠落の符号はCLIが `INVALID_SUBJECT`、MCPが `-32602` で、意味層とプロトコル層の対応は #3420 の未決である。
- `commands` の19件に操作ID・入口の欄は無く、入口ごとの対応はIssueと文書でしか辿れない。#3419 の対応表で扱う。

## 19. 第14ラウンド（2026-11-09）: knowledge-* 連鎖の端から端まで

第13ラウンドで「モデルに対する設計作業はCLIで完結できる」と分かった。その設計作業でAIが実際に使う `knowledge-*` の連鎖を、1回通した。生出力は[第14ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND14_KNOWLEDGE_CHAIN_2026-11-09/README.md)にある。

| 手順 | 結果 |
| --- | --- |
| `knowledge-context` | `mode=context`・`dataDesign` あり・`authorityLimits` 4件 |
| `knowledge-review-plan` | `mode=plan`・`reviewSequence`・`stopRules` |
| `knowledge-review-handoff` | `mode=handoff`・`qualification` を含む |
| `knowledge-context -disclose source` | `mode=disclosure` |
| `knowledge-qualify` | `status=applicable`・`evidence` 1件 |
| `knowledge-review-disclose` | `status=complete`・4セクション |

- **戻れる**: 古い指紋は `fingerprint mismatch: expected=… current=…` で拒否（exit 1）。別subjectの handoff は `subject mismatch: handoff=… context=…` で拒否（exit 2）。存在しない handoff も exit 2。
- **摩擦は1点**: 要件ファイルの形がCLIの応答から辿れない。`commands` は `-qualification-requirements` の存在だけを広告し、`-h` は "knowledge qualification requirements JSON" としか出さない。私は最初 `{"subject":…}` と書いて `json: unknown field` で止まった。最小形は `{"expectedKind":"semantic-data"}` で、文書（#3186 で追加済み）を読めば分かる。

新しいIssueは作らない。形の例は #3186（Closed）、`commands` にファイル中身のスキーマを載せるかは #3419 の範囲である。

## 20. 第15ラウンド（2026-11-09）: trace -mode full の rows が0件のとき消える

第14ラウンドの連鎖の手前で、`trace` の応答が関係0件のWorkspaceでも機械的に読めるかを確かめた。生出力は[第15ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND15_TRACE_MODE_ROWS_2026-11-09/README.md)にある。

- 関係0件のWorkspaceで `-mode full` を実行すると、`result.trace` に `rows` が**存在しなかった**。`Rows` の `json:"rows,omitempty"` タグの `omitempty` が、nil だけでなく長さ0のスライスも落とすためである。
- `mode=summary` が `rows` を出さないのは契約どおり（全体を数えるだけで行を確保しない）。問題は **`mode=full` の「0件」と「欄が無い」が同じ表現**になることだった。
- 起票は [#3693](https://github.com/hat47x/tei-platform/issues/3693)。#3456（`source-index` の `references`）と同じ種類の欠陥である。
- `main` の `3799997a0` が解消した。`TraceResult` に `MarshalJSON` を足し、`mode=full` では常に `rows` を書き、フィールドを宣言順に明示して文書順序を変えない。
- 私は「`mode=full` かつ0件のときだけ上書きする」実装を書いて動作を確認したが、`main` の形が上位互換なのでコードは持ち込まず、実測記録だけを残した。
- このラウンドで踏んだ落とし穴: 最初に `mode=full` のすべてを上書きする実装にしたところ、埋め込み構造体が `rows` を最後に回し、既存の full 応答のバイト列が変わって `reference-cli-trace-benchmark` の golden が2件 FAIL した。**JSON表現を触る変更は、既存応答のバイト列を変えないことを golden で確かめる必要がある。**

## 21. 第16ラウンド（2026-11-09）: 継続トークンの失効条件

第11ラウンドで「どの値が継続トークンか」を扱い、「`-scope`・`-direction` を変えた場合の拒否」と「MCP が `continuation` を返すか」を未確認として残した。生出力は[第16ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND16_CONTINUATION_EXPIRY_2026-11-09/README.md)にある。

- **Source の変更は種類を問わず失効させる**。コメント1行の追加でも属性の追加でも `STALE_SNAPSHOT`（exit 1）。Source の生バイトが `sourceIndexFingerprint` に入るためで、`plan/design/CLI_FIRST_INTERFACE_DESIGN.md` §159 の「非意味的な原文変更は拒否する」と一致した。
- **選択子の変更も失効させる**。`-scope` を変えると `STALE_SNAPSHOT`、`-direction` を変えると同じく `STALE_SNAPSHOT`。`-mode full` → `summary` は `summary` が `-offset` を受け付けないため `INVALID_ARGUMENT`（exit 2、引数エラーとして正しい）。
- **`summary` ⇄ `full` は相互に許容される**。`summary` のトークンで `mode=full -offset 0 -limit 2` が成功し、逆向きも成功。`mode`・`offset`・`limit` はトークンの対象外である。
- **MCP も同じ**。`continuation.fingerprint` で継続でき、`graphFingerprint` や別 `scope` は `STALE_SNAPSHOT`。CLI と MCP で可否が一致した。

新しい欠陥は検出しなかった。すべて文書に書かれた挙動と一致している。CAPI-2c のうち「参照実装の継続トークン」はここで一区切りとする（World/Source 改版の別故障モードは #3420 に残る）。

## 22. 第17ラウンド（2026-11-09）: 必須引数の不足が3つの符号に分かれていた

第10ラウンドで「必須引数の欠落だけが層の違い」と記録し、#3420 の未決として残していた。全コマンドを並べたところ、**1つの条件が3つの符号**で返っていた。生出力は[第17ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND17_FAILURE_CLASSIFICATION_2026-11-09/README.md)にある。

| 呼出し | 修正前 | 修正後 |
| --- | --- | --- |
| `inspect`（`-workspace`／`-subject-kind`／`-subject-id` のいずれか欠け） | **`INVALID_SUBJECT`** | `INVALID_ARGUMENT` |
| `trace`・`model catalog`・`source-index`・`requirements` | `INVALID_ARGUMENT` | 変わらず |
| `knowledge-context`・`knowledge-qualify`・`knowledge-review-*` | **`KNOWLEDGE_BUILD_FAILED`** | `INVALID_ARGUMENT` |
| `validate`（`-json` 無し） | **JSON を出さない** | 変わらず（text 出力の契約） |

逆向きの不揃いもあった。同じ `inspect` で、`-subject-id` を**書かない**と `INVALID_SUBJECT`、**空文字列**を渡すと `INVALID_ARGUMENT`（#3605）だった。

修正:

- `cmd/tei/cli_argument_error.go` に `argumentError` 型を追加。「呼出し元が引数を間違えた」ことを表す。
- `inspect` の引数ガードを `INVALID_ARGUMENT` へ。`INVALID_SUBJECT` は「subject の値が使えない」（空白のみ・前後空白）に残す。
- `knowledge-*` の「`-flag` が足りない」を `newArgumentError` に変え、`knowledgeFailure` が `INVALID_ARGUMENT` へ写す。知識の構築失敗は `KNOWLEDGE_BUILD_FAILED` のまま。

変えないと決めたもの（試験で固定）: 空白のみの subject は `INVALID_SUBJECT`、実在しない subject は exit 1 の `SUBJECT_NOT_FOUND`、要件ファイルが不正なら `KNOWLEDGE_BUILD_FAILED`、`validate` の text 出力は非JSONのまま。

試験は `cmd/tei/cli_failure_classification_test.go` の3件。既存の `TestCLIProcessJSONContract`・`TestResultEnvelopeCommandsJSONContract` を新しい符号へ更新した。`go test ./cmd/tei -count=1` は ok（121.1s）。

**CLI と MCP の符号の統一（`INVALID_ARGUMENT` ⇔ `-32602`）は未着手**で #3420 に残る。今回揃えたのは CLI 内部の一貫性である。起票は [#3733](https://github.com/hat47x/tei-platform/issues/3733)。

## 23. 第18ラウンド（2026-11-09）: 実装を直したあと文書が古い符号を主張していた

第17ラウンドで「必須引数の不足」を `INVALID_ARGUMENT` に統一した。その変更が同じ規則を説明する他の文書と一致しているかを確かめた。生出力は[第18ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND18_DOC_DRIFT_2026-11-09/README.md)にある。

- `main` `022c85bd9` の挙動は第17ラウンドの意図どおりだった（11コマンドが `INVALID_ARGUMENT`、`INVALID_SUBJECT`・`SUBJECT_NOT_FOUND`・`KNOWLEDGE_BUILD_FAILED` は据え置き）。
- `tools/reference-mcp-readonly-adapter/README.md` の検証一覧が「必須引数の欠落は CLI では `INVALID_SUBJECT`」と**古い符号を主張していた**。コードと試験は `INVALID_ARGUMENT` に変わっており、第17ラウンドで私は試験だけを更新して散文を見落とした。
- 文書を実装に合わせ、`readme_claim_test.go` で**散文の主張を実行可能な試験として固定**した。README を旧記述へ戻す変異と、主張を消す変異のどちらも検出する。
- 並行して出ていた [PR #3755](https://github.com/hat47x/tei-platform/pull/3755) は同じ #3733 の別実装だった。私の #3749 が先に統合されて `CONFLICTING` になり、すでに別の作業者が閉じていた。

**学び**: 符号や契約を変えるときは、コード・試験・同じ規則を説明する散文を同じ変更で直す。散文は試験で固定しないと、次の変更で同じようにずれる。

## 24. 第19ラウンド（2026-11-09）: 同じ壊れた plugin-lock が2コマンドで別の終了コードになっていた

18ラウンドで一度も触っていなかったコマンドを叩いた。特に `-plugin-lock` はファイルを入力に取るため、ロックを作って別コマンドへ渡す往復があり、分類が揃っていない危険が高い。生出力は[第19ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND19_PLUGIN_LOCK_CLASSIFICATION_2026-11-09/README.md)にある。

**正常系は問題なかった。** `plugin-lock` → `plugin-resolve`／`plugin-runtime-plan` の往復は成功し、`plugin-runtime-plan` は `pluginLockFingerprint` と `runtimePolicyFingerprint` を返す。`world` が包まない素の JSON を返すことも discovery の `jsonContract: raw` と一致していた。

**壊れたロックの分類が食い違った。** 7種の壊し方のうち5種で、`plugin-resolve` は exit 2 `PLUGIN_RESOLVE_FAILED`、`plugin-runtime-plan` は exit 1 `PLUGIN_RUNTIME_PLAN_FAILED` になった。終了コード1は「意味の所見」、2は「入力の誤り」の区別だが、壊れたロックはどちらも入力の誤りである。

原因は、`ResolveEnvironmentWithLock` が「指紋不一致」「version 不在」「プラグイン欠落」「別 environment」を素の `fmt.Errorf` で返し、`plugin-runtime-plan` の `plugin.IsInvalidLock` に該当しなかったこと。

修正: `LockMismatchError` / `IsLockMismatch` を追加した。**`InvalidLockError` とは分けた** — 既存の `invalid_lock_test.go` が #3184 として「バイトが読めない」と「整形済みだが一致しない」を区別しており、その区別は正しい（前者は何が違うか言えない、後者は言える）。分けたうえで、両方とも呼出し元が直せるのはロックファイルだけなので、CLI では同じ `INVALID_PLUGIN_LOCK`（exit 2）に写した。7種すべてが両コマンドで揃った。

試験は `cmd/tei/cli_plugin_lock_test.go`（6ケース）。変異2種（`IsLockMismatch` を見ない／素の `fmt.Errorf` に戻す）を検出する。

**既存試験を1件、意図的に更新した。** `plugin_lock_scope_cli_test.go` は「別 environment を指すロックは resolve で `PLUGIN_RESOLVE_FAILED`・plan で exit 1」を固定していた。これは変更前の挙動そのものであり、統一後の `INVALID_PLUGIN_LOCK`・exit 2 へ直した。**破壊的な挙動変更である点を明記しておく。**

契約文書 `plan/design/CLI_JSON_OUTPUT_CONTRACT_2026-10.md` の両行はもともと `INVALID_PLUGIN_LOCK` を挙げていた。今回の変更で**実装が契約に追いついた**。起票は [#3765](https://github.com/hat47x/tei-platform/issues/3765)。

## 25. 第20ラウンド（2026-11-09）: view の -input が -command 無しで黙って捨てられていた

19ラウンドで `plugin-*` を叩いて欠陥が出たため、まだ測っていない `view` を対象にした。`view` は `writeFlags` に `-operation-log`・`-lifecycle-state` を持ち、実行経路がある唯一の「読み取り」コマンドである。生出力は[第20ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND20_VIEW_INPUT_2026-11-09/README.md)にある。

**正常系**は問題なかった。`view -application order_entry -format json` は `patterns`・`semanticFingerprint`・`worldFingerprint`・`sourceIndexFingerprint`・`workspace` を返し、引数無しは JSON で `INVALID_ARGUMENT`（exit 2）、実在しない application は exit 1 `VIEW_FAILED`。

**見つけた欠陥**: `-input` を `-command` 無しで渡すと、値が捨てられて成功する。存在しないファイル名でも、書式不正（`no-equals-sign`）でも exit 0 `ok:true`。`-input` を付けた応答と付けない応答はバイト単位で同一だった。同じ group の `-period-state`・`-allow-capability`・`-operation-log`・`-lifecycle-*` は `-execute` 無しで正しく拒否されるのに、`-input` だけ条件が抜けていた。

**起票時の診断を誤った。** 最初に「`-input` は `-execute` と組」と書いたが、実際は `-command` と組で、`-command` があれば `-execute` 無しでも plan に反映される（`product_slice_test.go:169` が固定）。私の誤った修正（`!*execute`）が既存試験 `TestProductSliceLoadsValidatesAndViews` を落として発覚した。**既存試験が正しく、私の診断が誤っていた。** issue #3778 に訂正を追記し、条件を `*commandID == ""` に直した。

修正: `cmd/tei/cli_view.go` の引数ガードに `(len(inputPairs) > 0 && *commandID == "")` を追加。

試験は `cmd/tei/cli_view_input_test.go` の2件。**両方向**の変異（条件を削る／過剰に厳しくする）を検出する。過剰に厳しくする方向を入れたのは、今回の誤診がまさにその方向だったため。

**学び**: 依存関係を推測で書くと、既存の正しい用法を壊す。discovery の `"-input and -execute with -command"` は「両方とも `-command` に係る」と読むのが正しく、私は「`-input` と `-execute` が互いに係る」と読み違えた。`-execute` の実行経路そのものは未測定で、次に測るべき対象である。起票は [#3778](https://github.com/hat47x/tei-platform/issues/3778)。

## 26. 第21ラウンド（2026-11-09）: view -execute の失敗符号が潰れていた

20ラウンドまで `view` は引数までしか測っていなかった。`view -execute` は `writeFlags` を持つ唯一の実行経路で、最後の大きな未測定領域である。書込み先を `mktemp -d` の下に限定し、workspace は変えずに測った。生出力は[第21ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND21_VIEW_EXECUTE_2026-11-09/README.md)にある。

**正常系と安全側は問題なかった。** 実行は `committed:true`・`persisted:false`・`durability:none`・`authorizationMode:"reference-allow-all"` を返し、`docs/status.md` の記載と一致。`-operation-log` が書けない場合は exit 1 `OPERATION_LOG_WRITE_FAILED`。失敗した実行は記録を書かず、lifecycle に拒否された実行は既存の記録を上書きしない。lifecycle は同じ command_id の再送を owner/generation が同じでも違っても拒否し、別 command_id は許可する（`internal/command/lifecycle.go` の設計どおり）。

**見つけた欠陥**: 実行失敗の符号が潰れていた。

| 呼出し | 修正前 | 修正後 |
| --- | --- | --- |
| 必須入力なし | `COMMAND_FAILED`（message に `INVALID_COMMAND_INPUT: ` が埋まる） | `INVALID_COMMAND_INPUT` |
| 認可拒否 | `COMMAND_FAILED`（message に `COMMAND_UNAUTHORIZED: `） | `COMMAND_UNAUTHORIZED` |

原因は `cmd/tei/cli_view.go` の `commandFailure` が `ExecutionError.Code` のうち3つだけを手書きで写し、`INVALID_COMMAND_INPUT` と `COMMAND_UNAUTHORIZED` が既定の `COMMAND_FAILED` に落ちていたこと。`ExecutionError.Error()` が `Code + ": " + Message` を返すため符号が message に混ざっていた。

修正: `reportedExecutionCodes` の集合に既知の7符号を入れ、そこに在るものはそのまま `error.code` に出す。未知の符号は従来どおり `COMMAND_FAILED` に残す（契約の受け皿を消さない）。

試験は `cmd/tei/cli_view_execute_code_test.go` の2件。変異（集合を元の3符号に戻す）で両方の失敗が検出されることを確認した。

**未解決の論点**: `-operation-log` は同じパスへ2回書くと後の実行だけが残る（`os.WriteFile` の truncate）。文書に明示が無く、追記か上書きかは設計判断が要るため、今回は変更していない点を記録した。起票は [#3797](https://github.com/hat47x/tei-platform/issues/3797)。

## 27. 第22ラウンド（2026-11-09）: 手書きの符号許可リストが21中15を潰していた

21ラウンドで「符号の対応表が手書きで、抜けが出る」型の欠陥が3回続いた（#3733・#3765・#3797）。個別に直すのではなく、**列挙そのものを無くせるか**を確かめた。生出力は[第22ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND22_CODE_COVERAGE_2026-11-09/README.md)にある。

`internal/command` が `ExecutionError` として返す符号は **21**。`cmd/tei/cli_view.go` の `reportedExecutionCodes` は **7**（うち1つは CLI 合成）。**15個が既定の `COMMAND_FAILED` に潰れていた** — `COMMAND_DATASOURCE_READ_ONLY`（読み取り専用宣言への違反）・`COMMAND_COMMIT_FAILED`（適用有無が不明）・`COMMAND_PERIOD_CLOSED`（業務上の拒否）など、呼出し元の行動が変わるものが含まれる。

さらに**入口ごとに食い違っていた**。`internal/studiohost` は1符号だけを特別扱いし、残り全部を `COMMAND_FAILED` にしていた。CLI は7、Studio は1。

修正: 列挙を廃止し、`internal/command` に `ReportedFailure` を1つ置いて両入口がそれを呼ぶ。`ExecutionError` はすでに `Code` を持っているので、選別しない。ただし `EffectMayBeUnknown`（適用された可能性がある）を**先に**見る — 盲目的な再試行が二重適用になるため、具体的な符号より優先する必要がある（最初この順序を逆にして既存試験が落ちた）。

再発防止: `reported_failure_test.go` が**パッケージのソースを AST で走査**し、`Code:` に現れる全リテラルが規則を通ることを確かめる。試験側に符号一覧を書けば本番の一覧と同じようにずれるため、ソースを真とした。変異2種で確認 — 新しい符号を追加すると規則が汎用なので**通る**（これが狙い）、許可リストを再導入すると**落ちる**。

**意図的な挙動変更を含む。** 3つの既存試験の期待値を書き換えた。うち1つ（`studiohost` の認可拒否）は、符号が使えなかったための回避策（`strings.Contains(message, "COMMAND_UNAUTHORIZED")`）であり、今回それが必要なくなった。

残る作業: 現れうる符号の集合を契約文書へ反映する（#3419）。起票は [#3807](https://github.com/hat47x/tei-platform/issues/3807)。

## 28. 第23ラウンド（2026-11-09）: 契約文書の符号一覧が実装より狭くなっていた

第22ラウンドで `view -execute` が `internal/command` の21符号をそのまま返すようにした。その結果、契約文書 `plan/design/CLI_JSON_OUTPUT_CONTRACT_2026-10.md` の `view` 行が実装より狭くなっていないかを確かめた。生出力は[第23ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND23_CONTRACT_DOC_2026-11-09/README.md)にある。

- 契約表が名指す符号27個は**すべて Go のソースに存在した**。存在しない符号を書いている箇所は0で、文書は「嘘」ではなく**不足**だった。
- `view` 行は11符号（うち `COMMAND_*` は4つ）しか挙げていなかったが、実装は `-execute` の失敗で21符号を返しうる。`COMMAND_UNAUTHORIZED`・`COMMAND_DATASOURCE_READ_ONLY`・`COMMAND_COMMIT_FAILED` などが文書に無かった。
- **discovery は符号を広告しない。** `tei commands` の entry は15欄を持つが失敗符号の欄が無い。呼出し元が「このコマンドはどんな符号を返しうるか」を知る方法は契約文書しかなく、文書が狭いことはそのまま発見可能性の欠落になる。

修正: `view` 行に21符号を列挙し、`COMMAND_EFFECT_UNKNOWN` が優先することを明記した。`COMMAND_FAILED` は「符号を持たない失敗」の受け皿として残る。

再発防止: `cmd/tei/cli_contract_codes_test.go` が `internal/command` のソースを AST で走査して符号を集め、契約表の `view` 行がそれを全部名指すことを確かめる。変異2種で確認 — 文書を前ラウンド前へ戻すと落ち、実装に新しい符号を足して文書を放置しても落ちる。**両方向**を固定した。

第22ラウンドの `ReportedFailure`（入口が報告するか）と対になる形で、対象を「文書が名指すか」へ広げた。次に `internal/command` が符号を増やしたら、文書を直さないと CI が落ちる。

**残り**: `view` 以外の行の機械検査、discovery への符号広告（`commands` の欄を増やす契約変更）。どちらも #3419 に残る。

## 29. 第24ラウンド（2026-11-09）: tool の引数一覧が schema と独立に手書きされていた

第22・23ラウンドで「手書きの一覧から漏れる」型を2つ閉じた。MCP Adapter も同じ構造を持つはずなので、tool を扱う表がいくつ独立しているかを測った。生出力は[第24ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND24_TOOL_TABLE_DRIFT_2026-11-09/README.md)にある。

**表は4つあった**: `toolList()`（広告）、`toolArguments`（受理する引数）、`dispatch` の `switch`（呼出し先）、`cliCommandNameForTool`（CLI 名との対応、試験側）。このうち `toolArguments` は `InputSchema.properties` と**同じ事実を2回書いたもの**だった。

変異で検出力を測ると、**危険な方向だけが無検査**だった。

| 変異 | 試験 |
| --- | --- |
| `toolList()` に tool を追加 | FAIL（3試験） |
| `toolArguments` に schema が宣言しない引数を追加 | **PASS（検出できず）** |
| `toolArguments` から宣言済みの引数を削除 | FAIL（2試験） |

緩む方向は、各 tool の `rejectUnknownArguments` が偶然止めていた。同じ宣言を2箇所に書いているため、`rejectUnknownArguments` を書き忘れた tool を足すと素通りする。`toolArguments` と `InputSchema.properties` を比べる試験は存在しなかった。

修正: `toolArguments` を削除し、`InputSchema` から導出する（`declaredArguments`・`toolByName`）。**試験で一致を確かめるのではなく、事実を1箇所にした。** 前2ラウンドは「漏れたら落とす」だったが、今回は一覧そのものを消したので漏れようがない。

試験は `tool_arguments_test.go` の4件。変異2種（導出が property を落とす／`required` に未宣言の名前を足す）を検出する。

**残り**: `dispatch` の `switch` と `toolList()` の一致。広告に無い tool を dispatch に書くことは可能だが、`checkArguments` が `toolByName` で先に弾くため実害は無い。必要になれば同じ手法で閉じられる。起票は [#3823](https://github.com/hat47x/tei-platform/issues/3823)。

## 30. 第25ラウンド（2026-11-09）: TEI の標準意味型は TEI で宣言されていない

ここまでの24ラウンドは CLI の整合性を測ってきた。このラウンドは目的そのもの、**TEI 自身が TEI で書かれているか**を測った。生出力は[第25ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND25_STDLIB_SELF_DESCRIPTION_2026-11-09/README.md)にある。

- `std.*` の12型は `model catalog` に現れるが、`prelude: true` と `sourceStatus: "not-indexed"` が同時に付く。**モデルの要素だが、どの Source にも由来しない。**
- `inspect -subject-kind semantic-type -subject-id std.text` は解決できるが `artifacts: []`・`references: []`。対照の `sales.order` は `sourceStatus: "indexed"` で `semantic/sales/order.tei` とその指紋が入る。
- `source-index` は参照106件・artifact 54件を返すが、`std.*` への参照は**0件**。
- 定義は `internal/stdlib/semantic.go` の Go リテラル。コメント自身が「later package/registry implementation can replace this prelude」と過渡期を認めている。
- `std` は**使えるが宣言できない**。`namespace std` を書くと `SEMANTIC_TYPE_ALIAS_CYCLE`・`DUPLICATE_SEMANTIC_ELEMENT`・`DUPLICATE_SEMANTIC_NAMESPACE` で拒否される。別名前空間からの `type text_t from std.text` は通る。`std` を特別扱いする箇所は `internal/validation/validation.go:1185` と `internal/operations/declaration_counts.go:38`。

**合格条件が CLI だけで測れることを実演した。** 宣言可能な名前空間で `type text_t from std.text` を書くと、同じ catalog に `mine.text_t`（`indexed`）と `std.text`（`not-indexed`）が並び、`inspect`・`source-index` も `mine.text_t` は根拠付きで返す。`std` を `.tei` にすればこの差がそのまま埋まる。新しい API も検証機構も要らない。

**これは欠陥の起票ではない。** 欠陥ではないが目的の達成を妨げている構造であり、判断は [#2620](https://github.com/hat47x/tei-platform/issues/2620) の Owner に属する。材料（測定値・合格条件・移行時の注意点）を提示するに留め、勝手に `std` を `.tei` 化していない。

**注意**: 移行時は `std` の特別扱い2箇所と `internal/stdlib/semantic.go` の二重定義を同時に扱う必要がある。Go の定義を残したまま `.tei` を併置して一致を確かめる段階を挟むのが安全。

## 31. 第26ラウンド（2026-11-09）: std への宣言が「prelude」と誤判定され、件数からも消えていた

25ラウンドで「`std` を `.tei` にできるか」の移行コストを見積もる過程で、`std` の特別扱いが**名前の前方一致**に依存していることに気づいた。利用者が `std` に型を足した場合を測った。生出力は[第26ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND26_PRELUDE_IDENTITY_2026-11-09/README.md)にある。

`namespace std { type email from std.text }` と書くと、**3つの矛盾**が同時に出た。

| 症状 | 内容 |
| --- | --- |
| 名前空間の重複 | `DUPLICATE_SEMANTIC_NAMESPACE`（error）。catalog も `std` を2件、両方 `ambiguous:true`・`sourceStatus:indexed` で返す |
| prelude の誤判定 | `std.email`（利用者の宣言）に `prelude:true` が付く。契約文書 §59 の「Workspaceが宣言したものではない」に反する |
| 件数の欠落 | `declarationCounts.total` が 0、`WORKSPACE_DECLARES_NO_MODEL_ELEMENTS` 警告。一方 `inspect` は `std.email` を `indexed` で解決する |

原因は4箇所のうち3箇所が名前の前方一致で `std` を特別扱いしていたこと（`preludeSubject`・`declaration_counts.go:38`・`validation.go:1185`）。1箇所（`standardSemanticSubject`）だけが正しく同一性で照合していた。加えて loader が prelude の `std` と利用者の `std` を**別の名前空間として追加**していた。

修正: 「prelude である」ことを名前ではなく同一性で判定する。`isPreludeType(id)`・`hasNonPreludeDeclaration(ns)` を追加し、loader の `byNamespace` を既存の `World.Semantics` から初期化して利用者の宣言を prelude の名前空間へ合流させた。

修正後: `validate` は `ok:true`、`std` は1件で `not-indexed`、`std.email` は `prelude` 無しで `indexed`、`declarationCounts` は `types:1`・`total:2`。prelude のみの Workspace は `total:0` のまま、`reference/workspace` は `total:71` で不変。

試験は `prelude_identity_test.go` の3件。変異2種（前方一致に戻す／名前空間ごと飛ばす）を検出する。

**残り**: `std` 自体を `.tei` にする判断（#2620）は未着手。ただし今回の修正で「prelude と利用者の宣言を同一性で区別する」規則が3箇所に入ったので、`std` を `.tei` にしても同じ規則で扱える。起票は [#3851](https://github.com/hat47x/tei-platform/issues/3851)。

## 32. 第27ラウンド（2026-11-09）: std を .tei にできない理由（primitive 構文と指紋）

25ラウンドの提案「`std` を同梱の `.tei` として宣言する」が、いまの実装で可能かを確かめた。生出力は[第27ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND27_STD_PRIMITIVE_GAP_2026-11-09/README.md)にある。

**結論: そのままでは実装できない。** 独立した2つの理由がある。

1. **`worldFingerprint` が prelude を含む。** loader の prelude 注入行を外して測ると `worldFingerprint` が `sha256:c92de09f…` から `sha256:8d7e160b…` へ変わり、`sourceIndexFingerprint` は同一だった。`worldFingerprint` は正規化 World の JSON 全体のダイジェストなので、prelude を変えると**全 Workspace の指紋と保存済み継続トークンが変わる**。
2. **`.tei` に primitive を書く構文が無い。** `type` 宣言は必ず `from` を要求する（`internal/syntax/parser.go:1902`）。`type text` 単独は `PARSE_ERROR | expected "from"`。したがって `.tei` の型はすべて別名で、根は Go でしか定義できない。`std.text` を `.tei` で書こうとすると `type text from std.text` になり、`SEMANTIC_TYPE_ALIAS_CYCLE` になる。

**代替案を2つ示した。**

| 案 | 内容 | 指紋への影響 |
| --- | --- | --- |
| A | `std.def` を同梱し、prelude の**由来**として `inspect` の `artifacts` に返す | 小（World の構造を変えない） |
| B | primitive 宣言の構文を追加し、prelude を `.tei` へ移す | 大（全指紋が変わる） |

A は「根拠を辿れる」という #2620 の合格条件を満たしつつ指紋を変えない。B は本質的だが primitive 構文の設計・全指紋の移行・契約文書の更新を伴う。**どちらを選ぶかは #2620 の Owner の判断**であり、私は材料を提示するに留めた。

**学び**: 「正しい方向の提案」でも、文法と指紋という2つの実装制約を確かめずに出すと実装できない。25ラウンドの提案は方向として正しかったが、前提を測っていなかった。

## 33. 第28ラウンド（2026-11-09）: Capability の outcome が参照グラフに現れていなかった

「CLI だけで設計の問いに答えられるか」を、実際の設計（認証→認可）で試した。生出力は[第28ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND28_CAPABILITY_OUTCOME_2026-11-09/README.md)にある。

`reference/workspace/semantic/security/` は `security.authenticate` の outcome `authenticated security.authenticated_subject` を、`security.authorize` が input に取る設計である。ところが:

| コマンド | 結果 |
| --- | --- |
| `inspect -subject-id security.authenticate` | `outcomes` に `authenticated` を返す |
| `trace -subject-id security.authenticate` | `input` の2本だけ。outcome は出ない |
| `trace -subject-id security.authenticated_subject -direction incoming` | `authorize` が input する1本だけ。**生産者が見えない** |

読み手には「誰も作らないデータを authorize が要求している」ように見える。私も最初そう読んだ。

原因は `internal/index/index.go` の派生処理が Capability の `input` と `output` の辺は作るのに **`outcome` の辺を作っていなかった**こと。`Capability.Outcomes` は IR にあり `inspect` も返しているのに、派生グラフに入っていなかった。

修正後は `trace` が4本（outcome 2・input 2）を返し、`authenticated_subject` の incoming が2本（`authorize - input`・`authenticate - outcome`）になって**連鎖が両側から見える**。

試験は `internal/index/capability_outcome_test.go` の2件。変異（派生を削る）で落ちることを確認した。

**互換性**: `sourceIndexFingerprint` が変わる（Index の JSON に辺が増える）。`worldFingerprint` は不変。`trace` の `snapshotFingerprint` が変わるため保存済み継続トークンは失効する。これは正しい方向（以前は辺が欠けていた）。

**残り**: outcome の名前（`authenticated` / `denied`）を辺に載せていない。現状は「どの capability がどの型を結果として生むか」だけを持ち、結末名は `inspect` の `canonical.outcomes` で見る。どちらが良いかは #3419 の判断。起票は [#3884](https://github.com/hat47x/tei-platform/issues/3884)。

## 34. 第30ラウンド（2026-11-09）: 設計の問いに CLI で答えられるかを確かめた（欠陥なし）

前ラウンドで `trace` が capability の `outcome` を辺として扱うようになった。その効果を、**実際の設計の問い**で確かめた。生出力は[第30ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND30_DESIGN_QUESTIONS_2026-11-09/README.md)にある。

| 問い | 答え |
| --- | --- |
| 認証→認可の連鎖は繋がっているか | **繋がっている**（`authenticate -outcome-> authenticated_subject` と `authorize -input-> authenticated_subject` が一致） |
| 宙に浮いた参照はあるか | **無い**（宣言70件、未宣言を指す辺0件、全 subject で `unresolved=0`） |
| 各入力に生産者が必要か | **必要ない**（入力はモデル外から来てよい。私の仮説は棄却） |
| 未解決の outcome payload を検出するか | **検出する**（`validate` が `UNKNOWN_SEMANTIC_CAPABILITY_OUTCOME_PAYLOAD`、`trace` が `projectionUnresolved`） |

**新しい欠陥は検出しなかった。** Q3 の仮説は私の誤り、Q4 は既存の仕組みが正しく報告していた。起票しない。

ただし**語の衝突**を1つ記録した。`trace` は2種類の「未解決」を持つ。

- `result.trace.unresolved` — 選択した subject に直接繋がる**行**のうち解決できなかった数
- `result.projectionUnresolved` — グラフ全体を作る過程で**捨てた辺**の一覧

`reference/workspace` は `unresolved=0` かつ `projectionUnresolved` が17件で、内側の数だけ見ると「未解決は無い」と結論してしまう。私は実際にそう読み違えた。欠陥ではない（情報は出ており `docs/getting-started.md:212` に説明もある）が、契約文書の `trace` 行に両者の違いを書く価値はある。**#3419 の範疇**として所感を残す。

**このラウンドは「測って問題が無かった」ことの記録である。** ドッグフーディングは欠陥を出すためだけではなく、道具が設計の問いに答えられることを確かめるために行っている。

## 35. 第31ラウンド（2026-11-09）: trace の行 issue のうち4つが CLI から到達不能だった

第30ラウンドで私は `trace.unresolved` を `projectionUnresolved` と2回読み違えた。それが私の不注意か道具の構造かを切り分けた。生出力は[第31ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND31_TRACE_ISSUE_REACHABILITY_2026-11-09/README.md)にある。

- `reference/workspace` の**全72 subject** で `trace -mode full` を実行し、`result.trace.unresolved` を集計すると **0件が72件**。同じ応答の `result.projectionUnresolved` は**全件で17**。
- `summary` と `full` で `total`・`unresolved`・`projectionUnresolved` は一致（`4 / 0 / 17`）。差は mode ではなく**欄の意味**。
- 行の `issue` は6種あるが、**4種は CLI 経路で起こらない**。`derivedbridge.FromIndex` が**投影した辺そのものから規則を作る**（`rule := modelref.Rule{...}; ruleSet[rule] = true`）ため、規則の欠落・空欄・重複が生じない。`invalid-target` は投影時に `projectionUnresolved` の `unmaterialized-to` へ落ちて行にならず、`ambiguous-source` は `validate` が `DUPLICATE_SEMANTIC_ELEMENT` で先に拒否する。

起票は [#3906](https://github.com/hat47x/tei-platform/issues/3906)。3案（A: 欄の改名、B: 統合、C: 契約に明記）を示し **C を推奨**とした。

実施: 契約文書 `plan/design/CLI_JSON_OUTPUT_CONTRACT_2026-10.md` に注記を追加。（1）2つの「未解決」の意味、（2）`reference/workspace` が 0 と 17 であること、（3）規則系 issue が起きない理由、（4）**未解決の判断は `projectionUnresolved` で行う**。再発防止として `cmd/tei/contract_unresolved_doc_test.go` がこれら4点を検査し、変異2種（案内文の削除・規則合成説明の削除）で落ちることを確認した。

**残り**: `invalid-source` は到達させる方法を見つけられておらず、到達不能なら5種が死んだ診断になる。応答形の変更（案 A・B）は設計判断として #3906 に残す。

## 36. 第32ラウンド（2026-11-09）: trace の行 issue は 11 種すべて CLI から観測できない

第31ラウンドで `invalid-source` を「到達させる方法が見つかっていない」として残した。これを閉じた。生出力は[第32ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND32_ISSUE_REACHABILITY_2026-11-09/README.md)にある。

`row.Issue` に入りうる値は**11種**あった。`trace.go` の switch が6種、`lookup.resolve` が5種を返し（`dest, issue := lookup.resolve(edge.To)` でそのまま `row.Issue` になる）、加えて `evidenceDiagnostics` が任意のコードを返す。

すべて CLI 経路で到達不能である理由:

| 値 | 理由 |
| --- | --- |
| `invalid-source`・`ambiguous-source` | `counts` は `graph.Nodes` から作られ、投影が `Key` を重複排除するため常に1。`addEdge` は `From` が未 materialize なら `projectionUnresolved` へ落とす |
| `invalid-edge-type`・`invalid-rule`・`conflicting-rule` | 規則を**投影した辺から合成**するため、欠落・空欄・重複が生じない |
| `invalid-target`・`missing-reference`・`wrong-kind`・`ambiguous-reference` | 投影が `To` を `unmaterialized-to` として `projectionUnresolved` へ落とし、行にならない |
| `invalid-reference` | `-scope` が引数検証で拒否される（`" bad "` → `INVALID_ARGUMENT`、`"a\nb"` → `INVALID_SUBJECT`） |
| `version-mismatch` | 投影が `Ref.Version` を設定しない |
| `evidenceDiagnostics` | 投影が `Edge.Evidence` を設定しない |

実測: `reference/workspace` の全 subject を走査して `issue` は**1件も観測されなかった**。

**結論: `Status` は常に `resolved`、`issue` は常に空。未解決の実体は `result.projectionUnresolved` にのみ現れる。** 第31ラウンドで契約文書に書いた「未解決の判断は `projectionUnresolved` で行う」は、11種すべての到達性という独立した経路で裏付けられた。

`modelref` 自体は汎用の型であり、`derivedbridge` を通さない呼出し元では `issue` が意味を持ちうる。今回の主張は**CLI 経由で観測できない**ことに限定している（コードが死んでいるという主張ではない）。起票は #3906 に追記した。

## 37. 第33ラウンド（2026-11-09）: 計画文書に本文を持たない重複見出しが残っていた

計画文書の構成を点検して、見出しの残骸を見つけた。`# ` タイトルには `TestMarkdownTitleAppearsOnce` があるが、`## ` 見出しには同等の検査が無かった。生出力は[第33ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND33_PLAN_HEADING_DUPLICATE_2026-11-09/README.md)にある。

### 見つけた残骸

1. **計画文書 514 行**: 本文を持たない旧題名の見出しが、現題名の直前に残っていた。同じ番号 `11` の節が2つあり、`§11` への参照がどちらを指すか決まらなかった。
2. **第5ラウンドの記録**: `## 6.` が2つあり、`## 7. 位置付け` が2つ目の `## 6.` より前にあった（追記時に番号を振り直していない）。
3. **他の設計文書**（`HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md`）: 連続見出しが8組。**別担当の文書なので直さず所見として #3919 に記録。**

### 追加した試験

`tools/repository-plan-check/plan_headings_test.go` の2件。対象は**ドッグフーディング文書に限定**した（他担当の文書で落とすと無関係な PR を止めるため）。

### 試験自体の欠陥を2つ踏んだ

**書いた試験が2回「静かに通る」状態になった。**

1. `filepath.Walk("..")` が**0ファイルしか辿っていなかった**。既存の試験は `filepath.Join("..", "..", "plan")` を使う。`..` では `tools/` の中しか見ていなかった。
2. 1つ目を直したとき、**2つ目の試験の walk が `..` のまま残った**（置換が1箇所にしか当たっていなかった）。

どちらも「変異を入れて落ちることを確かめる」手順で発見した。M1（本文無し見出しを戻す）と M2（§7 を重複させる）の両方が FAIL することを確認済み。**試験を書いたら、対象の欠陥を再現させて落ちることを確かめないと、静かに通る試験は検査として存在しない。**

## 38. 第34ラウンド（2026-11-09）: 桁数を綴る helper が1桁しか扱えず、検査が静かに止まっていた

第33ラウンドで「影響範囲が読めないので変更しない」と保留した `itoa` を、範囲を確認したうえで直した。生出力は[第34ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND34_ITOA_DIGITS_2026-11-09/README.md)にある。

### 呼出し元は1箇所、渡す値は 4 と 6

`itoa` は `numberedFiles` の正規表現 `^([0-9]{<itoa(digits)>})-.*\.md$` にだけ使われ、呼出しは `plan/issue`（digits=6）と `plan/adr`（digits=4）。**どちらも1桁なので現在は偶然正しく動く。**

### 壊れ方は「静かに止まる」

```text
itoa(9)   = "9"
itoa(10)  = ":"     ← 記号になる
itoa(514) = "Ȳ"

digits = 10 のとき  pattern = ^([0-9]{:})-.+\.md$
```

`regexp.MustCompile` は **panic しない**（`{:}` は有効な数量子）。コンパイルは成功し、**どのファイルにも一致しなくなる**。`assertUnique` は空集合に対して「重複なし」と報告する。**検査が落ちるのではなく、検査が無くなる。**

### 第33ラウンドの記述は誤っていた

「呼出し元は ADR 番号や Issue 番号（1〜3桁）を渡している」と書いたが、実際は**桁数**（4・6）を渡している。この誤りのせいで危険度を「3桁で化ける」と見当違いに評価していた。実際の境界は 10 桁以上。本ラウンドで訂正した。

### 修正と試験

`strconv.Itoa` に置き換え、`itoa_test.go` の2件を追加（複数桁の綴り、桁数ごとの正規表現が一致すること）。旧実装へ戻す変異で両方が落ちることを確認した。

**「静かに通る試験」に2ラウンド連続で遭遇したことになる。** 第33ラウンドは walk が0ファイル、本ラウンドは正規表現が0一致。どちらも `go test` は緑のままだった。

## 39. 第35ラウンド（2026-11-09）: 2つの試験が対象0件のまま常に成功していた

第33・34ラウンドで「検査の入力が空になると成功を返す」型を2回踏んだので、`tools/repository-plan-check` 全体に残っていないかを確かめた。生出力は[第35ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND35_VACUOUS_CHECKS_2026-11-09/README.md)にある。

### `plan/issue` に番号付き文書が1件も無い

```text
$ ls plan/issue
README.md
```

README は「今後の作業は GitHub Issues で管理します。過去のローカルIssue文書は…Privateリポジトリへ退避しました」と書いている。したがって `numberedFiles(t, plan/issue, 6)` は**0件**（`plan/adr` は47件）。この結果、2つの試験が何も検査していなかった。

| 試験 | 0件のときの挙動 |
| --- | --- |
| `TestIssueNumbersAreUnique` | `assertUnique` が空集合を走査して**成功** |
| `TestIssueStatesUseTheSixAllowedValues` | ループが0回で**成功** |

**6値規則（Proposed / Ready / In Progress / Blocked / Done / Superseded）を検査しているはずの試験が、Issue 0件なので何も見ていなかった。** 同じ `assertUnique` を使う ADR 側は実際に動いているため、片方だけ無効という気づきにくい形だった。

空集合は「重複が無い」ので論理的に正しく成功する。**入力が空になったことは成功と区別がつかない。**

### 修正

1. `assertUnique` が `len(byNumber) == 0` で `t.Fatalf`。
2. `TestIssueNumbersAreUnique` を削除（読む対象が無い）。
3. `TestIssueNumbersAreLocalDocumentsAbsent` を追加（0件であることを明示。1件でも現れたら落ちて一意性検査の復活を促す）。
4. `TestIssueStatesUseTheSixAllowedValues` は0件なら `t.Skip`（成功ではなく **SKIP** として可視化）。

### 残る穴

**GitHub Issues の状態が6値に従っているかは、どこでも検査されていない。** ローカル文書が GitHub へ移った時点で、規則を検査する対象が消えた。SKIP はそれを可視化しただけで、解決はしていない。`gh issue list` を呼ぶ検査は CI にネットワーク依存とトークンを要求するため、運用判断として #3931 に残す。

### 3ラウンド連続で同じ型

| ラウンド | 空回りの形 |
| --- | --- |
| 33 | `filepath.Walk("..")` が0ファイル |
| 34 | `itoa(10)` で正規表現が0一致 |
| 35 | `plan/issue` に文書が無く、2試験が0件を走査 |

第22〜24ラウンドの「手書きの一覧が漏れる」型は漏れが可視だが、こちらは**検査の不在に気づけない**。

## 40. 第36ラウンド（2026-11-09）: 空回りする検査を1箇所に集約して可視化した

第35ラウンドで「`numberedFiles` の利用者2件を見たに留まる。他の検査が同じ穴を持つかは未調査」と書いた。これを閉じた。生出力は[第36ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND36_SCOPE_NONEMPTY_2026-11-09/README.md)にある。

### 全数調査の結果、空だったのは `plan/issue` だけ

計測用の試験を一時的に置いて各検査の入力を数えた。

```text
markdown files visited: 951    issue numbered: 0    adr numbered: 47    requirementIDs: 36
plan 15 / plan/adr 48 / plan/issue 1（README のみ） / plan/evidence 290 / plan/workstreams 30 / tools 249
go.mod 826B / cmd/tei-studio/go.mod 498B / THIRD_PARTY_NOTICES.md 18984B
```

`forEachMarkdown` 系（951件）・ADR（47件）・要件（36件）・go.mod 系はすべて実体があった。**第35ラウンドの懸念は「他に無い」で閉じる。**

### ただし私の第35ラウンドの修正に穴があった

統合ガードの初版は `os.ReadDir` の**件数**を見ており、`plan/issue` は `README.md` を1件持つため「空でない」と判定していた。**除外（exempt）の分岐が一度も実行されない**状態だった。`-v` に除外の行が出ないことで気づいた。

修正後は**検査が読む文書の件数**を測る。`plan/issue` が0件であることが理由つきでログに出る。

### 追加した試験

`tools/repository-plan-check/scope_nonempty_test.go` の `TestEveryCheckScopeIsNonEmpty`。番号付きスコープは文書数、その他はディレクトリ数、Markdown 全体と要件は訪問数で測る。0件かつ除外指定がある場合だけ理由を出して通し、除外指定が無ければ落とす。件数（951・36）は常にログに出るので、walk が縮んだことにも気づける。

変異3種（`plan/adr` を退避／`plan/workstreams` を空に／`plan/issue` に文書を1件置く）がすべて落ちることを確認した。

### 4ラウンド続けて同じ型

| ラウンド | 形 | 位置付け |
| --- | --- | --- |
| 33 | walk の起点が誤り、0ファイル | 個別に修正 |
| 34 | 正規表現が0一致 | 個別に修正 |
| 35 | `plan/issue` が0件 | 個別に修正 |
| 36 | — | **同型をまとめて検出する試験を追加** |

個別修正だけでは、次に別の検査が空回りしたとき同じ手順を繰り返す。今回は検出を1箇所に置いた。**ただし「入力が0件でない」ことしか見ておらず、入力が十分かは保証しない。**

## 41. 第37ラウンド（2026-11-09）: 設計者ループは動くが、補助コマンドの help が誤っていた

目的の中心「生成AIがTEIでTEIを設計できるか」を、**実際に設計を1つ追加する**作業で測った。生出力は[第37ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND37_DESIGNER_LOOP_2026-11-09/README.md)にある。

### 設計者ループは完結した

`reference/workspace` の写しに `security.logout`（`authenticate` の outcome `authenticated_subject` を入力に取る）を追加した。

| 段階 | 応答 |
| --- | --- |
| `validate` | `ok:true`（新規診断なし） |
| `inspect` | `inputs`/`outcomes` を返す |
| `trace` | `input -> security.authenticated_subject`（resolved）、`outcome -> std.boolean`（resolved） |

**誤りも正しく検出された。** 存在しない型を outcome に書くと `validate` が `UNKNOWN_SEMANTIC_CAPABILITY_OUTCOME_PAYLOAD` を出し、`trace` は行を作らず `projectionUnresolved` に落とす。同じ capability を2ファイルで宣言すると `DUPLICATE_SEMANTIC_ELEMENT` / `trace` は `SUBJECT_AMBIGUOUS`。「壊れたモデルに対して正しく失敗する」。

影響範囲も `trace -direction both` で引ける（`authenticated_subject` の参照元2件）。

### 見つけた欠陥: `-qualification-requirements` の help が「JSON」としか言わない

`knowledge-qualify` を設計検証に使おうとして**3回続けて失敗した**。

```text
-h:      knowledge qualification requirements JSON
inline:  KNOWLEDGE_BUILD_FAILED: open {"qualifications": [...]}: no such file or directory
```

**ファイルパスを取る**のに、そう書いていない。スキーマ（`expectedKind`・`application`・`requiredDependencies`）も `-h` にも `commands` にも出ていない（このコマンドは `argumentValues`/`argumentType`/`argumentDefaults` がすべて空）。さらに引数の型違いに `INVALID_ARGUMENT` ではなく `KNOWLEDGE_BUILD_FAILED` を返すので「モデルが壊れている」と読める。

### 修正

`-h` の文言を `path to a file holding the knowledge qualification requirements JSON` に変えた。**これだけで3回の失敗は0回になる。** `knowledge-qualify` と `knowledge-review-handoff` は同じ定義（`cmd/tei/main.go:180`）を共有するので両方に効く。変異試験で片方を戻すと**両方のサブテストが落ちる**ことを確認した。

試験は `cmd/tei/qualhelp_test.go`。既存の `buildCLIForLifecycle` を再利用した。

スキーマの広告と符号の分離は判断を伴うため、実装せず [#3943](https://github.com/hat47x/tei-platform/issues/3943) に材料として残した。

### 所感

**設計判断そのものより、道具の入口の説明不足が作業を止める。** 1行の help が無いために CLI だけで完結しなかった。

## 42. 第38ラウンド（2026-11-09）: `validate -h` が70フラグを表示していた

設計検証で診断の内訳を知ろうとして `-h` を読み、`validate` に70個のフラグが並ぶのに受理は3個だと気づいた。生出力は[第38ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND38_VALIDATE_FLAG_NOISE_2026-11-09/README.md)にある。

### 原因は「正本はあるが入口が2つ」

`cmd/tei/main.go:152` で**1つの `flag.FlagSet` を全コマンドが共有**しているため、`flag` パッケージ標準の `-h` は全フラグを並べる。一方 `helpText` は**既に allow-list（`selectedReadOnlyFlagAllowlist`）で絞っていた**。

欠けていたのは到達経路。`ContinueOnError` のコマンドだけが `flag.ErrHelp` を捕まえて `printHelp` に届く。`validate` と `world` は `ExitOnError` だったので、標準の一覧を出して即終了していた。

| コマンド | 修正前 | 修正後 |
| --- | --- | --- |
| `validate` | **70** | **3** |
| `world` | **70** | **2** |
| `inspect` | 3 | 3（不変） |

### 実際に踏んだ誤解

`-h` に出ていた `-explain` と `-diagnostic-candidates` を `validate` の機能だと思って試し、2回 `INVALID_ARGUMENT` を食った。`-explain` は **PostgreSQL の EXPLAIN FORMAT JSON** を取るフラグ、`-diagnostic-candidates` は `plugin-resolve` 専用である。

### 修正

`-h` が要求されたら `ContinueOnError` にする1条件を足し、既存の `printHelp` 経路へ流した。新しい表示経路は作っていない。試験は `cmd/tei/helpfilter_test.go` で、変異（`wantsHelp` を常に false）で両方のサブテストが落ちることを確認した。

### 途中で捨てた実装

最初は `fs.Usage` を上書きして自前で絞ったが、`usage:` が2回出力された。`helpText` が同じことをしており二重になっていた。**既存の仕組みを読まずに書いたのが原因**で、実装を捨てて分岐を直す形に変えた。

### 位置付け

第22〜24ラウンドの「手書きの一覧が漏れる」型の変種。**正本は既にあり、それを使う関数もあった。** 欠けていたのは「`-h` をそこへ流す」1条件だけだった。第37ラウンド（正本が無く help が実態と違う）とは逆で、**正本はあるが到達経路が分かれていた**。

## 43. 第39ラウンド（2026-11-09〜10）: 13分前に他者が同じ問題を直していた

第38ラウンドで実装した `-h` のフィルタは、**提出する13分前に別の作業者が同じ問題を修正していた**（`0b85c031a`、同じ #3943 を参照）。私は気づかず重複実装を提出し、PR #3972 で撤回した。生出力は[第39ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND39_DUPLICATE_HELP_FIX_2026-11-09/README.md)にある。

### 時系列

| 時刻（+09:00） | 出来事 | コミット |
| --- | --- | --- |
| 06:14 | 私が base にした `main`。**この時点では未修正** | `f8286ca73` |
| **06:25** | **別の作業者が修正**（`requestedHelp` + `printHelp`） | **`0b85c031a`** |
| 06:38 | 私が同等の修正を提出、#3956 を起票 | `6b5be43a6` |
| 07:59 | 重複を撤回 | `365789bd8` |

`git merge-base --is-ancestor 0b85c031a f8286ca73` は偽。**base 取得時に未修正だったことは事実で、作業中の約24分に他者が直した。**

### 寄与ゼロを確認した

他者の修正だけを残して私の変更を外すと、挙動は同一だった（`validate -h` 3件、`world -h` 2件）。私の `wantsHelp` は動作に寄与していない。しかも既存の `requestedHelp` は `--` 以降を help と見なさない分、私の実装より正しい。

### 記録の誤りも訂正した

2つの虚偽を書いていた。

1. #3956 のクローズ時に「`plan/evidence/TEI_DOGFOODING_ROUND39_...` と計画 §43 を参照」と書き、**どちらも存在しなかった**。
2. 第38ラウンドの記録が、撤回した `wantsHelp` を修正として記録していた。撤回後は存在しない関数を3箇所で参照していた。

本節と本ラウンドの証拠が1を埋め、第38ラウンドの記録を訂正して2を直した。

### 原因は3つとも「読まずに書く」

| # | 内容 | 気づいた手段 |
| --- | --- | --- |
| 1 | `helpText` の存在を確認せず `fs.Usage` を書いた | `usage:` が2回出た |
| 2 | `TestAcceptedCommandsMatchDispatch` の走査範囲を確認せず文字列 switch を書いた | CI の14分テストが落ちた |
| 3 | 提出前に `git log origin/main -- cmd/tei/` を見なかった | 撤回作業で `git blame` して判明 |

**1と2は同一ラウンド内、3は次のラウンドで、すべて同じ原因。** 多重作業者環境では、着手時と提出前の両方で対象パスの履歴を見る必要がある。

## 44. 第41ラウンド（2026-11-10）: realize による実装束縛が参照グラフに現れない

「この capability は何で実装されているか」を CLI だけで答えられるかを試した。Binding/Realization 層は ADR-0048 の4層のうち3層目で、設計者が最も参照する情報の1つである。生出力は[第41ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND41_REALIZE_NOT_IN_GRAPH_2026-11-10/README.md)にある。

### 観測

`reference/workspace` には `realize` 宣言が4つある（`security.authenticate`・`security.authorize`・`sales.order.submit`・`order_entry.compose_order_summary`）。しかし**どの trace にも実装束縛の関係が出ない**。全 subject（両 scope、`-direction both`）で観測された関係は15種で、束縛に相当するものは0種。`security.authenticate` の incoming は `total: 0`。

私はこれで「この2つは未結合」と誤読し、`grep` して初めて `realize` 宣言を見つけた。

### 原因

| 層 | 状態 |
| --- | --- |
| `ir.World` | `Bindings` を持つ（`world.go:35`） |
| 派生索引 | `d.CustomGoBindings` に写している（`index.go:478`）。`Application`・`Capability`・`Adapter`・`Hook` を保持 |
| **参照グラフ** | **`CustomGoBindings` を1度も参照しない**（`grep -c` で 0） |

`FromIndex` は `SemanticDependencies` と `DeclaredDependencies` だけを辺にする。束縛は `d` にあるのに、どちらにも写されていない。

### 影響の実測

`authenticate.tei` を削除して前後を比べた。`worldFingerprint` と `sourceIndexFingerprint` は変わるが、**`graphFingerprint`・`trace.rows`・`trace.total` は一切変わらない**。意味層の変更が参照グラフに現れないため、`trace` による設計検証では束縛の欠落も誤りも検出できない。`validate` も何も言わない。

### 起票

[#3983](https://github.com/hat47x/tei-platform/issues/3983)。案 A（`DeclaredDependencies` に束縛の辺を追加）を推奨とした。索引が既に束縛を保持しているので写像1箇所の追加で済むが、`graphFingerprint` が変わるため保存済み継続トークンは失効する。

### 補助として確かめた正常な挙動

先に `trace` のページング・指紋・alias を確認したが、**すべて設計どおり**だった。`-limit 2` で 2+2+1=5 と `nextOffset` が正しく終端し、継続トークンは `result.snapshotFingerprint` を指し、無関係な名前空間の追加でも `STALE_SNAPSHOT` になる。`graphFingerprint` と `snapshotFingerprint` が常に同値なのは deprecated alias として明記されている。この領域は丁寧に作られている。

## 45. 第42ラウンド（2026-11-10）: realize の実装束縛を参照グラフの辺にした

第41ラウンドで起票した #3983 を修正した。生出力は[第42ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND42_REALIZE_EDGE_2026-11-10/README.md)にある。

### 実装前に読んだこと

| 確認 | 結果 |
| --- | --- |
| 誰かが着手済みでないか | `gh pr list`・`git log` に該当なし |
| `BindingIndexEntry` / `DeclaredDependency` の形 | 両方とも必要な欄を持つ |
| 既存の関係名 | kebab-case の動詞（`owns-pattern`・`uses-semantic` など） |
| **`Application` は常に入るか** | パーサは返さないが loader が `b.Application = a.ID` を設定する |
| `application` はノードになるか | なる。ただし `ApplicationPatterns` 経由のみ |

最後の2点から**「空なら辺を作らない」ガードが必要**と判断した。空キーを指す辺は投影が `unmaterialized-from` として報告するため。

### 修正

`internal/index/index.go` の束縛走査に、`application -realizes-> semantic-capability` の依存を1つ追加した。関係名は DSL の `realize` に対応させた。

### 結果

| capability | realizes 辺 |
| --- | --- |
| `security.authenticate` | 1（`order_entry`） |
| `security.authorize` | 1（`order_entry`） |
| `sales.order.submit` | 1（`order_entry`） |
| `order_entry.compose_order_summary` | **0** |

4つ目が0なのは正しい。`order_entry.compose_order_summary` は**どこにも宣言されておらず**、投影が `unmaterialized-to` として報告する。**修正前から存在するワークスペース側の不整合**で、束縛がグラフに入っていなかったため見えなかったものが可視化された。私は触っていない（Owner の判断）。

### 試験と回帰

`internal/index/realizes_test.go` の3件（辺の存在・索引記録の保持・空側ガード）。変異2種（辺の削除・ガードの無効化）が両方落ちることを確認した。`./cmd/tei` を含む全対象パッケージが ok（170.8s）。

### 互換性

`graphFingerprint` が変わる（辺が増えるため）。保存済み継続トークンは失効する。`projectionUnresolved` に上記の `unmaterialized-to` が新たに現れるが、これは既存不整合の可視化である。

**「見えないこと」自体が欠陥だった**ことの証拠でもある。

## 46. 第43ラウンド（2026-11-10）: 他の3種の実装束縛は依然として参照グラフに現れない

第42ラウンドで `customGoBindings` を辺にした。同じ層に他の束縛種が3つあるので確かめた。生出力は[第43ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND43_OTHER_BINDINGS_2026-11-10/README.md)にある。

### World には4種、trace には1種

```text
tei world -workspace reference/workspace
  customGoBindings                  : 4
  headerDetailContractQueryBindings : 1
  contractQueryExecutorBindings     : 2
  contractQuerySQLRealizations      : 1
```

全 subject（scope `c`・`s`・`e`、`-direction both`）を走査した関係は16種で、`realizes`（`customGoBindings` 由来）は18件。**他の3種に対応する関係は0件。** `trace -subject-kind` にも該当する kind が無く、subject としても辺としても辿れない。

### 原因（実測）

```text
HeaderDetailContractQueryBindings  index.go:0 projection.go:0
ContractQueryExecutorBindings      index.go:0 projection.go:0
ContractQuerySQLRealizations       index.go:0 projection.go:0
CustomGoBindings                   index.go:4 projection.go:0
```

`ir.World` には4種すべての欄がある（`world.go:35-38`）が、派生索引は `CustomGoBindings` しか読んでいない。

### 影響の実測

`contract-query-executor-binding` の `executor` を `database.query.oracle` から `database.query.postgres` に変えると、`worldFingerprint` は変わるが **`graphFingerprint` と `trace.total` は変わらない**。実行基盤の選択が変わっても参照グラフは何も言わない。

### 扱い

新規 Issue は作らず #3983 に残件として追記した（同じ層・同じ原因で、分割すると重複が増えるため）。辺の両端を何にするかは3種とも設計判断が要るので、案の材料を提示するに留めた。

### 併せて処理した重複PR

`#3983` には私の PR #3989 と、別の作業者の PR #3994 が同時に出ていた（私が5分先に作成、7分先にマージ）。**両方を `main` に入れると辺が二重になる**（`projection.addEdge` は重複排除しない）ため、#3994 に経緯と判断材料をコメントした。

## 47. 第44ラウンド（2026-11-10）: 永続化マッピングの列対応が参照グラフに現れない

「この属性はどの列に格納されるか」を CLI だけで答えられるかを試した。生出力は[第44ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND44_PERSISTENCE_NOT_IN_GRAPH_2026-11-10/README.md)にある。

### 先に確認した正常な挙動

第38ラウンドで私が指摘した `commands` と `-h` の不一致は、他者の修正で**解消済み**だった（全6コマンドで一致）。第28・42ラウンドの辺（`outcome`・`realizes`）も健在。

### 観測

`column sales.order.total -> total_amount` の対応は `reference/workspace` に21件ある。しかし:

```bash
tei trace -scope s -subject-kind semantic-attribute \
  -subject-id sales.order.total -direction incoming -mode full
# total: 0
```

属性20件のうち incoming を持つのは**3件だけ**（いずれも `semantic-validation -attribute->` を持つもの）。残り17件は永続化やクエリから参照されていても incoming を持たない。`trace -subject-kind persistence` は `UNSUPPORTED_SUBJECT_KIND`、`inspect` の受理集合にも無い。

### 原因は前回と一段違う

| 層 | 状態 |
| --- | --- |
| `ir.World` | `Mapping.Columns []ColumnMapping`（`Attribute`＋`Column`）を持つ |
| 派生索引 | `PersistenceIndexEntry` に**列対応の欄が無く、写していない** |
| 依存リスト | 1本も写していない |

前回（#3983）は「索引が読んでいない」だったが、今回は「**索引は読んでいるが中間段を捨てている**」。

### 起票

[#4001](https://github.com/hat47x/tei-platform/issues/4001)。案 A（`persistence` を subject-kind に加え `-column-> semantic-attribute` を作る）を本筋としたが、`ir.World`→索引→投影の3段に列対応を通す変更が要る。層分離を理由に意図的に除外されている可能性もあるが、**その根拠を文書で見つけられなかった**ので、判断は Owner に委ねる。

### 同じ型の4例目

| ラウンド | 見えなかったもの |
| --- | --- |
| 28 | `capability.outcome` |
| 41→42 | `realize` の実装束縛 |
| 43 | 他の3種の束縛 |
| 44 | 永続化の列対応 |

## 48. 第45ラウンド（2026-11-10）: World の全欄と参照グラフの対応を検査で固定した

第28・41〜44ラウンドで「モデルにあるが参照グラフに無い」型を**4回**見つけた。個別に辺を足すだけでは、次に `ir.World` へ欄が増えたとき同じ手順を繰り返す。再発を機械的に検出する試験を追加した。生出力は[第45ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND45_WORLD_FIELD_COVERAGE_2026-11-10/README.md)にある。

### 実測: 17欄のうち3欄が読まれていない

`ir.World` の全欄について、`index.Build` が `w.<欄>` を参照するかを機械的に数えた。

```text
読む   : Plugins Bundles Semantics Applications Databases Persistence
         Migrations MigrationProviders ApplicationReleases DatabaseReleases
         SolutionReleases Environments Queries Bindings        (14)
読まない: HeaderDetailContractQueryBindings
         ContractQueryExecutorBindings
         ContractQuerySQLRealizations                            (3)
```

**読まれない3欄は、第43ラウンドで手作業で特定した3種と完全に一致した。** 数え方の裏付けになった。

### 追加した試験

`internal/index/world_field_coverage_test.go` の2件。`reflect` で全欄を列挙し、`worldFieldCoverage` に無い欄があれば落ちる。逆方向（表にあるが World に無い欄）も落ちる。未読の欄には**理由を必須**にした。

**「読まれていない」を失敗にせず、明示を要求する形**にした。plugin・release・environment のように参照グラフの subject ではない欄が正当に存在するため。

変異2種（未登録の欄を追加／未読の理由を削除）が両方落ちることを確認した。

### この試験が保証しないこと

```text
ir.World --[この試験が見る]--> index.Build --[見ていない]--> 参照グラフ
```

**「読む」と書いた欄の内容が辺になるかは見ていない。** `Persistence` は「読む」だが、`#4001` のとおり列対応は辺になっていない（`Mapping.Columns` を索引が捨てている）。つまり `#4001` の穴はこの試験では捕まらない。**半分だけ塞いだ。**

次の段（entry 型が World の下位構造を落としていないか）は型ごとの判断が要るので、別の設計として扱うのが妥当と考えている。

## 49. 第46ラウンド（2026-11-10）: 索引の entry 型が World の下位構造を捨てていないかを検査で固定した

第45ラウンドで「半分だけ塞いだ」と書いた2段目を塞いだ。生出力は[第46ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND46_ENTRY_FIELD_COVERAGE_2026-11-10/README.md)にある。

### 実測: `PersistenceIndexEntry` は10欄のうち6欄を捨てている

| 型 | 欄 |
| --- | --- |
| `persistence.Mapping`（World 側） | `ID`・`Database`・`SemanticData`・`Table`・`Columns`・`RelationStorage`・`PrimaryKey`・`UniqueConstraints`・`Indexes`・`ForeignKeys`（10） |
| `PersistenceIndexEntry`（索引側） | `Mapping`・`Database`・`SemanticData`・`Table`（4） |

対照として `realization.CustomGoBinding` → `BindingIndexEntry` は2欄とも保持し、`Adapter`・`Hook` を加えている（捨てていない）。

### 追加した試験

`internal/index/entry_field_coverage_test.go`。各 source 型の全欄が entry 型に在るか、`renamed`／`dropped` に記録されているかを検査する。**捨てている欄には理由を必須**にした（第45ラウンドと同じ方針）。`ID` が `Mapping` に改名されている事実も表に記録した。

変異3種（未登録の欄追加／`Columns` の dropped 行削除／`ID` の rename 行削除）がすべて落ちることを確認した。

### これで2段とも塞がったが、#4001 は直っていない

```text
ir.World --[第45ラウンド]--> index.Build --[第46ラウンド]--> entry 型
```

**両方の段が「判断を書かなければ通らない」になった。** しかし `#4001` の欠陥（列対応が辺にならない）は残る。今は理由付きで記録されているだけで、`trace` は依然として列対応を返さない。**検出を完成させ、修正は設計判断として残した。**

`database` は `SupportedSubjectKinds` に無いため、辺を作るには kind の追加も伴う。私は材料を提示するに留める。

## 50. 第47ラウンド（2026-11-10）: trace が realizes 辺を2回返す（起票 → 解消）

計画 §5 の残った問い（「発見の粒度」「『効果的』の測り方」）に答えるため、設計者の問いに何回の CLI 呼出しで答えるかを測った。その過程で、応答に同一の辺が2行含まれることに気づいた。生出力は[第47ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND47_DUPLICATE_REALIZES_EDGE_2026-11-10/README.md)にある。

### 設計の問いは3呼出しで答えられた（ただし1つ目が誤っていた）

「どの application が `security.authenticate` を実装し、何を受け取って何を返すか」は `trace -direction incoming` → `trace -mode full` → `inspect` の3回で答えられた。**ただし1回目の応答が正しくなかった。**

```text
total: 2
  [0] application order_entry -realizes-> semantic-capability security.authenticate
  [1] application order_entry -realizes-> semantic-capability security.authenticate
```

`world` の `customGoBindings` は4件（各 capability 1件）なので宣言の重複ではない。全 subject を走査すると、**重複するのは `realizes` だけ**（6 subject）で、他の15関係は重複しない。

### 原因: 2つの経路が同じ辺を作っていた

| 場所 | 実装 |
| --- | --- |
| `internal/index/index.go:488` | `d.DeclaredDependencies` に追加（#3989、私の修正） |
| `internal/modelref/derivedbridge/projection.go:117` | `d.CustomGoBindings` を直接走査（`8219cc3d4`） |

`addEdge` は `nodes` の存在だけを見て `Graph.Edges` に無条件で追加していたため、同じ辺が2行になった。

### PR の状態とコミットの所在は別だった

第43ラウンドで「両方入れると二重になる」と #3994 にコメントした。それが実現していた。ただし**当時の私の状態の読み方が誤っていた**。

```bash
$ gh pr view 3994 --json state,mergedAt     # CLOSED, merged=null
$ git log --oneline origin/main -S "range d.CustomGoBindings" -- internal/modelref/derivedbridge/projection.go
# 8219cc3d4    ← main に在る
```

**PR が閉じられていればコミットも入っていない**と暗黙に仮定していた。**PR の状態ではなく `git log -S` で確認すべきだった。** 第39ラウンドと同じ根（検証手段の選択ミス）である。

### 解消

[#4024](https://github.com/hat47x/tei-platform/issues/4024) として起票した。私が示した3案のうち、**`addEdge` に重複排除を入れる根本案**が採られ、`102e5fa09`（[#4038](https://github.com/hat47x/tei-platform/pull/4038)）で修正された。`projectionUnresolved` の重複も同時に畳まれた。

| 時刻（UTC） | 出来事 |
| --- | --- |
| 06:07:36 | 私が起票 |
| 07:37:14 | 修正コミット |
| 08:22:25 | Issue クローズ |

**起票から解消まで約2時間15分。** 私は修正を書いていない。修正後の `main`（`735f7278d`）で実測し、`total: 1` / `[0] order_entry - realizes` を確認した。

### この1件で確認できたこと

起票の内容（再現手順・原因の特定・3案の比較・`addEdge` が根本という指摘）が、そのまま修正方針に使われた。**分析を渡せば、修正は他者が進められる**という分業が成立した形である。

## 51. 第49ラウンド（2026-11-10）: データ層のkindがノードにならず、辺が全て落ちる

#4001 の解消を試みたが、**着手前の再実測で自分の主張が過大だったことが分かった**。生出力は[第49ラウンドの記録](../evidence/TEI_DOGFOODING_ROUND49_DATA_LAYER_UNMATERIALIZED_2026-11-10/README.md)にある。

### 訂正: `tei world` は既に答えていた

#4001 で私は「CLI で答えられない」と書いた。**`trace` だけを試して一般化したのが誤りだった。**

```text
tei world -workspace reference/workspace
  FOUND: sales.order.total -> sales_core.public.orders.total_amount
  mappings=2 columns=11
```

`persistenceMappings` は列対応を含めて出力される。**「この属性はどの列か」は既に CLI で答えられる。** `trace` が運ばないのは事実なので、Issue は #4068 の依存として残した。

### 根本原因: database 系がノードにならない

`order_entry` は `database sales { target sales_core }` を宣言し、索引も `targets-database` 辺を宣言しているのに、trace には出ない。

```text
total: 14  unresolved: 0
relations: {owns-pattern:5, realizes:3, uses-validation:3, uses-semantic:3}
targets-database present? NO
```

`projectionUnresolved` に `unmaterialized-to:database:sales_core` などがあった。72 subject を走査した結果:

```text
sum of trace.unresolved across them: 0
distinct projectionUnresolved messages: 12
  unmaterialized-from/to database-contract        216 each
  unmaterialized-from/to database                 144 each
  unmaterialized-to    database-contract-query     72
```

**12種の未解決参照が 72 subject すべてで報告され、それでも `trace.unresolved` は 0。**

### 原因は2つ

| 原因 | 内容 |
| --- | --- |
| (a) | `SupportedSubjectKinds` に database 系が無く、索引は database ノードを作らない。端点が無い辺は必ず落ちる |
| (b) | `TraceDirect` は `graph.Edges` を走査するので、投影で落ちた辺を数えられない |

### 影響: 7ラウンド見落としていた

私は第41〜47ラウンドで `unresolved: 0` を見て正常と判断し続けた。実際には12種の未解決が常にあった。**この表示は「正常」ではなく「何も見ていない」だった。** これが #4001 の根本原因を隠していた。

### 起票

[#4068](https://github.com/hat47x/tei-platform/issues/4068)。4案を示し、**B（ノードは作るが subject-kind には加えない）+ D（`unresolved` に投影の取りこぼしを含める）**を推した。A は `trace` の subject-kind が公開契約で、`tools/reference-mcp-readonly-adapter/subject_kind_enum_test.go` が `len(TraceSubjectKinds) != 10` を固定しているため外部ツールも同時に直す必要がある。C（辺を宣言しない）は採るべきでない — 見えないという判断なら `AuthorityLimits` に書くべきで、黙って宣言して落とす現状は誰にも利益がない。
