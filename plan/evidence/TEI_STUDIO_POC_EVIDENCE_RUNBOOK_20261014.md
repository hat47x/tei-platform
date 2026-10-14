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
| データ変換と静的検査 | **単体検査PASS（7件）** | Node.js 22.16.0で `node --test *.test.mjs`。読書ノート選択・Graph入力変換・依存解析の正負例を検証（ローカル同内容複製） |
| TEI専用import | **限定した静的検査PASS** | `node audit-imports.mjs main.mjs notes.mjs journal-graph.mjs`。対象3モジュールのimport graphで該当依存なし。未生成のGraphモジュールや実際のbundleまでは監査していない |
| #9 Graph APIへの接続 | **コード接続準備あり、動作BLOCKED** | #9 `graph.ts` の実公開 `mountReadonlyGraph` に合わせて `graph-entry.mjs` を作成。ただし本PR headには#9ソースがないため `node build-graph.mjs` はBLOCKED（exit 2）。Graph実描画は未実行 |
| #4 Editor / #5 Command | **BLOCKED** | 対象コミットの実exportが未確認。代用部品・仮想APIを作らない |
| C-05 / C-06 の正式再利用判定 | **NOT RUN** | Studioでの同一共通部品実利用、別consumer実ブラウザー描画、差替え・拒否・unmount等の独立検証を実施していない |

#9 Graphの確認元: `feat/tei-studio-poc-c-graph-reference`（確認時head `e652a1b655c6dd5f3e52afde9390f1f6cd0771dd`）。そのbranchでのProviderの存在と、このPR headでの利用成立を混同しない。#9が統合または正規の統合ブランチに揃ったら同一SHA・同一checkoutでSource/build/ブラウザー操作を再実行する。ここでPoC Cの合格・再利用成立を宣言しない。
