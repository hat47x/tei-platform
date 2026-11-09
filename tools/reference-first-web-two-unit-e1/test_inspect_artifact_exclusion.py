"""Negative/positive tests for inspect_artifact_exclusion.py (Issue #1182).

Go fixtures are REAL binaries built from tiny modules named `tei`, so the
`go tool nm` path is exercised end to end. Expected forbidden names are
hard-coded here on purpose: removing an entry from the inspector's table must
turn these tests red.
"""
import importlib.util
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

sys.dont_write_bytecode = True
NAME = "inspect_artifact_exclusion"
SOURCE = Path(__file__).with_name(NAME + ".py")
spec = importlib.util.spec_from_file_location(NAME, SOURCE)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

GO = os.environ.get("TEI_GO") or shutil.which("go")
ENV = dict(os.environ, GOENV="off", GOWORK="off", GOFLAGS="", GOTOOLCHAIN="local",
           CGO_ENABLED="0")
if os.environ.get("TEI_GO"):
    ENV["GOROOT"] = str(Path(os.environ["TEI_GO"]).resolve().parents[1])

DRIVER = "package driver\n\n//go:noinline\nfunc Run() string { return \"driver\" }\n"


def pkg_source(name):
    # noinline keeps a named symbol of the package in the linked binary.
    return ("package %s\n\n//go:noinline\nfunc Run() string { return \"%s\" }\n"
            % (name, name))


def main_source(imports, use):
    body = "".join('\t"%s"\n' % i for i in imports)
    return 'package main\n\nimport (\n%s)\n\nfunc main() {\n%s}\n' % (body, use)


class ApiBinaryTests(unittest.TestCase):
    """Builds real Go executables. `go tool nm` is part of what is tested."""

    @classmethod
    def setUpClass(cls):
        if not GO:
            raise unittest.SkipTest("go toolchain not found (set TEI_GO)")
        cls.tmp = tempfile.TemporaryDirectory()
        cls.addClassCleanup(cls.tmp.cleanup)

    def build(self, name, files, ldflags=None):
        root = Path(self.tmp.name) / name
        root.mkdir()
        (root / "go.mod").write_text("module tei\n\ngo 1.21\n", encoding="utf-8")
        files = dict(files)
        files.setdefault("internal/driver/driver.go", DRIVER)
        for rel, text in files.items():
            path = root / rel
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(text, encoding="utf-8", newline="\n")
        exe = root / ("fixture.exe" if os.name == "nt" else "fixture")
        cmd = [GO, "build", "-trimpath"]
        if ldflags:
            cmd += ["-ldflags=" + ldflags]
        cmd += ["-o", str(exe), "./cmd/app"]
        done = subprocess.run(cmd, cwd=root, env=ENV, capture_output=True, timeout=300)
        self.assertEqual(done.returncode, 0, done.stderr.decode("utf-8", "replace"))
        return str(exe)

    def inspect(self, exe, profile="json-api"):
        return module.inspect_api(exe, profile, GO)

    def clean_files(self, extra_imports=(), extra_use="", extra=None):
        imports = ["fmt", "tei/internal/driver"] + list(extra_imports)
        files = {"cmd/app/main.go": main_source(
            imports, "\tfmt.Println(driver.Run())\n" + extra_use)}
        files.update(extra or {})
        return files

    def test_clean_binary_passes_with_symbol_count(self):
        result = self.inspect(self.build("clean", self.clean_files()))
        self.assertTrue(result["passed"], result["failures"])
        self.assertEqual(result["forbiddenHitCount"], 0)
        self.assertGreater(result["symbolsInspected"], 500)
        self.assertEqual(result["knownIntrusionPackageCount"], 0)

    def test_html_template_is_rejected(self):
        files = self.clean_files(
            ["html/template"],
            '\ttemplate.Must(template.New("x").Parse("{{.}}")).Execute(nil, 1)\n')
        result = self.inspect(self.build("html", files))
        self.assertFalse(result["passed"])
        self.assertIn("html/template", result["forbiddenHits"])
        self.assertIn("text/template", result["forbiddenHits"])

    def test_studio_host_package_is_rejected(self):
        files = self.clean_files(
            ["tei/internal/studiohost"], "\tfmt.Println(studiohost.Run())\n",
            {"internal/studiohost/s.go": pkg_source("studiohost")})
        result = self.inspect(self.build("studio", files))
        self.assertFalse(result["passed"])
        self.assertIn("tei/internal/studiohost", result["forbiddenHits"])

    def test_test_framework_is_rejected(self):
        files = self.clean_files(["testing"], "\tfmt.Println(testing.Short())\n")
        result = self.inspect(self.build("testing", files))
        self.assertFalse(result["passed"])
        self.assertIn("testing", result["forbiddenHits"])

    def test_presentation_entry_symbol_is_rejected(self):
        main = main_source(["fmt", "tei/internal/driver"],
                           "\tfmt.Println(driver.Run(), handleCustomSubmit())\n")
        main += "\n//go:noinline\nfunc handleCustomSubmit() string { return \"x\" }\n"
        result = self.inspect(self.build("symbol", {"cmd/app/main.go": main}))
        self.assertFalse(result["passed"])
        self.assertIn("main.presentation-symbols", result["forbiddenHits"])

    def workspace_files(self):
        return self.clean_files(
            ["tei/internal/workspace"], "\tfmt.Println(workspace.Run())\n",
            {"internal/workspace/w.go": pkg_source("workspace")})

    def test_known_intrusion_is_allowlisted_and_counted_in_json_api(self):
        result = self.inspect(self.build("known", self.workspace_files()))
        self.assertTrue(result["passed"], result["failures"])
        self.assertEqual(result["knownIntrusionPackageCount"], 1)
        self.assertIn("tei/internal/workspace", result["knownIntrusionAllowlisted"])

    def test_known_intrusion_is_forbidden_in_frozen_world_profile(self):
        exe = self.build("known-frozen", self.workspace_files())
        result = self.inspect(exe, "frozen-world")
        self.assertFalse(result["passed"])
        self.assertIn("frozen-world:tei/internal/workspace", result["forbiddenHits"])

    def test_new_unlisted_tei_package_is_a_new_deviation(self):
        files = self.clean_files(
            ["tei/internal/brandnew"], "\tfmt.Println(brandnew.Run())\n",
            {"internal/brandnew/b.go": pkg_source("brandnew")})
        result = self.inspect(self.build("newpkg", files))
        self.assertFalse(result["passed"])
        self.assertIn("tei/internal/brandnew", result["unexpectedTeiPackages"])

    def test_stripped_binary_fails_closed(self):
        exe = self.build("stripped", self.clean_files(), ldflags="-s -w")
        # `go tool nm` itself refuses a stripped image; that must not read as PASS.
        with self.assertRaises(SystemExit):
            self.inspect(exe)
        self.assertFalse(module.analyze_symbols([])["passed"])

    def test_cli_exit_status_and_receipt(self):
        browser = Path(self.tmp.name) / "cli-browser"
        browser.mkdir()
        (browser / "index.html").write_text("<p>ok</p>", encoding="utf-8")
        good = self.build("cli-good", self.clean_files())
        bad = self.build("cli-bad", self.clean_files(
            ["testing"], "\tfmt.Println(testing.Short())\n"))
        out = Path(self.tmp.name) / "receipt.json"
        script = str(SOURCE)

        def run(api):
            return subprocess.run(
                [sys.executable, "-I", script, "--api", api, "--browser", str(browser),
                 "--go", GO, "--out", str(out)],
                env=ENV, capture_output=True, text=True, timeout=300)

        done = run(good)
        self.assertEqual(done.returncode, 0, done.stdout + done.stderr)
        import json
        receipt = json.loads(out.read_text(encoding="utf-8"))
        self.assertTrue(receipt["passed"])
        self.assertGreater(receipt["api"]["symbolsInspected"], 0)
        self.assertEqual(receipt["api"]["forbiddenHitCount"], 0)
        self.assertEqual(run(bad).returncode, 1)


class SymbolMatcherTests(unittest.TestCase):
    """Every expected forbidden name is listed here independently of the table."""

    BASE = ["main.main", "tei/internal/driver.Run"]

    def analyze(self, extra):
        return module.analyze_symbols(self.BASE + extra)

    def test_each_expected_forbidden_name_is_detected(self):
        samples = {
            "html/template": "html/template.(*Template).Execute",
            "text/template": "text/template/parse.Parse",
            "tei/internal/studiohost": "tei/internal/studiohost.Run",
            "tei/internal/desktophost": "tei/internal/desktophost.Run",
            "tei/internal/workbench": "tei/internal/workbench/database.Open",
            "tei/internal/xeadsource": "tei/internal/xeadsource.Classify",
            "tei/internal/scenariotest": "tei/internal/scenariotest.Run",
            "tei/internal/sourceacceptance": "tei/internal/sourceacceptance.Verify",
            "tei/cmd": "tei/cmd/tei.run",
            "testing": "testing.(*common).Fail",
            "net/http/httptest": "net/http/httptest.NewServer",
            "go/parser": "go/parser.ParseFile",
            "go/types": "go/types.(*Checker).Files",
            "go/build": "go/build.Import",
            "main.presentation-symbols": "main.(*webTargetServer).handleCustomSubmit-fm",
        }
        for ident, symbol in samples.items():
            with self.subTest(ident=ident):
                result = self.analyze([symbol])
                self.assertIn(ident, result["forbiddenHits"])
                self.assertFalse(result["passed"])

    def test_similar_looking_names_are_not_false_positives(self):
        for symbol in ("example.com/html/template.X", "internal/testing.X",
                       "tei/internal/studiohosting.X", "net/http.(*Server).Serve"):
            with self.subTest(symbol=symbol):
                self.assertNotIn("forbidden", " ".join(self.analyze([symbol])["failures"]))

    def test_every_known_intrusion_package_is_allowlisted_in_json_api(self):
        for pkg in ("tei/internal/workspace", "tei/internal/syntax",
                    "tei/internal/sourceindex", "tei/internal/plugin/implementation",
                    "tei/core/subject", "tei/internal/subject",
                    "tei/internal/deployment/executor", "tei/internal/stdlib",
                    "tei/internal/validation"):
            with self.subTest(pkg=pkg):
                result = self.analyze([pkg + ".Load"])
                self.assertTrue(result["passed"], result["failures"])
                self.assertIn(pkg, result["knownIntrusionAllowlisted"])
                frozen = module.analyze_symbols(self.BASE + [pkg + ".Load"], "frozen-world")
                self.assertFalse(frozen["passed"])

    def test_eq_and_hash_prefixed_names_match(self):
        for ident, symbol in (
                ("html/template", "type:.eq.html/template.state"),
                ("tei/internal/studiohost", "type:.eq.tei/internal/studiohost.X"),
                ("text/template", "type:.hash.text/template.Template"),
                ("main.presentation-symbols", "type:.eq.main.handleIndex")):
            with self.subTest(symbol=symbol):
                self.assertIn(ident, self.analyze([symbol])["forbiddenHits"])

    def test_type_only_runtime_packages_are_expected_but_new_ones_are_not(self):
        for pkg in ("tei/internal/query", "tei/internal/semantic", "tei/internal/resource",
                    "tei/internal/environment", "tei/internal/application",
                    "tei/internal/database/physical"):
            with self.subTest(pkg=pkg):
                result = self.analyze(["type:.eq." + pkg + ".T"])
                self.assertTrue(result["passed"], result["failures"])
        result = self.analyze(["type:.eq.tei/internal/brandnew.T"])
        self.assertIn("tei/internal/brandnew", result["unexpectedTeiPackages"])

    def test_empty_symbol_list_fails_as_no_symbols(self):
        result = module.analyze_symbols([])
        self.assertFalse(result["passed"])
        self.assertTrue(any("no symbols" in f for f in result["failures"]))

    def test_required_runtime_symbols_are_checked_individually(self):
        no_driver = module.analyze_symbols(["main.main", "runtime.x"])
        self.assertIn("required runtime symbol missing: tei/internal/driver",
                      no_driver["failures"])
        no_main = module.analyze_symbols(["tei/internal/driver.Run"])
        self.assertIn("required runtime symbol missing: main.main", no_main["failures"])
        self.assertEqual(module.analyze_symbols(self.BASE)["failures"], [])

    def test_nm_line_parsing(self):
        text = ("  4a0000 T main.main\n"
                "         U runtime.x\n"
                "  4a0010 R go:string.\"a b\"\n")
        self.assertEqual(module.parse_nm(text),
                         ["main.main", "runtime.x", "go:string.\"a b\""])


class BrowserScanTests(unittest.TestCase):
    def setUp(self):
        d = tempfile.TemporaryDirectory()
        self.addCleanup(d.cleanup)
        self.root = Path(d.name)

    def write(self, rel, data):
        path = self.root / rel
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(data if isinstance(data, bytes) else data.encode("utf-8"))

    def clean(self):
        self.write("index.html", '<form id="json-command" method="post" action="/scenario/submit">'
                                 '<input name="credential" type="password"></form>')
        self.write("browser-command.js",
                   'fetch("/scenario/submit",{credentials:"omit"}); // no secrets\n')

    def test_clean_assets_pass_and_are_counted(self):
        self.clean()
        result = module.scan_browser(str(self.root))
        self.assertTrue(result["passed"], result["failures"])
        self.assertEqual(result["filesScanned"], 2)
        self.assertEqual(result["hitCount"], 0)
        self.assertGreater(result["bytesScanned"], 0)

    def leak(self, rel, payload, expect):
        self.clean()
        existing = (self.root / rel).read_bytes() if (self.root / rel).exists() else b""
        self.write(rel, existing + payload)
        result = module.scan_browser(str(self.root))
        self.assertFalse(result["passed"], payload)
        self.assertIn(expect, result["hits"], payload)

    def test_synthetic_credentials_in_html_and_js(self):
        self.leak("index.html", b'<!-- editor-secret -->', "synthetic-credential")
        self.leak("browser-command.js", b'const p="viewer-secret";', "synthetic-credential")
        self.leak("browser-command.js", b'"wrong-secret"', "synthetic-credential")
        self.leak("index.html", b'reference/legacy-auth/users.json', "synthetic-credential")

    def test_secret_shapes(self):
        self.leak("browser-command.js", b"-----BEGIN RSA PRIVATE KEY-----", "private-key-block")
        self.leak("browser-command.js", b"AKIAABCDEFGHIJKLMNOP", "cloud-access-key")

    def test_dsn_forms(self):
        self.leak("browser-command.js", b'"postgres://u:p@db/x"', "dsn")
        self.leak("browser-command.js", b'"POSTGRESQL://h/x"', "dsn")
        self.leak("index.html", b"host=db sslmode=disable", "dsn-keyword")
        self.leak("index.html", b"user=a password=hunter2", "dsn-keyword")

    def test_server_only_import_names(self):
        for payload in (b"tei/internal/workspace", b"github.com/jackc/pgx/v5",
                        b"database/sql", b"workspace.Load", b"driver.NewResolver"):
            self.leak("browser-command.js", payload, "server-only-import")
        for payload in (b'import pg from "pg";', b'require("child_process")',
                        b"import('node:fs')"):
            self.leak("browser-command.js", payload, "server-only-js-module")

    def test_source_map_reference_and_file(self):
        self.leak("browser-command.js", b"\n//# sourceMappingURL=a.js.map\n",
                  "source-map-reference")
        self.leak("browser-command.js", b"\n//# sourceURL=webpack://x\n",
                  "source-map-reference")
        self.clean()
        self.write("browser-command.js.map", b'{"version":3}')
        result = module.scan_browser(str(self.root))
        self.assertIn("source-map-file", result["hits"])
        self.assertFalse(result["passed"])

    def test_secret_in_an_unexpected_extra_file_is_found(self):
        self.leak("assets/deep/app.bundle.txt", b"editor-secret", "synthetic-credential")

    def test_executable_or_wasm_inside_browser_unit(self):
        for magic in (b"\x7fELF\x02\x01", b"\x00asm\x01\x00"):
            self.clean()
            self.write("module.bin", magic + b"rest")
            self.assertIn("native-executable", module.scan_browser(str(self.root))["hits"])

    def test_hidden_file_is_scanned(self):
        self.leak(".x.js", b"editor-secret", "synthetic-credential")

    def test_secret_after_a_large_prefix_is_found(self):
        self.clean()
        self.write("big.js", b"a" * (3 << 20) + b"viewer-secret")
        result = module.scan_browser(str(self.root))
        self.assertIn("synthetic-credential", result["hits"])
        self.assertGreater(result["bytesScanned"], 3 << 20)

    def can_symlink(self, directory):
        probe = self.root.parent / ("probe-link-%d" % os.getpid())
        try:
            os.symlink(str(self.root), str(probe), target_is_directory=directory)
        except (OSError, NotImplementedError):
            return False
        os.unlink(str(probe))
        return True

    def test_directory_symlink_is_reported_not_silently_skipped(self):
        if not self.can_symlink(True):
            self.skipTest("cannot create symlinks here (Windows needs privilege)")
        self.clean()
        outside = tempfile.TemporaryDirectory()
        self.addCleanup(outside.cleanup)
        Path(outside.name, "a.js").write_text("editor-secret", encoding="utf-8")
        os.symlink(outside.name, str(self.root / "ext"), target_is_directory=True)
        result = module.scan_browser(str(self.root))
        self.assertFalse(result["passed"])
        self.assertEqual(result["hits"].get("symlink"), ["ext/"])

    def test_file_symlink_is_reported(self):
        if not self.can_symlink(False):
            self.skipTest("cannot create symlinks here (Windows needs privilege)")
        self.clean()
        outside = tempfile.TemporaryDirectory()
        self.addCleanup(outside.cleanup)
        target = Path(outside.name, "a.js")
        target.write_text("editor-secret", encoding="utf-8")
        os.symlink(str(target), str(self.root / "linked.js"))
        result = module.scan_browser(str(self.root))
        self.assertFalse(result["passed"])
        self.assertEqual(result["hits"].get("symlink"), ["linked.js"])

    def test_missing_index_fails(self):
        self.write("browser-command.js", "x")
        self.assertFalse(module.scan_browser(str(self.root))["passed"])


if __name__ == "__main__":
    unittest.main()
