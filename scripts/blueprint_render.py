"""Deterministic Blueprint presentation; sample fixtures never grant proof."""

import html
import hashlib
import json
import re
from html.parser import HTMLParser
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
INPUTS = Path(__file__).with_name("blueprint")
VIEWS = ("room", "exchange", "downloads", "cartography", "personas", "pipeline")
LABELS = ("Assurance Room", "Exchange", "Sample downloads", "Cartography", "Personas", "Pipeline")
ASSETS = ("blueprint.js", "blueprint.css", "blueprint-views.css")


def asset_digests(root):
    return {name: hashlib.sha256((root / name).read_bytes()).hexdigest() for name in ASSETS}


def verify_assets(root):
    expected = json.loads((INPUTS / "enhancement-sha256.json").read_text())
    if asset_digests(root) != expected:
        raise ValueError("Blueprint enhancement assets differ from tracked producer inputs")


def escaped(value):
    return html.escape(str(value), quote=True)


def load(name, inputs=INPUTS):
    value = json.loads((inputs / "data" / (name + ".json")).read_text())
    if not isinstance(value, list) or not value:
        raise ValueError("missing Blueprint fixture rows: " + name)
    return value


def action(text, primary=False):
    cls = "act ck-action-unwired" + (" primary" if primary else "")
    return (f'<span class="{cls}" aria-disabled="true" '
            'title="Illustrative action; not wired in this Blueprint">'
            + escaped(text) + '</span>')


def download_card(row):
    if row.get("signed") is not False or row.get("url"):
        raise ValueError("Blueprint fixture cannot grant download eligibility")
    return ('<article class="dl-card unsigned"><div class="name">' + escaped(row["name"])
            + '</div><div class="lang">' + escaped(row["lang"])
            + '</div><div class="digest">digest <span class="dig">'
            + escaped(row["digest"]) + '</span></div><div class="row">'
            '<span class="nolink">not downloadable</span>'
            '<span class="mark">○ unsigned — illustrative fixture</span></div></article>')


def card(row):
    cls = "ecard" + (" sealed" if row.get("sealed") else "") + (" alarm" if row.get("alarm") else "")
    content = (f'<article class="{cls}"><div class="head"><div class="title">'
               + escaped(row["id"]) + '<span class="sub">' + escaped(row["title"])
               + '</span></div><span class="chip ' + escaped(row["chipCls"]) + '">'
               + escaped(row["chip"]) + '</span></div>')
    for key in ("meta", "role", "evidence", "diff", "links"):
        if row.get(key):
            content += '<div class="' + key + '">' + row[key] + '</div>'
    if row.get("blocked") or row.get("next"):
        content += '<div class="act-footer">'
        if row.get("blocked"):
            content += '<div class="act-blocked"><span class="why">' + escaped(row["blocked"]) + '</span></div>'
        if row.get("next"):
            content += '<span class="act-label">illustrative next actions</span><div class="act-list">'
            content += ''.join(action(text, index == 0) for index, text in enumerate(row["next"])) + '</div>'
        content += '</div>'
    return content + '</article>'


def surface_link(text):
    mappings = ((r"Exchange|inbound|verify row|outbound|disposition|quarantine", "exchange"),
                (r"Downloads|SDK|framework", "downloads"),
                (r"Cartography|SBOM Card|Scanner Card|VEX Card|AI Tool Card|AI-BOM|Attestation Card|Defensive UX|stale evidence", "cartography"),
                (r"Pipeline|Dagger|CI|workflow|Release|Pages|Reconcile|gate|evidence producer", "pipeline"))
    target = next((view for pattern, view in mappings if re.search(pattern, text, re.I)), "room")
    return f'<a class="jsurface" href="#view-{target}">{escaped(text)}</a>'


def step(row):
    cls = "defensive" if row.get("defensive") else ("complete" if row["state"] == "advance" else "")
    states = {"advance": "→ advances state", "blocked": "⊘ blocked", "inspect": "⊢ inspect", "export": "↗ export"}
    return (f'<li class="{cls}"><div class="jtitle">' + escaped(row["title"]) + '</div>'
            + surface_link(row["surface"]) + '<div class="jdetail">' + escaped(row["detail"])
            + '</div><span class="jstate ' + escaped(row["state"]) + '">'
            + states[row["state"]] + '</span></li>')


def persona(row):
    return ('<article class="persona-card" data-pid="' + escaped(row["id"]) + '">'
            '<div class="phead"><div class="avatar" aria-hidden="true">' + escaped(row["glyph"])
            + '</div><div class="pinfo"><div class="pname">' + escaped(row["name"])
            + '</div><div class="prole">' + escaped(row["role"] + " — " + row["jurisdiction"])
            + '</div></div></div><div class="pgoal"><span class="lab">goal</span>'
            + escaped(row["goal"]) + '</div><ol class="journey">'
            + ''.join(step(s) for s in row["steps"]) + '</ol><div class="act-footer">'
            '<span class="act-label">illustrative outputs</span><div class="act-list">'
            + ''.join(action(text, index == 0) for index, text in enumerate(row["produces"]))
            + '</div></div></article>')


def pipe_card(row):
    content = ('<article class="pipe-card" data-pc-tags="' + escaped(','.join(row["tags"])) + '">'
               '<div class="pc-head"><div><div class="pc-name">' + escaped(row["name"])
               + '</div><div class="pc-fn">' + escaped(row["fn"]) + '</div></div>'
               '<span class="pc-chip ' + escaped(row["cls"]) + ' unsigned">'
               + escaped(row["status"]) + '</span></div><div class="pc-body">')
    for field, label in (("cmd", "command"), ("inputs", "inputs"), ("outputs", "outputs"), ("desc", "what it does")):
        content += '<span class="lab">' + label + '</span>' + escaped(row[field])
    return content + '</div><div class="pc-oscal"><span class="lab">OSCAL touchpoint</span>' + escaped(row["oscal"]) + '</div></article>'


def workflow(row):
    return ('<article class="wf-card"><div class="wf-head"><div><div class="wf-name">'
            + escaped(row["name"]) + '</div><div class="wf-trigger">'
            + escaped(row["file"] + " — trigger: " + row["trigger"]) + '</div></div>'
            '<span class="wf-chip unsigned">' + escaped(row["status"]) + '</span></div>'
            '<div class="wf-body"><span class="lab">OSCAL touchpoint</span>' + escaped(row["oscal"])
            + '</div><ol class="journey">' + ''.join('<li><div class="jtitle">' + escaped(s["step"])
            + '</div><div class="jdetail">' + escaped(s["detail"]) + '</div></li>' for s in row["journey"])
            + '</ol><div class="act-footer"><span class="act-label">produces</span><div class="act-list">'
            + ''.join('<span class="act">' + escaped(text) + '</span>' for text in row["produces"]) + '</div></div></article>')


def table(headers, rows, caption):
    return ('<caption>' + escaped(caption) + '</caption><thead><tr>'
            + ''.join('<th scope="col">' + escaped(h) + '</th>' for h in headers)
            + '</tr></thead><tbody>' + ''.join('<tr>' + ''.join('<td>' + escaped(c)
            + '</td>' for c in row) + '</tr>' for row in rows) + '</tbody>')


def framework_rows(root=ROOT):
    # The enforcing release policy already pins this exact producer inventory.
    policy = json.loads((root / "docs/release-policy.json").read_text())
    ids = policy["framework_ids"]
    if len(ids) != len(set(ids)) or not all(re.fullmatch(r"[A-Za-z0-9_.-]+", value) for value in ids):
        raise ValueError("invalid exact framework inventory")
    paths = [value + "/catalog.json" for value in ids] + ["cccs-medium-cloud-pbmm/profile.json"]
    return [{"name": path, "lang": "OSCAL profile — illustrative producer inventory" if path.endswith("profile.json") else "OSCAL catalog — illustrative producer inventory", "digest": "not supplied", "signed": False} for path in paths]


def content_slots(inputs=INPUTS, root=ROOT):
    slots = {key: ''.join(download_card(row) for row in load(name, inputs))
             for key, name in (("release-artifacts", "release_artifacts"), ("sdks", "sdks"), ("prov", "prov"))}
    slots["frameworks"] = ''.join(download_card(row) for row in framework_rows(root))
    for slot, name in (("canonical-cards", "canonical"), ("interop-cards", "interop"), ("ext-cards", "ext")):
        slots[slot] = ''.join(card(row) for row in load(name, inputs))
    slots["defensive-cards"] = ''.join(row["html"] for row in load("defensive", inputs))
    slots["interop-matrix"] = table(("Artifact", "Primary Card", "OSCAL Touchpoint", "Extension"), load("interop_matrix", inputs), "Illustrative interoperability decisions")
    slots["dux-grid"] = ''.join('<div class="dux-item"><div class="id">' + escaped(row[0]) + '</div><div class="rule">' + escaped(row[1]) + '</div></div>' for row in load("dux", inputs))
    slots["persona-grid"] = ''.join(persona(row) for row in load("personas", inputs))
    slots["pipe-flow"] = ''.join('<div class="pipe-stage ' + escaped(s["cls"]) + '"><div class="pnode">' + escaped(s["node"]) + '</div><div class="pname">' + escaped(s["name"]) + '</div><div class="pcmd">' + escaped(s.get("cmd", "")) + '</div></div>' for s in load("pipe_flow", inputs))
    slots["pipe-grid"] = ''.join(pipe_card(row) for row in load("pipe_cards", inputs))
    slots["wf-list"] = ''.join(workflow(row) for row in load("workflows", inputs))
    slots["trace-table"] = table(("Producer", "Output", "Surfaces in", "Admission / presentation"), load("trace", inputs), "Producer-to-portal evidence trace")
    return slots


def navigation(index):
    links = []
    for offset, direction in ((-1, "previous"), (1, "next")):
        target = index + offset
        if 0 <= target < len(VIEWS):
            links.append(f'<a class="{direction}" href="#view-{VIEWS[target]}"><span class="vlabel">{direction}</span><span class="vname">{LABELS[target]}</span></a>')
    return '<nav class="view-nav" aria-label="Adjacent Blueprint sections">' + ''.join(links) + '</nav>'


def substitute(template, slots):
    def replace(match):
        if match[1] not in slots:
            raise ValueError("unknown Blueprint template slot: " + match[1])
        return slots[match[1]]
    return re.sub(r"\{\{([a-z-]+)\}\}", replace, template)


def inert_actions(text):
    def replace(match):
        attrs = re.sub(r'\s+href="#"', '', match[1])
        if 'class="' in attrs:
            attrs = attrs.replace('class="', 'class="ck-action-unwired ', 1)
        else:
            attrs += ' class="ck-action-unwired"'
        return ('<span' + attrs
                + ' aria-disabled="true" title="Illustrative action; not wired in this Blueprint">'
                + match[2] + '</span>')
    return re.sub(r'<a\b([^>]*\bhref="#"[^>]*)>(.*?)</a>',
                  replace, text, flags=re.S)


class SafeMarkup(HTMLParser):
    """Reject executable fixture markup instead of relying on review alone."""

    def handle_starttag(self, tag, attrs):
        values = dict(attrs)
        if tag in {"iframe", "object", "embed", "base"}:
            raise ValueError("unsafe Blueprint markup tag: " + tag)
        if tag == "script" and values != {"src": "blueprint.js", "defer": None}:
            raise ValueError("Blueprint runtime may only enhance prebuilt HTML")
        for key, value in attrs:
            if key.startswith("on") or (key in {"href", "src"} and value and re.match(r"\s*(?:javascript|data):", value, re.I)):
                raise ValueError("executable Blueprint fixture attribute")


def render(inputs=INPUTS, root=ROOT, preview=False):
    slots = content_slots(inputs, root)
    for index, name in enumerate(VIEWS):
        template = (inputs / "views" / (name + ".html")).read_text()
        slots["view-" + name] = substitute(template, dict(slots, **{"view-navigation": navigation(index)}))
    text = inert_actions(substitute((inputs / "shell.html").read_text(), slots))
    SafeMarkup().feed(text)
    if preview:
        text = re.sub(r'((?:href|src)=")((?!#|https?:)[^"]+)(")', r'\1site/\2\3', text)
        text = text.replace("Governed Assurance Room Blueprint</title>", "Governed Assurance Room Blueprint (Sample)</title>")
    return text
