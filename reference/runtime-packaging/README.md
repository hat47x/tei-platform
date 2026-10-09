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
- `ArtifactDigest` は生成済み成果物の識別子、`BuildPlanFingerprint` はまだ成果物がない段階の固定されたビルド入力を指す。少なくとも一方が必要で、両方あればそれぞれ別の事実として保持する。これらの値の真正性・相互対応は既存BuildPlan/BuildArtifactLock/Plugin Lockで検査し、ここで第三のLockを作らない。

## 確認する振る舞い

```sh
go test -count=1 ./reference/runtime-packaging
```

局所Go 1.23.2環境で、合成入力を使った**17テストがPASS**した。ファイルのGit blob SHAを照合した同一バイト列についての結果であり、GitHubの全リポジトリcheckoutに対する健全性検査ではない。

- BrowserへServer専用のDB処理が混入しないこと。
- 到達不能なEditor用実装が選択されないこと。
- 入力の列挙順序が変わっても、出力JSONが同一になること。
- 欠落契約、曖昧な実装選択、動的依存の未確定、Targetの機能不足、通信先の欠落、通信境界の参照漏れ、不正な成果物識別やBuild Planの識別子を拒否すること。生成済み成果物がなくても、固定Build Planからの選択を保てること。
- 依存グラフに循環があっても、到達できる各実装を一度だけ含めること。

## まだ確認していないこと

実Reference Application `order_entry`を、実Plugin Lock、Capability契約、BuildArtifactLockからこの形式へ変換する入口は未実装である。Goコードのimport graph、静的結合、実際の実行ファイル、Browserでの権限分離、実ネットワーク接続、実DB、Desktop/Wasm/IoT、運用・監視については**検証していない**。

特に、同じ一つの物理成果物に複数Capabilityが含まれる場合、論理的に未到達のCapabilityを除外しただけでは、バイナリからその実装が削除されたと証明できない。また、宣言されたHost importを検査するだけでは、OS権限を共有するネイティブプラグインを隔離できない。

次は、`order_entry`の既存Bindingを読み取るAdapterを用意するかを既存Ownerと評価し、その結果を静的結合時の生成コードへ渡して、生成後の成果物を直接調べる。追加の汎用依存解決機構は設けない。
