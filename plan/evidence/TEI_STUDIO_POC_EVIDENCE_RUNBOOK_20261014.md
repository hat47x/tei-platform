# TEI Studio PoC A/B/C 検証証跡取得手順（開発#8）

- 作成日: 2026-10-14
- 状態: **実行手順案。未採用・未実施・未判定**
- 対象: [Studio協調 Issue #1824](https://github.com/hat47x/tei-platform/issues/1824) の PoC A/B/C
- 目的: 開発担当から独立した観測者が、対象版・実環境・失敗条件をそろえて根拠を取得できるようにする
- 所有境界: #8 は証跡取得手順と負例整理を担当。PoC A/B/C 本体・契約・Resolver・UI Kit・Host の実装は所有しない

**この文書は製品要求・設計判断・新しい Gate・エビデンス schema の正本ではない。** 受入条件は #1824 と参照先の既存正本に従い、既存条件が変わったときはその参照を追跡する。実行可能なコード、固定済み CLI オプション、新しい型・Core kind・UI Registry を規定しない。表の試験 ID はこの手順内だけの識別子である。

## 1. 実施前に固定する情報

別々の実験の結果を一つの PoC の成立証拠に誤結合しないよう、**試験実施ごと**に次を記録する。値を取得できない場合は「未取得」と明記する。

| 項目 | 記録する内容 |
| --- | --- |
| 実装版 | リポジトリ、exact commit SHA、branch、dirty/clean、並行 PR・追加 patch の有無 |
| 依存とビルド | Go、Node、パッケージマネージャーと lock digest、Wails v3 の exact version、Go/TS/frontend build のコマンドと終了コード |
| 実行環境 | OS/architecture、Desktop WebView の実装・版、Webブラウザー・版、loopback/公開ネットワーク、実験で許可した利用者・接続元 |
| 入力と受理 | Workspace の版・パス（公開時は適切に伏せる）、Source revision / fingerprint が関わるか、CLI / Host が指す同一入力の同定方法 |
| 操作の追跡 | 検証ケース ID、実行者、実行時刻と timezone、再現操作、期待される観測、実際の観測、ログ・録画・スクリーンショット・テスト実行結果への参照 |
| 判定の根拠 | PASS/FAIL/NOT RUN/NOT APPLICABLE、結果不一致と原因、運用範囲の制約、後続の所有者・Issue/PR |
| 証拠の種別 | static review / automated test / real Go service / real browser / real Desktop WebView / human value review を分離 |

スクリーンショットだけで実サービス呼出しを確認したと扱わない。モック応答・単体試験と、実 Workspace に対する CLI/Go/ブラウザー/WebView の観測を区別する。テスト用固定データの使用と実利用者による価値確認を区別する。

## 2. ベースラインと差異の比較方法

1. PoC A 担当が確定した Workspace を用いて、比較対象の CLI 操作を **同一 revision** で実行する。現行の参照例は `go run ./cmd/tei validate -workspace reference/workspace`。採取時の CLI オプションは対象コミットで `-help` と実装を確認する。
2. 同じ Go Operation を経由した Web と Desktop の操作を実施し、結果の業務上の意味を突き合わせる。例えば workspace/subject の識別、検証の成功・失敗の分類、診断対象と理由を比較する。時刻、描画、動的な識別子、表示順のような非意味差は**先に根拠を示した場合に限り**除外する。
3. 観測された UI 応答、ネットワーク・Host 経由の実呼出し、サービスの結果を対応づける。失敗が UI に隠れたり誤って成功として表示されないことを確認する。
4. Web/Desktop で同じ結果になっても、認証・権限・ファイル I/O 境界の同等性まで推定しない。

## 3. PoC A — 共通サービスと二つの Host

各項目は実装完成後に実施する。未実装の認証・マルチユーザー分離を試験できない場合は「限定した loopback/read-only のみ」と書き、公開 Web の成立を宣言しない。

| ID | 操作または失敗入力 | 必要な観測と失敗判定 |
| --- | --- | --- |
| A-01 | 固定 Workspace を CLI で検証・参照 | 成功/失敗分類と入力の同定を保存し、比較基線を得る |
| A-02 | **実ブラウザー**で同じ操作 | Go Operation 実呼出し、UI表示と A-01 の意味一致。静的な画面だけでは不可 |
| A-03 | **実Desktop WebView**で同じ操作 | A-02 と同じ Operation / TS UI 資産の利用、起動環境と A-01 との意味一致 |
| A-04 | CLI/Web/Desktop へ同じ不正 Workspace を渡す | エラーの隠蔽、別データへの暗黙 fallback、成功誤表示がない |
| A-05 | 二つのブラウザー接続で異なる Workspace を選択 | 各要求・選択・結果・path が混線しない。tab ID を認証された利用者 ID とみなさない |
| A-06 | 別 scope の Workspace/操作を要求 | Host で許可対象を検証し、許可されない request を拒否。Web 単一利用者限定の場合も許可範囲を明示 |
| A-07 | ブラウザー間のイベント監視、再接続 | 機微な path/検証結果/Draft/権限内容が他接続へ broadcast されない。非機微通知だけなら再取得時の scope 保護も確認 |
| A-08 | session 失効・再接続・購読の再開 | 古い資格/接続で非公開結果を取得できない。認証を実装していなければ公開 Web 判定は不可 |
| A-09 | Desktop に許可 root 外・相対 path 逸脱・symlink・置換を試行 | Go Host の resource boundary で拒否。renderer や file picker の戻り値を権限とみなさない |
| A-10 | WebView 再起動と複数窓の状態遷移 | 選択 Workspace と Draft marker の混同、意図しない再送/受理がない。実装しない操作は N/A と理由を記録 |
| A-11 | 編集・受理が追加されたとき、stale revision/Source fingerprint 不一致を与える | 既存 Preview/Acceptance と Host 権限で拒否。**初期 read-only では N/A** |
| A-12 | 依存/ライセンス/再接続の不適合を記録 | Wails の機能や単体成功を全 OS・全環境の成立へ広げない |

Wails Server Build のイベント仕様には複数ブラウザーへの配信が含まれるため、**Broadcast を安全な利用者別通知と推定しない**。各イベントの payload と購読経路の観測が必要。初期は read-only のローカル単一利用者向けとして試験できるが、それだけで公開 Web 適合とはならない。

## 4. PoC B — TEI宣言からの静的合成

PoC A の**通常ビルド**を比較元とし、宣言合成の Builder は既存 Contract/Provider/Target 解決を利用する。#951/#1016 の移行中の契約を正式標準とみなさない。

| ID | 操作または失敗入力 | 必要な観測と失敗判定 |
| --- | --- | --- |
| B-01 | TEI宣言から 2 Go service と 2 UI部品を静的に組込む | 宣言・解決対象・登録・実動作の対応を提示する。単なる名前列挙は不可 |
| B-02 | 同一宣言・lock・依存で複数回生成 | 解決と登録順の決定性を確認。生成物の byte 一致の有無と、意味の一致を別に記録 |
| B-03 | 未知の部品/Provider を参照する | 明示的拒否。暗黙の別 Provider 選択をしない |
| B-04 | 同一ID・同一slot・不適合する置換を与える | 重複/slot衝突を検出し、静かに後勝ちさせない |
| B-05 | 依存欠落と循環を導入する | 順序の決定不能を検出して拒否する |
| B-06 | version / fingerprint / scope / lock の不整合を与える | 不適合を黙認しない。既存の検証機能が使われたことを確認 |
| B-07 | 通常ビルドと宣言合成ビルドで同じ操作を実行 | CLI/Host の意味結果が等価。Build方法の違いを利用者データ差と混同しない |
| B-08 | 宣言・lock・Provider・assets・生成物の来歴を追跡 | 対象 exact SHA と依存を再同定できる証拠。digest 未取得は未取得と明記 |
| B-09 | 任意 Go 共有ライブラリを要求する | 初期 PoC B の非目標として扱い、動的ロード成立を主張しない |

新たな Core enum、第二 Resolver、動的 Go 共有ライブラリロードを PoC B の成功条件にしない。

## 5. PoC C — Studio最小画面と別 consumer

| ID | 操作または失敗入力 | 必要な観測と失敗判定 |
| --- | --- | --- |
| C-01 | 同一実画面で Workspace一覧・Editor・Command・Graph を操作 | 見た目だけでなく選択と表示が実 Go Operation に結びつくこと |
| C-02 | Editor/Graph/Command が同じ選択対象を扱う | 選択遷移が UI 内の第二 Source 正本を生成しない |
| C-03 | 不明Command、対象不適合Command、Operation拒否 | busy / denied / failure を適切に表示し、成功扱いしない |
| C-04 | read-only 表示の編集操作を試みる | 保存・副作用が発生しない。編集対応時は既存 Preview/Acceptance 経由を別に検証 |
| C-05 | **TEI意味モデルを持たない別アプリ**へ共通UI部品を組込む | 異なる2 consumerで使われ、TEI固有import/意味ロジックへの依存を作らない |
| C-06 | 別アプリ側で Host/部品差替え・エラー・権限拒否を試す | 同一契約とライフサイクルで振る舞うか観測。Web/Desktopの2 Hostだけでは2 consumerとは数えない |
| C-07 | キーボード操作・フォーカス・アクセシビリティと依存版を確認 | 使用した Editor/Graph OSS の版・ライセンスと操作できない範囲を記録 |

高度 LSP/DAP、拡張 Marketplace、AI Agent連携、完全マルチユーザー化を C の受入条件へ追加しない。

## 6. 証拠の整理と判断

- **試験の担当者**: 各 PoC 本体担当が再現環境・実行入口・fixture を提示し、#8 が独立した負例・意味結果の突合方法と欠落を確認する。実機を操作した者とレビューした者を区別する。
- **合格しない例**: source の存在だけ、mock テストだけ、単一 Host だけ、別版の Web と Desktop の結果を結合、Wails の公式文書だけ、実画面を操作していない画像だけ、Studio 専用の部品を Web/Desktop で使っただけ。
- **記録方法**: #1824 の該当担当コメントまたは PoC PR に、ケース ID、対象 exact SHA、入力同定、コマンド/手順、期待値、実際の観測、証拠リンク、PASS/FAIL/NOT RUN/NOT APPLICABLE、残課題を記録する。**未実行のチェックリストを PASS としない。**
- **採否**: A の実環境・安全境界が成立せず修正不能なら Wails Server Build 一体方式の採用は保留し、共通 Operation/UI を維持した別 Web Host を検討する。Desktop の問題は Desktop Host の差替えを検討する。B/C の未達を A 成功で代替しない。
- **他 Target への波及**: Studio 上の Wails Host 成立を #259/#1182 の生成業務 Desktop Target 合格へ転用しない。

## 7. 現在の取得状況

**2026-10-14時点、#8が確認した範囲では本手順の A/B/C ケースはいずれも NOT RUN。** 本ファイルの追加は自動検査・PoCビルド・実ブラウザー・実WebView・人による利用検証の実施を意味しない。実装や結果の報告を受けたら、対象の版を再取得してから対応する証拠を採取する。

## 参照（各内容の正本はリンク先）

- [協調用 Issue #1824](https://github.com/hat47x/tei-platform/issues/1824): PoCと担当境界、受入条件。
- [共通 Operation #1158](https://github.com/hat47x/tei-platform/issues/1158)、[Draftと復旧 #1529](https://github.com/hat47x/tei-platform/issues/1529)、[三視点編集 #1124](https://github.com/hat47x/tei-platform/issues/1124)。
- [UI Composition Kit #997](https://github.com/hat47x/tei-platform/issues/997)、[Contract/Provider移行 #951](https://github.com/hat47x/tei-platform/issues/951)、[Application Composition #1016](https://github.com/hat47x/tei-platform/issues/1016)。
- [生成Desktop Target #259](https://github.com/hat47x/tei-platform/issues/259)、[参照実装の現状](../../docs/status.md)。

## 8. #8 別consumer初回検証結果（2026-10-14、PoC Cの一部のみ）

**対象**: `reference/tei-studio-independent-consumer/`、[Draft PR #1825](https://github.com/hat47x/tei-platform/pull/1825)。以下の実行結果はローカル作業用複製でのNode試験とGitHub上のファイル再取得による照合であり、実ブラウザー/WebView、TEI Studio画面の受入証拠ではない。

| ケース/検査 | 結果 | 実行・観測 |
| --- | --- | --- |
| 別consumerの独立入口 | 実装あり、ブラウザー未確認 | TEIモデルを持たない読書ノート3件を選択・参照する `index.html`/`main.mjs`、共通部品領域は未提供時にBLOCKEDと表示 |
| データ変換と静的検査 | **単体検査PASS（7件）** | Node.js 22.16.0で `node --test *.test.mjs` および全 `.mjs` に `node --check` を実施。読書ノート選択・Graph入力変換・依存解析の正負例を検証（ローカル同内容複製） |
| TEI専用import | **限定した静的検査PASS** | `node audit-imports.mjs main.mjs notes.mjs journal-graph.mjs`。対象3モジュールのimport graphで該当依存なし。未生成のGraphモジュールや実際のbundleまでは監査していない |
| #9 Graph APIへの接続 | **コード接続準備あり、動作BLOCKED** | #9 `graph.ts` の実公開 `mountReadonlyGraph` に合わせて `graph-entry.mjs` を作成。ただしこの単体PR headには#9ソースがないため `node build-graph.mjs` はBLOCKED（exit 2）。Graph実描画は未実行 |
| #5 Command APIへの接続 | **コード準備あり、動作BLOCKED** | #5の実export `mountCommandPanel` へ `command-entry.mjs` で接続し、consumer側に読取り結果success／削除拒否deniedを置いた。原本は別branchで、本PR headでは `node build-command.mjs` がBLOCKED（exit 2）。実DOM呼出し未実行 |
| #4 Editor | **BLOCKED** | CodeMirror依存の確定・ビルド・実DOM接続未実施。代用部品・仮想APIを作らない |
| C-05 / C-06 の正式再利用判定 | **NOT RUN** | Studioでの同一共通部品実利用、別consumer実ブラウザー描画、差替え・拒否・unmount等の独立検証を実施していない |

#9 Graphの確認元: `feat/tei-studio-poc-c-graph-reference`（確認時head `e652a1b655c6dd5f3e52afde9390f1f6cd0771dd`）。そのbranchでのProviderの存在と、このPR headでの利用成立を混同しない。Draft PRに対してmerge操作を行わず、各依存PRのexact headから専用pathだけを検証専用checkoutへ取り出して固定snapshotを作る。各SHAと取得pathを記録してからSource/build/ブラウザー操作を再実行する。ここでPoC Cの合格・再利用成立を宣言しない。

#5 Commandの確認元: `feat/tei-studio-poc-c-command-diagnostics`（実exportは `mountCommandPanel`）。#4 Editorは `feat/tei-studio-poc-c-editor` に原本あり、CodeMirror依存の別途固定・ビルドが必要。#8単体のheadでは本体が利用可能とは扱わず、固定snapshotで検証する。

## 9. Windows固定snapshotの失敗と#8側修正（2026-10-14）

**検証の同一性を区別:** #WorkがWindowsで実行した**修正前**の固定入力は次のとおり。各Draft PRへのmerge操作は行っていない。対象の専用pathだけを検証用checkoutへ取り出す方式であり、元PRやmainの統合状態を変えない。

| 入力 | exact head |
| --- | --- |
| `main` | `128429482e02d96afe0123ee7da023256094dfca` |
| #1825 consumer（修正前） | `7714c79e7e28b335e6706af642efc9e7e285bfff` |
| #1828 Graph | `e652a1b655c6dd5f3e52afde9390f1f6cd0771dd` |
| #1829 Command | `26f47772ed49f5c4a9e1c50026c335783bc5e153` |
| #1830 Editor | `4970921e613c68e6da29fb603536f2e71fc2d462` |

### #Workが報告したWindows観測（修正前）

- 環境: Windows / Node v24.21.0。固定snapshotで **25 tests中24 PASS、1 FAIL**。
- 失敗: `audit-imports.test.mjs` の「sample imports contain no TEI-specific modules」。`auditImportGraph` が `ENOENT lstat 'C:\\C:'`。ファイルURLの `pathname` をWindows FS pathとして渡したことに起因。
- 同じsnapshotでは `build-graph.mjs` と `build-command.mjs` は共に成功（#Work報告）。これは実DOM操作・実ブラウザー・二つのconsumerの成立を証明しない。
- Windows実行結果は#Workからの報告であり、#8側では同じWindows環境を直接再実行していない。

### #8側で変更した原本とローカル再検証

- `audit-imports.test.mjs` の `repoRoot` を `fileURLToPath(new URL('.', import.meta.url))` 経由に変更し、Windows URL形式のドライブパス変換の負例を追加。
- 同種のWindows問題があった `check-components.mjs` のCLI実行判定も `fileURLToPath(import.meta.url)` に変更。CLIのBLOCKED終了コード2とimport CLI起動の回帰テストを追加。
- 検証環境: **Linux、Node v22.16.0**。同じ内容のローカル作業用コピーで `node --test *.test.mjs` **10/10 PASS**、`node audit-imports.mjs main.mjs notes.mjs` PASS、`node check-components.mjs` **exit 2（想定どおりBLOCKED）**。
- WindowsドライブURLの模擬変換（`fileURLToPath('file:///C:/...', {windows:true})`）はLinux上のNodeでもPASS。ただし**Windows OS上での回帰検査PASSを意味しない**。
- 修正した二つのファイルのGit blob SHAを、ローカル試験コピーとGitHub #1825の原本で照合し一致を確認。#1828/#1829/#1830とmainは変更していない。

### 未確認と次の検査

#Workの**修正後Windows v24.21.0での再実行はNOT RUN**（#8が確認した範囲）。次の固定snapshotでは#1825の修正後exact headへ差し替え、他の3PRのheadは固定したまま、`node --test *.test.mjs` とCLI起動・Graph/Command build・実DOM・実ブラウザーを段階的に検査する。結果の再取得前にPoC C合格や本番対応を宣言しない。

## 10. #8 別consumerのCodeMirror実接続・bundle準備（2026-10-14）

この項は**新しいGateではなくC-05/C-06に対する途中結果**。報告した元コードと、実行した環境・未実行を区別する。

### 非merge検証snapshotの現在の固定入力

| 対象 | exact SHA | 取得対象 |
| --- | --- | --- |
| main baseline | `128429482e02d96afe0123ee7da023256094dfca` | base |
| #1825 consumer | **このPRの検証時最新head**（次のコメント/PRにexact SHAを記録） | `reference/tei-studio-independent-consumer/` と本手順 |
| #1828 Graph | `36799dba33fe0e19db3f7b56e7af1e18da134a71` | `reference/tei-studio-graph/` |
| #1829 Command | `26f47772ed49f5c4a9e1c50026c335783bc5e153` | `reference/tei-studio-command/` |
| #1830 Editor | `a7084fe1711cc48eff56fd20e3dc81c508f520a3` | `reference/tei-studio-editor/` |

すべてのDraft PR原本は保持し、通常mergeは実施しない。Graphは修正前 `2d7148c3…` からポインター中心クリックを直したheadへ、Editorは修正前 `4970921e…` からCRLF座標変換を直したheadへ更新した。両旧headの失敗を新headのPASSに置き換えない。

### 今回#8が変更したconsumer

- `editor-entry.mjs`: #4の**実ソース** `createReadOnlyTextEditor` + `codeMirrorRenderer`をimport。`journal-editor.mjs` がノートIDから `{id,version,text,diagnostics}` を作る。初期表示後、picker切替で `setDocument`、ボタン/ページ破棄で `destroy`。書込みAPIは供給しない。
- `graph-entry.mjs` / `command-entry.mjs`: #9/#5の実TypeScriptをstatic importし、**バンドル後のブラウザー**から使用する。既存の型除去 `build-graph.mjs` / `build-command.mjs` は同一snapshotでのsource digest確認用に保持。
- `package.json`: consumer配下だけで `esbuild@0.25.12` とCodeMirror直接依存3件（state `6.7.6`、view `6.43.14`、lint `6.9.7`）をexact指定。frontend root manifest/lockには変更なし。
- `bundle.mjs`: esbuildの実bundling（`nodePaths`でconsumer専用 `node_modules` のbare imports解決）。Editor/CodeMirror/Graph/Commandを実入力として含めたかmetafileで確認し、TEI意味・Host専用importを拒否。生成物 `dist/index.html` / JS / `build-provenance.json` を作成し、元ソースSHA-256を記録。
- `journal-editor.test.mjs`: document projectionの正常/拒否。#1830の実 `editor.ts` が同一snapshotにある時だけPortの `setDocument`、同一版別本文拒否、`destroy`冪等を検査。
- `browser-e2e.py`: 実Chromium+Playwrightで、CodeMirror編集不可・本文切替・Graph内幅480と14pxラベル/横スクロール・ノードclick→picker/Editor・Command success/denied・destroy/disposeを検査する**実行可能な手順**。モック部品は使用しない。

### 実測と未実施

| 検査 | 観測 | 結果の区分 |
| --- | --- | --- |
| ローカル簡易document projection | Linux Node v22.16.0でノート `map` / `sketch` のID/本文と不明ID拒否を直接検査、3アサーション成功 | **限定したローカルPASS**。共通Editor実描画ではない |
| offline lockfile試行 | Linux npm v10.9.2で `npm install --package-lock-only --offline --ignore-scripts --no-audit --no-fund` → `ENOTCACHED`（`@codemirror/lint` metadataがcacheにない） | **BLOCKED**。lockfile未生成、推移依存は未固定 |
| esbuild実bundle | 同一snapshotでnpm依存が未導入のため `npm run build` を完走できない | **NOT RUN**。sourceコード追加だけでbuilt assetsの成立を主張しない |
| 実Chromium / CodeMirror接続 | `browser-e2e.py`を用意したが、上記bundle未生成 | **NOT RUN**。実DOM合格ではない |
| Graph旧幅対応 `2d7148c3…` | #Work独立ブラウザーで `clientWidth=225 / scrollWidth=480 / svgWidth=480 / fontSize=14px`、キーボード選択連携を確認。ノード中心のポインター選択はFAIL | **#Work報告の旧headの結果**。#9新 `36799dba…` の再検証待ち |
| Editor旧 `4970921…` | #Work実CodeMirrorではCRLF診断と選択オフセット不整合が再現。#4が `a7084fe1…` で修正を提出 | **旧headのFAIL**。新head未再検証 |

**次の検証実行**: 固定snapshotのconsumer配下で `npm install` →生成lock内容・依存版/版のdigest採取→ `npm test` → `npm run build` → `python3 browser-e2e.py`。Node版、npm版、Playwright/ブラウザー版、実出力、DOMスクリーンショット、入力SHAを保存する。依存が揃った時点でのみC-05/C-06を判定し、Studio側の共通部品利用の実証がなければ2 consumer成立を主張しない。

## 11. StudioとReading Journalの同一共通部品を用いた別用途検証（2026-10-15）

**位置づけ**: この節は既存PoC C-05/C-06の実装・証拠更新であり、新たな受入Gateではない。§10の「consumer lock未生成」は**当時の状態**であり、下記のWindows取得と#1825原本への反映で解消した。Draft PRのmerge・main統合・他担当原本変更は実施していない。

### 11.1 各snapshot・主体の分離

| 証拠 | 固定入力 | 環境・実行主体 | 観測 |
| --- | --- | --- | --- |
| Studio実Host+ブラウザー | Workbench #1849 `9a290dc61f585dfbf292e9032e5c5bbbad04351a`; Host #1832 `d520689cfb0a2e2919aa86b0beeba54802b31465`; Editor #1830 `cc57052f23b69582fc0d3800ab92b2210a28bb4f`; Graph #1828 `36799dba33fe0e19db3f7b56e7af1e18da134a71`; Command #1829 `26f47772ed49f5c4a9e1c50026c335783bc5e153` | **#Work** / Windows Node24.21.0、Go1.26.9 Linux-native Wails Server、実ブラウザー | 実Host接続、一覧/検索/Workspace切替、Graph選択→InspectSubject、Command成功を確認。<https://github.com/hat47x/tei-platform/issues/1824#issuecomment-6181206457> |
| Reading Journal別consumer実ブラウザー | Consumer #1825 `8cb42899ca2799c4ac4082db51412a8d45a3d6a1`; Editor #1830 `a7084fe1711cc48eff56fd20e3dc81c508f520a3`; Graph #1828 `36799dba33fe0e19db3f7b56e7af1e18da134a71`; Command #1829 `26f47772ed49f5c4a9e1c50026c335783bc5e153` | **#Work** / Windows Node24.21.0、esbuild0.25.12、CodeMirror state6.7.6/view6.43.14/lint6.9.7、実ブラウザー | `npm test` **12/12 PASS**、`npm run build` **PASS**。Tea tasting切替→CodeMirror本文連動、Graph click/Enter→picker/本文/Editor連動、Command成功・拒否、Editor unmount→reload後表示を観測。<https://github.com/hat47x/tei-platform/issues/1824#issuecomment-6181303406> |
| #8自身の原本検査 | #1825 `test/tei-studio-poc-evidence`（現行headはIssue追記時に明示） | **#8** / GitHub原本取得・構造検査 | Editor source 3ファイルのGit blob SHAがStudio/Journal両snapshotで一致、GraphとCommandの固定headも一致。lockのSHA-256を共有本文と照合後、原本へ登録。**#8による新E2E実行ではない**。 |

実際のEditor source 3ファイルのGit blob: `editor.ts=146c4fb631a55913a3f0e0aeb82043096f5b118e`、`codemirror.ts=194daf0d7354cf46163d28b3e2382abf4a227575`、`codemirror-coordinates.ts=686a54a685e0078975623e2ed130b2c33f33bc89`。#1830のsnapshot commit SHA（`cc570…` / `a708…`）は異なるが、**実際にimportする同じEditor実装source**である。証拠の異なる2つのconsumer/Hostを「同一実行」とは呼ばない。

### 11.2 #8 consumer原本に反映した独立検査と依存

- **再利用の最小入力**: Editor は `{id,version,text,diagnostics:[]}`、Graph は `{width,height,nodes:[{id,label,x,y}],edges:[{id,source,target}]}`、Command は `{commands,diagnostics,invoke(commandId,signal)}`。Graphの選択callbackは読書ノートIDのみを返し、pickerの `change` を介してEditorの `setDocument` を動かす。Commandの結果意味と拒否はconsumer固有の `journal-command.mjs` で判断する。共通部品にTEI Workspace/Source/Acceptanceや書込み/Host権限を追加していない。
- **明示破棄と再表示**: #1825内 `editor-entry.mjs`、`graph-entry.mjs`、`command-entry.mjs` にそれぞれUnmount/Remount操作を追加。旧インスタンスの `destroy/dispose` とイベント解除、現在選択中ノートに基づく再生成、`pagehide`での最終破棄を接続。独立 `shared-ports.test.mjs` は実Graph/Command sourceをNodeで読み、最小データ・選択通知・破棄後抑止/再マウント・Command成功/拒否を検査する。ただし提供元source未取得の単体#1825では明示SKIPとなる。
- **実DOM自動検査**: `browser-e2e.py`へEditor/Graph/Command各自のUnmount/RemountとGraph Enter選択を追加。#Workが確認した旧consumerのページreloadとは**異なる新操作**なので、その旧実画面PASSを本新コードのPASSへ転用しない。
- **lock原本反映**: #Workが実npm導入で生成したconsumer用`package-lock.json`全文を <https://github.com/hat47x/tei-platform/pull/1825#issuecomment-6181305849> から取得。CRLFと末尾改行を維持してSHA-256 **`7cb7257ef6e71def71ddf5d24414be596832d6c1f3aef36015f51b340d1f18c1`** が一致することを#8が確認し、#1825の`reference/tei-studio-independent-consumer/package-lock.json`としてcommitした。lockfileVersion 3・35 packagesで、consumer `package.json` と依存/Node要件が一致。frontend root manifest/lockには変更なし。
- **ビルド来歴**: `bundle.mjs`でlock内のCodeMirror/esbuild exact版一致を検査し、`dist/build-provenance.json`へlock SHA-256と提供元TypeScript SHA-256を保存する。実esbuildでの新変更版の成否は未検査。

### 11.3 再現・未確認

固定入力snapshot（各PRの専用pathのみ、mergeなし）でconsumerのディレクトリへ移動し、`npm ci --no-audit --no-fund && npm test && npm run audit:imports && npm run build`。実Chromium/Playwrightを準備した環境で `python3 browser-e2e.py`。stdout/stderr、Node/npm/esbuild/Chromium版、`dist/build-provenance.json`、画面観測を保存する。

**#8の追加remount/Command分離/lock来歴検査を含む最新consumer headでのnpm ci・Node全件・実ブラウザーE2E: NOT RUN**（#8実行環境ではnpm依存をキャッシュ・レジストリから取得できない）。#Workの先行 **12/12 PASS・bundle PASS・実ブラウザー観測** は上記旧consumer固定版に限定。#8は新実DOMの合格、Desktop WebView、公開Web認証、保存権限、アクセシビリティ全般、PoC C全条件、製品採用を主張しない。引き続き#Work等の独立実行結果をexact revision付きで追加する。

### 11.4 #8自身の最小入力検証（2026-10-15）

**環境**: Linux container / Node.js `v22.16.0`。GitHub #1825原本の `notes.mjs`（Git blob `243ea07c9584f385271ed665f7fd6290baddc931`）、`journal-command.mjs`（`c98909154ba2e8f6d8abcbc798dd5ac613105fcb`）、`journal-command.test.mjs`（`ee756285b958a48e4379e89da39b6f280b33ffd4`）を隔離ローカルディレクトリへ同内容コピーし、`git hash-object`で各blob SHAがGitHub原本と一致することを照合した。

**実行**: `node --test journal-command.test.mjs` → **2/2 PASS、fail 0、skip 0、exit 0**。ノートtea→sketchに応じた `2 related entries`→`0 related entries` の変化、表示上 `enabled=true` でも削除が `denied`、未知commandの拒否を実測。これは**#8独立実行のconsumer側入力/拒否試験**であり、実Command DOM/権限Hostの検証ではない。

**新規で追加したが未実行の範囲**: `shared-ports.test.mjs`の#1828/#1829実ソースを使ったGraph最小入力・click/Enter・dispose/remountとCommand Portの成功/拒否、更新後`browser-e2e.py`の3部品のUnmount/Remount連鎖。これらはProviderのexact pathを抽出した検証snapshotと実依存/Chromiumが必要で、#Workの旧固定版12/12 PASSへ含めない。次の独立実行でPASS/FAILを更新する。

### 11.5 #Work独立実行：新版Graph/Commandと別用途アプリの三部品再表示（2026-10-15）

**受領元**: [Issue #1824 / #Work実行報告 #6181785625](https://github.com/hat47x/tei-platform/issues/1824#issuecomment-6181785625)。以下は**#Workによる固定snapshotの独立実測**であり、#8が別環境で実行した成績ではない。§11.3/§11.4の「未実行」はそれぞれ当時の状態・#8自身による実行の範囲を示すもので、以下の後続#Work検証とは区別する。**Draft PRをmergeせず、各固定原本のGit blob生バイトを検証専用snapshotへ抽出**。元branchと他担当原本は変更しない。

#### 11.5.1 Reading Journal / 別用途の固定入力と環境

| 依存元 | exact revision |
| --- | --- |
| consumer / Draft PR #1825 | `78f7cb55f3fdf23c720379f937feba1c4a328cd9` |
| Editor / Draft PR #1830 | `a7084fe1711cc48eff56fd20e3dc81c508f520a3` |
| Graph / Draft PR #1828 | `786d8d82e19e8b08690aceadf9d180374e211448` |
| Command / Draft PR #1829 | `01f5022240fd86c6ac38c1107310e299a1615b48` |

**環境**: Windows、Node.js **24.21.0**、consumer専用lockfileを使った `npm ci`、esbuild **0.25.12**、CodeMirror `@codemirror/state@6.7.6` / `@codemirror/view@6.43.14` / `@codemirror/lint@6.9.7`。実ブラウザーは#WorkのIAB操作環境で、loopback **`127.0.0.1:4177`** の生成済みdistを開いた。**`browser-e2e.py`自体は未実行**。

| 検査 | #Workの観測と判定 |
| --- | --- |
| 依存取得と固定 | **PASS**: lockを使用した `npm ci` |
| consumer自動試験 | **17/17 PASS**: 実Graph/Commandの最小契約と、dispose後の古い選択イベント拒否を含む |
| consumer実バンドル | **PASS**: 実Editor / Graph / Commandを含む最新版dist |
| 3部品のUnmount | **PASS（実DOM）**: Editorの `.cm-editor`、Graph group、Command regionがそれぞれ0 |
| Unmount中のノート切替とRemount | **PASS（実DOM）**: EntryをTea tastingに変更後、三部品を再表示。Editor本文とGraph選択が現在ノートを反映 |
| Commandの成功 | **PASS（実DOM）**: Countで `成功: 2 related entries` |
| Graphキーボード選択 | **PASS（実DOM）**: Window sketchでEnter→Entry・見出し・Editor本文が同期。省略ノード補足欄に `ラベル全文: Window sketch` を表示 |
| Commandの拒否 | **PASS（実DOM）**: Deleteで `拒否: The reading journal does not permit deletion.` |
| 自動Chromium E2Eスクリプト | **NOT RUN**: #Workはブラウザー操作ツールを使用。Python `browser-e2e.py`のPASSとは呼ばない |

#Workの画面証拠: `outputs/reading-journal-lifecycle-20261015.jpg`（#Work実行環境内）。これは**Reader用途での同一共通部品再利用**の根拠であり、StudioのSource/Acceptance/実Host契約、Desktop WebView、全フォント・スクリーンリーダー・アクセシビリティ、完全なメモリリーク検査、利用者価値の合格を意味しない。#8側でこの画像を再取得・独立撮影したものではない。

**再現入口**: 上表のexact SHAの各所有pathだけを非mergeで新しいsnapshotへ抽出後、`cd reference/tei-studio-independent-consumer && npm ci --no-audit --no-fund && npm test && npm run build`。生成したdistをloopback配信し、三部品Unmount→ノートTea tastingへ切替→Remount→Count→Graph Window sketch Enter→Deleteを順に検査する。自動E2Eの別実行をする場合は `python3 browser-e2e.py` の結果を区別して記録する。

#### 11.5.2 Studio実Host / 新版Graphは別snapshotで未確認部分あり

#WorkはStudio側の**別固定snapshot**でもGraph新版 `786d8d82e19e8b08690aceadf9d180374e211448` を試験した。入力: Workbench #1849 `c55ad228`、静的合成 #1833 `fe8e37b1`、Command `01f5022240fd86c6ac38c1107310e299a1615b48`、Host #1832 `d520689cfb0a2e2919aa86b0beeba54802b31465`（各短縮SHAは#Work報告の表記。検証再現時は対象の完全SHAを再取得して固定する）。

- **PASS**: Graph **8/8 tests**、生成入口strict型検査、実bundle、Go **1.26.9** / Wails **beta29** / `CGO_ENABLED=0` / server / readonly build。
- **PASS（配信資産）**: loopback `127.0.0.1:43129` でServer起動ログとPID `27615`を観測。配信 `main.js` とdistのSHA-256一致：`1166d9489c76028ba21c9eaf44d25c6f36f98d5845d1f5a0599ea51bdfb1f1ee`。
- **NOT VERIFIED（Studio新版Graphの実画面）**: 既存tabのnavigationと後続CDP読取りがtimeout。**ブラウザー操作timeoutはServer停止やGraph失敗の証明ではない**。#WorkはServerを再起動せず、同Server/tabでのGraph選択・更新UI確認を保留。
- **別件の実行手順上の失敗を分離**: シェル起動用ファイルを誤ってCRLF保存して最初の起動手順が失敗したが、LFへ修正後に実行成功。原本実装や成果物のFAILではない。

**結論の境界**: 新版Graphの**Reading Journalでのブラウザー実操作はPASS**。同じGraphを組み込んだ**Studio新版Graphの実ブラウザーは未確認**。過去の#1849 Studio実Hostブラウザー成功（別Graph revision）の証拠も消さず、今回の新版GraphがStudioで動いた証拠として流用しない。#2/#1の生成Go実Host内部接続とDesktop実WebViewは残課題であり、PoC全体は未完了。

##### 後続の別固定snapshot：生成Go Host経由の新版Graph実画面を確認（#Work / 2026-10-15）

**受領元**: [Issue #1824の#Work報告 #6181914721](https://github.com/hat47x/tei-platform/issues/1824#issuecomment-6181914721)。**この段落は先行する `d520689…` Host・`127.0.0.1:43129`・CDP timeoutの「NOT VERIFIED」を上書きしない後続の独立snapshot**である。#8自身の再実行ではない。各Draftの所有pathを隔離snapshotへ抽出し、mergeしていない。

| 後続snapshotの入力 | exact revision |
| --- | --- |
| Host #1832 | `4bc437c72e63a3b5d578b1efe2c60be510a9f8a9` |
| Static composer #1833 | `fe8e37b1c735dc138c36cb2d37db94ee44424b9e` |
| Workbench #1849 | `c55ad2285268533f5f1c90fbc9fe766bae57f009` |
| Editor #1830 | `a7084fe1711cc48eff56fd20e3dc81c508f520a3` |
| Graph #1828 | `786d8d82e19e8b08690aceadf9d180374e211448` |
| Command #1829 | `01f5022240fd86c6ac38c1107310e299a1615b48` |

**固定生成物**: composer生成 `services.generated.go` のSHA-256は **`867501c5a98d71337178d8bd7995e2e4eb6d276accd25151a2c18d8cb5adee50`** で、#2 CI報告のものと一致。Workbench生成UI bundleのSHA-256は **`1166d9489c76028ba21c9eaf44d25c6f36f98d5845d1f5a0599ea51bdfb1f1ee`**。生成モードWails Serverから配信されたbundle SHAも後者と一致した。

**実行環境とGo検証（実行主体：#Work）**: WSL Linux、Go **1.26.9**、Wails **v3 beta.29**。CGO無効の `go test -count=1 -v -tags 'studio_generated,server' .` における **`TestGeneratedGoServicesRunThroughRealHost` PASS**、`go vet` PASS、生成モードServer build **PASS**。ただし **`-race` 試験はNOT RUN / BLOCKED**：実行環境に `pkg-config` がなく開始できなかったためであり、race検査PASSともrace由来の欠陥とも扱わない。

**標準Host経路の追加検証（#Work続報 [#6181937969](https://github.com/hat47x/tei-platform/issues/1824#issuecomment-6181937969)）**: 同じ後続固定snapshot（Host `4bc437c72e63a3b5d578b1efe2c60be510a9f8a9`、composer `fe8e37b1c735dc138c36cb2d37db94ee44424b9e`）に対し、生成タグ付き実Host試験とは**別に**、`CGO_ENABLED=0 go test -count=1 ./internal/studiohost/...` および `go vet ./internal/studiohost/...` を実行し、**いずれもexit 0 / PASS**。allowlist・Factory・読取りHostのpackage suiteを含む。**Wails本体を含む `cmd/tei-studio` の `-race` 試験は `pkg-config` 不足でローカル未実施**、非race版PASSとの区別を維持する。これはWeb Server版のHost検証であり、Desktopの実起動・終了確認を補完した証拠ではない。

**実Wails Server＋実ブラウザー（IAB・`127.0.0.1:43130`）**: 生成タグを使用するServerへ接続した固定UIについて、以下を**実DOM操作PASS**として受領した。

- workspace-1 の生成Catalog **72件**、Validate成功、警告 **3件**を表示。
- workspace-2 へ切替後、Catalog **19件**、Validate成功、警告 **0件**。workspace-1の診断を残留表示しない。
- workspace-1へ戻り、新版Graphの **customerId** ノードを選択。実Go Hostから **`sales.order.customer_id`** の投影を取得し、Graphが直接親 **Order** との**2ノード関係**に更新されることを確認。
- 実配信bundleのSHA-256が上記固定UI bundleと一致。

画面証跡は#Workローカル `outputs/tei-studio-generated-host-20261015.png`。この後続snapshotは**新版GraphのStudio生成モードWeb Server実画面成功**と、生成GoのCatalog/Validateを実Hostから実UI経由で呼べた範囲を証明する。一方、旧 `d520…` / port 43129のブラウザーtimeoutは**その時点の未確認として記録を維持**する。結果を異なる入力・環境・ブラウザー操作間で合算しない。

**Desktop / OS境界**: Windows Desktopバイナリの通常・生成モードbuildは独立CI [#38662384542](https://github.com/hat47x/tei-platform/actions/runs/38662384542)で**PASS（buildのみ）**。**native WebViewの実起動・自然終了はNOT VERIFIED**であり、IABのWeb Server版の画面結果はDesktop UI・OS統合の資格ではない。全PoCの完成・Source本文取得/受理等を示すものでもない。

**再現方法と次の依存**: 表のexact revisionから各所有pathだけを新しいsnapshotに抽出。生成した `services.generated.go` と配信UI bundleを上記SHA-256で照合し、WSL Linux/Go1.26.9/Wails v3 beta.29で `go test -count=1 -v -tags 'studio_generated,server' .`、`go vet`、CGO無効の生成モードServer buildを実施。loopbackの生成モードServerでworkspace切替、Catalog/Validate、Graph customerId選択後の投影・直接親2ノード更新を確認する。`-race`の実施には `pkg-config` を含む必要なrace/cgo環境が別途必要。Desktop実WebViewは#2/#10側の独立確認とする。

本節は独立検証の受領記録であり、新しいADR/GateやStudio外のIssue対応を発生させない。Studio側の残件は既存担当が継続する。

## 12. Reading Journalの独立Graphフォーカス再描画検証（#8 / 2026-10-15）

この項目は既存のC-05/C-06に対する**追加の契約回帰**であり、新Gateではない。前節の#Work固定証拠（Graph `786d8d82…`）と今回の#8検証入力（Graph `87ab7cdc…`）を同一実行として扱わない。

**固定ソース**: consumer #1825の変更前head `178e8406f9bcb64f5432cd64e5baeb4198a78a27`（新規test差分は後続のheadで管理）、Graph #1828 `87ab7cdc4277ee70aebe535d92dca55fcbecb36f`（`graph.ts` Git blob `4048d35e7c4e1c76774a8b64eb7b0613d17ca0ff`）。Command #1829 `837e81d5d71d51aed484003dd4978da3c7cbe9b4` とEditor #1830 `1009e5f3407c985e00b3a3257b3cc5eee89db2ba` は後続の追加テストを含むが、既存Journalでimportする実装ソースについて今回の機能変更・独立Node全件再実行を主張しない。

**不足を見つけた場所**: Reading Journalの既存 `shared-ports.test.mjs` はGraphのclick/Enter・dispose/remountを確認していたが、選択通知内で `graph.update()` が同期的に走る場合の**フォーカス継続→次のSpace**を検査していなかった。#9は旧Graph `786d8d82…` でfocus落ちを実Chromium再現し、新 `87ab7cdc…` に修正済み（[担当の独立報告](https://github.com/hat47x/tei-platform/issues/1824#issuecomment-6182036391)）。この契約をJournal自身のノートIDでも保持するか追加確認する。

**#8変更**: 所有 `reference/tei-studio-independent-consumer/shared-ports.test.mjs` **1ファイルのみ**に、Fake DOMの`focus()`/activeElement追跡と、`journalGraph()`の実Graph Portを用いて`onSelect(id)`が同一journalグラフを同期`update`する回帰を1件追加。teaノードのEnter→DOM再描画→同IDの新ノードへフォーカス復元→Spaceで2回目通知、旧ノードからの遅延click不採用、外部buttonフォーカス中のbackground updateはフォーカスを奪わない、dispose後root空を検査する。新規意味モデル、TEI Subject、権限判断、Graph原本変更はない。

**#8自身の確認（今回の範囲）**: GitHub固定Graph `graph.ts` blobを取得し、**型注釈を一時除去したコードを単発V8 isolateで評価する縮小再現**では、`tea`のEnter→`graph.update`→新SVGのfocus復元、続くSpace通知（`events=['tea','tea']`）、旧SVGのイベント無効、外部フォーカス保持、disposeのDOM除去がいずれも期待どおりだった。これは**一時type-erasureしたロジックでの限定PASS**であり、正式なNode `npm test`、tsc、esbuild、実CodeMirror DOM、Playwright/Chromium、Wails/Go HostおよびDesktop WebViewのPASSとは区別する。旧Graph版でのREDは今回#8自身は再実行していない。

**正式試験の状態**: 新規 `shared-ports.test.mjs` を含む固定consumer snapshotでのNodeテスト全件と実Chromiumの再実行は **NOT RUN**（#8ローカルに元のGraph/Command/Editorソースを揃えた依存snapshotおよびnpm依存がない）。先行#Workが報告した **17/17 PASS** はconsumer `78f7cb55…` + Graph `786d8d82…` + Command `01f50222…` に限定し、今回の追加1件や新Graphへ転用しない。

**再現**: mainの同一基線から各Draftの所有pathだけを検証用snapshotへ抽出（Draft mergeなし）。consumerを本検証head、Graphを `87ab7cdc4277ee70aebe535d92dca55fcbecb36f`、Commandを `837e81d5d71d51aed484003dd4978da3c7cbe9b4`、Editorを `1009e5f3407c985e00b3a3257b3cc5eee89db2ba` に固定し、`cd reference/tei-studio-independent-consumer && npm ci --no-audit --no-fund && npm test && npm run build` を実行。`shared-ports.test.mjs`でGraph sourceが欠落した場合は該当テストがSKIPされるので、**PASS件数だけでなくskip=0とGraph provider pathの存在**を確認。Python `browser-e2e.py`と実生成Studio Host画面は別試験として扱う。

**依存と境界**: #9の新Graphは単体/隔離実Chromiumでフォーカス修復PASS、生成モードStudio実Hostでの新head Enter→Space連続操作は未確認。#2の別Desktop固定bootstrap native起動/終了証拠（[Issue報告](https://github.com/hat47x/tei-platform/issues/1824#issuecomment-6182081215)）もWorkBench Graphを載せたDesktopの資格とは別。PoC外のADR/Issue追加引継ぎなし。

### 12.1 固定Providerの合成Node/Chromium独立CI（2026-10-15）

**実行主体と固定入力**: #8が自己所有の新しい隔離検証workflow `.github/workflows/tei-studio-journal-evidence-8-ci.yml` をDraft #1825に追加し、GitHub Actions [run #38686075764](https://github.com/hat47x/tei-platform/actions/runs/38686075764)、job `116261107211` **completed / success**。これは前節の一時V8型注釈除去試行とも、#Workの旧consumer 17件PASSとも**独立した正式Node24/実Chromium実行**である。作業snapshotには以下の**各Draftの所有pathだけ**を取り出した（`git archive`でファイルのみ、Git mergeなし、他担当原本への書込みなし）。

| 資産 | 実行したexact SHA / 検査blob |
| --- | --- |
| consumer Draft #1825 | `c780696bfa5bf8a5c8ed6f9b5497ab6be9beac41`（Workflow自体もこのheadに含む） |
| Graph Draft #1828 | `de9ef8f0ac40b164d60ad5d43637ba3afc585c5e` / `graph.ts=436c7d07c56802dbbc6d1dfeb1251470fca75cf6` |
| Command Draft #1829 | `63f7ce3af1e6e5a27124aa4aa8ae1331d405f83c` / `command.ts=bb56f3dba6387f7d9bf9604de9b519fb68feb50d` |
| Editor Draft #1830 | `472c5d90b0b196f44b203e710c88ff05821d74fa` / `codemirror.ts=eddc3c3e9320b9f61a6b99d970224072528500a0` |

**実行環境**: GitHub Actions Ubuntu **24.04**、Node **24.21.0**、consumer `package-lock.json`に対する`npm ci --no-audit --no-fund`、esbuild **0.25.12**、CodeMirror state **6.7.6** / view **6.43.14** / lint **6.9.7**。Python Playwright **1.57.0**でインストールした実Chromium **143.0.7499.4**（headless）。生成配布物はループバックの一時HTTP Serverから配信した。

| 実コマンド／操作 | 実測結果 |
| --- | --- |
| 各PR固定pathの `git archive` 抽出 + `git hash-object`（Graph/Command/Editor） | **PASS**：全3 providerのGit blobが固定値に一致 |
| `npm ci --no-audit --no-fund` | **PASS**：依存をlockに従い導入 |
| `npm test`（`node --test *.test.mjs`） | **18/18 PASS、FAIL 0、SKIP 0**。新規 `journal selection remains keyboard-operable when the actual Graph is synchronously re-rendered` もPASS。実Graph/Command/Editor Portを読み込む試験がSKIPされていない |
| `npm run audit:imports` | **PASS（静的指定ファイル範囲）**。全依存の意味分離証明ではない |
| `npm run build` | **PASS**：実Editor/CodeMirror/Graph/Commandを含むbrowser bundle。 `dist/build-provenance.json`内のlock digestが **`7cb7257ef6e71def71ddf5d24414be596832d6c1f3aef36015f51b340d1f18c1`** |
| `python browser-e2e.py` | **PASS（実Chromium）**：CodeMirrorの`contenteditable=false`、ノート切替とEditor反映、Graph選択と狭幅表示、Command成功/拒否、3部品それぞれのUnmount/Remount、Graph Enter選択、pagehideで最終破棄、ページJSエラーなし |

**実ブラウザーレイアウト値**: Graph `width=480`、viewport `clientWidth=245`、`scrollWidth=480`、`fontSize=14px`。幅を半分に縮めずに横スクロールで表示した。Chromium出力は `{"result":"PASS","browser":"143.0.7499.4"}`。

**重要な証拠境界**: 新たに追加した正式なGraph Port回帰は**合成Fake DOMの自動Node試験**であり、`browser-e2e.py`が実ブラウザーで確認したのはGraphのEnter選択と部品再表示などである。**実ブラウザーでフォーカス中のノードをGraph自身が同期再描画し、その直後のSpaceで2回目選択通知まで到達する経路はこのE2Eには含まれない**。#9の独立Chromiumフォーカス再描画成功（別実行）と区別する。

これはTEIと独立したReading Journal用途で共通3部品がNode24・Chromiumに接続できることの**固定snapshot資格**であり、正式製品採用・全a11y・メモリリーク完全否定・保存/認可・Studio実Wails Host・native Desktop操作の資格には転用しない。特に[Windows native #2の別固定実測](https://github.com/hat47x/tei-platform/issues/1824#issuecomment-6183933605)でWorkbenchの起動/画面/自然終了・マウスInspect投影が確認されたが、それはReading JournalのDesktop確認ではなく、さらにcross-OS esbuild bundle digest不一致も報告されている。後者のビルド再現性とADR-0032の整理は#2が既に#1824へ**PoC非ブロッカーとして引継ぎ理由と必要対応を記録**しており、#8で同一ADR対応を重複して開始しない。

**再現**: PR #1825の `.github/workflows/tei-studio-journal-evidence-8-ci.yml` を上表の4固定SHAで実行。各providerのsource blobを照合後、consumer dirで `npm ci --no-audit --no-fund && npm test && npm run audit:imports && npm run build`、Playwright+Chromium導入後に `python browser-e2e.py`。PRへの通常マージは不要。Nodeと実Chromium両試験の出力をGitHub Actions runに保存している。
