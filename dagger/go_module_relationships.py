"""Replay declared SDK Go requirements from the exact delivered manifest bytes."""
import hashlib
from pathlib import Path
import re
from urllib.parse import quote


def module_path(value):
    if not isinstance(value, str) or not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9._~/-]*', value) or any(p in ('', '.', '..') for p in value.split('/')):
        raise ValueError('unsupported Go module path')
    return value


def requirement(tokens, indirect):
    if len(tokens) != 2 or not isinstance(tokens[1], str) or not re.fullmatch(r'v[0-9]+\.[0-9]+\.[0-9]+(?:[-+][A-Za-z0-9.-]+)?', tokens[1]):
        raise ValueError('unsupported Go requirement')
    return {'name': module_path(tokens[0]), 'version': tokens[1], 'indirect': indirect}


def parse_manifest(data):
    """Accept the generated SDK subset; unsupported Go syntax requires review."""
    project = None
    directives, requirements, block = set(), {}, False
    for line in data.decode('utf-8').splitlines():
        code, separator, comment = line.partition('//')
        tokens = code.split()
        if not tokens and not separator:
            continue
        if separator and comment.strip() != 'indirect':
            raise ValueError('unsupported Go manifest comment')
        if tokens == [')'] and block and not separator:
            block = False
            continue
        if tokens == ['require', '('] and not block and not separator:
            block = True
            continue
        if block or tokens[:1] == ['require']:
            item = requirement(tokens if block else tokens[1:], bool(separator))
            if item['name'] in requirements:
                raise ValueError('duplicate Go requirement')
            requirements[item['name']] = item
            continue
        if separator or len(tokens) != 2 or tokens[0] not in ('module', 'go', 'toolchain') or tokens[0] in directives:
            raise ValueError('unsupported or duplicate Go directive')
        directives.add(tokens[0])
        if tokens[0] == 'module':
            project = module_path(tokens[1])
        elif not re.fullmatch(('go' if tokens[0] == 'toolchain' else '') + r'[0-9]+\.[0-9]+(?:\.[0-9]+)?', tokens[1]):
            raise ValueError('unsupported Go toolchain version')
    if block or project is None or 'go' not in directives or project in requirements or not requirements:
        raise ValueError('incomplete or contradictory Go manifest')
    return project, [requirements[name] for name in sorted(requirements)]


def source_module(component):
    properties = component.get('properties', [])
    names = [p['name'] for p in properties if p['name'] in ('syft:package:foundBy', 'syft:location:0:path')]
    if len(names) != len(set(names)):
        raise ValueError('duplicate Go inventory property')
    values = {p['name']: p['value'] for p in properties}
    return values.get('syft:package:foundBy') == 'go-module-file-cataloger' and values.get('syft:location:0:path') == '/go.mod'


def exact_component(components, name, version=None):
    matches = [c for c in components if c.get('name') == name and source_module(c)]
    if len(matches) != 1:
        raise ValueError('Go declaration lacks unique manifest inventory')
    component = matches[0]
    actual = component.get('version')
    if version is not None and actual != version:
        raise ValueError('Go requirement version differs from inventory')
    expected = 'pkg:golang/' + name
    if actual not in (None, '', 'UNKNOWN'):
        if not isinstance(actual, str):
            raise ValueError('invalid Go inventory version')
        expected += '@' + quote(actual, safe='')
    if component.get('type') != 'library' or component.get('purl') != expected:
        raise ValueError('Go declaration contradicts package identity')
    return component


def merge_edges(raw, project, required):
    refs = {c['bom-ref'] for c in raw['components']} | {raw['metadata']['component']['bom-ref']}
    seen = set()
    dependencies = raw.setdefault('dependencies', [])
    for edge in dependencies:
        children = edge.get('dependsOn', [])
        if edge['ref'] not in refs or edge['ref'] in seen or not isinstance(children, list) or len(children) != len(set(children)) or any(c not in refs or c == edge['ref'] for c in children):
            raise ValueError('contradictory existing dependency graph')
        seen.add(edge['ref'])
    existing = next((e for e in dependencies if e['ref'] == project), None)
    if existing is None:
        dependencies.append({'ref': project, 'dependsOn': sorted(required)})
    else:
        existing['dependsOn'] = sorted(set(existing.get('dependsOn', [])) | set(required))


def observed_closure(raw, project, cataloged, declared):
    """Keep additional scanner observations only within its retained graph."""
    edges = {e['ref']: e.get('dependsOn', []) for e in raw['dependencies']}
    visited, pending = set(), [project]
    while pending:
        ref = pending.pop()
        if ref not in visited:
            visited.add(ref)
            pending.extend(edges.get(ref, []))
    if any(c['bom-ref'] not in visited for c in cataloged):
        raise ValueError('Go inventory contains modules outside its observed dependency graph')
    extra = []
    for component in cataloged:
        if component['bom-ref'] not in declared | {project}:
            requirement([component.get('name'), component.get('version')], False)
            exact_component(raw['components'], component['name'], component['version'])
            extra.append({'bom_ref': component['bom-ref'], 'name': component['name'], 'version': component['version'], 'role': 'additional module in retained scanner dependency graph; not a manifest declaration or installed-byte claim'})
    return sorted(extra, key=lambda r: r['bom_ref'])


def declared_graph(raw, publisher_dir):
    components = raw['components']
    files = [c for c in components if c.get('type') == 'file' and c.get('name') == '/input/unpacked/go.mod']
    cataloged = [c for c in components if any(p.get('name') == 'syft:package:foundBy' and p.get('value') == 'go-module-file-cataloger' for p in c.get('properties', [])) and c.get('purl', '').startswith('pkg:golang/')]
    if not files and not cataloged:
        return [], None
    if publisher_dir is None or not (publisher_dir / 'go.mod').is_file():
        raise ValueError('Go manifest inventory lacks exact delivered go.mod')
    data = (publisher_dir / 'go.mod').read_bytes()
    digest = hashlib.sha256(data).hexdigest()
    if len(files) != 1 or [h['content'] for h in files[0].get('hashes', []) if h.get('alg') == 'SHA-256'] != [digest]:
        raise ValueError('Go manifest lacks unique exact file hash')
    refs = [c['bom-ref'] for c in components]
    if len(refs) != len(set(refs)) or not all(isinstance(r, str) and r for r in refs):
        raise ValueError('ambiguous Go inventory references')
    name, requirements = parse_manifest(data)
    project = exact_component(components, name)
    records, selected = [], []
    for item in requirements:
        component = exact_component(components, item['name'], item['version'])
        selected.append(component)
        records.append(dict(item, bom_ref=component['bom-ref'], role='declared indirect requirement' if item['indirect'] else 'declared direct requirement'))
    merge_edges(raw, project['bom-ref'], [c['bom-ref'] for c in selected])
    observed = observed_closure(raw, project['bom-ref'], cataloged, {c['bom-ref'] for c in selected})
    for component, record in zip(selected, records):
        component.setdefault('properties', []).extend([{'name': 'xoscal:go:manifest-sha256', 'value': digest}, {'name': 'xoscal:go:relationship-role', 'value': record['role']}])
    return [project['bom-ref']], {'version': 1, 'manifest_sha256': digest, 'file_ref': files[0]['bom-ref'], 'project_ref': project['bom-ref'], 'requirements': records, 'observed_additional_modules': observed, 'role': 'declared requirements and retained scanner relationships; no installed upstream bytes or inferred transitive graph'}
