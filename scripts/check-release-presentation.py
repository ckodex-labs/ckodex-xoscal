#!/usr/bin/env python3
"""Reject duplicate release presentations or lost prebuilt inventory metadata."""

import argparse
from html.parser import HTMLParser
import json
from pathlib import Path


class Presentation(HTMLParser):
    def __init__(self, text):
        super().__init__(convert_charrefs=True)
        self.headings, self.records, self.downloads, self.ids = [], {}, [], []
        self.heading, self.current, self.depth, self.in_main = None, None, 0, False
        self.feed(text)

    def handle_starttag(self, tag, pairs):
        attrs = dict(pairs)
        if tag == "main":
            self.in_main = True
        if attrs.get("id"):
            self.ids.append(attrs["id"])
        if tag == "h1":
            self.heading = ""
        if "download" in attrs:
            self.downloads.append(attrs.get("href"))
        if attrs.get("data-artifact-path"):
            path = attrs["data-artifact-path"]
            if not self.in_main:
                raise ValueError("artifact presentation must be inside main: " + path)
            if path in self.records:
                raise ValueError("duplicate artifact presentation: " + path)
            self.records[path] = {"text": "", "links": []}
            self.current, self.depth = path, 1
        elif self.current and tag not in {"br", "hr", "img", "input", "meta", "link"}:
            self.depth += 1
        if self.current and attrs.get("href"):
            self.records[self.current]["links"].append(attrs["href"])

    def handle_endtag(self, tag):
        if tag == "main":
            self.in_main = False
        if tag == "h1":
            self.headings.append(self.heading.strip())
            self.heading = None
        if self.current:
            self.depth -= 1
            if self.depth == 0:
                self.current = None

    def handle_data(self, data):
        if self.heading is not None:
            self.heading += data
        if self.current:
            self.records[self.current]["text"] += data


def downloadable(item):
    return item["kind"] in {
        "cli-archive", "release-evidence", "sdk-source", "sdk-package", "sdk-metadata",
        "oscal-catalog", "oscal-profile", "api-spec", "installer",
    } or (item["kind"] == "checksum" and item.get("release_asset_name"))


def check_page(root, name, title, items):
    page = Presentation((root / name).read_text())
    expected = {item["path"]: item for item in items}
    if page.headings != [title] or len(page.ids) != len(set(page.ids)):
        raise ValueError(name + ": distinct title and unique anchors required")
    if set(page.records) != set(expected):
        raise ValueError(name + ": exact artifact coverage differs")
    for path, item in expected.items():
        record = page.records[path]
        for value in (path, item["kind"], item["producer"], item["digest"],
                      item["oscal_touchpoint"], str(item["size"]) + " bytes"):
            if value not in record["text"]:
                raise ValueError(name + ": lost artifact metadata: " + path)
        if not set(item.get("evidence_refs", [])).issubset(record["links"]):
            raise ValueError(name + ": lost artifact evidence references: " + path)
    return page


def check(root):
    items = json.loads((root / "payload-manifest.json").read_text())["artifacts"]
    check_page(root, "downloads.html", "Downloads", [item for item in items if downloadable(item)])
    transparency = check_page(root, "transparency.html", "Supply-chain Transparency", items)
    if transparency.downloads:
        raise ValueError("Transparency must link to canonical Downloads without repeating download actions")
    print("check-release-presentation: distinct pages and complete prebuilt metadata passed")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, required=True)
    args = parser.parse_args()
    try:
        check(args.root)
    except (ValueError, OSError, KeyError) as error:
        parser.exit(1, "check-release-presentation: " + str(error) + "\n")
