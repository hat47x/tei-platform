# reference-projection-edit-check（#2497 / TASK 7）

Property/Table行の編集を、**既存のSource書込入口の手前で**判定する参照です。書込は`reference-shared-acceptance`（`preview`・`review`・`accept`）だけが行い、ここにParser・Writer・Acceptanceの写しはありません。

```text
Source ─(既存のsyntax anchor)→ 行(Subject ID, 値のspan, Revision)
行の変更 ─→ Judge ─→ 候補(set-attribute-type) ─→ 既存preview ─→ 既存review/accept ─→ Source
                                                                              └→ 別Read・Reparse・再Projectionで照合
```

## 判定の状態

`APPLIED`以外では、Workspaceの1byteも変わりません。

| 状態 | 条件 |
| --- | --- |
| `APPLIED` | 既存Acceptanceが書いた |
| `ELIGIBLE` | （Judgeのみ）Acceptanceへ渡せる |
| `NOOP` | 新しい値が現在の値と同じ |
| `STALE` | 行を読んだ後にSourceのRevisionまたは値・spanが変わった。再Projectionで復帰できる |
| `AMBIGUOUS` | 対象がSource上の定義ちょうど1つに解決しない |
| `UNSUPPORTED` | 対象にSource定義が無い、またはその操作が定義されていない |
| `WITHHELD` | 呼び手が書込Grantを持たないと申告した（Grantの真偽はここでは検証しない） |
| `REJECTED` | 既存の`preview`が拒否した（診断コードを保持） |

## 逆編集が成立する条件（TASK 7）

次がすべて揃う操作だけを渡します。

1. 対象がSource定義ちょうど1つに解決する。
2. 編集する値が、その対象が所有する1属性の恒等な表示である。
3. 変更がSource上の1 spanに閉じる（既存の`set-attribute-type`が型参照トークン1つだけを置換する）。
4. 行のRevisionと現在のRevision、span内の値が一致する。
5. 既存`preview`が受理可能で、適用後に再読込・再Projectionで編集値が出る。

成立しても、Source書込権限とOwnerの意味受理は別に必要です。このツールは両方を与えません。

## 写像側の入口

`EditableThroughMapping`は、既存`reference-facet-mapping-check`の判定が`preserved`かつ逆写像`eligible`のときだけ真を返す**写像全体の門**です。損失のある写像の中から安全な部分操作を探しません。

**既知のGap**: 損失はStructure・RepresentationのFacetから決まるため、Graph→Treeの写像を「Nodeの属性」のSlotに絞り直しても`lossy`・`ineligible`のままです（`TestNodeAttributeSlotOfALossyGraphTreeMappingStaysIneligible`）。操作相対の編集可能性を写像行として表すには、#4351でSlot単位の損失が要ります。

## 試験

```bash
go test ./tools/reference-projection-edit-check -count=1
```

試験は`reference/workspace`の隔離コピーに対して、実際に`reference-shared-acceptance`をビルドして呼びます。Expectedは手書きで、Judge・Applyの結果から作っていません。

## 何を言っていないか

- **実UI・実Provider・実DBは使っていません。** Property/Table行はこのパッケージ内の最小の型です。
- **Owner意味受理と正式Acceptance判断は含みません。** 隔離コピーでの機械的な受理だけです。
- **A→B→Aは検出できません。** Revisionは内容から導くため、元のbyteに戻れば同じ版に見えます（`TestABAIsNotDetectedByContentRevision`）。
- **`AMBIGUOUS`を実Sourceでは作れていません。** 有効なSourceでは1 Subjectが複数の定義を持てないため、判定の入口だけを行の申告で試しています。
- **Graph→Treeの候補列挙は未実施です。** 実データに複数Edgeの例が無く、判定は写像checkerの全体判定までです。
- **操作は`set-attribute-type`だけです。** 他の操作、Markdown、Graph・Chartの編集は対象外です。
- Linux（WSL2）の1回の観測です。
