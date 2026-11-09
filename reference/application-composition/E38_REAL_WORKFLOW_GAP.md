# E38：実案件PRのSource差分と、まだ計測されていない作業を分ける

Issue [#1016](https://github.com/hat47x/tei-platform/issues/1016) の、独立Composition Profileを追加する前に実案件の追加工数を実測する検討を継続する。

前段の[E34固定実GitHub差分](e34_project_bridge_pr1567_delta.json)は、案件Go＋Litの[PR #1567](https://github.com/hat47x/tei-platform/pull/1567)で、新規Goファイル2件、既存Host・READMEの更新2件、計4変更パスを示す。E38は**既存E28の申告スキーマと集計器**だけを利用して、変更前後のGit Blobを持つ更新2件を、実Sourceの参照イベントに変換する。新規追加の2件は変更前Blobが存在しないため、E28の`changed-source`として捏造せず別件数に保留する。

- 実際の入力：E34でGitHubから採取した**固定改訂のSourceメタデータ**。後続HEADを自動反映しない。
- E28への投入：`complete:false`、`coverage`のすべての種類を`untracked`とし、適格な2件の更新イベントだけ提出する。
- 出力：`modifiedSourceRecords=2`、`newlyAddedSourcePaths=2`、`e28Scope=partial-not-verified`。**全作業の変更ファイル数4件を、E28の完了済み2件と誤表示しない**。
- 人的判断数・AI質問数・手動接続数・検証済み工数・改善率・実行権限は引き続き`null`または未検証。GitHubコミットやPRのFile Diffからは導出しない。

同じGo関数を試験用の2案件Scopeで再利用し、曖昧なProvider IDだけ選択を要求した[E37](E37_REAL_PROJECT_SCOPE_CHOICES.md)とは役割が異なる。E37は構成の**必要な判断項目**を示すが、その判断に利用者が何回応答したかを実測した証拠ではない。

```sh
node --test reference/application-composition/e38_real_workflow_gap.test.mjs
node reference/application-composition/e38_real_workflow_gap.mjs .
```

既存E28への接続・固定PR差分・不正Path・欠落・重複・前後Blobの取り違え・工数補間拒否を試験する。Source分類と現在のProduct Authorityを混同せず、既存のResolver/Lock/Profileを複製しない。

**次の実測**：生成AIとの案件構成会話で利用者に必要だったProvider選択・Scope確認と、その根拠、実際の質問・手動操作・再検証を記録する。実施されていない対照群の生産性向上率を報告しない。