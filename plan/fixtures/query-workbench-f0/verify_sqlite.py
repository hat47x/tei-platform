#!/usr/bin/env python3
"""Synthetic Query/CTE fixture: independent SQLite observation, NOT TEI execution."""
import json
import sqlite3

CTE = """WITH line_totals AS (
  SELECT l.order_id, SUM(l.quantity * l.unit_price) AS amount
  FROM order_lines l GROUP BY l.order_id
), open_orders AS (
  SELECT o.order_id, o.customer_id,
    CASE WHEN t.order_id IS NULL THEN 0 ELSE t.amount END AS amount
  FROM orders o LEFT JOIN line_totals t ON o.order_id = t.order_id
  WHERE o.status = 'OPEN'
)"""
STEPS = {
    "line_totals": "SELECT order_id, amount FROM line_totals ORDER BY order_id",
    "open_orders": "SELECT order_id, customer_id, amount FROM open_orders ORDER BY order_id",
    "final": """SELECT q.order_id, c.customer_name, q.amount
                 FROM open_orders q JOIN customers c ON q.customer_id = c.customer_id
                 WHERE q.amount >= :threshold ORDER BY q.order_id""",
}
# Hand-authored expectations, not calculated using the query under test.
EXPECTED = {
    "F0": {
        "line_totals": [["O01", 250], ["O02", 50], ["O04", 300]],
        "open_orders": [["O01", "C01", 250], ["O02", "C01", 50], ["O03", "C02", 0]],
        "final": [["O01", "青空商事", 250]],
    },
    "N1_nullable_model": {
        "line_totals": [["O01", 250], ["O02", None], ["O04", 300]],
        "open_orders": [["O01", "C01", 250], ["O02", "C01", None], ["O03", "C02", 0]],
        "final": [["O01", "青空商事", 250]],
    },
}

def make_db(*, quantity_nullable: bool) -> sqlite3.Connection:
    con = sqlite3.connect(":memory:")
    con.execute("PRAGMA foreign_keys=ON")
    nullability = "" if quantity_nullable else " NOT NULL"
    con.executescript(f"""
        CREATE TABLE customers(customer_id TEXT PRIMARY KEY, customer_name TEXT NOT NULL);
        CREATE TABLE orders(order_id TEXT PRIMARY KEY, customer_id TEXT NOT NULL
            REFERENCES customers(customer_id), status TEXT NOT NULL);
        CREATE TABLE order_lines(order_id TEXT NOT NULL, line_number INTEGER NOT NULL,
            quantity INTEGER{nullability}, unit_price INTEGER NOT NULL,
            PRIMARY KEY(order_id, line_number), FOREIGN KEY(order_id) REFERENCES orders(order_id));
        INSERT INTO customers VALUES('C01','青空商事'),('C02','山田商店');
        INSERT INTO orders VALUES('O01','C01','OPEN'),('O02','C01','OPEN'),
            ('O03','C02','OPEN'),('O04','C02','CLOSED');
        INSERT INTO order_lines VALUES('O01',1,2,100),('O01',2,1,50),
            ('O02',1,1,50),('O04',1,3,100);
    """)
    return con

def query_steps(con: sqlite3.Connection) -> dict:
    return {
        name: [list(row) for row in con.execute(CTE + "\n" + sql, {"threshold": 200})]
        for name, sql in STEPS.items()
    }

def main():
    evidence = {"evidenceNature": "synthetic-sqlite-in-memory", "actual": {},
                "notProven": ["TEI DerivedIR evaluation", "original-query intermediate materialization",
                              "provider equivalence", "accepted business semantics", "production permissions"]}
    with make_db(quantity_nullable=False) as con:
        evidence["actual"]["F0"] = query_steps(con)
        assert evidence["actual"]["F0"] == EXPECTED["F0"]
        # N1a: a NULL is invalid against F0's declared NOT NULL domain.
        try:
            con.execute("UPDATE order_lines SET quantity=NULL WHERE order_id='O02'")
        except sqlite3.IntegrityError:
            evidence["actual"]["N1_forbidden_null"] = "rejected-by-NOT-NULL"
        else:
            raise AssertionError("F0 NULL value was not rejected")
        # N2: do not interpret invalid duplicated customer IDs as a valid business model.
        try:
            con.execute("INSERT INTO customers VALUES('C01','重複顧客')")
        except sqlite3.IntegrityError:
            evidence["actual"]["N2_duplicate_customer_key"] = "rejected-by-PRIMARY-KEY"
        else:
            raise AssertionError("duplicate customer key was not rejected")
        # N3: an early LIMIT is provably not equivalent to the authored final WHERE.
        premature = CTE + "\n, premature AS (SELECT * FROM open_orders ORDER BY order_id DESC LIMIT 1)\n" \
            + STEPS["final"].replace("FROM open_orders q", "FROM premature q")
        evidence["actual"]["N3_premature_limit_result"] = [
            list(row) for row in con.execute(premature, {"threshold": 200})
        ]
        assert evidence["actual"]["N3_premature_limit_result"] == []
        assert evidence["actual"]["F0"]["final"] != []
    # N1b: use an EXPLICIT ALTERNATE MODEL REVISION that permits NULL in quantity.
    with make_db(quantity_nullable=True) as con:
        con.execute("UPDATE order_lines SET quantity=NULL WHERE order_id='O02'")
        evidence["actual"]["N1_nullable_model"] = query_steps(con)
        assert evidence["actual"]["N1_nullable_model"] == EXPECTED["N1_nullable_model"]
    evidence["passed"] = True
    print(json.dumps(evidence, ensure_ascii=False, indent=2, sort_keys=True))

if __name__ == "__main__":
    main()
