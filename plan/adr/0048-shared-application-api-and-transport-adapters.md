# ADR-0048: CLI・MCP・API・Web・Desktopの操作を共通アプリケーションAPIに集約する

- 状態: 採用（設計方針、2026-11-08）。全入口の実装完了・同値性の実証を意味しない。
- 判断主体: TEI Ownerの共通操作方針に基づく。取り消せる設計判断は開発エージェントが代行した。
- 実施の親Issue: [#1158](https://github.com/hat47x/tei-platform/issues/1158)。MCP固有の移行は [#3400](https://github.com/hat47x/tei-platform/issues/3400)。
- 関連: [CLI主入口の設計](../design/CLI_FIRST_INTERFACE_DESIGN.md)、[CLI JSON出力契約](../design/CLI_JSON_OUTPUT_CONTRACT_2026-10.md)、[ADR-0022](0022-ai-proposal-canonical-acceptance-boundary.md)、[ADR-0031](0031-tool-neutral-git-native-development.md)、[ADR-0036](0036-delivery-forms-as-plugin-composition.md)、[ADR-0043](0043-target-native-artifacts-without-tei-host.md)、[ADR-0044](0044-authentication-not-a-core-concern.md)。

## 1. 背景

CLI/MCP/API/Web/Desktopでは、原則として共通の操作・インターフェイスを提供する。同じ機能を各入口へ個別に実装すると、検証の成否、権限、診断、版、再送条件が食い違う。共通機能をAPIとして実装し、各入口がそのAPIを呼ぶ構成を採用する。

ここでいうAPIはHTTP APIに限定しない。型付きのアプリケーション操作の契約を指す。通信経路を統一するために、ローカルCLIをHTTPクライアントだけにしたり、Desktopや組込み利用に常駐TEIサーバーを要求したりしない。

本ADRは既存のCLI-first設計のApplication-facing operationsを、MCP・HTTP・Web・Desktopを含む全入口の規則として具体化する。既存の採用ADRを置き換えず、CLI-firstを「意味処理の所有者がCLI」という意味にはしない。

## 2. 決定

### 2.1 共通APIと依存方向

```text
CLI ────────────────────────┐
MCP Adapter ────────────────┤
Web → HTTP Adapter ─────────┤
Desktop → IPC/直接Adapter ──┤
外部Consumer/リモートCLI ──→ HTTP等のAdapter
                            ↓
                共通アプリケーションAPI
              型付きRequest / Result / Failure
                            ↓
     既存の意味・Workspace・Source Index・Validation
     Proposal / Shared Acceptance / Command / Plugin等のowner
```

共通APIの実装は `internal/operations` を出発点とする。既存の関数が十分なら薄いwrapperを増やさず利用する。配置を変える必要があるときは対象操作だけを小さく移す。巨大な共通Service、文字列による万能dispatcher、第二のPlugin Registryを先に作らない。

依存方向はAdapter→アプリケーションAPI→既存ownerとする。共通APIから `cmd/`・`tools/`・HTTP・JSON-RPC・Wails・WebView・画面状態への依存を加えない。既存の `internal/operations/dependency_test.go` の禁止依存を維持する。契約の共有は中央プロセスへの集約ではない。

### 2.2 呼出し形態

| 利用形態 | 既定の呼出し | 条件 |
| --- | --- | --- |
| ローカルCLI・組込み | プロセス内 | HTTPサーバー・ネットワーク・ログインを前提にしない |
| リモートCLI | 明示した接続先のクライアント→HTTP等 | 接続先・対象・実行場所を結果に結び付け、通信失敗時にlocalへ黙って切り替えない |
| MCP | stdio等のAdapter→共通API | サーバー起動時にHostが許可する対象を固定。モデル入力で読取り範囲を広げない |
| Web | HTTP等のAdapter→共通API | Cookie/CSRF/CORS/要求上限等はHostで強制。HTTPからCLIをshell実行する方式を本実装にしない |
| Desktop | プロセス内、またはIPC→共通API | rendererの入力は信頼しない。IPC先でも対象・権限を検査する |

リモートCLI・全HTTP経路・全Desktop経路が実装済みという説明ではない。未対応の組合せは明示して拒否し、利用可能な操作として広告しない。ネットワーク呼出しはAdapterに置き、通信のために共通APIの依存規則を緩めない。

### 2.3 共通にする範囲

「同じ操作」は同じ対象・同じ版・同じHost方針・同じ主体と許可範囲のもとで比較する。別の権限や別版の結果が違うことを入口間の不整合とはしない。

| 契約 | 共通APIが持つ内容 | Adapterが変換する内容 |
| --- | --- | --- |
| 操作の識別 | 安定した操作ID、契約version、対応する要求・結果型 | argv、URL、MCP tool名、IPC method、画面のAction |
| 入力 | 対象ID、選択条件、値、不在/NULL、期待版、継続条件 | flag・JSON・フォーム等の構文解析 |
| 対象の解決 | Hostが検証したWorkspace/環境/対象への参照 | local path、remote target名、rendererのWorkspace IDからの解決 |
| 結果 | 意味上の値、警告、未解決、source、観測した版 | JSON包み、text/HTML、HTTP status、CLI exit、MCP content |
| 成否 | 操作の完了、検証の合否、業務Outcome、効果の観測を区別 | transport失敗の表現。共通の成否を勝手に再判定しない |
| 失敗 | 文字列の安定したfailure.code、分類、開示可能な詳細 | JSON-RPC数値code・HTTP status等のprotocol用code |
| 作用 | Source変更、外部作用、実行記録/成果物へのファイル書込み、出力先の条件 | 作用を説明する表示と確認UI。annotationだけで許可しない |
| 実行条件 | 必要権限、expected revision、idempotency/再送条件、取消・期限 | signal、HTTP切断、MCP取消等の共通contextへの変換 |

型の名前や全操作共通の巨大envelopeは本ADRで新設しない。操作ごとの型を維持しつつ、共通で意味を持つ欄の定義と失敗分類を再利用する。時刻、ランダムなrequest ID、画面の並び以外に差が出た場合は、比較から除く前に理由を記録する。

### 2.4 成否と副作用

検証は「読取り処理を完了したが検証に不合格」という結果を持つ。単一の `ok` へ次を混ぜない。

1. transportが要求を届けたか。
2. アプリケーション操作が完了したか。
3. モデルの検証が合格したか。
4. 業務Outcomeが成立したか。
5. 実際の効果が独立に観測されたか。

構文エラーにもsource診断を残す。未知kind、対象不在、namespace不在、stale、権限拒否、部分結果、効果不明は別の状態として扱う。未知のJSON欄・型不正を黙って既定値へ落とさない。既定値の適用は共通契約で定め、正常系と同じく入口間で比較する。

読取りでも、記録ファイルを作る場合はその作用を記述する。`view -execute`の `writeFlags` は既存の事実として維持し、Source非変更とファイル非書込みを混同しない。権限、受理の記録、認証、意味の検証は互いの代用にならない。

### 2.5 認証・認可と対象の閉じ込め

主体の確認方法はHost/Pluginに置く。ローカルStudioを無認証で動かすADR-0044を維持し、全入口にログインや新しいIdPを要求しない。無認証でも対象の許可範囲・Resource・Source受理の検査は省かない。

保護された操作では、Hostが解決した主体・許可範囲・方針の参照を、共通APIから既存の認可/受理/Command境界へ渡す。許可を検査する実装をCLI/MCP/HTTPへ複製しない。検査をプロセス内に集中することや、TEI Coreへ新しいPermission Engineを内蔵することも要求しない。

`actor`・`role`・`scope`等の外部文字列は認証済みcontextではない。`trace -scope`は読取り文脈であって権限ではない。Hostだけが構築できるcontextの生成経路とテスト用の代替を区別する。参考Adapterの固定Workspaceは範囲指定にすぎず、symlink/root置換に対するOSレベルの保護を証明しない。既存のpinned reader・allowlistを再利用する。

HTTPの経路保護、CSRF、IPCの接続元確認、入出力のbyte上限は入口側でも強制する。拒否時に秘密のpathや生データをresponseへ含めない。

### 2.6 発見と拡張

操作ID・要求型・作用・利用可能な入口・対応versionを、同じ操作定義から辿れるようにする。CLI catalog、MCP tool schema、HTTP schema、Desktop bindingが独立した意味の正本にならないよう、生成または一致試験でずれを検出する。

発見できることと、現在の主体が実行できることは分ける。Plugin由来の機能も既存の契約・解決結果を使う。未選択Pluginを公開せず、選択結果を認可済みと解釈しない。

## 3. 現行実装と不足

基準は `4fe4d37f5`。以下はコードの静的確認であり、このADRを追加しただけでは修正されない。

| 対象 | 存在するもの | 後続で埋める差 |
| --- | --- | --- |
| `internal/operations` | catalog/inspect/trace/validateの型付き読取りと依存試験 | 操作完了と検証合否、共通failure/作用/文脈の契約を操作単位で整理 |
| `cmd/tei` | 既存CLI JSON v0、検証診断とsourceの投影、exit変換 | 成否・診断・Proposal等の意味を共通側へ移し、text/exitだけを入口へ残す |
| `internal/studiohost/readonly.go` | HostのWorkspace allowlist、`OK/Value/Failure`。操作完了と検証合否を区別 | 共通結果と結び付け、HTTP/IPCごとの再判定をなくす |
| `tools/reference-mcp-readonly-adapter/main.go` | 4操作のstdio参照Adapter | validateはCLIと異なるenvelope。文字列failure.codeがmessageへ埋められる。意味エラーでも `isError:false` になりうる。未知欄・型・既定値・通知の負例も必要 |
| 既存MCP実測 | 指紋・一部値の一致 | 完全なJSON同値・全失敗系・実Harness接続・MCP準拠・安全性の証明ではない |

MCP参照Adapterが `tei.cli.*` を名乗ることを完成済みの共通契約とみなさない。対応する型・意味が違う場合は共通APIへ収束させ、wire互換の扱いも明示する。通信プロトコルの数値codeと意味上の文字列codeを別欄として保持する。

## 4. 移行順序と着手先

実施の親は #1158。実装の担当者は次の順で進める。子Issueのチェックリストを実行単位の正本とし、本ADRへ実績を転記して二重管理しない。

### CAPI-1（[#3419](https://github.com/hat47x/tei-platform/issues/3419)） 棚卸しと代表4操作の共通契約

`internal/operations`、`cmd/tei/cli_validation.go`、`cmd/tei/cli_result.go`、`internal/studiohost/readonly.go`、MCP参照Adapterを読む。catalog/inspect/trace/validateについて操作ID→要求/結果→既存owner→各入口の対応表を作る。

最初はvalidateを縦切りにする。operationCompletedとvalidationPassedをどう分けるかを決め、診断・source・宣言件数・source-root finding・Workspace/World/Sourceの版・Proposalの関係を共通型に保持する。関数や型を追加したら直接呼出しのfixtureで先に確認する。Go moduleの分割や全DTOの一括改名をしない。

### CAPI-2（[#3420](https://github.com/hat47x/tei-platform/issues/3420)） CLI/MCPの負例を含む収束

CAPI-1のvalidate契約から始め、CLI v0互換projectionとMCP projectionを同じ結果へ接続する。MCPの `isError` と `ok` は共通の失敗分類から変換する。検証不合格の表し方をtoolの説明にも明記する。catalog/inspect/traceへ展開し、4操作の実プロセス往復で共通意味を比較する。

### CAPI-3（[#3421](https://github.com/hat47x/tei-platform/issues/3421)） HTTP/Web/Desktopの同一操作

実HTTP Adapterと実Desktop IPCに同じ4操作を接続する。WebやDesktopに別の業務処理を実装せず、Hostのallowlist解決後に同じ共通APIへ渡す。HTTP fixture、IPC fixture、実ブラウザー/実WebViewの確認を別に記録する。リモートクライアントとプロセス内クライアントを呼出し元で選べるかも確認する。

### CAPI-4（[#3422](https://github.com/hat47x/tei-platform/issues/3422)） 変更候補・受理・実行

既存のProposal/Shared Acceptance/Command/Service Contractへ接続する。受理のowner・期待版・scope・現在権限・Operation ID・再送可否・観測receiptを省かない。接続失敗時にlocalへ切り替える挙動や、効果不明時の自動再実行を禁止する。CAPI-1と並行して負例を準備できるが、read-onlyの成功だけを根拠にwrite入口を公開しない。

## 5. 同値性の受入試験

同値性は「同じJSONが返った」だけで判断しない。wire形式を各Adapterの契約で復号し、共通の型へ戻して比較する。意味・分類・診断・版は完全比較し、除外する観測時刻やrequest IDは明示する。snapshotは原子的な読取りや受理の証明ではない。

| ID | ケース | 共通で確認するもの |
| --- | --- | --- |
| T01 | 正常読取り | Canonical内容・宣言件数・Workspace identity・World/Sourceの指紋 |
| T02 | 警告付きvalidate | 操作完了/検証合格/severityを分離し、警告とsourceを落とさない |
| T03 | 意味エラー・構文エラー | 不合格・source範囲・診断code。World未成立時に架空の版を付けない |
| T04 | 不在・重複・未知namespace・未対応kind | それぞれのfailure.codeと分類。空の成功にしない |
| T05 | 未知欄・型不正・不在/NULL・引数既定値 | 入口で黙って補正せず、共通契約どおりに拒否/既定化する |
| T06 | traceのページ継続・stale | 対象/方向/版を結ぶ外側tokenとnextOffset。不正tokenを拒否 |
| T07 | 部分結果・未解決・未対応入口 | Unknown/unsupported/NOT RUNを隠さず返す |
| T08 | 主体偽装・無権限・失効・localからの迂回 | 共通API経由でも既存ownerが拒否。executorが未到達 |
| T09 | 書込み・名前付き出力・read-only拒否 | 宣言した作用と実ファイル/Provider作用の一致。writeFlagsとschemaを照合 |
| T10 | stale受理・効果不明・重複送信 | 拒否/unknown/再送可能を区別。二重作用を起こさない |
| T11 | deadline・取消・通信断 | 取消がrollback証明にならない。効果不明を保持し、local fallbackしない |
| T12 | 入力/出力上限・秘密情報・root置換 | 上限と開示方針を各入口で強制し、root外を読まない |
| T13 | 旧CLI/witnessとschema移行 | v0のbyte証拠を無断変更せず、別version/移行告知/並行期間を検査 |

試験結果には exact commit、入力byte、共通APIの契約version、各実行物、argv/URL/IPC method、実行環境、期待/実値、PASS/FAIL/NOT RUNを記録する。API直呼び→CLI/MCP→HTTP/IPC→実UIの順に証拠を分ける。実装していない入口の同値性を推定でPASSにしない。

## 6. 互換性と採用しない案

既存のCLI JSON v0、text/exit、保存済みwitness、MCP参照出力を無断で包み直さない。共通結果への内部移行は互換projectionで行う。既存欄の削除・型・意味・失敗分類を変えるなら、新しいwire version、読み手の一覧、移行期間、廃止条件をIssue/PRに書く。単なる内部型の追加は全入口の契約更新を意味しない。

採用しない案を次に示す。

- 全入口をHTTPサーバー必須にする案。local/組込み/オフライン利用とADR-0043に不要な配備依存を加える。
- Web/MCP/DesktopがCLIをshell実行してtextを解析する案。開発時のホストToolからCLIを使うことは許すが、製品Adapterの共通実装にはしない。
- 各入口で同じ判定を複製し、テストの定数だけ揃える案。上記MCPの差を検出できない。
- 全操作を一括で汎用envelope・Registry・新Frameworkに載せ替える案。代表4操作から段階的に移す。

## 7. 既存判断との整合と残る選択

ADR-0031のGit-native開発、ADR-0036のPlugin構成、ADR-0043の対象環境固有成果物、ADR-0044の認証非内蔵を維持する。共通APIはこれらのownerを迂回しない。開発用操作と生成された業務アプリの操作は同じ設計原則を使うが、別のscope・権限・公開面を持つ。#2093のApplication MCP Targetを開発用MCPで代替しない。

MCPも共通APIの正式な接続対象として採用し、最初の公開範囲は既存の4つのread-only操作とする。#3400の「作るか」の判断はここで解消する。製品としての公開・権限・protocol準拠の実装完了は #3420 の証拠で別に判断する。

本ADRで決めないものは、HTTPの具体的なrouteとREST/RPCの選択、Desktop framework、全操作のwire schema、SDK公開方法である。ただし、これらを理由にCAPI-1/2を止めない。既存方式を既定として代表4操作の証拠から選ぶ。共通APIの採用は決定済みであり、HTTPへの全面集中を再検討する前提にはしない。

本変更はIssue/ADRと案内の更新のみ。実装の完成は各子Issueの受入証拠で判断する。
