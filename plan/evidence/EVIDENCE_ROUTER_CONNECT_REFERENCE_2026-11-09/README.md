# 証拠の宣言から、既存の Router で実際に入れるか（2026-11-09）

- 対象: [観測側の同じ試み](../OBSERVATION_ROUTER_CONNECT_REFERENCE_2026-11-09/README.md)、[6つの宣言の調査](../BINDING_FORMS_CONSUMER_SURVEY_2026-11-09/README.md)、[判断材料の第12節](../../workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)
- 位置づけ: **参照実装と実行記録**。製品 CLI へ繋ぐべきかは示しません。
- 計測した版: `raw/head.txt`（`origin/main` `9ece5adc` の作業ツリー＋このPRの追加）。

前の記録は `ObservationProviders` の断絶を参照実装で閉じました。この記録は、**同じ形が `EvidenceProviders` でも成り立つか**を試します。観測側と違い、**Router が `durable.ActionRecord` を要求する**ため、`Environment` だけでは入り口に立てません。

## 1. 何を作ったか

`tools/reference-evidence-router-connect`（新規・読み取り専用）。観測側と同じ4段を分けて数えます。

| 段 | 使う既存の関数 | 数える事実 |
| --- | --- | --- |
| 前提 | `rec.Mode == durable.ModeEvidence` | 証拠ゲートか（違えば別の状態で拒否） |
| 解決 | `evidence.Binding` | `resolved` |
| 選択 | Host の registry を引く | `selected` |
| 入場 | `evidence.Router.Collect` | `invoked` |

## 2. 観測

```text
== registered ==
  collected=1 refused=0 invoked=1
    action=reference-gate  provider=health.reference  status=COLLECTED       resolved=True selected=True  invoked=True  outcome=pass

== unregistered ==
  collected=0 refused=1 invoked=0
    action=reference-gate  provider=health.reference  status=NOT_REGISTERED  resolved=True selected=False invoked=False outcome=-
```

## 3. この観測が示すこと

- **同じ4段の形が `EvidenceProviders` でも成り立ちます。** 既存の関数だけで、宣言 → 解決 → 選択 → 入場が通りました。
- **`resolved` と `invoked` の分離は、証拠側でも同じように効きます。** 未登録でも `resolved=True` のまま `invoked=False` です。
- **証拠ゲートでない記録は、専用の状態で拒否します**（`NOT_EVIDENCE_GATE`）。Router の汎用エラーに流すと「なぜ証拠が出ないか」が読めなくなるため、参照は手前で理由を分けました。**これは観測側には無かった段です** — 観測の `Request` は kind だけで、記録の種別を持たないためです。

## 4. この記録が示さないこと

- **証拠が外部状態を反映していること**。CLI の Provider は `Outcome: pass`, `SourceRef: "reference-only"` を返すだけで、何も読みません。
- **`pass` が正しいこと**。`pass` は provider の主張で、外部状態と一致する証明ではありません。
- **製品 CLI へ `Collect` を足すべきか**。C-4（外部 Host に委ねる）も候補のままです。
- 開示 Scope、Artifact の信頼、現在の Grant、認証・権限・永続 receipt。
- `ResourceBindings`／`Instances`／`Bindings` についての同様の確認（未実施）。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（2つの場合） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域 |
| `raw/registered.json`・`raw/unregistered.json` | 2つの場合の出力 |
| `raw/summary.txt` | 上の表を作る元の一覧 |

`*.err.txt` はすべて0バイトです。

## 6. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `2f7020a503974dc23f7c5bf3c06d3fbd1a634a1e480f2ac05218c4ebec839b2e` | 2296 |
| `raw/head.txt` | `c249fed14e40690c8a101434c40a357f70e1bbd63153bd2b9f44c9e74cbdf07b` | 73 |
| `raw/pre-status.txt` | `8c1d17b6785ac232268365ba79ac4578ff6307a725630c82428cdf920c255ae6` | 229 |
| `raw/registered.json` | `84852e998143ee7e9d19246648f643dbb2781fc825800db6e933e8330fab8e98` | 1071 |
| `raw/unregistered.json` | `df4492b7cd2ad485e09abff3b299835709fa16864224149fd7720d6c988eba8e` | 1123 |
| `raw/summary.txt` | `eb18b78d67bd6397f07a7ef2bffd43db148d81d49979d0c7b33a3ef35d6afa05` | 462 |

規則の指紋は `sha256:3fa298f421e731ee3d612eabe474134e579d265995a49d8605da6f54378308bb` です。
