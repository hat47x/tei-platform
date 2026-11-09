// TEI非依存consumerにも渡せる、位置指定済みの小さな試験入力。
// 実行・保存・権限を表すものではありません。
export const buildPipelineGraph = {
  width: 480,
  height: 160,
  nodes: [
    {id: "source", label: "Source", x: 70, y: 65},
    {id: "check", label: "Check", x: 240, y: 65},
    {id: "publish", label: "Publish", x: 410, y: 65},
  ],
  edges: [
    {id: "source-check", source: "source", target: "check"},
    {id: "check-publish", source: "check", target: "publish"},
  ],
};
