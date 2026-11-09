# 実行先の現状調査（2026-11）

- 状態: **調査のみ**。新しい実行先を実装したものではありません。
- 目的: 「二つ目の実行先と多様な配備先」に関する現在地を、リポジトリに実在するものだけから整理する。
- 関連: [全体構成](../../docs/architecture.md)

## 1. 実在するもの

| 実行先 | 実在するもの | 何を確かめているか | 何を確かめていないか |
| --- | --- | --- | --- |
| **Web** | `tools/reference-first-web-target`（Reference-specific adapter。HTTP / HTML へ投影） | `order_entry` の Canonical / Driver の意味を、Web 固定・第二実装を作らずに投影できること。標準・Custom UI 共通の Command 境界 | Production Web Runtime、Target-neutral Presentation IR、複数の Web アプリ |
| **デスクトップ** | `cmd/tei-studio`（Wails v3）、`internal/desktophost`（**非権威**の Studio host 状態: discovery・draft） | 起動・Native 固定・WebView2 のスモーク（ワークフロー `tei-studio-wails-desktop-webview-smoke`・`tei-studio-desktop-native-fixed-smoke`） | 初めて使う人の一連の操作、三つの視点の編集と再反映、編集結果の受理 |
| **サービス** | `internal/servicehost`（`SelectedServiceCarrierRoute`） | 固定した Generation の束縛（論理 Provider ID）から、物理 Service 選択と Runtime Environment Plan への**ホスト所有の対応** | 新しい解決器・受理済み契約・適合性の発行・実行許可ではない（型のコメントで明示） |
| **Wasm（プラグイン）** | 変換プラグインの適合性（ワークフロー `tei-transform-wasm-conformance`・`tei-transform-wasm-approved-invoke`） | プラグインの変換が Wasm 上で適合すること | 製品全体の実行先としての Wasm（UI・保存・権限を含む） |
| **モバイル・組み込み** | **なし** | — | 該当するパッケージ・ワークフロー・検証根拠はありません |

## 2. 分かったこと

1. 「実行先」は一つではなく、**意味の層（受理済み）→ 投影 → ホスト**という同じ形の上に、性格の違う四つが載っています。Web は投影の参照実装、デスクトップは編集環境、サービスは配備の対応、Wasm はプラグインの実行形態です。
2. `internal/desktophost` と `internal/servicehost` は、いずれも自らを**権威でない**と明示しています。権威は受理済みの意味と Driver にあり、ホストは対応や状態を持つだけです。この分離は実行先を増やすときの前提になります。
3. モバイル・組み込みには**何もありません**。したがって「多様な配備先で動く」という主張は、現時点で根拠がありません。

## 3. 次に確かめる順序（提案）

1. **Web**: 同じ投影から二つ目のアプリを出す（投影が target-neutral でないことの証拠になる）。
2. **デスクトップ**: 初回利用の一連の操作（起動 → 編集 → 受理 → 再反映）を、第三者が実行できる手順として固定する。
3. **サービス**: `SelectedServiceCarrierRoute` を使った配備の実測（解決器ではないことの確認を含む）。
4. **モバイル・組み込み**: 着手するなら、まず「何を同じに保つか」（受理済みの意味・Driver の契約）を決める。UI の移植から始めない。

## 4. この調査が主張しないこと

- 新しい実行先を実装したこと。
- 複数の実行先で同じ業務が動くことの検証。
- モバイル・組み込みへの対応方針の決定。