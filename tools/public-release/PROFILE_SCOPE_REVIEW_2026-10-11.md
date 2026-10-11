# Public開発用Source構成の判定 — 2026-10-11

**暫定判定: 公開不可（NO-GO）。** これは、Private repositoryを将来Publicの開発正本へ移す場合の、SourceとGoテストの選択根拠です。配布用の軽量Sourceとは条件が異なります。

## 根拠のスナップショット

- 調査対象: Private main `def7efc6b864c0fc4da61b451909fc002531159b`（Git recursive tree: `truncated=false`）。
- 追跡通常ファイル: **1,762件**。公開開発用の暫定許可リスト: **1,158件**（約66%）。保留: **604件**。
- Goファイル: **941/941件を候補に含む**。そのうちGoテスト: **587/587件を候補に含む**。
- 通常の `testdata/`、`fixture/`、`fixtures/`、`golden/` の計**16件**を候補に含む。
- Source規模の目安: 選択約8.97MB、保留約6.22MB。Git Treeのblob sizeの合算であり、ダウンロードサイズではない。
- 全選択パスはSource Git Tree上に存在し、ファイルモードは`100644/100755`、大小文字無視の重複は0件。既知のsymlink / submoduleは0件。
- 元の軽量配布候補（Draft PR #1195）は**675件**で、上記基準時点より古いrevisionに固定されたもの。Goテストを大幅に省略するため、Public開発正本としてそのまま採用しない。

## 公開する理由があるファイル

| 位置付け | 判断 |
| --- | --- |
| `cmd/`、`internal/`、`examples/` | CLI、意味モデルと実装、直接のテストを維持 |
| `reference/workspace/`と選択したReference | CLI実行／再現例。権利・機密情報確認を別に実施 |
| `product/`、`requirements/`、Architecture、導入・運用の説明 | 利用者・開発参加者の入口 |
| 正本ADRと現在のGate判定 | 設計判断と過去に固定したEvidenceを区別して掲載 |
| `tools/`のGoソース・テスト | 本番用機能とは限らないが、既存テスト範囲を失わず公開開発を始めるために暫定維持 |
| 公開開発用の限定したTool README | First Web Target、Shared Acceptance、Chat CLI Hostなど、実装を理解するための入口 |

## 初期公開から保留するもの

| 種類 | 保留件数 | 判定理由 |
| --- | ---: | --- |
| `plan/` | 314 | 個別作業の積み重ね・調査記録。正本と必要なEvidenceは別に確保 |
| `bootstrap/` | 126 | 過去のSource再構築用断片。通常利用者が操作する対象ではない |
| `tools/` | 76 | Go実装・Goテスト以外を中心とする運用補助や単発の証跡・案内 |
| `docs/` | 49 | 移動案内と旧HTML資料、監査履歴等 |
| `reference/` | 34 | 補助Reference、第三者資料の移管等。実行・権利確認まで保留 |
| その他 | 5 | `.github`、`CLAUDE.md`、research補助1件、skill 2件 |
| **合計** | **604** | **Privateから削除したことを意味しない** |

`tools/`のGo testを「一般利用者が直接使わない」という理由だけで消すと、元の`go test ./...`と公開側テストの差を不可視にする。開発用Sourceと単なる配布用Sourceの違いとして維持する。

## 検査器と実施結果

- 暫定Pathリスト: [`proposed-development-public-paths.txt`](proposed-development-public-paths.txt)。
- オフライン監査: [`audit-profile-coverage.py`](audit-profile-coverage.py)（固定SHA、パス安全性、Git Treeの存在、README開発状況表示、全Go/test/通常fixtureのcoverage）。
- 合成テスト: [`audit-profile-coverage-test.sh`](audit-profile-coverage-test.sh)。正常、Goテスト欠落、Go source欠落、fixture欠落、異なるrevision、重複、不正パス、README表示不足の拒否を検証。

**合成テストPASS:** ローカルのLinux環境で`python3 -m py_compile`、`bash -n`、合成Gitリポジトリでのテストを実行。GitHub上に登録した2つのスクリプトのblob SHAとローカル実行ファイルの`git hash-object`が一致することも確認済み。

実Gitリポジトリの完全checkoutをこの環境へ配置できていないため、`audit-profile-coverage.py`を実TEIの`def7efc6...`に対して**実行したわけではない**。上記の1,158件の存在とcoverageは、Private GitHub Git TreeのAPI応答と許可リストの照合結果に基づく。

チェックアウトを利用できる信頼済みLinux/WSL2 Hostでは、次の順で検査する。

```bash
sha=def7efc6b864c0fc4da61b451909fc002531159b
python3 tools/public-release/audit-profile-coverage.py \
  . "$sha" \
  tools/public-release/proposed-development-public-paths.txt \
  --profile development
bash tools/public-release/audit-profile-coverage-test.sh
```

ただし、このコマンドのツールファイルは現段階ではDraft PR内にあり、固定したSource commit自身に含まれるとは限らない。必ずReview済みのツールファイルを用意した上で、入力Git commit SHAは変更せずに実行する。

## 未達Gate

1. 許可リストが参照する**全**コード・Goテスト・Runbook・Source Fixtureの依存閉包（Go packageと非Goデータ）。単に全`*.go`を選んでも`go:embed`、動的なfile reader、外部サービス依存は保証しない。
2. 本ツールと候補生成器の**同一ソースファイル**をtrusted full checkoutで実行し、生成後に独立検証すること。
3. `go list ./...`のPackage集合、`go test -count=1 -timeout 60m ./...`、`go vet ./...`、CLIの`reference/workspace` Validate。元Repoと公開候補を比較し、静かなPackage/Test消失を発生させない。
4. Markdown/HTMLとコード内の参照閉包。Public向けの最新Sourceには、作業経緯へのリンクや非選択パス参照が残り得る。
5. 全Sourceの機密・PII・API key・第三者利用条件・著作権表示の監査。**特にReferenceの外部由来・研究関連の利用条件が不明なら公開しない**。
6. 現行mainの変更を固定Sourceから適切に継承し、新Public Repo上で別のexact HEAD CIとbranch protectionを新規確立すること。

以上を満たすまで、Public向けGo test同等性や公開適合性を**PASSと判定しない**。Private Git履歴削除・Public化・既存PRの自動マージは行わない。

参照: #1197（Public文書）、#1190（リポジトリ構成）、#1188（機密監査）、#1189（公開用CI）、#941（Reference第三者資産）、Draft PR #1195（候補生成器）。
