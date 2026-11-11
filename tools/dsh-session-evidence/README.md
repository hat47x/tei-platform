# DSH Session Evidence ツール（TEI #2509 / TAI #168 検証基盤）

外部AIハーネス **DeepSeek Harness（DSH）** からTEI Product CLIを実呼び出しする
H2（Tool-only）／H3（Product Skill付き）比較試験の**証拠を取得・相関・分類**する
小さなツール群です。TAI #168／SK-02 Runbook §19〜§24の実行票に対応し、
**新Runner・独立Verifier・共通Permission Engine・第二Oracleは作りません**。

## 構成

| ファイル | 役割 |
| --- | --- |
| `build-participant-root.mjs` | 参加者Sessionに見せる**唯一の内容**（固定バイナリ・公開Workspace byteコピー・固定Task・H3はProduct Skill）を組み、digestのmanifestを書く。pin不一致は**build拒否** |
| `sync-participant-slot.mjs` | 各Session前に、固定slotパスへステージング内容を**検証済みコピー**（byte digest・件数の全件照合。腕残留は拒否） |
| `run-dsh-session.mjs` | 独立したDSH Sessionを1本起動し、CLI・起動引数・cwd・exit code・stdout/stderr（byte digest）・Session ID・usage・最終回答・永続Session Logを1件のrun記録に束縛する。`--collect-only`で保存済み証拠から再構成できる |
| `collect-session.mjs` | 永続Session Log（`session.v4.jsonl.zstd`）の**全zstd frame**を展開し、Tool call↔resultのcallId相関、TEI argv分類（PowerShell変数・call operator対応）、Skill Tool呼出し、切詰め・混在stderr・exit suffixの判別、model identity（provider解決後model）を抽出する |
| `build-trial-group.mjs` | run記録からSK-02 Checker投入用のtrial-group JSONを組み立てる（観測値を隠さず・書き換えない） |
| `score-answers.mjs` | 最終回答に対するExpected必須項目の機械事前点検（Reviewer採点の補助） |
| `collect-session.test.mjs` 等 | 上記の**独立回帰テスト**（合成fixtureのみ。実Session・実Provider・実Product応答は使わない） |

## 使い方

```bash
# 0. 回帰テスト（合成のみ）
node --test "tools/dsh-session-evidence/*.test.mjs"

# 1. 腕ごとのステージングRootを組む（H2: Skill無し / H3: Skill付き）
node tools/dsh-session-evidence/build-participant-root.mjs \
  --out <stage> --binary <tei.exe> --binary-digest sha256:... \
  --workspace-src reference/workspace --task <task.txt> --task-digest sha256:... \
  [--skill-src skills/yi-tei-modeling --skill-digest sha256:...]

# 2. 各Session前にslotへ検証済みコピー
node tools/dsh-session-evidence/sync-participant-slot.mjs --stage <stage> --slot <slot>

# 3. 独立Sessionを1本実行し、証拠を保存する
node tools/dsh-session-evidence/run-dsh-session.mjs \
  --root <slot> --overlay <overlay.yml> --task <task.txt> \
  --label h2-a --out <evidenceDir> --store <storeDir> [--arm H2]

# 4. trial-group JSONを組み、SK-02 Checkerで分類する
node tools/dsh-session-evidence/build-trial-group.mjs <evidenceDir> <stagesParent> -o trial-group.json
node <tai-repo>/evaluations/tools/check-sk02-pair-eligibility.cjs trial-group.json \
  --task-file <participant task>
```

## 回帰テストが守るもの（合成fixture）

- **全frame展開**: 連結zstd frameのうち先頭だけを展開する単一frame復号は
  後続frameを落とす偽陰性になる（TEI #2515 H6-E6の観測）。全frame展開を検証する
- **call/result相関**: 件数が一致してもcallId対応しないものをunpairedとして記録
  （過去の「Tool call 56／result 55」を型として保持）
- **切詰め**: pruner閾値（8,192文字）ちょうどのstdout、またはJSONがparseできない
  stdoutを**疑い**として記録。`[stderr]`混在・`EXIT=` suffixは切詰めと誤判定しない
- **argv分類**: 初回許可群（`version`／`commands`／`model catalog`／`inspect`／
  `trace`／`validate`）外（`source-index`・`dbcheck`等）を`outside-initial-readonly-profile`
  として記録。Workspace targetsの混在も記録する。PowerShellの
  `$exe = ...; & $exe <cmd>` 形式を解決してから分類する
- **Root構築/slot sync**: pin不一致時のbuild拒否、再構築時の腕残留消去、
  tree digestの決定性、改変の検出性

## 限界（このツールが証明しないこと）

- 記録の**真正性**（ログが改ざんされていないこと）は外部の独立Observationが担う
- モデルへ渡った**全Byte**の受領は、Session Logのresultと回答内のresult固有値の
  引用で裏付るが、Provider側の請求額・実model revisionの同一性はProviderの応答metadataが必要
- 正答・Skill効果・Owner意味受理・OS/Network強制・Network egressは**評価しない**
  （SK-02 Checkerの`eligible-to-score`は比較資格の入口に過ぎない）
- OS隔離の検証は実行環境の強制機構（例: Windows NTFS ACL）に依存する
