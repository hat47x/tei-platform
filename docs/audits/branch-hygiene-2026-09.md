# Branch Hygiene Audit — 2026-09

この文書は、TEI Repositoryに蓄積したremote branchを安全に整理するための**監査記録**です。

Branch運用の恒久ルールは[`CONTRIBUTING.md`](../../CONTRIBUTING.md)と[`AGENTS.md`](../../AGENTS.md)を正とします。この文書は、2026年9月時点で何が確認でき、何がまだ未確認かを残すsnapshotです。

## 1. なぜ監査するのか

TEIでは過去の作業で、同じworkstreamについて`-final`、`-sync`、`-current`、`-rebased`、`-v2`等の派生branchが多数作られてきました。

Branchが履歴保管場所になると、次の問題が起きます。

- どのbranchが現在の作業面なのか分からない
- mainへすでに入った変更と、まだ入っていない変更を名前から判別できない
- stale branchを誤って次の作業baseにする
- 同じ修正が複数PRへ重複する
- branch名の`final`や`ready`が、Issue / Gateの正式stateと誤認される

したがって、履歴はmainとPRへ残し、remote branchは現在進行中の責務だけを表す運用へ移行します。

## 2. 既に得られたEvidence

PR #578の監査では、2026-09-12時点で次が記録されました。

```text
remote branches examined: 528
branches with commits not already represented in main: 15
cleanly integrable branches: 7
conflicting branches requiring separate review: 8
```

この監査の重要な示唆は、**branch数の大半が「未統合作業」ではなく、mainへ内容が入った後も残っていたstale refだった**ことです。

ただしPR #578には、8本のconflicting branch名が記録されていません。このため「8本残っている」という数だけを根拠にbranchを削除してはいけません。残存branchは再監査してidentityを確定する必要があります。

## 3. PR #578 / #569の整理結果

PR #578は当初7 branch分の統合を目的としていましたが、その後mainへ関連文書が別PRで統合されたため、2026-09-13時点のnet diffは次の2 file / 6 substitutionsだけになっていました。

```text
tools/reference-business-procedure-check/business_action_crud_projection_pressure_test.go
  DatabaseByAlias -> DatabaseByAliasExact x3

tools/reference-business-procedure-check/business_action_ownership_boundary_projection_pressure_test.go
  DatabaseByAlias -> DatabaseByAliasExact x3
```

PR #578には次のactual test記録があります。

```text
go vet ./...                                                 clean
go test ./...                                                mainと同じ既存failureのみ
go test ./tools/reference-business-procedure-check/... \
  -run 'BusinessActionCrudProjection|BusinessActionOwnershipBoundary'  PASS
```

この差分は2026-09-13にPR #578からmainへsquash mergeしました。

同一差分を持っていたdraft PR #569は、#578で統合済みとしてcloseしました。

その後、次のbranchはcurrent mainへresetし、未統合差分を持たない状態へ戻しました。

- `claude/gracious-bell-mjpby4`
- `fix/lane-c-exact-datasource-authority-probes`

これはbranch削除の代替ではありません。Repositoryからbranch名を除去する場合は、別途remote ref deletionが必要です。

## 4. Branchを分類する基準

今後の監査では、名前や更新日時だけで削除可否を決めません。

### A. `MERGED_ANCESTRY`

branch tipがcurrent mainのancestorである。

```text
branch tip
   ↓
 ...
   ↓
current main
```

原則として削除候補です。ただしopen PR、release / external integration等の特別用途がないことを確認します。

### B. `IDENTICAL_TREE`

branch tip commit自体はmainのancestorではないが、current mainとtreeが同一である。

squash merge後にworkstream branchをmainへresetした場合などが該当します。

現在進行中の継続workstreamでなければ削除候補です。

### C. `PATCH_EQUIVALENT_REVIEW`

commit topologyは異なるが、non-merge commitについてmain側にpatch-equivalentな変更が存在し、固有patchが検出されない。

squash mergeやcherry-pick後のstale branchでよく発生します。

これは**削除候補ではありますが、自動削除対象にはしません**。merge-only change、open PR、外部参照の有無を確認してから扱います。

### D. `UNMERGED_REVIEW`

mainにpatch-equivalentがないcommitを持つ。

削除してはいけません。

次を確認します。

- 現在もProduct / planning上必要か
- 別PRですでにより新しい形へ置換されていないか
- current architectureと衝突していないか
- conflictを解いて統合すべきか
- 意図的に廃棄するなら、その理由をPR / Issue / auditに残したか

## 5. 削除前の必須確認

remote branchを削除する前に、最低限次を確認します。

1. 対応するopen PRがない
2. mainに未包含の固有変更がない、または明示的に廃棄判断済み
3. Evidence取得・release・external integration等でbranch refそのものを必要としていない
4. branch名をIssue / Gate stateの正本として使っていない
5. 必要な履歴はPR / commit / auditから追跡できる

`merged`という名前、`final`というsuffix、古い更新日時だけでは削除根拠になりません。

## 6. 自動削除をまだ導入しない理由

現在のRepositoryには、squash merge、cherry-pick、merge commit、長期間のbranch継承が混在しています。

そのため、最初から「mainに見える変更があるので自動削除」という仕組みにすると、merge-onlyの固有変更や未記録作業を失う可能性があります。

初回は[`tools/repository-branch-audit.sh`](../../tools/repository-branch-audit.sh)でread-only分類を行い、`UNMERGED_REVIEW`と`PATCH_EQUIVALENT_REVIEW`を人が確認します。

安全性が十分に確認できてから、明示的なbranch listを入力するcleanup commandを別途検討します。

## 7. 今後の運用

新しいbranchについては、監査で後から掃除することより、そもそも増やさないことを優先します。

```text
workstream branch
   ↓
small PR
   ↓
merge
   ↓
一回限りなら削除
継続workstreamならmainへ戻して再利用
```

同じworkstreamでsuffix branchを作りません。

Repository hygiene自体は`work/repository-hygiene`を継続branchとし、一度に一つのPRだけを扱います。

## 8. 未解決事項

- PR #578で「conflict」とされた8 branchのidentity再確定
- 全remote branchをcurrent mainに対して再分類したcurrent report
- `PATCH_EQUIVALENT_REVIEW`のうちmerge-only固有変更を持つbranchの確認
- remote refを実際に削除する安全な手順の確定
- 将来branch数が再増加した際のCI / scheduled auditの要否

これらを解決するまで、branch数だけを減らす目的で一括force-reset / 一括削除は行いません。
