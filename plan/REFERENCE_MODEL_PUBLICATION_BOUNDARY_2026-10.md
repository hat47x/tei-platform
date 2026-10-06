# CONCEPTWARE Reference Model公開境界 — 2026-10

- 状態: **Safe boundary fixed / model-embedded terms observed / redistribution scope pending**
- 対象: X-TEA / XEAD系Reference Model「CONCEPTWARE／受注生産」
- 関連Issue: [#941](https://github.com/hat47x/tei-platform/issues/941)
- 関連Gate: [Reference Model Transfer Evidence Gate](REFERENCE_MODEL_TRANSFER_GATE_2026-10.md)

## 1. 結論

2026-10-06時点では、XeadModeler / XeadEditor / XeadDriverのsource codeと、CONCEPTWARE Reference Model本体を分けて扱う。

XeadModeler / XeadEditor / XeadDriverは、Reference Model Transferで固定した各revisionの`LICENSE`を確認し、BSD-3-Clauseの再配布・改変条件を確認した。この条件は各source repositoryへ適用する。

一方、Libraryに保存されたexact `MfgOrder.xeaf`（SHA-256 `f4b73b36abd8398aa25ad86ecd7570ae0334e7fd93c3ee204c5a05bddebdb754`）の`System.Remarks`には「利用条件」が含まれている。そこでは、本データの著作権・所有権の帰属、利用時の免責に加え、再配布・転載時に著作権表示と利用条件表示を改変しないこと、条件を守る限り無償で利用できること、商業的な再配布・転載等は事前相談とすることが示されている。

したがって「明示条件が存在しない」とは扱わない。ただし、この独自条件がpublic GitHub Repositoryでの再配布、TEIのMIT Licenseとの併存、変換artifact・抜粋fixture・manualへの適用範囲、商業利用され得るOSSからの配布をどこまで許すかは、この文言だけから確定しない。**利用可能であることと、TEI Repositoryへ安全に同梱・再配布できることを分ける。** これらの範囲を確認するまで、原model / manualはRepository外sourceとして扱う。

本書はTEI Repositoryで採る公開方針を定めるものであり、一般的な著作権法上の結論を示すものではない。

## 2. 確認したEvidence

### 2.1 XEAD source code

次のexact revisionで`LICENSE`を確認した。3 repositoryのlicense本文は同じBSD-3-Clauseである。

| Source | Revision |
| --- | --- |
| `xead/XeadModeler` | `68f1535b1c5dd80a327dfced37be401ecd3093c9` |
| `xead/XeadEditor` | `ee19a27609b2749110ec39b62d4540693347fe09` |
| `xead/XeadDriver` | `6c077773b74c6d7dbcc253835cf8d347e4ffb1ef` |

Source codeを再利用する場合は、BSD-3-Clauseのcopyright notice、条件、disclaimer等を維持する。

### 2.2 CONCEPTWARE Reference Model

Repository外のexact source snapshotを確認した。

`MfgOrder.xeaf`の`System.Remarks`には、Reference Model自身の利用条件が記載されている。観測した内容は次の範囲である。

- 本データの著作権・所有権の帰属を明示している
- 利用に伴う損害について免責を置いている
- 再配布・転載時には著作権表示と利用条件表示を改変しないよう求めている
- 条件を守る限り無償利用できる旨を示している
- 商業的な再配布・転載等については事前相談を求めている

このEvidenceにより、「Reference Modelに利用条件がない」という仮説は棄却する。

一方、この条件はSPDX等で識別できる標準OSS licenseではなく、少なくとも今回確認した文面だけでは次を確定できない。

- public MIT Repositoryへ原modelを同梱する場合のlicense併存方法
- XMLの一部をtest fixtureとして公開する場合の扱い
- Table / Field / Function / Task等の名称・関係を大量に転写したartifactの扱い
- 原modelをTEI形式へ変換した派生artifactの扱い
- installer同梱manualへ同じ条件が及ぶか
- 商業利用されるOSSでの参照・同梱・再配布が「商業的な再配布・転載等」に当たる範囲

したがって、TEIのpublication policyは引き続き安全側へ置く。利用条件の存在は記録するが、曖昧な範囲をTEI側で許諾済みと解釈しない。

## 3. 現時点のpublic Repository境界

### 3.1 公開してよいEvidence

明示的な追加許諾がなくても、Reference Model Transferでは次をpublic Repositoryへ残す。

| 種類 | 扱い |
| --- | --- |
| artifact identity / SHA-256 | 公開可 |
| element数等の集計値 | 公開可 |
| source format種別、revision、取得元へのlink | 公開可 |
| TEI独自のClassification / pressure / mapping contract | 公開可 |
| TEI側のloss / unresolved分類 | 公開可 |
| 原文を含まないcross-view診断結果 | 公開可 |
| 原modelを複製しない独自fixture | 公開可 |
| 最小限のsource locator | traceabilityに必要な範囲で公開可 |

source locatorは、可能な限りelement kind、legacy ID、snapshot内位置、locator digest等で表す。個別名称をtraceability上どうしても必要とする場合は最小限に留め、名称一覧・説明文・順序・関係をまとめて原modelの代替物になる形では公開しない。

### 3.2 Repository外に留めるもの

明示的な許諾または適用licenseを確認するまでは、次をpublic Repositoryへ入れない。

- `MfgOrder.xead` / `MfgOrder.xeaf`の原物
- 配布archiveやinstaller同梱model / manual
- raw XML、HTML、PDF等のまとまった抜粋
- Script、SQL、説明文、画面文言、manual本文のコピー
- screen captureやmanual pageの複製
- field / Function / Task等の網羅的な名称catalog
- 原modelの選択・配列・関係を実質的に再現する公開fixture
- 原modelを直接変換しただけのTEI形式artifact
- 原modelを改変して作った派生版を、再配布許諾があるものとして公開すること

少量であっても、原文をtest fixtureへコピーする必要が生じた場合は、この境界を自動的に緩めず個別に確認する。

## 4. RMT artifactの作り方

RMT-1〜3は、この公開境界を理由に停止しない。full source inventoryと転写判断に原modelの内容が必要な場合、原sourceはRepository外で読む。public Repositoryへは、元の内容を複製しないreceipt / mapping Evidenceを残す。

概念上は次の二層に分ける。

~~~text
Repository外 source snapshot
  exact file + digest
  full source content
        ↓ local analysis
public Evidence
  source artifact digest
  source locator / locator digest
  aggregate inventory
  classification
  TEI target / native / unresolved
  diagnostic / loss reason
~~~

public `source-inventory.json`や`transcription-map.json`を作る場合も、「完全なinventory」という名前を理由に原modelの名称・説明・Script本文を複製しない。転写coverageはsource locatorとdigestで照合し、必要な元内容はRepository外sourceへ戻って確認する。

## 5. TEI独自fixtureの条件

Reference Modelから得たpressureを再現するために、TEI独自fixtureを作ることはできる。ただし、原modelのデータを別形式へ写したものを「独自fixture」とは呼ばない。

独自fixtureでは、少なくとも次を守る。

- 原modelの文章、Script、SQL、画面文言をコピーしない
- 原model固有の名称一覧や配置をそのまま再現しない
- fixtureが検証するTEI上のpressureを明記する
- fixture自体のauthoritative sourceをTEI Repository内に持つ
- 原modelとの一致を、互換性や原物再現のEvidenceとして主張しない

構造をどこまで参照できるか判断できない場合は、Reference Model由来のfixtureとして公開せず、一般化した最小pressureだけを独立に設計する。

## 6. 明示確認が必要な事項

公開境界を広げるには、model内に記載された利用条件の適用範囲を十分に説明できる追加資料を確認するか、権利者から書面で条件を確認する。既に利用条件が存在するため、「licenseの有無」ではなく「TEIで予定する公開・変換・再配布形態へどう適用されるか」を確認対象にする。

確認時には少なくとも次を分けて尋ねる。

1. `MfgOrder.xead` / `MfgOrder.xeaf`の原物をTEIのpublic GitHub Repositoryで再配布してよいか。
2. XML等の一部をtest fixtureとして公開してよいか。
3. Table / Field / Function / Task / Business Procedure等の名称と関係をmapping artifactとして公開してよいか。
4. 原modelをTEI形式へ変換・改変したartifactを公開してよいか。
5. 商用利用されるOSSから参照・同梱・再配布してよいか。
6. 必要なcopyright notice、license notice、attribution、link等は何か。
7. CONCEPTWARE modelに適用する既存licenseがある場合、その名称と対象範囲は何か。

連絡先は、公開されているXEAD repositoryの`LICENSE`や公式portalに記載された経路を優先する。個人間の回答をそのままpublic Repositoryへ転載せず、公開してよい範囲と条件も併せて確認する。

## 7. 境界を更新する条件

次のいずれかを確認した場合に、この文書を更新する。

- model内の既存利用条件について、TEIの公開・変換・再配布形態への適用範囲を明確にする公式説明
- 対象model / manualへ明示的に適用される追加license
- 権利者による再配布・改変・公開範囲の明示
- 公式配布物内にある、対象artifactと適用範囲が特定できる追加条件

「以前から公開されていた」「別製品が実装した」「無償で利用できる」といった事実だけでは、公開境界を広げない。

確認できない場合は、この安全側boundaryを維持する。その状態でもRMTはRepository外sourceを用いて継続し、権利条件が未確認であることをEvidenceへ残す。

## 8. 参照した公開資料

- X-TEA portal: <https://dbc.in.coocan.jp/>
- XeadModeler: <https://github.com/xead/XeadModeler>
- XeadEditor: <https://github.com/xead/XeadEditor>
- XeadDriver: <https://github.com/xead/XeadDriver>
- TALON「カスタマイズ放題 生産管理」: <https://talon.jp/products/businesspackage/businesspackage-mfs/>
- TALON「業務システムのモデリングを考える会」: <https://talon.jp/info/334/>
- ライブモデリングin大阪: <https://benkyoenkai.connpass.com/event/133860/>

公開資料の説明は利用意図を確認する補助Evidenceとして使う。原modelの再配布条件は、明示的なlicenseまたは権利者確認を優先する。
