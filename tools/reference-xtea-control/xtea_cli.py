#!/usr/bin/env python3
"""Read-only X-TEA model inspection for AI comparison.

The CLI reads an .xead file directly. It does not start X-TEA, does not use
Swing, and does not modify the model.
"""

from __future__ import annotations

import argparse
import json
import sys
import xml.etree.ElementTree as ET
from pathlib import Path
from typing import Any, Dict, List, Optional

SCHEMA_VERSION = 1


def _s(element: Optional[ET.Element], name: str, default: str = "") -> str:
    if element is None:
        return default
    return element.get(name, default)


def _bool_op(value: str) -> bool:
    return value not in ("", None)


def _index(elements: List[ET.Element]) -> Dict[str, ET.Element]:
    return {_s(element, "ID"): element for element in elements if _s(element, "ID")}


def load_model(path: Path) -> Dict[str, Any]:
    root = ET.parse(path).getroot()
    if root.tag != "System":
        raise ValueError(f"not an X-TEA System document: {root.tag}")

    subsystems = _index(root.findall("Subsystem"))
    data_types = _index(root.findall("DataType"))
    function_types = _index(root.findall("FunctionType"))
    tables = _index(root.findall("Table"))
    functions = _index(root.findall("Function"))

    def subsystem_name(subsystem_id: str) -> str:
        return _s(subsystems.get(subsystem_id), "Name")

    def data_type(data_type_id: str) -> Dict[str, str]:
        element = data_types.get(data_type_id)
        return {
            "id": data_type_id,
            "name": _s(element, "Name"),
            "sortKey": _s(element, "SortKey"),
            "basicType": _s(element, "BasicType"),
            "length": _s(element, "Length"),
            "decimal": _s(element, "Decimal"),
            "sql": _s(element, "SQLExpression"),
        }

    def table_info(table: ET.Element) -> Dict[str, Any]:
        table_id = _s(table, "ID")
        fields = []
        for field in table.findall("TableField"):
            fields.append({
                "id": _s(field, "ID"),
                "alias": _s(field, "Alias"),
                "name": _s(field, "Name"),
                "dataType": data_type(_s(field, "DataTypeID")),
                "notNull": _s(field, "NotNull").lower() == "true",
                "showOnModel": _s(field, "ShowOnModel").lower() == "true",
                "sortKey": _s(field, "SortKey"),
            })
        keys = []
        for key in table.findall("TableKey"):
            key_fields = [
                {
                    "fieldId": _s(child, "FieldID"),
                    "sortKey": _s(child, "SortKey"),
                }
                for child in key.findall("TableKeyField")
            ]
            keys.append({
                "id": _s(key, "ID"),
                "type": _s(key, "Type"),
                "sortKey": _s(key, "SortKey"),
                "fields": key_fields,
            })
        return {
            "id": table_id,
            "name": _s(table, "Name"),
            "alias": _s(table, "Alias"),
            "sortKey": _s(table, "SortKey"),
            "subsystem": {"id": _s(table, "SubsystemID"),
                          "name": subsystem_name(_s(table, "SubsystemID"))},
            "fields": fields,
            "keys": keys,
        }

    table_index = {table_id: table_info(table) for table_id, table in tables.items()}

    def function_info(function: ET.Element) -> Dict[str, Any]:
        function_type_id = _s(function, "FunctionTypeID")
        function_type = function_types.get(function_type_id)
        io_tables = []
        for io_table in function.findall("IOTable"):
            table_id = _s(io_table, "TableID")
            io_tables.append({
                "id": _s(io_table, "ID"),
                "table": {"id": table_id,
                          "name": _s(table_index.get(table_id, {}), "name")},
                "nameExtension": _s(io_table, "NameExtension"),
                "operations": {
                    "create": _bool_op(_s(io_table, "OpC")),
                    "read": _bool_op(_s(io_table, "OpR")),
                    "update": _bool_op(_s(io_table, "OpU")),
                    "delete": _bool_op(_s(io_table, "OpD")),
                },
                "sortKey": _s(io_table, "SortKey"),
            })
        panels = [
            {
                "id": _s(panel, "ID"),
                "name": _s(panel, "Name"),
                "size": _s(panel, "Size"),
                "background": _s(panel, "Background"),
                "sortKey": _s(panel, "SortKey"),
            }
            for panel in function.findall("IOPanel")
        ]
        calls = []
        for call in function.findall("FunctionUsedByThis"):
            called_id = _s(call, "FunctionID")
            calls.append({
                "id": called_id,
                "name": _s(functions.get(called_id), "Name"),
                "launchEvent": _s(call, "LaunchEvent"),
                "sortKey": _s(call, "SortKey"),
            })
        return {
            "id": _s(function, "ID"),
            "name": _s(function, "Name"),
            "sortKey": _s(function, "SortKey"),
            "type": {
                "id": function_type_id,
                "code": _s(function_type, "SortKey"),
                "name": _s(function_type, "Name"),
            },
            "subsystem": {"id": _s(function, "SubsystemID"),
                          "name": subsystem_name(_s(function, "SubsystemID"))},
            "parameters": _s(function, "Parameters"),
            "return": _s(function, "Return"),
            "summary": _s(function, "Summary"),
            "description": _s(function, "Descriptions"),
            "ioTables": io_tables,
            "ioPanels": panels,
            "calls": calls,
        }

    function_index = {function_id: function_info(function)
                      for function_id, function in functions.items()}
    return {
        "schemaVersion": SCHEMA_VERSION,
        "source": path.name,
        "system": {
            "name": _s(root, "Name"),
            "version": _s(root, "Version"),
            "formatVersion": _s(root, "FormatVersion"),
            "description": _s(root, "Descriptions"),
        },
        "subsystems": [
            {
                "id": _s(element, "ID"),
                "name": _s(element, "Name"),
                "sortKey": _s(element, "SortKey"),
                "description": _s(element, "Descriptions"),
            }
            for element in root.findall("Subsystem")
        ],
        "functionTypes": [
            {
                "id": _s(element, "ID"),
                "code": _s(element, "SortKey"),
                "name": _s(element, "Name"),
                "description": _s(element, "Descriptions"),
            }
            for element in root.findall("FunctionType")
        ],
        "tables": list(table_index.values()),
        "functions": list(function_index.values()),
    }


def model_summary(model: Dict[str, Any]) -> Dict[str, Any]:
    by_type: Dict[str, int] = {}
    for function in model["functions"]:
        code = function["type"]["code"] or "UNKNOWN"
        by_type[code] = by_type.get(code, 0) + 1
    return {
        "schemaVersion": model["schemaVersion"],
        "source": model["source"],
        "system": model["system"],
        "counts": {
            "subsystems": len(model["subsystems"]),
            "tables": len(model["tables"]),
            "functions": len(model["functions"]),
            "functionsByType": dict(sorted(by_type.items())),
        },
        "subsystems": [
            {"id": item["id"], "name": item["name"], "sortKey": item["sortKey"]}
            for item in model["subsystems"]
        ],
    }


def _print_json(value: Any) -> None:
    json.dump(value, sys.stdout, ensure_ascii=False, indent=2, sort_keys=False)
    sys.stdout.write("\n")


def _print_tables(tables: List[Dict[str, Any]]) -> None:
    print("id\talias\tname\tfields\tkeys\tsubsystem")
    for table in tables:
        print("\t".join([
            table["id"], table["alias"], table["name"],
            str(len(table["fields"])), str(len(table["keys"])),
            table["subsystem"]["name"],
        ]))


def _print_functions(functions: List[Dict[str, Any]]) -> None:
    print("id\tcode\tname\tinputs\tcalls\tsubsystem")
    for function in functions:
        print("\t".join([
            function["id"], function["type"]["code"], function["name"],
            str(len(function["ioTables"])), str(len(function["calls"])),
            function["subsystem"]["name"],
        ]))


def main(argv: Optional[List[str]] = None) -> int:
    parser = argparse.ArgumentParser(description="Read-only X-TEA model inspection")
    parser.add_argument("--json", action="store_true", help="JSON output (default: text for lists)")
    sub = parser.add_subparsers(dest="command", required=True)

    summary = sub.add_parser("summary", help="system counts and subsystems")
    summary.add_argument("model", type=Path)

    tables = sub.add_parser("tables", help="tables, fields and keys")
    tables.add_argument("model", type=Path)
    tables.add_argument("--subsystem")
    tables.add_argument("--id")

    functions = sub.add_parser("functions", help="functions, IO tables and calls")
    functions.add_argument("model", type=Path)
    functions.add_argument("--type", dest="type_code")
    functions.add_argument("--subsystem")
    functions.add_argument("--id")

    show_table = sub.add_parser("show-table", help="one table by ID")
    show_table.add_argument("model", type=Path)
    show_table.add_argument("id")

    show_function = sub.add_parser("show-function", help="one function by ID")
    show_function.add_argument("model", type=Path)
    show_function.add_argument("id")

    export = sub.add_parser("export", help="full normalized model as JSON")
    export.add_argument("model", type=Path)

    args = parser.parse_args(argv)
    try:
        model = load_model(args.model)
    except (OSError, ValueError, ET.ParseError) as error:
        print(f"error: {error}", file=sys.stderr)
        return 2

    if args.command == "summary":
        _print_json(model_summary(model))
        return 0
    if args.command == "export":
        _print_json(model)
        return 0
    if args.command == "tables":
        values = model["tables"]
        if args.id:
            values = [item for item in values if item["id"] == args.id]
        if args.subsystem:
            values = [item for item in values if item["subsystem"]["id"] == args.subsystem]
        if args.json:
            _print_json(values)
        else:
            _print_tables(values)
        return 0
    if args.command == "functions":
        values = model["functions"]
        if args.id:
            values = [item for item in values if item["id"] == args.id]
        if args.type_code:
            values = [item for item in values if item["type"]["code"] == args.type_code]
        if args.subsystem:
            values = [item for item in values if item["subsystem"]["id"] == args.subsystem]
        if args.json:
            _print_json(values)
        else:
            _print_functions(values)
        return 0
    if args.command == "show-table":
        values = [item for item in model["tables"] if item["id"] == args.id]
        if not values:
            print(f"error: table not found: {args.id}", file=sys.stderr)
            return 1
        _print_json(values[0])
        return 0
    if args.command == "show-function":
        values = [item for item in model["functions"] if item["id"] == args.id]
        if not values:
            print(f"error: function not found: {args.id}", file=sys.stderr)
            return 1
        _print_json(values[0])
        return 0
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
