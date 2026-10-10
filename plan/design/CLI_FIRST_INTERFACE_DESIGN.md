# CLIを主入口とするTEIインターフェース設計（初期案）

- **状態**: 評価・実装着手中（2026-10-10）。初期のCLI参照実装はPR内の候補であり、mainへ統合済みとは扱わない。
- **担当課題**: [#1155](https://github.com/hat47x/tei-platform/issues/1155)。
- **関連文書**: [全体構成](../../docs/architecture.md)、[案件ごとのPlugin構成](PROJECT_PLUGIN_COMPOSITION_DESIGN.md)、[Web構成](WEB_APPLICATION_COMPOSITION_DESIGN.md)、[開発手順](../../CONTRIBUTING.md)。

## 1. なぜCLIから始めるか

TEIは、最初から画面を揃えるよりも、業務モデルを読み取り、整合性を調べ、変更案を作り、受理可能か確かめる流れを先に成立させたい。その入口として、端末や生成AIが同じ操作を再現できるCLIを優先する。Goだけで作れる小さな呼び出し口を先に揃える方が、Webやデスクトップの画面設計に引きずられず、中核機能の不足を発見しやすい。

ただし「CLIを先に作る」は「すべてをコマンドの中に実装する」ではない。コマンド固有の文字列解析、表示形式、終了コードは周辺の関心事とし、意味の読取り、変更案の検証、業務の実行権限をCLIの内部へ閉じ込めない。

## 2. 将来のWebとデスクトップまで共有する境界

```text
LLM / human / automation        Browser                 Desktop/WebView
        |                         |                         |
   tei CLI adapter           HTTP/API adapter         local IPC adapter
        |                         |                         |
        +-------------------------+-------------------------+
                                  |
                     Application-facing operations
                      (Go service / typed contracts)
                                  |
            +---------------------+---------------------+
            |                     |                     |
     Workspace/Source       Proposal/Acceptance     Driver/Capability
       Canonical Model       existing authority      Runtime/Plugin
            |                     |                     |
            +---------------------+---------------------+
                                  |
                    Evidence / observation / state
```

この図は**目標とする依存方向**であり、共通Application Serviceがすべて完成しているという説明ではない。現行`cmd/tei/main.go`には直接各パッケージを呼んでいる箇所が多い。C0では最初の対象として`internal/operations.ValidateWorkspace`を設け、ソース読込み、既存Validation、Source Indexへの投影をCLIの表示処理から切り離した。CLIのテキスト表示とJSON表示は、同じ型付き結果を受け取る。ただし、これは共通Application Service全体が完成したという意味ではない。次に必要な操作から同様の境界を検証する。

共通にするのは、業務上の対象ID、版・fingerprint、Validation結果、Proposal／Acceptanceの成否、権限評価を求める対象、業務Action結果などの**意味**である。CLIのargv、HTTPのstatus・Cookie、WebViewのIPCメッセージ、画面の選択状態は共通契約へ持ち込まない。

特に、WebのHTTP要求をCLIコマンドの単なるラッパーとして実行しない。Web Hostは認証、CSRF対策、セッション、経路認可、入力制限などを独立して強制する必要がある。デスクトップでローカルに実行する場合も、端末操作権限が外部DBやPluginへの許可を自動的に与えるわけではない。

## 3. LLMに使いやすいCLIの小さな約束

### 3.1 コマンドを発見できる

`tei commands` は、初期対象とした**読み取り専用**コマンドをJSONで返す。名前、必要な引数、主要な任意引数、出力形式、作用を機械的に確認できるようにする。

```sh
go run ./cmd/tei commands
```

`schemaVersion=tei.cli.discovery/v0`、`scope=selected-read-only-commands` は、一覧が全コマンドやPlugin実装の完全な登録簿ではないことを示す。既存CLIの中には高度な配備関連操作もあるため、最初から「表示されたものは全部安全」とは扱わない。将来は既存の機能契約と拡張のメタデータから候補を得る方式を検討するが、Plugin Resolverの第二実装を作らない。

### 3.2 診断は機械可読、従来の表示も保持する

```sh
go run ./cmd/tei validate -workspace reference/workspace -json
```

`schemaVersion=tei.cli.validate/v0`、`ok`、`diagnostics`、必要な場合の`error`を持つ。Validationの`subject`が成立している場合はそのまま使い、Source Indexから位置を戻す。構文解析時点のエラーは意味上のSubjectがまだないため、出所の位置だけを付ける。

`validate -workspace`の従来のテキスト出力は互換性のため維持する。初版の`-json`はこの経路だけに対応する。既存`world`等のJSON結果を共通の`result`形式へ一斉に包み直さない。

終了コードは当面、`0`がValidation成功（Warningを含む）、`1`がValidation findingによる失敗、`2`が入力・読込み・出力等の実行上の失敗を意味する。JSONモードでは、読込み失敗に加え、不足・衝突した引数も可能な範囲でstdoutの一つのJSON文書として返し、人間向けエラー行や進行表示を混ぜない。Goのフラグパーサー自体が拒否した入力など、まだJSON化していない経路もある。パイプラインでは終了コードと`ok`の**両方**を確認する。

### 3.3 入力と副作用の条件

- **取得**: `inspect`、`source-index`、`validate`、`diff`など。原則として意味の正本を書き換えない。外部接続の有無や公開できる結果は別途宣言する。
- **提案**: `proposal`、`plan`、`preview`など。未受理の候補を返すだけで、Source更新・権限付与・配備を行わない。
- **受理・変更**: 対象、期待するrevision／fingerprint、変更範囲、判断主体、承認経路を明示する。成功を単なるプロセス終了と同一視しない。
- **実行・運用**: Resource、Environment、作用範囲、権限、監査情報を事前に確認する。外部作用が不明な失敗を無条件に再実行しない。

これは**操作を分類する設計方針**であって、上記の名前のコマンドすべてが実装済みという意味ではない。特にAIに与えるTool定義は、コマンド名の羅列でなく、実際に利用できる操作と当該利用者・実行環境の権限に基づいて組み立てる。

## 4. 初期の縦切りと次の優先順位

| 段階 | 作業 | 既存基盤への接続 | 確認すること |
| --- | --- | --- | --- |
| C0（着手） | `commands`、`validate -json` | `cmd/tei`、Workspace、Source Index、Validation | 読取り、diagnosticの構造、互換出力、失敗の分類 |
| C1 | `inspect`／`trace`／`diff` を整理 | Canonical、Source Index、三視点の既存Projection | LLMが対象を絞り、根拠へ戻れる |
| C2 | 変更候補と`preview` | 既存Shared Acceptance／設計Workspace | 仮説、候補、受理済みの区別 |
| C3 | 制約付き変更と安全な業務Action | Application Driver、Permission、Runtime Authority | stale revision拒否、権限拒否、外部作用の再実行防止 |
| W0（設計） | Web APIと画面のAdapter契約 | Web構成キット、Authentication、Driver | CLI固有のargv／exitを持ち込まない |
| D0（設計） | デスクトップIPC／ローカルWorkspace入口 | Wails/WebView候補、Go service | Host権限と業務権限を混同しない |

C1以降の名称・階層は候補であり、現行コマンドを大量に改名することは初回の目的ではない。とくにモデル編集は、[統合モデラー課題 #1124](https://github.com/hat47x/tei-platform/issues/1124)と同じSource／Acceptanceを使い、新たな編集正本を作らない。

## 5. 前倒しで決められること／保留すること

**先に固定する方向**は次のとおり。

1. Canonical meaning・正本への反映権限・Plugin解決をインターフェースごとに複製しない。
2. CLI/HTTP/IPCは同じGoのApplication-facing操作を呼ぶが、それぞれの入出力・本人確認・セキュリティ境界は独立させる。
3. 外部LLMには構造化された短い結果を優先し、詳しいEvidenceは必要時に取得する。ログ・秘密値は結果から分離する。
4. 書込み・実行は読取りとは異なる作用として契約化し、expected revision、権限、確認記録を省略しない。
5. JSON契約にはversionを付け、破壊的変更時に段階移行できるようにする。

**まだ固定しないもの**は、GUIフレームワーク、Webの標準画面構成、RPC方式、CLIサブコマンド階層の最終形、全PluginのTool schema自動生成方式、Remote CLI接続方式、AIの自動承認条件である。代表的な業務変更をCLIから最後まで実証し、その結果からWeb/デスクトップの最小契約を選ぶ。

## 6. 受入条件と未検証範囲

C0の受入は、正常系・構文エラー・意味エラー・入力読込み障害・不正な組合せの引数について、`-json`が一つの構造化結果を返し、従来のテキスト表示を壊さず、`commands`の契約がテストできることとする。加えて、独立したGoの読取り操作から同じValidation結果を取得できること、実バイナリのstdout・stderr・exit codeをプロセステストで確認できることを条件にする。

実装PRをmainへ取り込むには、対象exact headとcurrent mainについて、[リポジトリの統合規則](../../AGENTS.md)が要求するactual full Repository Healthの証拠が必要となる。コードの存在、合成テスト、レビュー、GitHub上のPR作成をその代用としない。

この文書もC0も、production-ready Editor、HTTP API、OS認証連携、デスクトップ版の実現、AIの安全な自律実行を証明するものではない。


## 7. C1: Canonical Subject inspection (draft)

Issue #1165 の第一段階では、`tei inspect` からモデル要素一件の意味情報とSource Index上の出所を取得する。初期対応は`semantic-data`、`application`、`query`の三種類で、kindとCanonical IDの完全一致を要求する。

`internal/operations.InspectSubject`が既存のNormalized Worldから要素を抽出し、同じWorkspaceロードで構築したSource Indexに結び付ける。CLIは引数・JSON・終了コードのみを処理し、新しい意味モデルを所有しない。不在は`SUBJECT_NOT_FOUND`、重複は`SUBJECT_AMBIGUOUS`として一件目を採用しない。

`worldFingerprint`は正規化Worldの内容ハッシュであり、書込み用revisionやAcceptanceを意味しない。`sourceStatus=not-indexed`は対応する出所参照の欠落を示すが、Canonicalの不存在とは異なる。結果は適切な情報開示制御の下で利用する。

この実装候補はグラフの参照先・参照元探索、差分比較、更新、認可、原子的Snapshotを含まない。テストの記述だけでは適合の証拠とせず、actual clean checkoutの全Repository Healthを確認してから統合する。


## 8. C1: 型付き参照traceをCLIへ接続する統合候補

Issue #1165に対し、既存 `index.Build` → `modelref/derivedbridge.FromIndex` → `modelref.TraceDirect` → `modelref/sourcebridge.NewResolver` を、一つの型付きGo操作 `operations.TraceWorkspace` でつなぐ。CLIの `tei trace` は引数・JSON schema・終了コードを変換する薄いAdapterとし、別のEdge正本は持たない。

- `-workspace`、`-scope`、`-subject-kind`、`-subject-id`を必須とする。`scope`はProjectionの文脈ラベルであり、業務namespace・ユーザーの認可scope・受理済みrevisionとの一致を証明しない。
- `direction=outgoing|incoming|both`、`mode=summary|full`。CLIはsummaryを既定とする。Fullは明示的な件数上限がなければ全件出力し、`limit`が正の場合のみ `nextOffset`を返す。
- Source Indexの既存Artifact fingerprintとRangeを検証してからSource Navigationに利用する。GraphとSource Indexが不一致ならエラー扱いとし、Sourceから業務関係を推測しない。元定義のLocationが見つかってもAcceptedや書込み許可を推定しない。
- JSONの `schemaVersion=tei.cli.trace/v0`、`ok`、`result.trace`、`result.source`、`result.relatedSources`（Fullの出力ページのみ）、`result.projectionUnresolved`を返す。意味的な直接参照だけでなく、Derived Indexの投影時点で脱落した関係の情報を保持する。
- 未存在・重複・stale snapshotを終了コード1、不正引数・読込失敗・Source Index検証失敗等を終了コード2とする。機械可読JSONの`error.code`で詳細を識別する。
- `snapshotFingerprint`は候補GraphとPolicyのバイト列に関するハッシュ。ページ間で変化を検出できるが、**Source Index自体を同一時点へ固定せず**、Sourceファイルの読取競合や同時書換えに対する完全な一貫性を保証しない。完全なSnapshot境界は#1017と別途調整する。
- `sourcebridge`に渡す読取関数で相対path正規化、symlinkによるWorkspace外参照拒否、regular file確認を実施する。ただしsymlinkのswapを原子的に防ぐ仕組みではなく、これをproductionのsecure source readerとは呼ばない。
- 初期の関係は既存Derived Indexがすでに提供する宣言済みData/Pattern/Relation等に限られる。Business Procedure/DFD、物理CRUD、共有Acceptance、暗黙のtransitive path探索は未実装。

### PRの統合順

本作業は、独立したDraft系列（[#1159](https://github.com/hat47x/tei-platform/pull/1159) → [#1169](https://github.com/hat47x/tei-platform/pull/1169) と [#1139](https://github.com/hat47x/tei-platform/pull/1139) → [#1171](https://github.com/hat47x/tei-platform/pull/1171)）を参照する**統合用Draft候補**である。統合ブランチで履歴を接続していても各元PRがmainへ統合可能と認定されたわけではない。元PRのheadを変更せず、上流の検証・マージ順序を尊重して最終的な差分を再評価する。

具体的には、executable CLIによる`commands`、`inspect`、`trace`、`validate`の戻り値/出力整合性と、typed Go操作/CLIの同等性を確認する。CIは停止中につき、Goのunit/プロセステストを追加した事実とclean checkoutにおける実行済みPASSを混同しない。`AGENTS.md`に定めるexact head/full repository healthが揃うまでDraftを維持し、mainへマージしない。
