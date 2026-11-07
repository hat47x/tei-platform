# TEIドッグフーディング：生成AIがCLIだけでTEI自身の設計を扱えるかの実測と課題

- **状態**: 2026-11-07 実測記録と課題の起票。read-onlyの実測は完了し、検出した課題は未修正・未受理。
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
- 検出した課題は10件。うち「CLI出力の契約」に属するものは既存の[#2354](https://github.com/hat47x/tei-platform/issues/2354)・[#2824](https://github.com/hat47x/tei-platform/issues/2824)と重なるため、本記録では**重複を明記して新規Issueを作らない**。それ以外は§6で分割先を示す。

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
- 意味の版を`semanticFingerprint`、世界全体を`worldFingerprint`と呼び分けているが、名前だけでは「どちらが正規化モデルか」は分からない。

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

---

## 4. 重複の確認と分割先

| 課題 | 既存Issueとの関係 | 分割先（案） |
| --- | --- | --- |
| D-1 | `#1165`（Closed）はtraceの初期実装。種類の非対応は未起票 | 新規Issue（CLI: traceの未対応kindをSUBJECT_NOT_FOUNDと区別する） |
| D-2 | `#2823`（Open）はsources診断、`#2605`はSkill側の記録 | 新規Issue（CLI: 応答にWorkspace同一性を含める） |
| D-3 | `#2354`（Closed）の残範囲。`#2824`（Open）は広告の一致 | `#2824`へ追記、または新規Issue（CLI: 版の欄を統一する） |
| D-4 | `#2998`（Open）は`committed`の意味。作用の宣言は未起票 | 新規Issue（CLI: effectを作用の範囲で分ける） |
| D-5 | 該当なし（`getting-started.md`は回避策を記載済み） | 新規Issue（CLI: plugin-lockのround-trip） |
| D-6 | 該当なし | D-5と同一Issue、または`#2354`系 |
| D-7 | `#2824`（Open）と近いが対象が異なる | 新規Issue（CLI: -hと引数契約） |
| D-8 | 該当なし | 新規Issue（文書: knowledge-qualifyの要件例） |
| D-9 | `#2619`・`#2621`（Open）の設計範囲 | 既存Issueへ実測結果を渡す（新規Issueは作らない） |
| D-10 | `#2822`（Open）と重なる | `#2822`へ追記 |

**本記録では新規Issueを作らない。** 分割は、この計画文書を統合した後に、重複の最終確認をしてから行う。理由は、`#3149`が「横断の実測記録と転記」を持ち、個別の修正は既存Issueまたは分割したIssueが持つと定めているため。今回の実測で得た生データを先に残し、分割時に参照できるようにする。

## 5. 設計支援の観点で残った問い

- **発見の粒度**: `commands`はコマンド名と引数の一部を返すが、対象の種類（`subject-kind`の許容値）は返さない。生成AIが「何を inspect できるか」を知るには`model catalog`の結果と文書を突き合わせる必要がある。これはD-7の一部である。
- **「効果的」の測り方**: 本実測は、成功・失敗・未対応の区別が付くことまでを確かめた。設計判断の質（どの要素を読むべきか、どの順で確かめるか）は測っていない。`#2605`のS1a（同一checkout・単一実行物での実測）と同じ範囲に留まる。
- **意味の受理との距離**: 本実測の全呼出しはread-onlyで、`view -execute`だけがローカルファイルを書いた。設計の変更を正本へ戻す経路（Proposal・Shared Acceptance）は試していない。`#3149`の完了条件にも含めていない。

## 6. 次に必要な作業

1. 本書と検証記録をブランチへコミットし、`#3149`へ結果を転記する（本作業の範囲）。
2. 課題の分割: D-1・D-2・D-4・D-5・D-7・D-8を新規Issueへ、D-3・D-10を`#2824`／`#2822`へ、D-9を`#2619`／`#2621`へ渡す。分割時に、本書の§3の再現手順と生データのパスを各Issueへ引用する。
3. 修正の採否はOwner判断。本記録は候補と根拠を提示するに留める。
4. 再実測は、`0b9c14a8…`ではなく修正後の`main`で行い、同じ実行物の識別（SHA-256）を記録する。改行の異なるcheckoutでの照合（`#2399`）と、大規模Workspaceでの完全性（`#1177`）は別途。
