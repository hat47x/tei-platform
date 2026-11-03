# reference-order-rule-design

技術検証用サンプル[`order-rule-design`](../../reference/order-rule-design/README.md)(サンプル2)の工程1〜6を、一つの通しの記録として実行する実装です。設計の正本は[技術検証用サンプルの統合設計 4.2節](../../plan/design/TECH_VALIDATION_SAMPLES_2026-10.md)です。

この実装はサンプル固有の処理で、`internal/`の基盤には入れていません。既存資産は変更せずに呼びます。共有入力(`reference/workspace`、合成の転写契約)は作業ディレクトリへ複製してから使い、リポジトリ、`reference/workspace`、`reference/world.json`、プラグインの固定情報、`.tei/decisions`へは書きません。

共通受理へは候補・根拠・署名つきの決定を**提出して止めます**。受理(`accept`)も退け(`reject`)も呼びません。`order-workflow`がmainに未統合のため、その共通受理への提出は未接続です(後述)。

## 実行

```bash
# 清浄でコミット済みの複製(例: 作業ブランチの浅いclone)で実行する
python3 tools/verify-order-rule-design.py [--repeat 5] [--out DIR]
```

- 工程4が`tools/reference-tool-neutral-development/run.sh`を変更せずに呼ぶため、清浄でコミット済みの作業ツリーが要ります。約1〜3分かかります(うち`run.sh`が約1.5分。ビルドの保存の状態で変わります)。
- Go(ローカルのGOROOTの実行体。取得はしない)、Node.js、`ssh-keygen`(`verify-time`対応)、git、bashが要ります。**Dockerは使いません**。足りない道具があれば合格にせず止めます。
- skipが0件で、通しの記録(`chain-record.json`)の`passed`が真のときだけ合格です。`TEI_ORD_GO`が無いと通しの試験はskipし、skipは証拠になりません。
- `--repeat N`は全体をN回行い、判定と診断(`stableSummaryDigest`)が同じことを確かめます。既存ツールには出力の位置や順序が一定でないものがあり得るため、生のバイト列では比べません。壁時計は読まず、時刻は固定の論理時刻です。

## 構成

| 役 | 実体 | 担当 |
| --- | --- | --- |
| 試験(編成) | `go test`のプロセス(`run.go`) | 工程1〜6、失敗注入、通しの照合 |
| 連鎖 | `chain.go` | 工程の記録を前の記録の照合値でつなぎ、識別と照合値が一つの値かを`VerifyThrough`で読むだけで確かめる |
| 提出の束 | `bundle.go`の`SubmissionBundle`と`Submitter` | 候補・出所・署名つきの決定を一つの型にまとめ、受け手を差し替えられるようにする |
| 資産の呼び出し | `tools.go` | 既存のコマンドを`go build`して呼ぶ。Gitは固定の日付で呼ぶ |

## 既存資産を呼べるか

実行ごとの記録(`chain-record.json`の`assetCalls`)と同じ内容です。

| 工程 | 既存資産 | 呼べるか | 理由と扱い |
| --- | --- | --- | --- |
| 1 | `tools/reference-xead-source-adapter` | CLIで呼べる | `package main`のためCLIを呼ぶ。受領記録、未解決の種別ごとの件数、節点ごとのtranscription evidence(`-candidate-evidence-node`)を読み、`rmtevidence.Decode`で検証する |
| 1 | `tools/reference-xead-cross-view` | CLIで呼べる | 公開用の集計から、業務作業→機能→表の経路の有無を読む |
| 1 | 共有の合成の転写契約 | 固定入力 | バイト列を複製して照合値を取る。書き換えない |
| 2 | `tools/reference-business-dfd` | CLIで呼べる | 受注業務の手順を描く。単価規則には触れず、対象外になる |
| 2 | `tools/reference-subsystem-cud` | CLIで呼べる(入力不足) | 受理済みの所有・機能の入出力の事実がなく、空の事実で`EMPTY_CANDIDATE_SCOPE`の未解決になる |
| 3 | `reference/business-spec-templates` | Node.jsで呼べる | `node --test`(8件)と`proposeAdoption`(受理なし、未解決の質問が残る) |
| 3 | `reference/studio-authoring` | Node.jsで呼べる | `cli.mjs propose`で表の編集から共通受理の候補を得る。内部で`go run`を使うためPATHにGoの実行体を置く |
| 4 | `tools/reference-tool-neutral-development/run.sh` | 呼べる | 清浄な作業ツリーで変更せずに実行し、直接編集の経路の結果を読む |
| 4 | 同`qualify.sh`、`integration-preflight.sh` | **呼ばない** | `qualify.sh`は`repository-health-check.sh`の全項目を要し、`AGENTS.md`は一律に実行しないとする。preflightはその受領記録を入力にする。サンプル独自の受領記録を置き、資格確認は未発行と記録する |
| 5 | `tools/reference-change-assurance` | CLIで呼べる | 前後の作業領域と固定入力`-input`を渡す。実行先ごとの派生は出力に無い |
| 5 | `tools/reference-procedure-expansion` | ライブラリとして呼べる | `Compare`で条件が決まる前後の実現先の準備状態を比べる |
| 6 | `tools/internal/opensshwitness` | ライブラリとして呼べる | `Payload`、`Sign`、`Verify` |
| 6 | `tools/reference-authoring-witness-openssh/pressure-test.sh` | 単独の探針として残す | 固定のダミー文面の7項目。このサンプルの署名は`opensshwitness`で別に行う |
| 6 | 共通受理の`witness-decision` | **呼べない** | 受理した決定記録(`reviewBinding`あり)だけが対象。このサンプルは受理しないため呼ばない |
| 6 | `tools/reference-spec-review`、`tools/reference-purpose-preflight` | CLIで呼べる | spec-reviewはcross-viewが欠けて入力不足が残る。purpose-preflightは作業領域の入口と承認規則の固定入力を呼ぶ |
| 6 | `tools/reference-shared-acceptance` | previewとreviewだけ呼ぶ | 使い捨ての基準版の複製へ提出し、再計算した値と署名の値を比べる |
| 6 | `order-workflow`の共通受理 | **呼べない** | 先行ブランチにあり、mainに未統合。「受理は未接続」と記録する |

## 工程の流れ

1. 工程1(`stage1.go`): 合成の旧資料と共有の合成契約を、source adapterとcross-viewで読みます。受注明細に相当する表(`ORDER_DETAIL`)の出所状態を三値で決めます。表の節点が契約で写り、機能と業務作業の経路が届けば「結び付いた」、経路がなければ「表だけ」、節点がなければ「対象なし」です。対象なしでは検証根拠を作りません。契約が扱えない記述(`LegacyMacro`)は未解決の種別ごとの件数で運びます。
2. 工程2(`stage2.go`): 基準版R0の複製を作り、単価の属性を宣言する名前空間を使うアプリケーションを評価対象にします。business-dfdとsubsystem-cudを呼び、権限を持たない未解決の一覧を作ります。工程1の値を入力として受け取ります。
3. 工程3(`stage3.go`): studio-authoringの表の編集で「単価は正」の候補(検証と、アプリケーションへの使用宣言)を作り、出所のevidenceを`attach-rmt-transcription`で付けます。出所状態と未解決の一覧は、候補の隣の添付(`Attachment`)に置きます。候補の意味の照合値には入りません。
4. 工程4(`stage4.go`): 候補を書き換えずに通常のGitのコミットへ載せ、別の複製が`fetch`と`merge --ff-only`で取り込み、その複製のソースで`cmd/tei`と共通受理を再構築します。受領記録(`Receipt`)は、先頭コミット、木、確認したmainのコミットを固定します。共通受理が示す版と、同じ変更を原文へ直接書いた版も比べて観測します。
5. 工程5(`stage5.go`): 使い捨ての複製だけで試験受理し(状態は`trial-acceptance-on-disposable-copy`)、前後の作業領域をchange-assuranceへ渡します。画面・サーバ・一括処理の三つの実行先と派生した版、旧版R0で開始した実行中の案件とその扱いを影響範囲へ並べます。実行先の派生と実行中の案件はサンプルの宣言です。
6. 工程6(`stage6.go`): 確認器の出力を、対象にできた・対象外・入力不足に分けて一つの一覧にし、審査パケットを作ります。署名の直前に基準版、main、資料の照合値、扱いの決定を再計算し(`Preflight`)、違えば署名しません。OpenSSHで署名した決定を`SubmissionBundle`にまとめ、共通受理へ提出して止めます。署名の理由には出所状態と未解決の件数が入ります。決定の値`accept`は、受理前の提出への署名であり、受理の記録ではありません。

## 共通受理への提出と差し替え点

`SubmissionBundle`(`bundle.go`)が提出物の型です。受け手は`Submitter`インターフェースで、現在は`sharedAcceptanceSubmitter`が`tools/reference-shared-acceptance`の`preview`と`review`だけを使い捨ての基準版の複製へ呼びます。そこで再計算した候補の意味の照合値、審査の照合値、基準版を、署名文面の値と比べます。決定記録の台帳(`.tei/decisions`)が変わらないことも確かめ、`accept`と`reject`は呼びません。

`order-workflow`(先行ブランチ`codex/reference-model-composition-20261012`)がmainに入ったときは、次の点だけを差し替えます。

| 差し替え点 | 内容 |
| --- | --- |
| `Submitter`の実装 | `sharedAcceptanceSubmitter`を、`order-workflow`の共通受理へ束を渡す実装に置き換える。現在の`notConnectedSubmitter`は「受理は未接続」を返す |
| `SubmissionBundle.Target` | 提出先の名前。置き換えた実装の名前にする |
| 署名文面の値の対応 | 束の`CandidateFingerprint`、`ReviewFingerprint`、`BaselineRevision`と、`order-workflow`が共通受理へ渡す値(`change-candidate.json`相当)の対応を決める。`order-workflow`の値が揃うまで、この一致は検証できない |
| 束そのもの | 型は変えない。候補、添付(出所状態と未解決)、審査パケットの照合値、署名つきの決定が一つの型にある |

## 失敗注入と完了条件

失敗注入は、`faults.go`が各工程の記録と既存資産に注入し、観測値を`chain-record.json`の`faultInjections`へ残します。

| 群 | 要件 | 内容 |
| --- | --- | --- |
| `provenance-` | 固有(1・3・6) | 契約が扱えない記述が増える、資料が工程1の後に差し替わる(cross-viewの`-expect-accounting`と署名前の再計算が止める)、対象なし・表だけが理由まで運ばれる、状態や件数を後の工程で変える、署名が理由に結ばれる |
| `checker-` | 固有(2・5・6) | 各確認器の区分が出力に従う。「指摘なし」だけの一覧、対象外を欠く一覧、対象にできたものが無い一覧を拒否する。対象外の「指摘なし」は根拠に数えない |
| `reuse-` | 固有(5) | 再利用候補を合格や実行許可として読むことと、試験受理の版を受理済みとして読むことを拒否する |
| `wf01-` | P-WF-01(4〜6) | 工程5の後に基準版を進める(署名しない)、工程4の後にmainを進める(受領記録の再照合が失敗)、作業環境の履歴を差し替える、署名を別の候補・別の署名者・拒否の決定・別の基準版へ流用する、再計算値の不一致 |
| `mix-` | P-MIX-01(5) | 三つの実行先のうち二つにだけ派生させ、派生していない実行先が再確認の範囲に残る |
| `wf02-` | P-WF-02(5・6) | 旧版R0で開始した案件が影響範囲に版とともに並び、扱いが決まるまで署名しない |
| `through-` | 通し | 識別を一つ差し替え、記録の照合値は再計算する。各工程の記録の単独の確認は通り、通しだけが不合格になる陰性対照 |

完了条件(`completionConditions`)は六つです。通しの完了条件(READMEの内容)を、別々の確認の足し算ではなく、一つの連鎖(`VerifyThrough`)と各群の失敗注入で確かめます。

## 基盤の不足

どの不足も、サンプル側の実装で通しを成立させただけで、基盤では解消していません。詳細は実行ごとの記録の`platformGaps`にあります。

- 出所状態と未解決の件数を、候補、審査、受理境界へ運ぶ欄が基盤にない(添付で運ぶ)。
- 確認器が自分の対象外を出力しない(サンプルが出力から区分を決める)。
- 実行先ごとの派生が影響範囲に入らない(change-assuranceの出力に無い。派生はサンプルの宣言)。
- 実行中の案件を設計側へ返す経路がない(案件の一覧はサンプルの入力)。
- 受理前の候補への署名を、共通受理の決定に接続できない(`witness-decision`は受理した記録だけが対象)。
- 共通受理の経路と直接編集の経路の比較記録がない(工程4で観測する)。
- 認識状態(確認の鮮度)が審査の入力にならない(署名前の再計算で代える)。

## 範囲外

実際のXeadの設計資料、実際の署名者の権限、派生物の生成と実行先の判定の突合、実行中の案件の移行、受理そのもの、`order-workflow`への提出、資格確認の受領記録(`qualify.sh`)、Docker、実環境。
