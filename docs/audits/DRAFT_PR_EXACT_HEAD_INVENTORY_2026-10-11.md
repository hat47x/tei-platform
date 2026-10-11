# Open Draft PR exact-head inventory

- 観測日時: 2026-10-11 (GitHub API; exact-head recheck completed against candidate HEAD `dc2c044c0b0f07941ad0d3257814995a36cb0349`)
- Candidate branch: `integrate/remote-unmerged`
- Search scope: `hat47x/tei-platform`, `is:open draft:true`
- Count: 33 open Draft PRs; each PR was fetched individually for its current state, draft flag, head SHA, base ref, and base SHA.
- Recheck result: 33/33 exact heads are ancestors of the candidate; 0 present-but-not-ancestor and 0 missing objects.
- Every listed head SHA is an ancestor of the candidate HEAD. This proves source history presence only; it is not PR qualification for merging to `main`, nor does it imply that every proposal was adopted unchanged.
- Draft PRs remain subject to the repository rule forbidding their merge into `main`.

| PR | Current head branch | Exact head SHA | Base ref | Observed base SHA |
| ---: | --- | --- | --- | --- |
| #1200 | `docs/public-status-consolidation-20261011` | `2af5a787bbf6f120ba9d308aee4ad7e6bd5abc1b` | `main` | `e9589c8814077c96d20469e352b31c9dc604c92b` |
| #1195 | `tools/public-source-candidate-20261011` | `d283cb4d5f036db00876600243da8089655682f6` | `main` | `4f7357a05d35c1fdb3370fa82fb55e67fe9a7314` |
| #1159 | `feat/cli-first-discovery-and-json-validation` | `358a44e407895ed09944b692432e10b8135b1a6c` | `main` | `e084b89f19d4ce53047906c03e28746af56afa4d` |
| #1172 | `feat/cli-c1-typed-trace-integration` | `ff075778fe6a9381aef908ded79bc59733571109` | `feat/cli-c1-inspect-subject` | `f0d7a0b3d97e5b82e05732ff91a3588105a3f7ed` |
| #1154 | `work/issue-1143-cross-view-reference-contract` | `2ac93bb3f1a7d91f414fed2497c9ac83f152db6d` | `main` | `e084b89f19d4ce53047906c03e28746af56afa4d` |
| #1196 | `work/issue-1180-driver-scenario-cli-reference` | `6cb13102a741917d1a2113552b031f9f8149abfa` | `main` | `3620f0dd3dd677fb1604c99809dde2c729e98b3d` |
| #1185 | `work/issue-972-git-history-qualification-hardening` | `87cef265d470ec797da62e2ebf36894b5a3d8380` | `main` | `53ce39c35851b1ec731eb6cd3066d9a708f3620d` |
| #1049 | `work/issue-988-operator-pin-convergence` | `5779ffa48bff8be8f171de026021f9a00fb6c678` | `main` | `53ce39c35851b1ec731eb6cd3066d9a708f3620d` |
| #1162 | `feature/first-model-change-1161` | `5e834a810f82ccb2fc4650a934d5fcd03d11c61b` | `main` | `2796703ed4549838dc82f94aa98aea95d1854bed` |
| #1141 | `work/issue-1121-lowering-contract` | `f41904aaf90ab1dd44c776b35aef4fcab6aa4da2` | `main` | `2796703ed4549838dc82f94aa98aea95d1854bed` |
| #1136 | `work/issue-1132-cud-declared-intent-probe` | `d6cd27958c0280cbcfadc6e70057c5a46b83ece3` | `main` | `2796703ed4549838dc82f94aa98aea95d1854bed` |
| #1114 | `work/issue-1043-web-principal-handoff` | `c6a95b4fc102d9e6f0b585398545a76105f46503` | `main` | `2796703ed4549838dc82f94aa98aea95d1854bed` |
| #1113 | `work/issue-000084-external-coordination` | `2895ee472700f437881b01b2cd2885aec3458d50` | `main` | `2796703ed4549838dc82f94aa98aea95d1854bed` |
| #1169 | `feat/cli-c1-inspect-subject` | `f0d7a0b3d97e5b82e05732ff91a3588105a3f7ed` | `feat/cli-first-discovery-and-json-validation` | `a6478c6ba64e7cac489759e24266998e4409f0f7` |
| #1171 | `feat/modelref-typed-trace-query` | `5ca38e7ed6f27be5f019f304d80ebaed59ae6c81` | `issue-1120-typed-reference-contract` | `30139f4c1903cc982789d308a4f75b04a5ead8eb` |
| #1115 | `work/issue-982-baseline-alias-guard` | `56318e9f6764e3c2d4b00b528a06473ba76bc7de` | `main` | `2796703ed4549838dc82f94aa98aea95d1854bed` |
| #1170 | `feature/first-value-ux-audit-1168` | `309d1b475c97b04df4c2198523ff0024c20a0e4e` | `main` | `2796703ed4549838dc82f94aa98aea95d1854bed` |
| #1160 | `work/issue-000080-mutating-capability-protocol` | `cc04cb0e81c87d695eedbf9b4d60621d5a3085a5` | `main` | `e084b89f19d4ce53047906c03e28746af56afa4d` |
| #1135 | `work/issue-1119-critical-path-product-gate` | `46a117573a11bea481db366957340cf5e9e2fd0f` | `main` | `2796703ed4549838dc82f94aa98aea95d1854bed` |
| #1156 | `work/issue-000040-second-application-reference` | `c669d91b52d49d1a5d4690f00e0dfe1e79228ad3` | `main` | `e084b89f19d4ce53047906c03e28746af56afa4d` |
| #1138 | `work/1125-business-dfd-readonly-v0` | `63038d35a7b117a647f372b667c3dfa05caf935e` | `main` | `5fc5a0e1cb4fb42897c76dd3334292e2ad4a66e4` |
| #1128 | `work/model-example-slide-m1` | `a0a419f5129925c0c1ddaeac76a95a22167f3a74` | `main` | `5fc5a0e1cb4fb42897c76dd3334292e2ad4a66e4` |
| #1139 | `issue-1120-typed-reference-contract` | `30139f4c1903cc982789d308a4f75b04a5ead8eb` | `main` | `5fc5a0e1cb4fb42897c76dd3334292e2ad4a66e4` |
| #1144 | `research/rmt-executable-scenario-contract` | `46b6d6bb495e0d5446c66a850443e9e67dc83eaa` | `main` | `5fc5a0e1cb4fb42897c76dd3334292e2ad4a66e4` |
| #1137 | `work/issue-1124-studio-authoring-projection` | `1b580482d363a0174b7d5ed1996ae6004f8efa3f` | `main` | `5fc5a0e1cb4fb42897c76dd3334292e2ad4a66e4` |
| #1055 | `work/issue-1045-epistemic-detail-resolver` | `81164512d1d5666ca118f955e81443911297c50b` | `main` | `5fc5a0e1cb4fb42897c76dd3334292e2ad4a66e4` |
| #1147 | `work/issue-000086-capability-transport-reference` | `623e4704ac1f7792c645218f4482ae15d601881c` | `main` | `5fc5a0e1cb4fb42897c76dd3334292e2ad4a66e4` |
| #1077 | `work/issue-001031-rmt-cross-view` | `d63811c2b44e3193a721efe0b165055e8cb7323c` | `main` | `5fc5a0e1cb4fb42897c76dd3334292e2ad4a66e4` |
| #1116 | `work/issue-000080-runtime-authority-bridge` | `1302cb1989c73a9e566f5a9c4ab72712d559c763` | `main` | `5fc5a0e1cb4fb42897c76dd3334292e2ad4a66e4` |
| #1111 | `work/issue-1017-loader-source-snapshot` | `cbdb40b8825f103a4c897b03fff773a35f14ea50` | `main` | `02e4160a529d69835916193aec83b5f932fef46e` |
| #1112 | `work/issue-000047-data-design-candidate-review` | `99d59649326647edfb8bf5d4cb75be730fde1fc8` | `main` | `02e4160a529d69835916193aec83b5f932fef46e` |
| #1071 | `work/issue-000047-runtime-object-identity` | `e65d30d06418843d4ade4c4ef6e25eeea0ecdbc8` | `main` | `4f8c6ba830a2996187e0e2ea79fb3c1a609d5db7` |
| #1061 | `work/issue-915-ref-spec-01-review` | `79e0d4d26dd28d1428326230014572c81aeb2ecb` | `main` | `aac0a7f075afd69d17b91ee49e24483c70542672` |

The inventory is a time-bounded snapshot. Re-fetch every PR and the current `main` before using it for any later integration decision.
