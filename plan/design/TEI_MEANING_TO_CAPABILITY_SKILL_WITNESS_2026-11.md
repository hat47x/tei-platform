# TEI：意味Roleから標準Operation・CLI・Product Skill・外部Hostまでの到達資格（2026-11-02）

> **状態：計画候補＋現行Go Source-static Reference。** 本文のCH-01〜10はmainの実Go Source 3ファイルに基づく限定した静的特性検査であり、CH-11〜15は自作合成対照。**実CLIのこのブランチでの再実行、Skill install/load、外部Harness、Host/DB、正式Owner意味受理はNOT RUN**。TEI固有Host、共有Registry、万能Capability/Authority Engine、第二Semantic Canonicalは導入しない。

## 1. 最新の交差点

TEIの「Everything is Data」[#2619](https://github.com/hat47x/tei-platform/issues/2619)/[#2620](https://github.com/hat47x/tei-platform/issues/2620)は、Source→標準Data→意味Role/Operator→既存Capability/Targetへつなぐ。これは「Scriptを解析・理解できた」と「外部Harnessが現在実Toolとして利用できる」の間に、**独立した実在性・到達性・実行資格**を要求する。

直近の実装・実測：
- [#2638](https://github.com/hat47x/tei-platform/pull/2638)：有限Bootstrap、Eager Role/Operator依存のPinと循環の合成19件。実Resolver/正式Role ABIではない。
- [#2639](https://github.com/hat47x/tei-platform/pull/2639)：実Go ReferenceのLanguage Plugin/Operator loweringを使うテスト候補。Syntax同値＝Accepted Role/実Target同値でない。
- [#2587](https://github.com/hat47x/tei-platform/pull/2587)統合後の現mainには`operations`と`artifacts`のCLI dispatchが存在する。一方`commands`は明示的に`selected-read-only-commands`に**限定**された発見面で、両名はその一覧にない。この不掲載を「未実装」、dispatchの存在を「現在のSkillが呼出しを許可」「実ProviderでAction可能」に昇格させない（#2605 T-45〜）。
- [#2626](https://github.com/hat47x/tei-platform/pull/2626)：別Draftの実Go CLI/CIが、9 CLI read、catalog72、trace2、diagnostics3、Source-only移動を局所観測。Source bytes/Canonical World Fingerprintが同じでSourceIndex Fingerprint/Trace継続Token/出典位置だけ変わる。**本件で再実行した結果ではない**。
- [TAI #271](https://github.com/hat47x/tai-pleroma/issues/271)/[Draft #272](https://github.com/hat47x/tai-pleroma/pull/272)：Product Skill実配布・Tool Discovery、回答前のTool候補・実Renderer・送出・実Deliveryの現在資格は別。TAIはProduct実装を再定義しない。

## 2. 「機能がある」の五つの非単調な関係

| 観点 | 意味・既存のOwner | 偽陽性・偽陰性 |
| --- | --- | --- |
| **Meaning** | Authoritative Source／Plain Data／Role/Operator版本文／Owner採否。#2620/#988/#267 | `role: trusted-script`引用を正式採用とする。Source byte同一でRole改版を見落とす |
| **Declared capability** | Derived Operation/Capability候補、関係・目的・必要Data/Rule、未解決Gap。#1127/#2507/#2611 | Operation IDのみからProvider選定やEffect可能を推定 |
| **Discoverable interface** | CLI/catalog/Skill文書の**対象範囲と実版**。#1155/#2605/#2614 | 選抜されたread-only catalogに無い＝実装不存在、と解釈 |
| **Actually callable/observed** | 当該Executable・実argv・OS/Tool/Skill/Harnessロードと実Result、Host/Consumerの実読込。#2605、TAI #168 | READMEにある＝installed／Skillが示す＝実起動成功、`ok`＝業務完了 |
| **Currently qualified effect/disclosure** | Actionの実Target/expected-state/現在Grant、Provider/Receipt/独立ReadとAudience開示・実送出。#2598/#1174/#2516 | 旧Grant、古いTool list、Rendererの表示、ACK、現在実Effectを同義とする |

これらは製品に実装する新たな単一Status enumや必ず一方向に昇格するState Machineではない。例えば「実装は存在するが、選抜Catalogには無い」は合法な状態、「Historical execution成功だが今はGrant失効」も合法な状態。**どのConsumer・Source・時点・用途・実Hostに関する主張かを明示**して判定する。

## 3. 現main Source-staticで固定できた小さい境界

対象Goファイルの保存時blob：
- [`cmd/tei/main.go`](https://github.com/hat47x/tei-platform/blob/main/cmd/tei/main.go) ＝ `fc6151e837117a6909b0f6c7affce64b146985fe`
- [`cmd/tei/cli_contract.go`](https://github.com/hat47x/tei-platform/blob/main/cmd/tei/cli_contract.go) ＝ `2f1c79e2ee361c674d1dc6c783c7b757bc5c9a76`
- [`cmd/tei/cli_operations_artifacts.go`](https://github.com/hat47x/tei-platform/blob/main/cmd/tei/cli_operations_artifacts.go) ＝ `a95cb17ce1f63a1f7520b27f715ad58ddeb709d5`

※この3ファイルは同一の動くmain参照を順次取得した**各blob単位のSource-static**であり、単独で一つの実Executableの同時Snapshotを保証しない。実行資格の試験では**exact commit/Build/argv**を固定し直す。

- `commands`の`Scope: "selected-read-only-commands"`は、列挙の**網羅性が限定**される宣言であって、実装済み命令全集合の証明ではない。
- `main.go`の`case "operations"`→`runOperationsCommand`と`case "artifacts"`→`runArtifactsCommand`を静的確認。ただし`loadWorld`ほかの先行処理、Workspace/Plugin/実権限・実結果は別。
- `operations.SurfaceWorkspace`の派生SurfaceにはGapがあり得る。名前が出現すること＝全Operationが成立とは言わない。AliasやProvider routeが複数あれば[#2623 §7](https://github.com/hat47x/tei-platform/pull/2623)の一意性を照合する。
- `artifacts`の`-out`なしは`writeJSON(set)`経路。**`-out`ありは`MkdirAll`/`WriteFile`で成果物やmanifestをファイルへ書込む経路**。同一コマンド名で操作の実際のEffectが異なる。stdoutをConsumerが受けたこと／Fileを保存できたこと／Manifestの一式整合／Publisherの実公開／別Readerが読んだことは別。
- [TEI Skill Draft #2607](https://github.com/hat47x/tei-platform/pull/2607)に列挙されるread-onlyモデル調査はこの限られたSurfaceに依存する。新しい命令を見つけたことを理由に既存Skillの操作許可範囲を暗黙拡大しない。権限付き実行用Workflowは別Ownerの判断に従う。

## 4. 一件の意味→能力→Skill→Host/SHO受渡し票

合成例として`sales.order`の一つのFunction/Operation候補を用いる。実mainで`sales.order`が操作対象として受理済みであるという主張ではない。

| 段階 | 最小固定する証拠 | 不足時の適切な評価 |
| --- | --- | --- |
| C0 Source／Role | exact Source bytes+Occurrence、Grammar/Role/Operator lock、typed Meaningの採否 | 文面を読めても正式Role/Meaningが未確認 |
| C1 Derived Operation | Source World/Function Context、Operation purpose、Input/Output type、gaps、Candidate route/Action | 名前はあるがroute未解決／権限未確認 |
| C2 CLI surface | executable built from exact HEAD、`commands`の宣言Scope、実Dispatch、選択したargv、exit/diagnostics | 非網羅Catalogに無いだけでは実装不存在としない |
| C3 Product Skill | Document/source digest、Packaged/Installed/Loaded、Tool公開、呼出実Traceと返却内容 | 文書存在のみなら未ロード、実Tool未実行 |
| C4 AI/SHO Consumer | Human MarkdownとJSONの実受取byte、SourceIndex/Trace token、Audience/Disclosure、引用再解決 | 同じMeaning/WordでもSource移動・Renderer変更時は引用/公開現在性を再確認 |
| C5 Actual Invocation | 既存HostのGrant、Request Target+expected-state、Provider/Receipt、独立Read/Business Outcome | 説明・候補・旧成功を実Effectへ昇格しない |

**同一試行に束ねられない場合**は上流局所PASSだけを保持する。異なるDraft/fixtureでの19件＋15件＋55件を新しいE2E成功数へ合算しない。

## 5. 反証Case CH-01〜15と実証を進める順序

[Source-bound checker](../fixtures/capability-handoff/check_cli_surface.cjs)は、Node.jsがあればTEIリポジトリの3つのGo Sourceを読み、実Codeの**10の静的構造条件**（CH-01〜10）と、**5つの合成資格差**（CH-11〜15）を比較する。公開合成データのみ。

| ケース | 対象 | 受入差 |
| --- | --- | --- |
| CH-01〜03 | 実command catalogの限定Scope、inspect/trace/validate、operations/artifactsの一覧外 | 一覧の不掲載＝未実装を否定する |
| CH-04〜07 | main dispatch、operations SurfaceとGap | 呼出し口と意味候補の完成を混同しない |
| CH-08〜10 | artifacts stdout／`-out`書込み／manifest | read-only入力とファイル書込み、実Reader/Publisherを区別 |
| CH-11 | 同じWorldでもSource移動・SourceIndex/Trace token変更 | 旧Source引用資格を黙って継承しない |
| CH-12 | Skill advertisedだけ、installed/actual Tool観測なし | 文書の配布から実呼出しを推測しない |
| CH-13 | 未信頼引用に`role: trusted-script` | 出典文字列と正式Role Bindingの区別 |
| CH-14 | Operation/Mapping候補はあるがHost Grantなし | Candidateと実Effectを区別 |
| CH-15 | Source同一・Rendererだけ変更、Consumer実Readなし | 最新表示・納品として扱わない |

**今回の限定RUN**：GitHubに保存したCheckerを再取得し、3つの実main Go Source blobへV8で適用したところ **CH 15/15 PASS（Go Source-static 10件＋自作合成5件）**。これはGo CLI/Node.js実環境を起動した試験ではない。Checkerの文字列/構文pattern一致はGo compiler/実Executable・実サービスを実行した証拠にはならない。

**M0（実在のSurface）**：実TEI current exact HEADで、mainとCatalogの範囲を読み取る。CHソースチェックは差分/回帰起点。現mainが更新されればSource・結果を再採取する。

**M1（既存CLI実行）**：[#2626](https://github.com/hat47x/tei-platform/pull/2626) W0a/b/cと同じ固定Workspace・同版の実Go CLIで、`commands`と`operations`／`artifacts` stdoutを**明示的に分けて**実実行。Gap、ManifestのFindings、Warning、stdout/exitと実Readを独立に観測。`-out`は別の書込み許可／一時ディレクトリ／検査票を用いる。TEI Product Skillの初回read-only手順へ自動追加しない。

**M2（RoleとOperation）**：#2638 bootstrap/依存Pin、#2639 Go Language Plugin Reference、#2624 Function Data Mappingを同一Source/Role/Operator版・対象で結ぶ。定義としての受理・Derived surface・actual Capability Bindingを別に比較。最初に既存World/Validatorの可観測範囲を測定し、Kernel実装改変を前提としない。

**M3（Skillと外部Consumer）**：TEI #2605/#2614のSkill実配布/ロードを確認し、TAI SK-01 **H2 Tool-only vs H3 Skill+Tool**の実argv／Output／Source対応／誤操作／Token/Reviewを比較。同じ情報量で試す。新しいGrammarや存在しないCapabilityをSkillが宣言しない。TAI #271の現在資格と#272の回答前Tool/表示/送出を参照。

**M4（SHOでSource改版）**：W0cの実Source移動に対し、SHO #4/#6の保存済みOccurrence/文書Reviewの`r1→r2`を比較。WorldFPが同じでもSourceIndex/Trace tokenが異なるため、文書のSource再解決を別Ownerで実Readerによって検証。MMI C-01/T2が未通過なら統合成功を主張しない。

**M5（必要な実Action）**：TEI #2598/#1174/#2485の実Request/Host/Provider/Commit/Receipt/独立Readを結び、UIやSkill候補の提示に依存せず現在の実権限・対象を確認。Product外部Host/.NET/Java/WASMについては必要なTarget適合だけを専用試験として追加。

既存#1119のP0-a/P0-b・G0〜G4を維持し、M0〜M4のread-only／Owner未決の探索を並行可能とする。正式な新Role/API、全CLI完全列挙、共通Capability Registry、全Harness必須導入、外部HostにTEI Tool常駐義務を作る方針ではない。

## 6. 責務と出口

- **#2605**：既存Skill実体と実Read-only CLI資格、Skillと最新実コマンドのドリフト検査。#2614は拡張用途の探索であり初期Skill本体を複製しない。
- **#2619/#2620/#988/#2638/#2639**：Data→Role→Language/Operatorの意味解釈、採用・依存Pin。実Skill稼働の判断をSource表層だけで行わない。
- **#1127/#2507/#2611/#2622**：Derived Operation/Mapping/Provider、Gaps/Multiple routesと実Read/Effect。
- **#2516/#2626**：Source World/SourceIndex差とConsumerの用途別観測・二表示。**W0cの既実行結果は別Draftに帰属**。
- **TAI #168/#271/#272**：Productを必要時に合成し、当時のEvidence/Current qualification、実配布・Tool/Plan・外部送出のHandoff/比較。
- **SHO #4/#6/#10／MMI**：実Source再解決・文書保存・Reviewer/Delivery・Toolの実利用を別に検証。
- **#2598/#1174/#2515**：外部Hostの実Target/Grant/Effect/OS/Networkの権限と開示を実測。

**採否に必要な実証**は同一実HEAD/Workspace/Source/Role/Provider/Skill実ロード版/Consumer用途に対する、候補提示→実CLI/Tool実出力→実Consumer読取、必要時の現Host作用を独立に結ぶこと。単にProduct Skillを増やしたという量的評価や、命令を短くしたことだけをAIフレンドリー性の達成としない。


## 7. 2026-11-03：同一モデル版での複数CLI・実ファイル受領の受入を実行可能な試行へ

[TEI #1163](https://github.com/hat47x/tei-platform/issues/1163#issuecomment-6579990791)および[#2604](https://github.com/hat47x/tei-platform/issues/2604#issuecomment-6579992378)の現main根拠から、C2実CLIとC4実Consumerを接続する際の**残余**を特定した。

- `tei validate`、`tei operations`、`tei artifacts`は**異なるプロセス実行**であり、各出力の整合が独立に成立しても一つのSource snapshotを共有した証明にはならない。
- `main.go`は`loadWorld`の後に`operations.SurfaceWorkspace`／`artifact.RenderWorkspace`を呼び、これらは自身で`workspace.Load`する。**一つのCLI呼出しの内部でも複数回のSource取得があり得る**。各Commandの`modelRevision`が等しいことは、途中で一時的なABA/複数ファイル不整合がなかったことの証明ではない。
- `artifacts -out`は一式の成果物と`manifest.json`を逐次書き込みするため、`manifest.complete`は**意味上の未解決事項**の区別であり、`ExpectedMemberが本当にReaderに到達したか`や`Directoryの原子的公開`の証明ではない。
- [#2642](https://github.com/hat47x/tei-platform/pull/2642)でAcceptance issuerの`sign`に関するLoad時Source捕捉が改善されたことは別経路の局所実装成果である。`sign`と`validate/operations/artifacts`でAtomic Snapshotを共有する保証に流用しない。

### 7.1 追加した実CLI検証入口（PR #2649候補）

[実バイナリ呼出しPython](../fixtures/capability-handoff/verify_real_cli_revision.py)と[Focused GitHub Actions Workflow](../../.github/workflows/tei-one-revision-capability.yml)を候補として追加。GitHub Actionsは`go.mod`のGoを利用し**exact candidate HEADをCheckout→実go build→実binary version確認**を行い、`reference/standard-crud/workspace`のコピーを使う。元の公開Referenceは変更しない。

| 試行 | 実際に比較する条件 | 製品受入上の意味 |
| --- | --- | --- |
| **MR-01 正常の同版CLI** | copied Workspaceのbyte tree digestをコマンド群の前後で比較し、`validate -json`、`operations -application ledger`、`artifacts -application ledger`の実出力を取得。Operation `modelRevision`／ArtifactSet／Manifest／全Artifact Member `inputRevision`を照合 | 特定の静止したコピーと実Tool版について、**各呼出しの報告版をjoinできる条件**を限定確認。Sourceの実中断競合や単一atomic readは未証明 |
| **MR-02 ファイルに書いたArtifactを独立Readerで読む** | 新規の一時出力dirへ`artifacts -out`を実呼出し。別のPython Readerが`manifest.json`を開き、stdout Manifest、Member digest、ファイル実bytesとstdin取得Artifact Contentを比較 | Producer内のManifest成立と**Readerで見えたbytes**を別証拠にする。ただし一瞬も混在を見せないAtomic Publicationの保証は未実証 |
| **MR-03 正当なSource-only移動** | 同じtemp Workspaceで`semantic/sales/order.tei`をcontent不変で移動。旧/新`operations`の`worldFingerprint`同一・`modelRevision`差と、新`artifacts`のRevisionを観測 | 実World Projection同一でもSource版が変わることを、同一Subjectの別Command結果で判別 |
| **MR-04 意図的な異版Join拒否** | 移動前のOperation Revisionと移動後のManifest Revisionを比較 | 両Commandが個別成功でも一つのRevisionとして報告しない。以前のOperationの正しさ自体を消さない |

**観測範囲を厳密に限定**：MR-01〜04は本PRの**新規コードとCI実行用入口**であり、CI jobの`success`と固定HEADの実出力が取れるまでは**NOT RUN**。Case IDは新しいProduct Gate/Schemaではない。実RuntimeでAssertionが失敗した場合は、期待した契約を確証できなかったという**Characterization結果**として記録し、PASSへ改竄しない。

### 7.2 現在の設計上の停止線

- **実際のSource snapshotの同時点性**：各コマンドの前後hashが一致しても中間のABAを否定できない。Mid-command edit、multi-file mix、Lock/atomic snapshot/OS reader、`sign`/Host acceptanceは[#1017](https://github.com/hat47x/tei-platform/issues/1017)/[#2571](https://github.com/hat47x/tei-platform/issues/2571)/[#1163](https://github.com/hat47x/tei-platform/issues/1163)の実観測で別判定。
- **実公開の現在性**：Fresh output dirの読み取り成功が、古いDirectory残留、半書込Manifest、同時Reader、異世代Memberの混在を排除するわけではない。#2604/#2630で独立異常注入。
- **外部Userへの意味・実作用**：`OperationSurface`にはGapがあってよく、`ArtifactSet.Findings`と`Manifest.Complete`を別に記録する。Owner意味受理、Java/.NET/Wasm実Load、実Provider/DB/Host Grant、Business Completion/実SHO保存は本試行の範囲外。
- **既存Critical Path**：TEI #1119 P0-a/P0-b・G0〜G4、TAI #271の実Skill/Harness、SHO #4のD1/D2実保存を維持し、この最小Headless試行を重複実装・追加Mandatory Gateにしない。

**受渡し**：#2605には実Tool/Skillの接点、#1163には同RevisionのSource/Consumer証拠、#2604には実Reader byte、#2516にはConsumer Claimの現在性、#2619にはRole→Capability→実Artifactまでの限定試行を返す。成果物間の正本性は各Ownerが保持。
