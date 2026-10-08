#!/usr/bin/env python3
"""Render disclosure pages from exact payload; signatures are verified before coloring."""

import argparse
import hashlib
import html
import importlib.util
import json
import re
from pathlib import Path

from release_inventory import digest, validate, write


def escaped(value):
    return html.escape(str(value), quote=True)


def replace_main(path, content):
    text = path.read_text()
    text = re.sub(r'(<main id="main"[^>]*>).*?(</main>)',
                  lambda match: match[1] + content + match[2], text, flags=re.S)
    text = re.sub(r'(<aside class="ck-shell__margin"[^>]*>).*?(</aside>)',
                  lambda m: m[1] + '<h2>Evidence Margin</h2><p>See the pinned manifest and proof envelope. Missing proof is incomplete.</p>' + m[2], text, flags=re.S)
    text = re.sub(r"<script>.*?</script>", "", text, flags=re.S)
    path.write_text(text)


def artifact_id(item):
    return "artifact-" + hashlib.sha256(item["path"].encode()).hexdigest()[:20]


def artifact_evidence(item, verified):
    refs = list(item.get("evidence_refs", []))
    if verified:
        refs += ["payload-manifest.json", "proofs/payload.sigstore.json"]
    return '<ul>' + ''.join(f'<li><a href="{escaped(ref)}">{escaped(ref)}</a></li>' for ref in dict.fromkeys(refs)) + '</ul>' if refs else '<p>Incomplete: no artifact evidence references supplied.</p>'


def artifact_details(item, verified):
    return ('<details class="ck-artifact-details"><summary>Digest, producer and evidence</summary>'
            + f'<p>Producer: {escaped(item["producer"])}</p>'
            + f'<p>OSCAL touchpoint: {escaped(item["oscal_touchpoint"])}</p>'
            + f'<p>{item["size"]} bytes</p><code class="ck-artifact-digest">{escaped(item["digest"])}</code>'
            + '<p>Evidence:</p>' + artifact_evidence(item, verified) + '</details>')


def download_action(item, verified):
    if verified:
        return f'<a href="{escaped(item["path"])}" download>Download {escaped(item["name"])}</a>'
    return '<span aria-disabled="true">Download unavailable</span>'


def artifact_card(item, verified):
    return (f'<article id="{artifact_id(item)}" data-artifact-path="{escaped(item["path"])}" class="ck-quiet ck-padded"><strong>{escaped(item["path"])}</strong>'
            + f'<p class="ck-caption">{escaped(item["kind"])}</p>'
            + '<p>' + download_action(item, verified) + '</p>' + artifact_details(item, verified) + '</article>')


def disclosure(manifest, verified, title):
    state = "Verified payload provenance" if verified else "Unsigned local candidate"
    return ('<div class="ck-measure ck-stack"><h1>' + escaped(title) + '</h1>'
            + '<p class="ck-caption">' + state + '</p>'
            + f'<p>Release: {escaped(manifest["release_tag"])}. Source: <code>{escaped(manifest["source_revision"])}</code>.</p>'
            + '<p>Downloads require successful signature verification against the expected repository, workflow, source and tag. Final site integrity is covered separately by the release manifest.</p>')


def render_downloads(root, manifest, verified):
    kinds = (("Binaries and release evidence", ("cli-archive", "release-evidence")),
             ("SDK sources and packages", ("sdk-source", "sdk-package", "sdk-metadata")),
             ("OSCAL catalogs and profiles", ("oscal-catalog", "oscal-profile")),
             ("API and installer", ("api-spec", "installer")),
             ("SHA-256 sidecars", ("checksum",)))
    content = disclosure(manifest, verified, "Downloads")
    content += '<p>Installable packages and distributable files belong here. See <a href="transparency.html">Transparency</a> for the complete inventory and analysis evidence.</p>'
    for title, allowed in kinds:
        rows = [a for a in manifest["artifacts"] if a["kind"] in allowed and (a["kind"] != "checksum" or a.get("release_asset_name"))]
        content += '<section><h2>' + title + '</h2>'
        if allowed == ("checksum",):
            content += '<details><summary>Show downloadable SHA-256 sidecars</summary>'
        if allowed in (("oscal-catalog", "oscal-profile"), ("checksum",)):
            content += artifact_table(rows, verified, downloads=True)
        else:
            content += '<div class="ck-data-grid">' + ''.join(artifact_card(a, verified) for a in rows) + '</div>'
        if allowed == ("checksum",):
            content += '</details>'
        content += '</section>'
    content += '</div>'
    replace_main(root / "downloads.html", content)


def render_transparency(root, manifest, verified):
    content = disclosure(manifest, verified, "Supply-chain Transparency")
    content += '<p>Inspect the exact inventory, producers and evidence. Use <a href="downloads.html">Downloads</a> to obtain packages and distributable artifacts.</p>'
    content += '<p><a href="payload-manifest.json">Payload manifest</a></p>'
    if verified:
        content += '<p><a href="proofs/payload.sigstore.json">Signature bundle</a> / <a href="proofs/payload-verification.json">Cryptographic verification results</a></p>'
    artifacts = manifest["artifacts"]
    content += f'<section id="release-inventory"><h2>Complete payload inventory</h2><p>{len(artifacts)} files. Every row retains its exact digest, producer, OSCAL touchpoint and evidence references.</p>'
    for kind in sorted({item["kind"] for item in artifacts}):
        rows = [item for item in artifacts if item["kind"] == kind]
        content += '<details class="ck-inventory-group"><summary>' + escaped(kind) + f' — {len(rows)} files</summary>'
        content += artifact_table(rows, verified, downloads=False) + '</details>'
    content += '</section></div>'
    replace_main(root / "transparency.html", content)


def artifact_table(items, verified, downloads):
    content = '<div class="ck-table-scroll" tabindex="0" role="region" aria-label="Artifact inventory table"><table class="ck-table ck-artifact-table"><caption>'
    content += ('Distributable files' if downloads else 'Exact payload files')
    content += '</caption><thead><tr><th scope="col">Artifact</th><th scope="col">Kind</th><th scope="col">' + ('Download' if downloads else 'Evidence') + '</th></tr></thead><tbody>'
    for item in items:
        anchor = artifact_id(item)
        content += f'<tr id="{anchor if downloads else "inventory-" + anchor}" data-artifact-path="{escaped(item["path"])}"><th scope="row"><code>{escaped(item["path"])}</code></th>'
        content += '<td>' + escaped(item['kind']) + '</td><td>'
        if downloads:
            content += download_action(item, verified)
        content += artifact_details(item, verified) + '</td></tr>'
    return content + '</tbody></table></div>'


def render_sbom(root):
    path = root / "sbom-assessment-results.json"
    content = '<div class="ck-measure ck-stack"><h1>SBOM Validation</h1>'
    if not path.is_file():
        content += '<p>Incomplete: no SBOM assessment supplied. No pass state is available.</p>'
    else:
        document = json.loads(path.read_text())
        results = document.get("assessment-results", {}).get("results", [])
        findings = [f for r in results for f in r.get("findings", [])]
        satisfied = sum(f.get("target", {}).get("status", {}).get("state") == "satisfied" for f in findings)
        observations = sum(len(r.get("observations", [])) for r in results)
        content += f'<p>Observed findings: {len(findings)}; satisfied: {satisfied}; other states: {len(findings) - satisfied}; observations: {observations}.</p>'
        content += '<p>These counts describe raw findings; release admission uses the enforcing policy and exact SBOM digest.</p>'
        content += '<p><a href="sbom-assessment-results.json">Raw OSCAL assessment</a>; <a href="transparency.html">release inventory and admission evidence</a>.</p>'
        content += '<details><summary>Inspect the complete raw assessment</summary><pre class="ck-quiet ck-code-block">' + escaped(json.dumps(document, indent=2)) + '</pre></details>'
    replace_main(root / "sbom-validation.html", content + '</div>')


def verified_payload(args):
    if args.bundle is None:
        return False
    module_path = Path(__file__).with_name("release-contract.py")
    spec = importlib.util.spec_from_file_location("release_contract", module_path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    receipt = module.verify_attestation(args.root, args.manifest, args.bundle, args.tag, args.revision)
    write(args.root / "proofs/payload-verification.json", receipt)
    return True


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, required=True)
    parser.add_argument("--manifest", type=Path, required=True)
    parser.add_argument("--bundle", type=Path)
    parser.add_argument("--tag")
    parser.add_argument("--revision")
    args = parser.parse_args()
    manifest = json.loads(args.manifest.read_text())
    validate(args.root, manifest)
    verified = verified_payload(args)
    # Legacy main-generated inventories refer to different release bytes and
    # must never travel alongside this candidate's exact pinned inventory.
    (args.root / "release-assets.json").unlink(missing_ok=True)
    render_downloads(args.root, manifest, verified)
    render_transparency(args.root, manifest, verified)
    render_sbom(args.root)
    index = dict(manifest, verification="verified" if verified else "incomplete")
    for item in index["artifacts"]:
        item["eligible"] = verified
        item["evidence_refs"] = list(item.get("evidence_refs", [])) + (["payload-manifest.json", "proofs/payload.sigstore.json"] if verified else [])
    write(args.root / "downloads.json", index)
    write(args.root / "provenance.json", index)
    print(f"render-release-site: {len(index['artifacts'])} artifacts; provenance {'verified' if verified else 'incomplete'}")


if __name__ == "__main__":
    main()
