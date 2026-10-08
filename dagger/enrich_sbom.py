#!/usr/bin/env python3
"""Record the release distributor for byte-backed components, never origin authors."""
import argparse
import hashlib
import json
import re
from urllib.parse import quote
from pathlib import Path

# These catalogers observe delivered package/build metadata, not a source
# dependency declaration. Keep this allowlist narrow and independently tested.
BUNDLED = {'go-module-binary-cataloger', 'java-archive-cataloger',
           'dpkg-db-cataloger', 'rpm-db-cataloger', 'apk-db-cataloger',
           'python-installed-package-cataloger'}


def enrich_publishers(raw, publisher_dir):
    authors_changed = []
    if publisher_dir is not None:
        from publisher_metadata import verify_record
        ledger = json.loads((publisher_dir / 'publishers.json').read_text())
        if ledger.get('version') != 1: raise ValueError('invalid publisher ledger')
        by_ref = {c['bom-ref']:c for c in raw['components']}
        seen_publishers=set()
        for record in ledger['records']:
            ref=record['bom_ref']
            if ref in seen_publishers or ref not in by_ref: raise ValueError('ambiguous publisher component')
            seen_publishers.add(ref)
            if not isinstance(record.get('raw_sha256'),str) or not re.fullmatch(r'[0-9a-f]{64}',record['raw_sha256']):raise ValueError('invalid publisher metadata digest')
            filename=record['raw_sha256']+'.metadata'
            if record.get('file')!=filename: raise ValueError('invalid publisher metadata filename')
            author,supplier=verify_record(by_ref[ref],record,(publisher_dir/filename).read_bytes(),(publisher_dir/'go.sum').read_bytes() if (publisher_dir/'go.sum').is_file() else None,(publisher_dir/'Package.resolved').read_bytes() if (publisher_dir/'Package.resolved').is_file() else None)
            if author and not by_ref[ref].get('author'):by_ref[ref]['author']=author;authors_changed.append(ref)
            if supplier and not by_ref[ref].get('supplier'):by_ref[ref]['supplier']=supplier
            if record.get('source')=='artifact:SDK-PRODUCER.json' and by_ref[ref].get('version') in (None,'','UNKNOWN'):
                by_ref[ref]['version']=record['version']
                if by_ref[ref].get('purl','').startswith('pkg:golang/') and '@' not in by_ref[ref]['purl']:by_ref[ref]['purl']+='@'+quote(record['version'],safe='')
    return authors_changed


def python_declarations(raw, publisher_dir, declared):
    if publisher_dir is not None and (publisher_dir/'requirements.txt').is_file():
        requirements=(publisher_dir/'requirements.txt').read_bytes()
        files=[c for c in raw['components'] if c.get('type')=='file' and Path(c['name']).name=='requirements.txt' and any(h.get('alg')=='SHA-256' and h.get('content')==hashlib.sha256(requirements).hexdigest() for h in c.get('hashes',[]))]
        if len(files)!=1:raise ValueError('requirements graph lacks exact delivered file evidence')
        for line in requirements.decode().splitlines():
            match=re.fullmatch(r'([A-Za-z0-9_.-]+)==([A-Za-z0-9_.+!-]+)',line.strip())
            if not match:raise ValueError('unsupported requirement relationship')
            name,version=match.groups()
            matches=[c for c in raw['components'] if c.get('purl','').startswith('pkg:pypi/') and c['name'].replace('_','-').lower()==name.replace('_','-').lower() and c.get('version')==version]
            if len(matches)!=1:raise ValueError('requirement lacks unique exact-version inventory')
            declared.append(matches[0]['bom-ref'])
    return declared


def observed_components(raw, distributor):
    changed, bundled = [], []
    for c in raw['components']:
        props = {p['name']: p['value'] for p in c.get('properties', [])}
        catalogers = {p['value'] for p in c.get('properties', []) if p['name']=='syft:package:foundBy'}
        cataloger = next(iter(sorted(catalogers & BUNDLED)), None)
        file_hash = c.get('type') == 'file' and any(h.get('alg') == 'SHA-256' and len(h.get('content', '')) == 64 for h in c.get('hashes', []))
        observed_os = c.get('type')=='operating-system' and props.get('syft:distro:id')==c.get('name') and props.get('syft:distro:versionID')==c.get('version')
        observed = file_hash or observed_os or (cataloger is not None and any(k.startswith('syft:location:') and k.endswith(':path') and v for k,v in props.items()))
        if not observed:
            continue
        bundled.append(c['bom-ref'])
        if not c.get('supplier'):
            c['supplier'] = dict(distributor)
            c.setdefault('properties', []).extend([
                {'name':'xoscal:release:supplier-role', 'value':'distributor/repackager; does not assert upstream manufacturer'},
                {'name':'xoscal:release:supplier-evidence', 'value':cataloger or ('observed installed OS release identity' if observed_os else 'SHA-256 of delivered file')},
            ])
            changed.append(c['bom-ref'])
        if observed_os and not c.get('purl'):
            c['purl']='pkg:generic/'+quote(c['name'],safe='')+'@'+quote(c['version'],safe='')
    return changed, bundled


def enrich(raw, policy, publisher_dir=None, subject=None):
    if raw.get('bomFormat') != 'CycloneDX' or not raw.get('components'):
        raise ValueError('no CycloneDX component inventory')
    if policy.get('version') != 1 or policy.get('role') != 'release-distributor':
        raise ValueError('invalid distributor policy')
    distributor = policy['entity']
    if not distributor.get('name') or not distributor.get('url'):
        raise ValueError('distributor must have an evidenced identity')
    refs = {c['bom-ref'] for c in raw['components']}
    root = raw['metadata']['component']
    root_ref = root['bom-ref']
    if len(refs) != len(raw['components']) or root_ref in refs:
        raise ValueError('ambiguous component references')
    authors_changed = enrich_publishers(raw, publisher_dir)
    from go_module_relationships import declared_graph
    declared, go_graph = declared_graph(raw, publisher_dir)
    declared = python_declarations(raw, publisher_dir, declared)
    changed, bundled = observed_components(raw, distributor)
    root['supplier'] = root.get('supplier') or dict(distributor)
    if subject and not root.get('version'):root['version']='sha256:'+subject
    if subject and not root.get('purl'):
        root['purl']='pkg:generic/'+quote(Path(root['name']).name,safe='')+'@'+quote(root['version'],safe='')
    deps = raw.setdefault('dependencies', [])
    for dep in deps:
        if dep['ref'] not in refs | {root_ref} or any(x not in refs | {root_ref} for x in dep.get('dependsOn', [])):
            raise ValueError('dependency graph references absent inventory')
    # The delivered container/archive contains these byte-backed components.
    # Declaration-only dependency edges remain intact; no fabricated upstream graph.
    existing = next((d for d in deps if d['ref'] == root_ref), None)
    if bundled or declared:
        if existing is None:
            deps.append({'ref':root_ref, 'dependsOn':sorted(set(bundled+declared))})
        else:
            existing['dependsOn'] = sorted(set(existing.get('dependsOn', [])) | set(bundled+declared))
    raw['version'] = raw.get('version', 1) + 1
    return raw, {'author_components':authors_changed, 'supplier_components':changed, 'root_contains':sorted(bundled), 'root_declares':sorted(declared),
                 'go_module_relationships':go_graph,
                 'unresolved_components':[c['bom-ref'] for c in raw['components'] if c.get('type') != 'file' and not c.get('supplier') and not c.get('manufacturer') and not c.get('author')]}


def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--raw',required=True,type=Path)
    p.add_argument('--policy',required=True,type=Path)
    p.add_argument('--subject',required=True,type=Path)
    p.add_argument('--output',required=True,type=Path)
    p.add_argument('--receipt',required=True,type=Path)
    p.add_argument('--publisher-dir',type=Path)
    a=p.parse_args()
    raw_bytes=a.raw.read_bytes(); policy_bytes=a.policy.read_bytes()
    subject=a.subject.read_text().strip()
    if len(subject)!=64 or any(x not in '0123456789abcdef' for x in subject):
        raise ValueError('invalid original artifact digest')
    result, receipt=enrich(json.loads(raw_bytes),json.loads(policy_bytes),a.publisher_dir,subject)
    output=(json.dumps(result,indent=2,sort_keys=True)+'\n').encode()
    a.output.write_bytes(output)
    receipt.update({'version':1,'role':'release-distributor','subject_sha256':subject,
                    'raw_sbom_sha256':hashlib.sha256(raw_bytes).hexdigest(),
                    'sbom_sha256':hashlib.sha256(output).hexdigest(),
                    'policy_sha256':hashlib.sha256(policy_bytes).hexdigest()})
    a.receipt.write_text(json.dumps(receipt,indent=2,sort_keys=True)+'\n')

if __name__=='__main__': main()
