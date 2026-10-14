#!/usr/bin/env python3
"""main保護ルールの定義と実設定を読み取り専用で確認する。"""

import argparse
import copy
import json
import subprocess
import sys
from pathlib import Path

REPOSITORY = "hat47x/tei-platform"
NAME = "main：PR経由の変更と履歴保全"
DEFINITION = Path(__file__).with_name("main-pr-only.json")
RULE_TYPES = {"pull_request", "non_fast_forward", "deletion"}
REVIEW_PARAMETERS = {
    "allowed_merge_methods": ["merge", "squash", "rebase"],
    "dismiss_stale_reviews_on_push": False,
    "require_code_owner_review": False,
    "require_last_push_approval": False,
    "required_approving_review_count": 0,
    "required_review_thread_resolution": False,
}


def violations(value):
    errors = []
    if not isinstance(value, dict):
        return ["ルールセットの応答がオブジェクトではありません"]
    if value.get("name") != NAME:
        errors.append("名前が定義と一致しません")
    if value.get("target") != "branch":
        errors.append("対象種別がbranchではありません")
    if value.get("enforcement") != "active":
        errors.append("保護がactiveではありません")
    condition = value.get("conditions") or {}
    refs = condition.get("ref_name") or {}
    if refs.get("include") != ["refs/heads/main"] or refs.get("exclude") != []:
        errors.append("対象がmainだけに限定されていません")
    # GitHubは読み取り権限が不足する場合、bypass_actors自体を省略する。
    if "bypass_actors" not in value:
        errors.append("迂回権限を取得できず、空であると確認できません")
    elif value["bypass_actors"] != []:
        errors.append("迂回権限が設定されています")
    rules = value.get("rules")
    if not isinstance(rules, list):
        return errors + ["規則一覧を取得できません"]
    kinds = [rule.get("type") for rule in rules if isinstance(rule, dict)]
    if len(kinds) != len(rules) or len(kinds) != len(RULE_TYPES) or set(kinds) != RULE_TYPES:
        errors.append("PR必須・強制更新禁止・削除禁止の規則構成と異なります")
    pr = next((r for r in rules if isinstance(r, dict)
               and r.get("type") == "pull_request"), None)
    if pr is None:
        errors.append("PR必須規則がありません")
    else:
        parameters = pr.get("parameters") or {}
        for key, expected in REVIEW_PARAMETERS.items():
            actual = parameters.get(key)
            if key == "allowed_merge_methods":
                match = (isinstance(actual, list)
                         and len(actual) == len(expected)
                         and set(actual) == set(expected))
            else:
                match = actual == expected
            if not match:
                errors.append("PR必須規則の値が異なります: " + key)
    return errors


def gh_json(endpoint):
    try:
        completed = subprocess.run(
            ["gh", "api", endpoint], check=True, text=True,
            capture_output=True, timeout=30)
        return json.loads(completed.stdout)
    except (OSError, subprocess.SubprocessError, ValueError) as exc:
        raise RuntimeError("GitHubから設定を読み取れません: " + type(exc).__name__) from exc


def verify_live():
    # 一覧APIは最大100件/頁。同名ルールが次頁に隠れても成功としない。
    # 上限に達した場合は判定不能とし、不完全な一覧で合格しない。
    all_rules = []
    for page in range(1, 21):
        batch = gh_json(f"repos/{REPOSITORY}/rulesets?per_page=100&page={page}")
        if (not isinstance(batch, list)
                or any(not isinstance(item, dict) for item in batch)
                or len(batch) > 100):
            return ["GitHubのルール一覧が不正または不完全です"], None
        all_rules.extend(batch)
        if len(batch) < 100:
            break
    else:
        return ["ルール一覧が2000件以上で、全件を照合できません"], None

    matches = [entry for entry in all_rules
               if entry.get("name") == NAME and entry.get("source_type") == "Repository"]
    if len(matches) != 1:
        return ["同名のリポジトリルールが1件ではありません: " + str(len(matches))], None
    rule_id = matches[0].get("id")
    if type(rule_id) is not int or rule_id < 1:
        return ["ルールセットIDが不正です"], None
    detail = gh_json(f"repos/{REPOSITORY}/rulesets/{rule_id}")
    if not isinstance(detail, dict) or detail.get("id") != rule_id:
        return ["詳細応答のルールセットIDが一覧と一致しません"], rule_id
    if detail.get("source_type") != "Repository":
        return ["詳細応答の管理元がリポジトリではありません"], rule_id
    return violations(detail), rule_id


def self_test():
    baseline = json.loads(DEFINITION.read_text(encoding="utf-8"))
    if violations(baseline):
        raise AssertionError("定義ファイル自体が無効です: " + str(violations(baseline)))
    tests = []
    for key, value in (("enforcement", "disabled"), ("target", "tag"),
                       ("bypass_actors", [{"actor_type": "RepositoryRole"}])):
        changed = copy.deepcopy(baseline)
        changed[key] = value
        tests.append(changed)
    changed = copy.deepcopy(baseline)
    del changed["bypass_actors"]
    tests.append(changed)
    changed = copy.deepcopy(baseline)
    changed["conditions"]["ref_name"]["include"] = ["refs/heads/*"]
    tests.append(changed)
    changed = copy.deepcopy(baseline)
    changed["rules"] = changed["rules"][:1]
    tests.append(changed)
    changed = copy.deepcopy(baseline)
    changed["rules"][0]["parameters"]["required_approving_review_count"] = 1
    tests.append(changed)
    changed = copy.deepcopy(baseline)
    del changed["rules"][0]["parameters"]["allowed_merge_methods"]
    tests.append(changed)
    changed = copy.deepcopy(baseline)
    changed["rules"][0]["parameters"]["allowed_merge_methods"] = ["merge"]
    tests.append(changed)
    changed = copy.deepcopy(baseline)
    changed["rules"][0]["parameters"]["allowed_merge_methods"] = ["octopus"]
    tests.append(changed)
    changed = copy.deepcopy(baseline)
    changed["rules"].append({"type": "required_status_checks"})
    tests.append(changed)
    for i, changed in enumerate(tests, 1):
        if not violations(changed):
            raise AssertionError(f"危険な設定の模擬ケース{i}を検出できません")
    print(f"定義検査と負例 {len(tests)} 件：成功（GitHub実設定は未検査）")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument("--local", action="store_true", help="定義ファイルだけを確認する")
    group.add_argument("--live", action="store_true", help="GitHub実設定を読み取る。書き込みはしない")
    group.add_argument("--self-test", action="store_true", help="検査自身の負例を実行する")
    args = parser.parse_args()
    try:
        if args.self_test:
            self_test()
            return 0
        local = json.loads(DEFINITION.read_text(encoding="utf-8"))
        errors = violations(local)
        if errors:
            for message in errors:
                print("不一致（定義）: " + message, file=sys.stderr)
            return 1
        if args.local:
            print("定義ファイルの検査：成功（GitHub実設定の適用は未確認）")
            return 0
        remote_errors, rule_id = verify_live()
        if remote_errors:
            for message in remote_errors:
                print("未適用または判定不能: " + message, file=sys.stderr)
            return 1
        print(f"GitHub実設定の定義一致を確認: ruleset ID={rule_id}")
        print("注意: 不正pushのサーバー側拒否試験は別に実施してください")
        return 0
    except (OSError, RuntimeError, ValueError) as exc:
        print("判定できません: " + str(exc), file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
