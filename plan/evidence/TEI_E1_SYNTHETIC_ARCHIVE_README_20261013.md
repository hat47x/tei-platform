# #1119 E1合成証跡の保全と再検証

## 取得元と原本

- GitHub Actions E1：[`#38267656696`](https://github.com/hat47x/tei-platform/actions/runs/38267656696)（SUCCESS）。PR #1442（**検証専用・未マージで閉鎖**）のGitHub仮統合commit `466b1d10080460390d46b3b342d8bc9c19e09208` を実行。
- PR #1442の元Sourceは #1163 [Draft PR #1262](https://github.com/hat47x/tei-platform/pull/1262) のhead `ef0ea673a988536558970cae86f00672f0599b8a`、仮統合のbase mainは `532c31c2e7d0667ffbab966b1af327e1de30b3f3`。
- [Artifact #11755671316](https://github.com/hat47x/tei-platform/actions/runs/38267656696/artifacts/11755671316) は **2026-10-19 15:05 UTCまでの短期保管**。ZIP全体のSHA-256 `8519e0bb06377f1943a97bb4b7f839f46df339ec12690df3f465630902ad235a` をGitHubから取得した記録と独立計算で突合済み。
- manifest原本は [`TEI_E1_SYNTHETIC_MANIFEST_20261012_38267656696.json`](TEI_E1_SYNTHETIC_MANIFEST_20261012_38267656696.json)、ZIP内の原本SHA-256 `60c2babe223fa89383bea24cd718407dbc8e037f8d29aa448858c52997a70584`。manifestは先行PR #1466でmainへ統合済み。
- この追加分はZIP内の**8 JSON原本をバイト単位でコピー**した [`tei-e1-synthetic-38267656696/`](tei-e1-synthetic-38267656696/)。8 JSONのGit blob SHAと、元ZIPから算出したGit blob SHAを全件照合した。これらは人の入力や実運用データではなく、実HTTPと合成Provider／試験用オーバーレイの証拠である。loopbackアドレスや一時PIDはCIの実行記録であり、後日再利用できるURLではない。

## 改ざんと誤昇格の拒否

新しい `tools/verify-critical-path-e1-archive.py` はmanifest原本のSHA-256を固定し、8件全てのraw byte SHA-256、余剰/欠落ファイル、原本のGit仮統合HEAD、R0→R1とR2の意味同値・異Source版、実Web正常/単価0拒否/N11、製品資格の**否定フラグ**を検査する。独立した負例テストは `tools/test-critical-path-e1-archive.py`。

```bash
python3 tools/verify-critical-path-e1-archive.py
python3 tools/test-critical-path-e1-archive.py
```

専用CI `.github/workflows/tei-e1-archive-ci.yml` が変更時に実行する。この検査はアーカイブの**バイトと内部整合性**の確認であり、以前のGitHub runnerが信頼できたこと、当時の実DBや認証の実効性を新たに保証しない。

## 製品Gateの非主張

- `productGate=UNQUALIFIED`、`runtimeVersionBindingVerified=false`、`semanticEquivalentR2RejectedByTarget=false`、`independentDatabaseRead=false`、`humanAcceptance=false` を保持。
- R0（単価規則の欠落）とR2（Sourceに空行だけ付加しWorld/Manifest意味同一）は**外部ハーネス**のSource版検査で不一致を検出。**Web自身は拒否していない**。
- 正常、単価0、N11はそれぞれ異なる合成Webバイナリ。実PostgreSQL、current Host Authority、実効的な受理済みSource束縛、独立したDB read、第二Targetおよび人間受理は未達。
- この8件は**特定のGitHub仮統合コミット**を再閲覧できる保存記録であり、現在のmain/PR headの自動的な資格更新や、後日の再実行が成功する保証にはならない。
