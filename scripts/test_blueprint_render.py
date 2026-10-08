"""Static-first contracts inspect actual markup, not embedded data availability."""

import html
import json
import re
import shutil
import subprocess
import sys
import tempfile
import unittest
from html.parser import HTMLParser
from pathlib import Path

import blueprint_render as blueprint


class Node:
    def __init__(self, tag, attrs):
        self.tag, self.attrs = tag, dict(attrs)
        self.children, self.parts = [], []

    def text(self):
        return ' '.join(self.parts + [child.text() for child in self.children])

    def descendants(self):
        return [node for child in self.children for node in [child] + child.descendants()]


class Document(HTMLParser):
    def __init__(self, text):
        super().__init__(convert_charrefs=True)
        self.nodes, self.stack = [], []
        self.feed(text)

    def handle_starttag(self, tag, attrs):
        node = Node(tag, attrs)
        self.nodes.append(node)
        if self.stack:
            self.stack[-1].children.append(node)
        if tag not in {"area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr"}:
            self.stack.append(node)

    def handle_endtag(self, tag):
        for index in range(len(self.stack) - 1, -1, -1):
            if self.stack[index].tag == tag:
                del self.stack[index:]
                break

    def handle_data(self, text):
        if self.stack:
            self.stack[-1].parts.append(text)

    def by_id(self, value):
        return next(node for node in self.nodes if node.attrs.get("id") == value)


def normalized(text):
    return re.sub(r"\s+", " ", html.unescape(text)).strip()


def class_nodes(node, cls):
    return [n for n in node.descendants() if cls in n.attrs.get("class", "").split()]


class BlueprintStaticContract(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.text = (blueprint.ROOT / "site/portal.html").read_text()
        cls.document = Document(cls.text)

    def test_tracked_live_and_preview_are_exact_deterministic_builds(self):
        blueprint.verify_assets(blueprint.ROOT / "site")
        for path, preview in (("site/portal.html", False), ("portal-preview.html", True)):
            expected = blueprint.render(preview=preview)
            self.assertEqual((blueprint.ROOT / path).read_text(), expected)
            self.assertEqual(blueprint.render(preview=preview), expected)

    def test_all_fifteen_content_containers_have_real_prebuilt_elements(self):
        contracts = {"release-artifacts": ("dl-card", 2), "sdks": ("dl-card", 6),
                     "frameworks": ("dl-card", 36), "prov": ("dl-card", 2),
                     "canonical-cards": ("ecard", 10), "interop-cards": ("ecard", 9),
                     "defensive-cards": ("defensive", 6), "ext-cards": ("ecard", 1),
                     "dux-grid": ("dux-item", 12), "persona-grid": ("persona-card", 6),
                     "pipe-flow": ("pipe-stage", 10), "pipe-grid": ("pipe-card", 22),
                     "wf-list": ("wf-card", 4)}
        for identifier, (cls, expected) in contracts.items():
            with self.subTest(container=identifier):
                container = self.document.by_id(identifier)
                self.assertEqual(len(class_nodes(container, cls)), expected)
                self.assertGreater(len(normalized(container.text())), 30)
        for identifier, expected in (("interop-matrix", 11), ("trace-table", 15)):
            with self.subTest(table=identifier):
                table = self.document.by_id(identifier)
                self.assertEqual(sum(node.tag == "tr" for node in table.descendants()), expected + 1)
                self.assertEqual(sum(node.tag == "th" and node.attrs.get("scope") == "col" for node in table.descendants()), 4)
                self.assertTrue(any(node.tag == "caption" and node.text().strip() for node in table.descendants()))

    def test_every_persona_step_detail_and_output_is_in_html(self):
        for fixture in blueprint.load("personas"):
            node = next(n for n in self.document.nodes if n.attrs.get("data-pid") == fixture["id"])
            text = normalized(node.text())
            for key in ("name", "role", "jurisdiction", "goal"):
                self.assertIn(normalized(fixture[key]), text)
            self.assertEqual(len([n for n in node.descendants() if n.tag == "li"]), len(fixture["steps"]))
            for step in fixture["steps"]:
                for key in ("title", "surface", "detail"):
                    self.assertIn(normalized(step[key]), text)
            for output in fixture["produces"]:
                self.assertIn(output, text)

    def test_complete_cards_workflows_and_trace_are_html_not_script_strings(self):
        for slot, name, fields in (("canonical-cards", "canonical", ("id", "title")),
                                   ("interop-cards", "interop", ("id", "title")),
                                   ("ext-cards", "ext", ("id", "title")),
                                   ("pipe-grid", "pipe_cards", ("name", "fn", "cmd", "inputs", "outputs", "desc", "oscal"))):
            text = normalized(self.document.by_id(slot).text())
            for row in blueprint.load(name):
                for field in fields:
                    self.assertIn(normalized(row[field]), text)
        workflow_text = normalized(self.document.by_id("wf-list").text())
        for row in blueprint.load("workflows"):
            for step in row["journey"]:
                self.assertIn(normalized(step["step"]), workflow_text)
                self.assertIn(normalized(step["detail"]), workflow_text)
        trace_text = normalized(self.document.by_id("trace-table").text())
        for row in blueprint.load("trace"):
            for cell in row:
                self.assertIn(normalized(cell), trace_text)

    def test_all_eighteen_sections_are_named_and_have_content_without_script(self):
        sections = [n for n in self.document.nodes if n.tag == "section"]
        self.assertEqual(len(sections), 18)
        for section in sections:
            self.assertTrue(section.attrs.get("aria-label") or section.attrs.get("aria-labelledby"))
            self.assertNotIn("hidden", section.attrs)
            self.assertNotEqual(section.attrs.get("aria-hidden"), "true")
            self.assertNotIn("display:none", section.attrs.get("style", "").replace(" ", ""))
            self.assertGreater(len(normalized(section.text())), 80)
        css = (blueprint.ROOT / "site/blueprint.css").read_text()
        self.assertRegex(css, r"\.view\{display:block;")
        self.assertIn('html[data-blueprint-enhanced] .view:not(.active){display:none}', css)
        self.assertNotIn("data-blueprint-enhanced", self.text)

    def test_page_heading_and_view_headings_have_a_static_outline(self):
        headings = [n for n in self.document.nodes if re.fullmatch(r'h[1-6]', n.tag)]
        self.assertEqual(len([n for n in headings if n.tag == 'h1']), 1)
        self.assertEqual([n.text() for n in headings if n.tag == 'h1'], ['Governed Assurance Blueprint'])
        for name in blueprint.VIEWS:
            heading = self.document.by_id('view-' + name + '-title')
            self.assertEqual(heading.tag, 'h2')
        self.assertFalse(any(n.tag == 'h4' for n in headings))

    def test_primary_navigation_and_brand_match_canonical_site_shell(self):
        header = next(n for n in self.document.nodes if n.tag == 'header')
        self.assertEqual(header.attrs.get('class'), 'ck-shell__header')
        brand = next(n for n in header.descendants() if 'ck-site-brand' in n.attrs.get('class', '').split())
        self.assertEqual(brand.tag, 'a')
        self.assertEqual(brand.attrs['href'], 'index.html')
        primary = next(n for n in header.descendants() if n.tag == 'nav' and n.attrs.get('aria-label') == 'Primary')
        links = [n for n in primary.descendants() if n.tag == 'a']
        expected = [('index.html', 'Overview'), ('cli.html', 'CLI Guide'), ('docs.html', 'API Docs'),
                    ('downloads.html', 'Downloads'), ('review.html', 'Live Review'),
                    ('transparency.html', 'Transparency'), ('sbom-validation.html', 'SBOM Validation'),
                    ('portal.html', 'Design Blueprint')]
        self.assertEqual([(n.attrs['href'], normalized(n.text())) for n in links], expected)
        self.assertEqual([n.attrs['href'] for n in links if n.attrs.get('aria-current') == 'page'], ['portal.html'])
        self.assertNotIn(self.document.by_id('nav'), header.descendants())
        self.assertEqual(self.document.by_id('nav').attrs['aria-label'], 'Blueprint sections')

    def test_views_share_one_flow_container_instead_of_overlapping_grid_cells(self):
        main = self.document.by_id('main')
        children = [n for n in main.children if n.tag != 'script']
        self.assertEqual([n.tag for n in children], ['h1', 'div', 'div', 'aside'])
        self.assertEqual(children[0].attrs['class'], 'ck-page-title')
        self.assertEqual(children[1].attrs['class'], 'banner')
        content = children[2]
        self.assertEqual(content.attrs['class'], 'ck-portal-content')
        self.assertEqual([n.attrs['id'] for n in content.children], ['view-' + name for name in blueprint.VIEWS])
        css = (blueprint.ROOT / 'site/styles.css').read_text()
        self.assertIn('grid-template-areas: "title title" "banner banner" "content margin"', css)
        self.assertNotRegex(css, r'\.ck-portal-main\s*>\s*\.view\s*\{[^}]*grid-area:')

    def test_theme_aliases_use_canonical_tokens_without_duplicate_palette(self):
        css = (blueprint.ROOT / 'site/blueprint.css').read_text()
        self.assertIn('--tone:var(--ck-fg-3)', css)
        self.assertIn('--surface:var(--ck-bg-1)', css)
        self.assertIn('--serif:var(--ck-ff-display)', css)
        self.assertNotRegex(css, r'--(?:tone|surface|paper|ink):\s*#')
        body = re.search(r'\bbody\{([^}]+)\}', css)[1]
        self.assertNotIn('transition', body)

    def test_navigation_and_every_journey_have_resolving_native_anchors(self):
        ids = {n.attrs["id"] for n in self.document.nodes if n.attrs.get("id")}
        self.assertEqual(len(ids), len([n for n in self.document.nodes if n.attrs.get("id")]))
        nav = self.document.by_id("nav")
        links = [n for n in nav.descendants() if n.tag == "a"]
        self.assertEqual({n.attrs.get("href") for n in links}, {"#view-" + view for view in blueprint.VIEWS})
        self.assertEqual(len(links), 6)
        self.assertEqual(len([n for n in self.document.nodes if n.tag == "nav" and n.attrs.get("class") == "view-nav"]), 6)
        for node in self.document.nodes:
            if node.tag == "a" and node.attrs.get("href", "").startswith("#"):
                self.assertIn(node.attrs["href"][1:], ids)
        journeys = [n for n in self.document.nodes if "jsurface" in n.attrs.get("class", "").split()]
        self.assertTrue(journeys)
        self.assertTrue(all(n.tag == "a" and n.attrs["href"].startswith("#view-") for n in journeys))

    def test_no_runtime_content_assembly_or_network_dependency(self):
        scripts = [n for n in self.document.nodes if n.tag == "script"]
        self.assertEqual(len(scripts), 1)
        self.assertEqual(scripts[0].attrs, {"src": "blueprint.js", "defer": None})
        self.assertFalse(scripts[0].text().strip())
        script = (blueprint.ROOT / "site/blueprint.js").read_text()
        self.assertNotRegex(script, r"\b(?:fetch|XMLHttpRequest|innerHTML|outerHTML|insertAdjacentHTML|createElement|eval)\b")
        self.assertNotIn("href=\"#\"", self.text)
        self.assertNotIn("{{", self.text)
        self.assertNotIn("undefined", self.document.by_id("pipe-flow").text())

    def test_sample_downloads_and_actions_never_gain_eligibility(self):
        for slot in ("release-artifacts", "sdks", "frameworks", "prov"):
            node = self.document.by_id(slot)
            self.assertFalse(any(n.tag == "a" for n in node.descendants()))
            self.assertEqual(len(class_nodes(node, "dl-card")), len(class_nodes(node, "unsigned")))
        for node in self.document.nodes:
            if "ck-action-unwired" in node.attrs.get("class", "").split():
                self.assertEqual(node.tag, "span")
                self.assertEqual(node.attrs.get("aria-disabled"), "true")

    def test_pipeline_producer_inventory_includes_every_catalog_and_pbmm_profile(self):
        text = self.document.by_id("frameworks").text()
        policy = json.loads((blueprint.ROOT / "docs/release-policy.json").read_text())
        for identifier in policy["framework_ids"]:
            self.assertIn(identifier + "/catalog.json", text)
        self.assertIn("cccs-medium-cloud-pbmm/profile.json", text)


class BlueprintRefusal(unittest.TestCase):
    def test_unverified_fixture_cannot_render_signed_download_or_url(self):
        fixture = blueprint.load("sdks")[0]
        for replacement in ({"signed": True}, {"url": "https://example.com/sdk.zip"}):
            with self.subTest(replacement=replacement), self.assertRaisesRegex(ValueError, "cannot grant"):
                blueprint.download_card(dict(fixture, **replacement))

    def test_unknown_template_slot_fails_instead_of_shipping_empty_content(self):
        with self.assertRaisesRegex(ValueError, "unknown.*slot"):
            blueprint.substitute('<div>{{missing-cards}}</div>', {})

    def test_empty_missing_or_executable_fixture_is_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            inputs = Path(directory) / "inputs"
            shutil.copytree(blueprint.INPUTS, inputs)
            fixture = inputs / "data/defensive.json"
            for unsafe in ('<script>alert(1)</script>', '<img src="x" onerror="alert(1)">', '<a href="javascript:alert(1)">run</a>'):
                fixture.write_text(json.dumps([{"html": unsafe}]))
                with self.subTest(unsafe=unsafe), self.assertRaisesRegex(ValueError, "Blueprint"):
                    blueprint.render(inputs=inputs)
            fixture.write_text('[]')
            with self.assertRaisesRegex(ValueError, "missing.*rows"):
                blueprint.render(inputs=inputs)
            fixture.unlink()
            with self.assertRaises(FileNotFoundError):
                blueprint.render(inputs=inputs)

    def test_actual_candidate_with_empty_container_is_refused(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            shutil.copytree(blueprint.ROOT / "site", root, dirs_exist_ok=True)
            path = root / "portal.html"
            original = path.read_text()
            for replacement in ('<div class="persona-grid" id="persona-grid"></div>', '<div id="persona-grid"><script type="application/json">{"personas":6}</script></div>'):
                path.write_text(re.sub(r'<div class="persona-grid" id="persona-grid">.*?</article></div>', replacement, original, flags=re.S))
                result = subprocess.run([sys.executable, str(blueprint.ROOT / "scripts/check-blueprint.py"), '--root', str(root)], capture_output=True, text=True)
                with self.subTest(replacement=replacement):
                    self.assertNotEqual(result.returncode, 0)
                    self.assertIn("complete deterministic build", result.stderr)

    def test_drift_check_refuses_stale_html_without_mutating_it(self):
        with tempfile.TemporaryDirectory() as directory:
            target = Path(directory) / "portal.html"
            target.write_text('<main id="main">prebuilt data only</main>')
            before = target.read_bytes()
            result = subprocess.run([sys.executable, str(blueprint.ROOT / "scripts/render-blueprint.py"), '--check', '--output', str(target)], capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn("differs", result.stderr)
            self.assertEqual(target.read_bytes(), before)

    def test_actual_candidate_with_tampered_enhancement_is_refused(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            shutil.copytree(blueprint.ROOT / "site", root, dirs_exist_ok=True)
            (root / 'blueprint.js').write_text('fetch("https://example.com/dynamic-cards");')
            result = subprocess.run([sys.executable, str(blueprint.ROOT / "scripts/check-blueprint.py"), '--root', str(root)], capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('enhancement assets differ', result.stderr)


if __name__ == "__main__":
    unittest.main()
