# ADR-0032: Cordis型の合成KernelをTEIの新しいプラグイン基盤とする

- **状態**：採用（目標アーキテクチャ、2026-10-12）。**現行production Runtimeへの移行完了を意味しない**。
- **関連**：[Issue #1223](https://github.com/hat47x/tei-platform/issues/1223)、[#951](https://github.com/hat47x/tei-platform/issues/951)、[#943](https://github.com/hat47x/tei-platform/issues/943)、[#259](https://github.com/hat47x/tei-platform/issues/259)。
- **置き換える判断**：実験開発中の旧Plugin ABI、固定CapabilityKind、RuntimeKind、Plugin Package中心の依存・旧Lock形状を長期制約として保護する判断。過去の判断・実装を消さず、移行完了まで現行状況を記録する。

## 背景

TEIは標準機能までPluginとし、Web/デスクトップ/サーバー/分散/IoT向けの最適な実行形態を生成する。強い可用性要件があるHostでは、機能更新の大部分をサービス継続中に行えるようにする。

現行のGo実装には、固定 `CapabilityKind`、`builtin|process`、Plugin Package IDへの `requires`、個別ドメイン構造を抱え込んだPlugin modelがある。この形を永久に残すと、後から任意のサービス契約を導入するだけで基盤の再ビルドやCore変更が必要になる。現行実装の安定性を「設計を変更してはならない理由」にしない。

CordisのRevertible Effects、Reactive Coeffects、Contextによる合成規律を参考にする。TypeScript APIやLoader実装の全面模倣はしない。論文：<https://arxiv.org/abs/2608.25512>。

## 決定

### 1. 契約を中心とする新モデルへ切り替える

- **Service Contract**：論理ID、version、shape・effect・failure semantics、scope、required context、Conformance。Plugin Package名やOS Processから独立した識別子。
- **Provider**：Contractを実装する実体。Plugin identity、実装ID、exact artifact、Target realization等は別軸。
- **Consumer requirement / Coeffect**：必要なContract、互換条件、必須/任意、適用Scope。Package間依存は配布上の制約として残せるが、実行依存の唯一の表現にはしない。
- **Effect Scope**：Provider・Consumerが生成する一時的な作用を登録し、終了時には逆順に解放する。永続的な業務上の副作用には適用しない。
- **Generation**：特定のContract・Provider・Artifact・Runtime instanceの束縛を固定したHost局所の実行世代。呼出し開始時に世代を保持し、切替中にすり替えない。
- **Context**：利用可能なサービスとEffect/Coeffectを同じ作用域で管理する。ただしUIや業務上の「文脈」と同一の正本へ融合しない。
- **Composition Profile**：案件の選択・接続を表す宣言。Runtime Authority、Accepted business meaning、Artifact Digestを上書きする第二の正本にはしない。

旧 `CapabilityKind` 固定列挙と旧Plugin Packageの `requires` を**新設計の仕様には引き継がない**。ただし古いSourceを現在のProductionで扱う間は、旧形式が具体的にどう解決されたかを記録する。新形式への明示的変換後は二重正本を維持しない。

### 2. Coreを極小化し、意味上の不変条件を定める

残すべき不変条件は、(a)契約識別と選択の一意性、(b)Effect/Scopeの終了、(c)世代の線形化と参照保持、(d)信頼できるHostによる現在の権限検査、(e)故障時の隔離/復旧を可能とする情報、(f)可観測な結果と不確実性の区別、に限定する。

認証方式、DBドライバー、Web middleware、UI、業務ルール、Logger、Builder、Loader、Updater等の実装は原則Plugin化できる。ただしRoot of Trustと実行隔離を担う最小Host機構をPlugin自身に自己保証させない。コア自体の更新にはHostの二重化/交替、再起動等が必要になる場合がある。

**「Coreが小さい」はSecurityチェックを削る意味ではない。** 安全条件を定義する境界と、その機能を実装する仕組みを分離する。

### 3. 更新は単なるreloadでなくGeneration protocolにする

計画更新では準備済みの別世代をReadyまで検証してから、Host局所の新規呼出しを原子的に切り替え、旧世代のin-flightをDrainして最後にEffectを解放する。旧世代の処理途中でProviderを黙って交換しない。切替失敗時には旧世代を継続する。旧世代のdrain失敗やcleanup失敗を「無停止更新成功」にしない。

予期しないProvider消失時は安全な依存解消・停止を行う。計画更新と予期しない故障を同じ無条件のConsumer dispose/restartにしない。Version互換、readiness、現在の権限、fencing、resource capacity、DB expand/contract、Rollback可能性、実際のAvailabilityはそれぞれ別の条件である。

### 4. 同じ契約を静的にも動的にも実現する

論理的なPlugin/Contract定義はTargetに依存しない。配置や更新単位は独立させる。静的リンクでは依存グラフをビルド時に解決・縮退し、不要なDependency Watcher/HMR/Loaderを除外する。サーバー/長時間稼働するWasm Hostでは動的構成と世代切替を選べる。単一IoT端末のfirmware再起動は無停止と呼ばない。

更新後の計測対象は、変更可能なPluginの割合だけではなく、**更新によって変更しなくて済む実行単位の割合、Availability、P99遅延、更新失敗時の復帰時間、影響範囲、実行・記憶資源の増加**とする。

## 破壊的移行の許可と順序

1. **M1: 新Kernelの検証**：Go標準ライブラリのみを用いるEffect Scope / Generation Routerの独立実装と否定試験を起点とする。今は`internal/composition`に隔離。コアの公開ABIには未接続。
2. **M2: Contract中心の依存解決**：旧 `CapabilityKind` から独立したcontract identity / provider resolution / version / scope / deterministic ambiguity rejectionをProduction候補へ昇格する。#951の成果は採否を判断し、旧形式が妨げになるなら変更する。
3. **M3: 旧Runtime接続を置換**：既存Plugin Lock/Runtime Engine identity/Authorityから必要な検証根拠を抽出して新Invocation入口を作る。完全な二重Authorityを恒久化せず、最初の縦断Consumerから切り替える。
4. **M4: 実際のHot replacement**：独立ProcessまたはWasm instanceを使って、readiness、切替、in-flight drain、cleanup、fail-close、復旧を負荷下で観測する。
5. **M5: 静的Lowering・軽量化**：同じContract/Plugin構成から動的部分を削ったstatic/Go/Wasm成果物を生成し、サイズ・起動時間・RAM・権限の差を測定する。
6. **M6: 旧形式削除**：新しい本番Consumerが必要な能力を満たした時点で旧 `CapabilityKind` /旧Lockの書き込みと旧Resolverを撤去する。互換入口が必要なら期限付き一方向変換に限る。

プロトタイプをProduction authorityへ昇格させる前に、source/consumer一覧、exact versionへの結び付け、error/unknown semantics、正負Reference、移行前後の回帰試験を要求する。必要なら旧SourceやGo APIに破壊的変更を入れる。**「旧仕様に合わせるために新しい設計を複雑化する」ことを原則として禁止する。**

## 残る不確実性

- Contractの互換性・副作用型・Context Scopeは、実ConsumerとProviderの圧力試験なしに完全固定しない。
- Consumer requirementが受け渡す項目、パラメータ、値の出どころ、対応付けと変換の方式は、[評価待ちの設計候補](../design/PLUGIN_DEPENDENCY_IO_MAPPING_DESIGN_2026-10.md)で検討する。このADRの判断には含めない。
- Go in-processの任意コード差し替えは一般に可能ではなく、更新単位の変更が必要。
- 単純なMutexによるRouterはHost局所Referenceであり、分散一致性・Availability SLO・本番認可を提供しない。
- 実際のService/Resource LifetimeとDBの永続状態のライフサイクルは一致しない。
- Plugin依存の動的変更による振動、循環、Provider競合と多重配備の整合性を別に検証する。
- KernelとRuntime Engineを論理的に区別することと、実行可能なEngineを後から交換可能にすることは両立する。既存Build identityの区別は設計上の出発点に過ぎず、構造は必要なら変える。

## 反対案と不採用理由

- **現行ABIへ少しずつフィールドを追加**：固定enum/Runtime分類を温存し、技術的負債を将来へ繰り越すため不採用。
- **CordisのTypeScript実装をGoへ逐語移植**：TEIのTarget/静的Lowering/セキュリティ要件に一致せず不採用。
- **何でも単一Runtimeに同梱**：IoT/Wasmの性能・可搬性を損なうため不採用。
- **無停止のため権限と安全性を緩和**：不採用。

## 検証と成果の状態

2026-10-12に孤立したGo試作を作成し、独立した一時モジュールにて`go test -race -count=1 ./...`と`go vet ./...`を実行。これは**単一Packageの候補検証**であり、TEI全体のtrusted full Repository Health、同一Git exact-headの検証、Production migration、無停止運用のEvidenceではない。コードのmain統合条件は`AGENTS.md`に従う。
