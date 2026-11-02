"""QX-30: independent JDK ArtifactSet / loaded-byte semantic-skew oracle.

Runs actual javac/java from a clean temporary work tree. No TEI generator,
TEI Store, accepted business model, Host authorization, or Provider is used.
"""

import hashlib
import json
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path


API = '''package qx30;
import java.math.BigDecimal;
public final class LineAmount {
    public static String calculate(String value) {
        return Rule.apply(new BigDecimal(value)).toPlainString();
    }
}
'''

RULE = '''package qx30;
import java.math.BigDecimal;
import java.math.RoundingMode;
final class Rule {
    static BigDecimal apply(BigDecimal n) {
        return n.setScale(2, RoundingMode.%s);
    }
}
'''

CONSUMER = '''import qx30.LineAmount;
public final class ExternalConsumer {
    public static void main(String[] args) throws Exception {
        System.out.println("amount=" + LineAmount.calculate(args[0]));
        Class<?> rule = Class.forName("qx30.Rule");
        System.out.println("api=" + LineAmount.class.getProtectionDomain().getCodeSource().getLocation());
        System.out.println("rule=" + rule.getProtectionDomain().getCodeSource().getLocation());
    }
}
'''


def run(*args, cwd=None):
    return subprocess.run(args, cwd=cwd, text=True, capture_output=True, check=False)


def sha(data):
    return hashlib.sha256(data).hexdigest()


def members(root):
    return {str(p.relative_to(root)): sha(p.read_bytes())
            for p in sorted(root.rglob("*.class"))}


def set_digest(member_map):
    # Finite oracle digest, NOT the TEI ArtifactSet canonical serialization.
    return sha(json.dumps(member_map, sort_keys=True, separators=(",", ":")).encode())


class JavaArtifactSemanticsOracle(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        if not shutil.which("javac") or not shutil.which("java"):
            raise unittest.SkipTest("JDK not installed: no success claim")
        cls.tmp = tempfile.TemporaryDirectory(prefix="qx30-")
        cls.root = Path(cls.tmp.name)
        cls.libs = {}
        for policy in ("HALF_UP", "HALF_EVEN"):
            source = cls.root / ("source-" + policy) / "qx30"
            source.mkdir(parents=True)
            (source / "LineAmount.java").write_text(API, encoding="utf-8")
            (source / "Rule.java").write_text(RULE % policy, encoding="utf-8")
            output = cls.root / ("lib-" + policy)
            output.mkdir()
            cmd = run("javac", "-encoding", "UTF-8", "-d", str(output),
                      str(source / "LineAmount.java"), str(source / "Rule.java"))
            if cmd.returncode:
                raise RuntimeError(cmd.stderr)
            cls.libs[policy] = output
        cls.consumer = cls.root / "external-consumer"
        cls.consumer.mkdir()
        cs = cls.root / "external-source" / "ExternalConsumer.java"
        cs.parent.mkdir()
        cs.write_text(CONSUMER, encoding="utf-8")
        cmd = run("javac", "-encoding", "UTF-8", "-cp", str(cls.libs["HALF_UP"]),
                  "-d", str(cls.consumer), str(cs))
        if cmd.returncode:
            raise RuntimeError(cmd.stderr)
        cls.mixed = cls.root / "mixed"
        cls.mixed.mkdir()
        cls.only_main = cls.root / "only-main"
        cls.only_main.mkdir()
        main = Path("qx30/LineAmount.class")
        rule = Path("qx30/Rule.class")
        for dest, policy in ((cls.mixed, "HALF_EVEN"), (cls.only_main, "HALF_UP")):
            (dest / "qx30").mkdir()
            shutil.copy2(cls.libs["HALF_UP"] / main, dest / main)
            if dest == cls.mixed:
                shutil.copy2(cls.libs[policy] / rule, dest / rule)
        cls.copy = cls.root / "verified-copy"
        shutil.copytree(cls.libs["HALF_UP"], cls.copy)

    @classmethod
    def tearDownClass(cls):
        if hasattr(cls, "tmp"):
            cls.tmp.cleanup()

    @classmethod
    def java(cls, dirs, value="1.005"):
        cp = str(cls.consumer) + ":" + ":".join(str(d) for d in dirs)
        return run("java", "-cp", cp, "ExternalConsumer", value)

    def test_01_primary_class_bytes_same_across_policy_variants(self):
        a, b = (members(self.libs[x]) for x in ("HALF_UP", "HALF_EVEN"))
        self.assertEqual(a["qx30/LineAmount.class"], b["qx30/LineAmount.class"])

    def test_02_dependency_bytes_differ_and_set_digest_differs(self):
        a, b = (members(self.libs[x]) for x in ("HALF_UP", "HALF_EVEN"))
        self.assertNotEqual(a["qx30/Rule.class"], b["qx30/Rule.class"])
        self.assertNotEqual(set_digest(a), set_digest(b))

    def test_03_external_consumer_runs_on_both_independent_libraries(self):
        up, even = (self.java([self.libs[x]]) for x in ("HALF_UP", "HALF_EVEN"))
        self.assertEqual(up.returncode, 0, up.stderr)
        self.assertEqual(even.returncode, 0, even.stderr)
        self.assertIn("amount=1.01", up.stdout)
        self.assertIn("amount=1.00", even.stdout)

    def test_04_mixed_secondary_member_silently_changes_value(self):
        result = self.java([self.mixed])
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("amount=1.00", result.stdout)
        self.assertEqual(members(self.mixed)["qx30/LineAmount.class"],
                         members(self.libs["HALF_UP"])["qx30/LineAmount.class"])

    def test_05_primary_only_cannot_run(self):
        result = self.java([self.only_main])
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("NoClassDefFoundError", result.stderr)
        self.assertIn("qx30/Rule", result.stderr)

    def test_06_verified_member_set_identical_after_isolated_copy(self):
        self.assertEqual(members(self.copy), members(self.libs["HALF_UP"]))
        self.assertEqual(set_digest(members(self.copy)), set_digest(members(self.libs["HALF_UP"])))
        run_result = self.java([self.copy])
        self.assertEqual(run_result.returncode, 0, run_result.stderr)
        self.assertIn("amount=1.01", run_result.stdout)
        self.assertIn(str(self.copy), run_result.stdout)

    def test_07_expected_manifest_rejects_mixed_member_set(self):
        expected, mixed = members(self.libs["HALF_UP"]), members(self.mixed)
        self.assertEqual(set(expected), set(mixed))
        self.assertNotEqual(expected["qx30/Rule.class"], mixed["qx30/Rule.class"])
        self.assertNotEqual(set_digest(expected), set_digest(mixed))

    def test_08_classpath_order_shadows_other_version(self):
        first = self.java([self.libs["HALF_UP"], self.libs["HALF_EVEN"]])
        second = self.java([self.libs["HALF_EVEN"], self.libs["HALF_UP"]])
        self.assertEqual(first.returncode, 0, first.stderr)
        self.assertEqual(second.returncode, 0, second.stderr)
        self.assertIn("amount=1.01", first.stdout)
        self.assertIn("amount=1.00", second.stdout)
        self.assertIn(str(self.libs["HALF_UP"]), first.stdout)
        self.assertIn(str(self.libs["HALF_EVEN"]), second.stdout)

    def test_09_non_tie_example_equal_does_not_prove_policy_equality(self):
        results = [self.java([self.libs[x]], "1.006") for x in ("HALF_UP", "HALF_EVEN")]
        self.assertTrue(all(x.returncode == 0 for x in results))
        self.assertTrue(all("amount=1.01" in x.stdout for x in results))
        self.assertNotEqual(set_digest(members(self.libs["HALF_UP"])),
                            set_digest(members(self.libs["HALF_EVEN"])))

    def test_10_negative_tie_detects_policy_difference(self):
        up = self.java([self.libs["HALF_UP"]], "-1.005")
        even = self.java([self.libs["HALF_EVEN"]], "-1.005")
        self.assertEqual((up.returncode, even.returncode), (0, 0))
        self.assertIn("amount=-1.01", up.stdout)
        self.assertIn("amount=-1.00", even.stdout)

    def test_11_member_record_independent_of_declaration_order(self):
        manifest = members(self.libs["HALF_UP"])
        reversed_manifest = dict(reversed(list(manifest.items())))
        self.assertEqual(set_digest(manifest), set_digest(reversed_manifest))

    def test_12_member_omission_is_distinct_from_byte_change(self):
        expected = members(self.libs["HALF_UP"])
        only_main = members(self.only_main)
        self.assertEqual(set(expected) - set(only_main), {"qx30/Rule.class"})
        self.assertNotEqual(set_digest(expected), set_digest(only_main))


if __name__ == "__main__":
    unittest.main()
