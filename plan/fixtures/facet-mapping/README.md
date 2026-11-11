# facet-mapping fixture（#4351・#2497）

`cases.json`は写像の入力、`expected.json`は**手書きの独立Expected**です。参照ツールの出力をExpectedへ写していません。

- 根拠：#4350の負例表、#2497（写像・損失・逆写像）、#4133（表現型トポロジー・時間軸）、#2506（Function Data Context）。
- すべて合成データです。実在の顧客・業務データ・非公開資料を含みません。
- XMAP-01〜10は#4350の負例表に対応します。XMAP-11〜15は受入条件が要求する追加ケース（合成写像の非単射、複数Relation、異なる粒度、時間軸不一致、Source変更）です。XMAP-16は正常系です。
- Expectedを変更する場合は、根拠となるIssueの記述を先に確認し、参照ツールの出力を見てから書き換えないこと。出力を見て直したExpectedは独立検証ではなくなります。

判定語彙・損失コード・12列の契約は [`tools/reference-facet-mapping-check/README.md`](../../../tools/reference-facet-mapping-check/README.md) を参照してください。
