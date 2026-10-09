# 配備単位ごとの依存閉包 — 読取り専用の参照実装

- **状態**: 実装前設計の局所的な検証。製品の配備機構、プラグイン解決器、コンパイラ、実行時の認可機構ではない。
- **設計**: [多環境向け最小ランタイムとバイナリ別配備](../../plan/design/RUNTIME_TARGET_BINARY_COMPOSITION_DESIGN.md)
- **主担当**: [#259](https://github.com/hat47x/tei-platform/issues/259)。プラグイン契約と解決は[#951](https://github.com/hat47x/tei-platform/issues/951)、ビルドと成果物の追跡は[#943](https://github.com/hat47x/tei-platform/issues/943)に委ねる。

## 何を確認するか

TEIの実際のプラグイン解決器が**すでに確定した**契約・Provider・成果物の情報を渡すと仮定し、配備先の実行入口から到達できる実装だけを選び出す。契約を実装するProviderの選択や署名・真正性の検査は、ここでは行わない。

合成例では、画面がAPIの業務操作を呼び、API内の業務操作がDBへ接続する。

```text
browser [ui] --明示した通信/信頼境界--> api [action] --> api [db]
                                           api [unused.editor] ← 到達できない
```

出力は実行単位別の**採用候補一覧**であり、そのままコンパイルできるGoパッケージ群ではない。バイナリからの実コード除去、Wasmの生成、起動時間、配布形式は別の実証を必要とする。

## 入力上の最小規約

- `Unit` は個別に配備・停止・更新する単位。入口となる契約 `Roots` を明示する。
- `Binding` は、既存のCapability/Plugin解決とLockで選んだ結果を**参照する写像**。この参照実装は選択し直さない。
- `Requires` は同じ実行単位で呼ぶ契約、`RemoteRequires` は別単位の契約を指す。後者には通信契約と信頼境界への参照を必須とする。ただし参照があるだけで通信保護・認可が成立するわけではない。
- `AvailableImports` は信頼できるHost側が提示する機能の候補であり、プラグイン自身の申告から無条件に増やしてはならない。
- `DynamicOpen` は動的依存を閉じられない状態を表す。この状態の実装に到達した場合は失敗する。静的に決まらない呼出しを黙って除外しない。
- `PluginFingerprint` は既存Plugin Lockの定義指紋、`ArtifactDigest` は生成済み成果物、`BuildPlanFingerprint` は固定済みのビルド入力を指す。**既存のBuiltin Pluginは成果物digestを持たない場合がある**ため、三つの識別のうち一つ以上を保持する。これらの値の真正性と対応関係は既存の解決・Lock検証へ委ね、ここで第三のLockを作らない。なお、`ContractFingerprint`は将来の契約境界の候補であり、現行Plugin Lockの定義指紋を代入してはならない。

## 確認する振る舞い

```sh
go test -count=1 ./reference/runtime-packaging
```

初期の依存閉包に対する**17件の合成テストは、当時の局所Go 1.23.2環境でPASS**した。その後に実Workspace接続とBuiltin Plugin識別を追加しているため、**現在のPR最新コミット全体に対するテスト成功はまだ取得していない**。以前のGit blob SHA照合は新しいファイルの検証根拠にはならない。

- BrowserへServer専用のDB処理が混入しないこと。
- 到達不能なEditor用実装が選択されないこと。
- 入力の列挙順序が変わっても、出力JSONが同一になること。
- 欠落契約、曖昧な実装選択、動的依存の未確定、Targetの機能不足、通信先の欠落、通信境界の参照漏れ、不正な成果物識別やBuild Planの識別子を拒否すること。生成済み成果物がなくても、固定Build Planからの選択を保てること。
- 依存グラフに循環があっても、到達できる各実装を一度だけ含めること。

## 実Workspaceへの接続（今回の追加）

`workspace_probe.go` は既存の `workspace.Load`、`plugin.Catalog.ResolveEnvironmentWithLock`、`plugin.Lock` をそのまま使用し、`order_entry`の**データベース要件に限って**次の情報を突き合わせる。

```text
Application.database alias
       ↓
Environment.bind
       ↓
DatabaseInstance.definition / provider
       ↓
既存のPlugin Lockで検証されたDatabase Provider実装
```

既存Sourceの読取りから期待する対応は、開発環境の `sales → sales_core.dev → postgres → tei.database.postgres` と、`master → master_data.dev → oracle → tei.database.oracle` の二つである。両者ともBuiltin実装であり、現在のPlugin Lockには成果物digestではなく定義指紋が記録されている。

`workspace_probe_test.go`には、実`reference/workspace`を読み取るテストに加え、Lock改ざん、Binding欠落、Database定義の不一致、Provider競合、Alias重複の負例を追加した。これらの**新しいテストは未実行**であり、成功は主張しない。

取得したのは**データベース接続に関する前提情報**であって、Action、認証、画面などの実装を一つの配備物へまとめる完全な依存閉包ではない。結果には配備準備完了を表す項目自体を設けず、DB接続の観測をアプリケーション全体の構築完了と混同させない。

## まだ確認していないこと

現時点では`order_entry`のAction実装（Custom Go等）やUI処理を、Plugin Lockで固定したCapabilityの実装と同一視できない。現行のプラグイン管理情報は**環境が要求するPlugin集合**を表すものであり、それだけでアプリケーションの実行入口からの到達可能性がわかるわけではない。

まず、アプリケーションのActionと実行入口を既存Driver・Realizationへ戻って正確に対応付ける必要がある。新たな定義指紋や契約指紋を便宜的に合成して`Derive`へ渡すことはしない。

Goコードのimport graph、静的結合、実際の実行ファイル、Browserでの権限分離、実ネットワーク接続、実DB、Desktop/Wasm/IoT、運用・監視についても検証していない。論理閉包を絞れても、同じ物理成果物内の未到達コードが除去された証拠にはならない。

次の優先作業は、実際のWeb Targetの起動点と既存Realizationから**実装への参照辺**を取得し、現在のPlugin Lockが説明できる範囲と説明できない範囲を明示することである。ビルド可能な依存閉包が成立して初めて、生成バイナリの差を測定する。
