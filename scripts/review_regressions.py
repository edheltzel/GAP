import base64
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from html.parser import HTMLParser
from unittest.mock import patch


ROOT = Path(__file__).resolve().parent.parent
DO_SKILLS = ROOT / ".vendor" / "do-skills"
if not (DO_SKILLS / "skills").is_dir():
    sys.exit("Do-Skills checkout missing at .vendor/do-skills; clone the pin in gap.lock first")
ART = DO_SKILLS / "skills/global/content/do-art/Tools"
PNG = base64.b64decode("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+a9m8AAAAASUVORK5CYII=")
sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location("illo", DO_SKILLS / "skills/global/content/do-illo/scripts/illo.py")
illo = importlib.util.module_from_spec(spec)
spec.loader.exec_module(illo)


class GeneratedReport(HTMLParser):
    def __init__(self, text):
        super().__init__()
        self.rows, self.text, self.row, self.cell = [], [], None, None
        self.feed(text)

    def handle_starttag(self, tag, attrs):
        if tag == "tr":
            self.row = []
        if tag in ("td", "th"):
            self.cell = []

    def handle_data(self, data):
        self.text.append(data)
        if self.cell is not None:
            self.cell.append(data)

    def handle_endtag(self, tag):
        if tag in ("td", "th") and self.row is not None:
            self.row.append("".join(self.cell).strip())
            self.cell = None
        if tag == "tr":
            self.rows.append(self.row)
            self.row = None


class ScriptRegressions(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="review-", dir=ROOT)
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.env = {**os.environ, "HOME": str(self.root), "XDG_CONFIG_HOME": str(self.root / "config")}

    def run_cli(self, *args):
        return subprocess.run(args, cwd=self.root, env=self.env, text=True, capture_output=True, timeout=30)

    def image_tools(self):
        stub = f'''#!{sys.executable}
from pathlib import Path
import base64, os, sys
args = sys.argv[1:]
tool = Path(sys.argv[0]).name
if os.environ.get("FAIL_IMAGE_TOOL"):
    sys.exit(1)
if tool == "identify":
    print(100)
elif tool == "rembg":
    Path(args[2]).write_bytes(base64.b64decode({base64.b64encode(PNG).decode()!r}) + b"|cutout|" + Path(args[1]).read_bytes())
else:
    data = Path(args[0]).read_bytes()
    marker = "|signature" if "-annotate" in args else "|background:" + args[args.index("-background") + 1]
    Path(args[-1]).write_bytes(data + marker.encode())
'''
        bindir = self.root / "bin"
        bindir.mkdir()
        for name in ("rembg", "magick", "identify"):
            path = bindir / name
            path.write_text(stub)
            path.chmod(0o755)
        self.env["PATH"] = str(bindir) + os.pathsep + self.env["PATH"]
        self.env["REMBG_BIN"] = str(bindir / "rembg")

    def test_remove_background_preserves_explicit_input_on_rerun(self):
        self.image_tools()
        source, output = self.root / "original.jpg", self.root / "transparent.png"
        source.write_bytes(b"original image")
        for _ in range(2):
            result = self.run_cli("bun", str(ART / "RemoveBg.ts"), str(source), str(output))
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertTrue(source.is_file(), "explicit input must survive even when output already exists")
            self.assertEqual(source.read_bytes(), b"original image")
            self.assertEqual(output.read_bytes(), PNG + b"|cutout|original image")
        same = self.run_cli("bun", str(ART / "RemoveBg.ts"), str(output), str(output))
        self.assertEqual(same.returncode, 0, same.stderr)
        batch = [self.root / f"batch-{i}.jpg" for i in range(3)]
        for image in batch:
            image.write_bytes(b"batch image")
        result = self.run_cli("bun", str(ART / "RemoveBg.ts"), *map(str, batch))
        self.assertEqual(result.returncode, 0, result.stderr)
        for image in batch:
            self.assertFalse(image.exists())
            self.assertTrue(image.with_suffix(".png").is_file())

    def test_generate_postprocesses_all_modes_and_providers(self):
        self.image_tools()
        for key in ("OPENAI_API_KEY_OPTIN", "OPENROUTER_API_KEY", "GOOGLE_API_KEY"):
            self.env[key] = ""
        self.env.update(XAI_API_KEY="fixture", OPENAI_API_KEY="fixture", GEMINI_API_KEY="fixture", ART_SIGNATURE="fixture")
        modes = [("grok", 1, "--thumbnail"), ("grok", 2, "--thumbnail"), ("compare", 2, "--thumbnail"), ("nano-banana", 2, "--add-bg")]
        for i, (model, count, option) in enumerate(modes):
            with self.subTest(model=model, count=count, option=option):
                output = self.root / f"run-{i}.png"
                args = ["bun", "--no-install", "--preload", str(ROOT / "scripts/fixtures/generate-providers.ts"), str(ART / "Generate.ts"),
                        "--freeform-confirmed", "--model", model, "--prompt", "fixture", "--output", str(output),
                        "--creative-variations", str(count), "--signature", option]
                if option == "--add-bg":
                    args.extend(["#112233", "--remove-bg"])
                result = self.run_cli(*args)
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                images = sorted(p for p in self.root.glob(f"run-{i}*.png") if not p.stem.endswith("-thumb"))
                self.assertEqual(len(images), count * (3 if model == "compare" else 1))
                for image in images:
                    data = image.read_bytes()
                    self.assertIn(b"|cutout|", data)
                    self.assertIn(b"|signature", data)
                    if option == "--thumbnail":
                        self.assertEqual(image.with_name(image.stem + "-thumb.png").read_bytes(), data + b"|background:#EAE9DF")
                    else:
                        self.assertIn(b"|background:#112233", data)
        self.env["FAIL_IMAGE_TOOL"] = "1"
        for model in ("grok", "compare"):
            result = self.run_cli("bun", "--no-install", "--preload", str(ROOT / "scripts/fixtures/generate-providers.ts"), str(ART / "Generate.ts"),
                                  "--freeform-confirmed", "--model", model, "--prompt", "fixture", "--output", str(self.root / "failed.png"), "--thumbnail")
            self.assertNotEqual(result.returncode, 0, "failed post-processing must fail the command")

    def test_memory_report_preserves_growth_and_uncertainty(self):
        for index, series in enumerate(([10, 11, 12, 13], [10, 13, 11, 14])):
            before = {"boot": {"launchToWindowMs": 1, "reloadToFirstCommitMs": 1, "heapMB": 10, "nodes": 10, "listeners": 10},
                      "bundle": {"bootCriticalKB": 1, "bootCriticalChunks": 1, "reachableTotalKB": 1, "orphanFiles": 0},
                      "rendererErrorCount": 0, "phases": {"work": {"taskMs": 1}}, "leak": {"cycles": 4, "heapKBPerCycle": 0, "nodesPerCycle": 0,
                      "listenersPerCycle": 0, "finalHeapMB": 10, "series": [{"heapMB": 10, "nodes": 10}] * 4}}
            after = {**before, "leak": {"cycles": 4, "heapKBPerCycle": 10, "nodesPerCycle": 0.25,
                     "listenersPerCycle": 0.125, "finalHeapMB": series[-1], "series": [{"heapMB": h, "nodes": 10} for h in series]}}
            for name, value in (("before", before), ("after", after)):
                (self.root / f"{name}.json").write_text(json.dumps(value))
            result = self.run_cli("node", str(DO_SKILLS / "skills/project/tooling/do-perf/scripts/perf-report.mjs"), "--before=before.json", "--after=after.json", f"--out=report-{index}.html")
            self.assertEqual(result.returncode, 0, result.stderr)
            report = GeneratedReport((self.root / f"report-{index}.html").read_text())
            self.assertIn(["Node growth per cycle", "0", "0.25", "GROWING"], report.rows)
            self.assertIn(["Listener growth per cycle", "0", "0.125", "GROWING"], report.rows)
            text = " ".join(report.text)
            self.assertIn("Heap growth changes from 0 to 10 KB per cycle.", text)
            self.assertIn("do not establish the cause or rule out retention", text)

    def test_codex_recovery_requires_invocation_identity(self):
        cache = self.root / "generated_images"
        output = self.root / "requested.png"
        for event, timeout, own_image, requested in ((None, False, False, False), (None, True, False, False),
                ("../foreign", False, False, False), ("own", False, False, False), ("own", False, True, False),
                ("own", True, True, False), (None, False, False, True)):
            with self.subTest(event=event, timeout=timeout, own_image=own_image, requested=requested):
                def invoke(*args, **kwargs):
                    foreign = cache / "foreign" / "image.png"
                    foreign.parent.mkdir(parents=True, exist_ok=True)
                    foreign.write_bytes(PNG)
                    if own_image:
                        own = cache / "own" / "image.png"
                        own.parent.mkdir(parents=True, exist_ok=True)
                        own.write_bytes(PNG)
                    if requested:
                        output.write_bytes(PNG)
                    stdout = json.dumps({"type": "thread.started", "thread_id": event}) if event else ""
                    if timeout:
                        raise subprocess.TimeoutExpired("codex", 1, output=stdout.encode())
                    return subprocess.CompletedProcess(args, 0, stdout=stdout, stderr="")
                with patch.dict(os.environ, {"CODEX_HOME": str(self.root)}), patch.object(illo, "codex_available", return_value=True), \
                     patch.object(illo, "_codex_binary", return_value="codex"), patch.object(illo.subprocess, "run", side_effect=invoke):
                    if own_image or requested:
                        path, _ = illo.codex_exec_generate("fixture", [], output)
                        self.assertEqual(path, output if requested else cache / "own" / "image.png")
                    else:
                        with self.assertRaises(illo.BackendUnavailable):
                            illo.codex_exec_generate("fixture", [], output)

    def test_grok_rejects_foreign_or_invalid_artifacts(self):
        output = self.root / "requested.png"
        for content in (None, b"not an image", PNG):
            with self.subTest(content=content):
                def invoke(*args, **kwargs):
                    foreign = self.root / "sessions" / "cwd" / "foreign" / "images" / "image.png"
                    foreign.parent.mkdir(parents=True, exist_ok=True)
                    foreign.write_bytes(PNG)
                    if content is not None:
                        output.write_bytes(content)
                    return subprocess.CompletedProcess(args, 0, stdout="", stderr="")
                with patch.dict(os.environ, {"GROK_HOME": str(self.root)}), patch.object(illo, "grok_available", return_value=True), \
                     patch.object(illo, "_grok_binary", return_value="grok"), patch.object(illo.subprocess, "run", side_effect=invoke):
                    if content == PNG:
                        path, _ = illo.grok_exec_generate("fixture", [], output)
                        self.assertEqual(path, output)
                    else:
                        with self.assertRaises(illo.BackendUnavailable):
                            illo.grok_exec_generate("fixture", [], output)


if __name__ == "__main__":
    unittest.main()
