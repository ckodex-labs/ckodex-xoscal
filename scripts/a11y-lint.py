#!/usr/bin/env python3
"""Static accessibility gates for the HTML delivery surfaces.

This is a WCAG 3.0 Working Draft-aligned regression check, not a claim of
formal conformance. User-agent and assistive-technology testing remains
necessary for a complete accessibility evaluation.
"""

from __future__ import annotations

from html.parser import HTMLParser
from pathlib import Path
import re


ROOT = Path(__file__).resolve().parents[1]
HTML_FILES = sorted((ROOT / "site").glob("*.html")) + [ROOT / "portal-preview.html"]


class Element:
    def __init__(self, tag: str, attrs: dict[str, str], order: int) -> None:
        self.tag = tag
        self.attrs = attrs
        self.order = order
        self.text = ""
        self.children: list[Element] = []


class SurfaceParser(HTMLParser):
    def __init__(self) -> None:
        super().__init__(convert_charrefs=True)
        self.elements: list[Element] = []
        self.stack: list[Element] = []

    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        element = Element(tag, {key: value or "" for key, value in attrs}, len(self.elements))
        self.elements.append(element)
        if self.stack:
            self.stack[-1].children.append(element)
        if tag not in {"area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr"}:
            self.stack.append(element)

    def handle_startendtag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        self.handle_starttag(tag, attrs)

    def handle_endtag(self, tag: str) -> None:
        for index in range(len(self.stack) - 1, -1, -1):
            if self.stack[index].tag == tag:
                del self.stack[index:]
                return

    def handle_data(self, data: str) -> None:
        if self.stack:
            self.stack[-1].text += data


def classes(element: Element) -> set[str]:
    return set(element.attrs.get("class", "").split())


def descendants(element: Element, tag: str | None = None) -> list[Element]:
    result: list[Element] = []
    for child in element.children:
        if tag is None or child.tag == tag:
            result.append(child)
        result.extend(descendants(child, tag))
    return result


def fail(errors: list[str], path: Path, message: str) -> None:
    errors.append(f"{path.relative_to(ROOT)}: {message}")


def check_surface(path: Path, errors: list[str]) -> None:
    parser = SurfaceParser()
    parser.feed(path.read_text(encoding="utf-8"))
    elements = parser.elements
    tags = [element.tag for element in elements]

    roots = [element for element in elements if element.tag == "html"]
    if not roots or roots[0].attrs.get("lang") != "en":
        fail(errors, path, "document language must be declared as lang=\"en\"")
    if not any(element.tag == "main" and element.attrs.get("id") == "main" for element in elements):
        fail(errors, path, "main landmark with id=\"main\" is required")
    for landmark in ("header", "nav", "aside", "footer"):
        if landmark not in tags:
            fail(errors, path, f"{landmark} landmark is required")

    skip = next((element for element in elements if "ck-skip" in classes(element)), None)
    header = next((element for element in elements if element.tag == "header"), None)
    if skip is None or skip.attrs.get("href") != "#main":
        fail(errors, path, "skip link must target #main")
    elif header is not None and skip.order > header.order:
        fail(errors, path, "skip link must precede the shell header")

    brand = next((element for element in elements if element.tag == "a" and ("ck-site-brand" in classes(element) or "wordmark" in classes(element))), None)
    if brand is None:
        fail(errors, path, "masthead logo must be an anchor")
    else:
        if not brand.attrs.get("href"):
            fail(errors, path, "masthead logo anchor needs a destination")
        if not brand.attrs.get("aria-label"):
            fail(errors, path, "masthead logo anchor needs an accessible name")
        mark = next((child for child in descendants(brand, "img") if "mark-a2-favicon.svg" in child.attrs.get("src", "")), None)
        if mark is None:
            fail(errors, path, "masthead logo needs the DS-3 A2 mark asset")
        else:
            mark_path = (path.parent / mark.attrs["src"]).resolve()
            if not mark_path.is_file():
                fail(errors, path, f"logo asset does not resolve: {mark.attrs['src']}")
            if "alt" not in mark.attrs or mark.attrs["alt"] != "":
                fail(errors, path, "decorative logo mark must use alt=\"\"")
            if mark.attrs.get("width") != "24" or mark.attrs.get("height") != "24":
                fail(errors, path, "logo mark must declare its 24px intrinsic size")

    for image in (element for element in elements if element.tag == "img"):
        if "alt" not in image.attrs:
            fail(errors, path, "every image needs an alt attribute")

    for element in elements:
        if element.tag in {"button", "a"} and element.attrs.get("tabindex", "").lstrip("-").isdigit() and int(element.attrs["tabindex"]) > 0:
            fail(errors, path, "positive tabindex is not allowed")
        if element.tag == "button" and not element.attrs.get("aria-label") and not element.text.strip() and not descendants(element):
            fail(errors, path, "button needs an accessible name")


def main() -> int:
    errors: list[str] = []
    if not HTML_FILES:
        print("a11y-lint: no HTML delivery surfaces found")
        return 1

    stylesheet = ROOT / "site" / "styles.css"
    css = stylesheet.read_text(encoding="utf-8") if stylesheet.is_file() else ""
    for marker, label in (
        (".ck-site-brand:focus-visible", "site-brand focus style"),
        (".wordmark:focus-visible", "portal-brand focus style"),
        (".ck-site-nav a", "navigation target-size rule"),
        ("nav.surfaces button", "portal navigation target-size rule"),
        (".toggle button", "theme control target-size rule"),
        (".card-filter button", "card filter target-size rule"),
        (".claims .claim", "claim target-size rule"),
        (".journey li .jsurface", "journey target-size rule"),
        (".claims .claim:focus-visible", "claim focus style"),
        ("@media (prefers-reduced-motion: reduce)", "reduced-motion rule"),
        ("@media (forced-colors: active)", "forced-colors rule"),
    ):
        if marker not in css:
            errors.append(f"site/styles.css: missing {label}")
    if not re.search(r"\.ck-site-brand[^}]*min-height:\s*44px", css, re.S):
        errors.append("site/styles.css: site-brand target must be at least 44px")

    for page in HTML_FILES:
        check_surface(page, errors)
        if page.name in {"portal.html", "portal-preview.html"}:
            text = page.read_text(encoding="utf-8")
            for marker, label in (
                ("setAttribute('aria-pressed'", "pressed-state announcement"),
                ("setAttribute('role','button')", "keyboard role for scripted controls"),
                ("addEventListener('keydown'", "keyboard handler for scripted controls"),
            ):
                if marker not in text:
                    fail(errors, page, f"missing {label}")

    if errors:
        print("a11y-lint: FAIL")
        print("\n".join(f"- {error}" for error in errors))
        return 1
    print(f"a11y-lint: PASS ({len(HTML_FILES)} surfaces; static WCAG 3.0 WD-aligned gates)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
