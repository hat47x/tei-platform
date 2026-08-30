#!/usr/bin/env python3
"""TEI SQL analysis worker.

The stable protocol exposes normalized scopes, column occurrences, output-field
lineage and a deliberately small relational vocabulary. SQLGlot AST objects
never cross the Go/Python boundary.
"""
import json
import re
import sys

PROTOCOL_VERSION = 1


def fail(query, code, message):
    return {
        "protocolVersion": PROTOCOL_VERSION,
        "status": "error",
        "query": query,
        "diagnostics": [{"code": code, "message": message}],
    }


def sql_text(expr, dialect):
    if expr is None:
        return ""
    try:
        return expr.sql(dialect=dialect or None)
    except Exception:
        return str(expr)


def table_name(table):
    parts = [p for p in (table.catalog, table.db, table.name) if p]
    return ".".join(parts)


def safe_part(value):
    value = re.sub(r"[^A-Za-z0-9_]+", "_", str(value or ""))
    return value.strip("_") or "anon"


def nearest_select(node, exp):
    p = getattr(node, "parent", None)
    while p is not None:
        if isinstance(p, exp.Select):
            return p
        p = getattr(p, "parent", None)
    return None


def ancestor_cte(node, exp):
    p = getattr(node, "parent", None)
    while p is not None:
        if isinstance(p, exp.CTE):
            return p
        if isinstance(p, exp.Select):
            return None
        p = getattr(p, "parent", None)
    return None


def query_parent(select, exp):
    p = getattr(select, "parent", None)
    while p is not None:
        if isinstance(p, exp.Select):
            return p
        p = getattr(p, "parent", None)
    return None


def collect_selects(tree, exp):
    out = []
    if isinstance(tree, exp.Select):
        out.append(tree)
    for s in tree.find_all(exp.Select):
        if all(s is not existing for existing in out):
            out.append(s)
    return out


def assign_scopes(tree, exp):
    selects = collect_selects(tree, exp)
    if not selects:
        return {}, []
    root = tree if isinstance(tree, exp.Select) else selects[0]
    scope_by_select = {id(root): "scope:root"}
    scopes = [{"id": "scope:root", "kind": "root"}]
    subquery_counter = 0

    # Assign parents before children where possible. Repeated passes handle a
    # nested select that happened to appear before its parent in traversal.
    remaining = [s for s in selects if s is not root]
    while remaining:
        progressed = False
        for s in remaining[:]:
            parent = query_parent(s, exp)
            parent_id = "scope:root" if parent is None else scope_by_select.get(id(parent))
            if parent_id is None:
                continue
            cte = ancestor_cte(s, exp)
            if cte is not None:
                name = getattr(cte, "alias_or_name", None) or "cte"
                scope_id = f"scope:cte:{safe_part(name)}"
                kind = "cte"
                scope_name = name
            else:
                scope_id = f"scope:subquery:{subquery_counter}"
                subquery_counter += 1
                kind = "subquery"
                scope_name = ""
            scope_by_select[id(s)] = scope_id
            item = {"id": scope_id, "parentId": parent_id, "kind": kind}
            if scope_name:
                item["name"] = scope_name
            scopes.append(item)
            remaining.remove(s)
            progressed = True
        if not progressed:
            # Defensive fallback: preserve scopes instead of losing analysis.
            s = remaining.pop(0)
            scope_id = f"scope:subquery:{subquery_counter}"
            subquery_counter += 1
            scope_by_select[id(s)] = scope_id
            scopes.append({"id": scope_id, "parentId": "scope:root", "kind": "subquery"})
    return scope_by_select, scopes


def immediate_tables(select, exp):
    return [t for t in select.find_all(exp.Table) if nearest_select(t, exp) is select]


def local_aliases(select, exp, cte_scopes):
    aliases = {}
    for table in immediate_tables(select, exp):
        source = table_name(table)
        alias = table.alias_or_name or table.name
        aliases[alias] = source
    # CTE names are lexical sources even when no physical table exists.
    aliases.update({name: name for name in cte_scopes})
    return aliases


def ancestor_alias_scope(select, qualifier, exp, scope_by_select, aliases_by_scope):
    parent = query_parent(select, exp)
    while parent is not None:
        sid = scope_by_select.get(id(parent))
        if sid and qualifier in aliases_by_scope.get(sid, {}):
            return sid
        parent = query_parent(parent, exp)
    return ""


def analyze_columns(selects, exp, dialect, scope_by_select, aliases_by_scope):
    refs = []
    outputs = []
    occurrence_by_object = {}
    counter = 0
    for select in selects:
        sid = scope_by_select[id(select)]
        local = aliases_by_scope.get(sid, {})
        for col in select.find_all(exp.Column):
            if nearest_select(col, exp) is not select:
                continue
            qualifier = col.table or ""
            correlated = False
            outer = ""
            if qualifier and qualifier not in local:
                outer = ancestor_alias_scope(select, qualifier, exp, scope_by_select, aliases_by_scope)
                correlated = bool(outer)
            cid = f"col:{counter}"
            counter += 1
            occurrence_by_object[id(col)] = cid
            item = {"id": cid, "scopeId": sid, "name": col.name}
            if qualifier:
                item["qualifier"] = qualifier
            if correlated:
                item["correlated"] = True
                item["outerScopeId"] = outer
            refs.append(item)

        for i, expr in enumerate(select.expressions):
            inputs = []
            for col in expr.find_all(exp.Column):
                if nearest_select(col, exp) is not select:
                    continue
                cid = occurrence_by_object.get(id(col))
                if cid:
                    inputs.append(cid)
            name = getattr(expr, "alias_or_name", None) or ""
            item = {
                "id": f"out:{safe_part(sid)}:{i}",
                "scopeId": sid,
                "expression": sql_text(expr, dialect),
                "inputs": inputs,
            }
            if name:
                item["name"] = name
            outputs.append(item)
    return refs, outputs


def lower_select(select, exp, dialect, scope_id, cte_scopes):
    operations = []
    sources = []
    tables = immediate_tables(select, exp)
    for i, table in enumerate(tables):
        source = table_name(table)
        alias = table.alias_or_name or table.name
        source_kind = "cte" if source in cte_scopes else "table"
        node_id = f"{safe_part(scope_id)}:source:{i}"
        op = {
            "id": node_id,
            "kind": "source",
            "scopeId": scope_id,
            "source": source,
            "sourceKind": source_kind,
            "alias": alias,
        }
        if source_kind == "cte":
            op["binding"] = cte_scopes[source]
        operations.append(op)
        sources.append(node_id)

    current = sources[0] if sources else ""
    joins = select.args.get("joins") or []
    for i, source_id in enumerate(sources[1:]):
        join = joins[i] if i < len(joins) else None
        join_kind = "cross"
        expressions = []
        if join is not None:
            side = str(join.args.get("side") or "").lower()
            kind = str(join.args.get("kind") or "").lower()
            join_kind = side or kind or "inner"
            on = join.args.get("on")
            if on is not None:
                expressions.append(sql_text(on, dialect))
        node_id = f"{safe_part(scope_id)}:join:{i}"
        operations.append({
            "id": node_id,
            "kind": "join",
            "scopeId": scope_id,
            "inputs": [current, source_id] if current else [source_id],
            "joinKind": join_kind,
            "expressions": expressions,
        })
        current = node_id

    where = select.args.get("where")
    if where is not None:
        node_id = f"{safe_part(scope_id)}:filter:0"
        operations.append({"id": node_id, "kind": "filter", "scopeId": scope_id, "inputs": [current] if current else [], "expressions": [sql_text(where.this, dialect)]})
        current = node_id

    group = select.args.get("group")
    aggregate_exprs = [a for a in select.find_all(exp.AggFunc) if nearest_select(a, exp) is select]
    if group is not None or aggregate_exprs:
        exprs = []
        if group is not None:
            exprs.extend(sql_text(e, dialect) for e in group.expressions)
        exprs.extend(sql_text(e, dialect) for e in aggregate_exprs)
        exprs = list(dict.fromkeys(e for e in exprs if e))
        node_id = f"{safe_part(scope_id)}:aggregate:0"
        operations.append({"id": node_id, "kind": "aggregate", "scopeId": scope_id, "inputs": [current] if current else [], "expressions": exprs})
        current = node_id

    node_id = f"{safe_part(scope_id)}:project:0"
    operations.append({"id": node_id, "kind": "project", "scopeId": scope_id, "inputs": [current] if current else [], "expressions": [sql_text(e, dialect) for e in select.expressions]})
    current = node_id

    if select.args.get("distinct") is not None:
        node_id = f"{safe_part(scope_id)}:distinct:0"
        operations.append({"id": node_id, "kind": "distinct", "scopeId": scope_id, "inputs": [current]})
        current = node_id

    order = select.args.get("order")
    if order is not None:
        node_id = f"{safe_part(scope_id)}:sort:0"
        operations.append({"id": node_id, "kind": "sort", "scopeId": scope_id, "inputs": [current], "expressions": [sql_text(e, dialect) for e in order.expressions]})
        current = node_id

    limit = select.args.get("limit")
    if limit is not None:
        node_id = f"{safe_part(scope_id)}:limit:0"
        operations.append({"id": node_id, "kind": "limit", "scopeId": scope_id, "inputs": [current], "expressions": [sql_text(limit.expression, dialect)]})
    return operations


def analyze_query(tree, exp, dialect):
    scope_by_select, scopes = assign_scopes(tree, exp)
    if not scope_by_select:
        return [], {}, [], [], [], "partial"
    selects = collect_selects(tree, exp)
    cte_scopes = {s.get("name"): s["id"] for s in scopes if s.get("kind") == "cte" and s.get("name")}
    aliases_by_scope = {}
    flat_aliases = {}
    for select in selects:
        sid = scope_by_select[id(select)]
        aliases = local_aliases(select, exp, cte_scopes)
        aliases_by_scope[sid] = aliases
        flat_aliases.update(aliases)

    column_refs, output_fields = analyze_columns(selects, exp, dialect, scope_by_select, aliases_by_scope)
    operations = []
    for select in selects:
        operations.extend(lower_select(select, exp, dialect, scope_by_select[id(select)], cte_scopes))

    # We now preserve nested scopes and qualified correlation. Constructs that
    # do not lower into the current relational vocabulary still return partial.
    lowering = "full"
    return operations, flat_aliases, scopes, column_refs, output_fields, lowering


def main():
    try:
        req = json.load(sys.stdin)
    except Exception as exc:
        json.dump(fail("", "INVALID_REQUEST", str(exc)), sys.stdout)
        return

    query = req.get("query", "")
    if req.get("protocolVersion") != PROTOCOL_VERSION:
        json.dump(fail(query, "PROTOCOL_VERSION_MISMATCH", "unsupported protocol version"), sys.stdout)
        return
    if req.get("operation") != "analyze_sql":
        json.dump(fail(query, "UNSUPPORTED_OPERATION", "expected analyze_sql"), sys.stdout)
        return

    try:
        import sqlglot
        from sqlglot import exp
    except Exception as exc:
        json.dump(fail(query, "SQLGLOT_UNAVAILABLE", str(exc)), sys.stdout)
        return

    dialect = req.get("dialect")
    if dialect == "generic":
        dialect = None
    try:
        tree = sqlglot.parse_one(req.get("source", ""), read=dialect)
    except Exception as exc:
        json.dump(fail(query, "SQL_PARSE_ERROR", str(exc)), sys.stdout)
        return

    tables = set()
    columns = set()
    for table in tree.find_all(exp.Table):
        tables.add(table_name(table))
    for column in tree.find_all(exp.Column):
        columns.add(f"{column.table}.{column.name}" if column.table else column.name)

    if isinstance(tree, exp.Select) or list(tree.find_all(exp.Select)):
        operations, aliases, scopes, column_refs, output_fields, lowering = analyze_query(tree, exp, dialect)
    else:
        operations = [{"id": "opaque:0", "kind": "opaque", "scopeId": "scope:root", "expressions": [sql_text(tree, dialect)]}]
        aliases, scopes, column_refs, output_fields, lowering = {}, [{"id": "scope:root", "kind": "root"}], [], [], "partial"

    json.dump({
        "protocolVersion": PROTOCOL_VERSION,
        "status": "ok",
        "query": query,
        "tables": sorted(tables),
        "columns": sorted(columns),
        "tableAliases": dict(sorted(aliases.items())),
        "scopes": scopes,
        "columnRefs": column_refs,
        "outputFields": output_fields,
        "operations": operations,
        "lowering": lowering,
        "diagnostics": [],
    }, sys.stdout)


if __name__ == "__main__":
    main()
