# representation-binding fixture（#4353・#4133・#2506）

`cases.json`は表現インスタンスとFunction Data Contextの束縛、`expected.json`は**手書きの独立Expected**です。参照ツールの出力をExpectedへ写していません。

- 根拠：#4350のRSP-01〜06表、#4133（表現型トポロジー・媒体時間軸）、#2506（Function Data Context）。
- すべて合成descriptorです。実在の画面・文書・音声・動画・顧客データを含みません。
- 判定語彙・15表現型・4時間軸roleは [`tools/reference-representation-binding-check/README.md`](../../../tools/reference-representation-binding-check/README.md) を参照してください。
- Expectedを変更する場合は、根拠Issueの記述を先に確認し、参照ツールの出力を見てから書き換えないこと。出力を見て直したExpectedは独立検証ではなくなります。
