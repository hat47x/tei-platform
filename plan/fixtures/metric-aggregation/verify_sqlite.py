#!/usr/bin/env python3
"""Synthetic, read-only counterexamples for metric-grain/rollup compatibility.

No TEI code, production Provider, accepted business rule or external data is loaded.
Expected values are computed independently of SQL with Python integers/Fraction.
"""
import json
import pathlib
import sqlite3
from fractions import Fraction

# Hand-fixed values shared with the Go reference test
# (reference/derivation-relationsets, TestHandFixedExpectedMatchesSharedFixture).
FIXED = json.loads(pathlib.Path(__file__).with_name("expected.json").read_text(encoding="utf-8"))["cases"]


def scalar(db, sql):
    return db.execute(sql).fetchone()[0]


def case(name, actual, expected, unsafe, note):
    assert actual == expected, (name, actual, expected)
    assert unsafe != expected, (name, "negative case did not discriminate")
    fixed = FIXED[name[:5]]
    assert str(expected) == fixed["expected"], (name, "differs from expected.json", str(expected))
    assert str(unsafe) in fixed["unsafe"], (name, "unsafe value is not listed in expected.json", str(unsafe))
    return {"case": name, "status": "PASS", "expected": str(expected),
            "unsafe": str(unsafe), "note": note}


def main():
    db = sqlite3.connect(":memory:")
    db.executescript("""
      CREATE TABLE line(line_id TEXT PRIMARY KEY, order_id TEXT, amount INTEGER);
      CREATE TABLE delivery(delivery_id TEXT PRIMARY KEY, order_id TEXT);
      INSERT INTO line VALUES('L1','H',100),('L2','H',100);
      INSERT INTO delivery VALUES('D1','H'),('D2','H'),('D3','H');

      CREATE TABLE member_value(group_id TEXT, member_id TEXT, val INTEGER);
      INSERT INTO member_value VALUES('A','a1',10),('A','a2',10),
        ('B','b1',20),('B','b2',20),('B','b3',20),('B','b4',20),
        ('B','b5',20),('B','b6',20),('B','b7',20),('B','b8',20);

      CREATE TABLE fraction_group(group_id TEXT, positive INTEGER, total INTEGER);
      INSERT INTO fraction_group VALUES('A',1,2),('B',8,10);

      CREATE TABLE assignment(group_id TEXT, person_id TEXT);
      INSERT INTO assignment VALUES('A','U1'),('A','U2'),('B','U2'),('B','U3');

      CREATE TABLE closing_balance(day INTEGER, account_id TEXT, balance INTEGER);
      INSERT INTO closing_balance VALUES(1,'A',100),(1,'B',200),(2,'A',110),(2,'B',190);
    """)
    out = []
    correct = scalar(db, "SELECT SUM(amount) FROM line WHERE order_id='H'")
    fanout = scalar(db, "SELECT SUM(l.amount) FROM line l JOIN delivery d ON l.order_id=d.order_id")
    by_value = scalar(db, "SELECT SUM(DISTINCT l.amount) FROM line l JOIN delivery d ON l.order_id=d.order_id")
    assert correct == 100 + 100 == 200 and fanout == 600 and by_value == 100
    out.append(case("MC-01 fanout", correct, 100 + 100, fanout, "SUM DISTINCT value is also wrong: " + str(by_value)))

    means = db.execute("SELECT group_id, SUM(val), COUNT(*) FROM member_value GROUP BY group_id ORDER BY group_id").fetchall()
    assert means == [("A",20,2),("B",160,8)]
    weighted = Fraction(sum(m[1] for m in means), sum(m[2] for m in means))
    naive = sum(Fraction(m[1],m[2]) for m in means)/len(means)
    out.append(case("MC-02 weighted average", weighted, Fraction(2*10+8*20,10), naive, "roll up SUM and COUNT, not AVG of AVG"))

    numer, denom = db.execute("SELECT SUM(positive),SUM(total) FROM fraction_group").fetchone()
    ratios = [Fraction(n,d) for n,d in db.execute("SELECT positive,total FROM fraction_group")]
    out.append(case("MC-03 ratio", Fraction(numer,denom), Fraction(1+8,2+10), sum(ratios)/2, "ratio of sums, not mean of ratios"))

    uniq = scalar(db, "SELECT COUNT(DISTINCT person_id) FROM assignment")
    summed_groups = scalar(db, "SELECT SUM(n) FROM (SELECT COUNT(DISTINCT person_id) AS n FROM assignment GROUP BY group_id)")
    out.append(case("MC-04 overlapping groups", uniq, len({"U1","U2","U3"}), summed_groups, "identity overlap prevents count rollup"))

    by_day = dict(db.execute("SELECT day,SUM(balance) FROM closing_balance GROUP BY day"))
    all_days = scalar(db, "SELECT SUM(balance) FROM closing_balance")
    out.append(case("MC-05 point-in-time stock", by_day[2], 110 + 190, all_days, "the balance at day 2 is not the sum of both days"))
    assert by_day == {1:300,2:300}

    db.execute("CREATE TABLE nullable_fact(x INTEGER)")
    empty_sum = scalar(db, "SELECT SUM(x) FROM nullable_fact")
    empty_count = scalar(db, "SELECT COUNT(*) FROM nullable_fact")
    db.execute("INSERT INTO nullable_fact VALUES(NULL)")
    null_sum = scalar(db, "SELECT SUM(x) FROM nullable_fact")
    null_count = scalar(db, "SELECT COUNT(*) FROM nullable_fact")
    assert (empty_sum,empty_count,null_sum,null_count)==(None,0,None,1)
    out.append({"case":"MC-06 empty versus NULL", "status":"PASS", "expected":"0 rows vs 1 NULL row", "unsafe":"both SUM(x) return SQL NULL", "note":"row existence is not recovered from SUM alone"})

    print(json.dumps({"status":"PASS", "case_count":len(out), "environment":"Python stdlib sqlite3 in-memory; synthetic only", "cases":out},ensure_ascii=False,indent=2))


if __name__ == "__main__":
    main()
