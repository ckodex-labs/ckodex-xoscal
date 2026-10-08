#!/usr/bin/env python3
"""Replay the exact published Scalar assembly and observed source-map components."""
import argparse
import base64
import hashlib
import io
import json
from pathlib import Path
import re
import posixpath
import tarfile
from urllib.parse import quote

VERSION = '1.72.4'
PACKAGE = '@scalar/api-reference'
TAR_URL = 'https://registry.npmjs.org/@scalar/api-reference/-/api-reference-1.72.4.tgz'
TAR_SRI = 'sha512-HKsUqCJbXhmz/5j6ETceXVgnlZKOX1uidF7jz8elSuzKE6FJjJYhK1G7WqxHk4zLabbSb13QHQ9yKUikf8XLBQ=='
JS_SHA = 'f5ac0c3504a7ca77ca9fa96a6bb7c8f98b42c271864df35583f6ed9e6da40c23'
MAP_SHA = '69db34bf9f7b86eca7ba675399016c441c5c0b0600cd2ed812deeac7bb1c13ac'
# Actual sibling source roots in the verified publisher map. They stay within
# the outer Scalar assembly; these names do not imply guessed package versions.
SCALAR_ROOTS = frozenset(('agent-chat api-client asyncapi-upgrader blocks code-highlight components helpers icons json-magic localization oas-utils openapi-to-markdown openapi-upgrader schemas sidebar snippetz themes types use-codemirror use-hooks use-toasts validation workspace-store').split())


def unique(pairs):
    obj = {}
    for key, value in pairs:
        if key in obj: raise ValueError('duplicate JSON property: ' + key)
        obj[key] = value
    return obj


def load(data):
    return json.loads(data, object_pairs_hook=unique)


def sha(data):
    return hashlib.sha256(data).hexdigest()


def encoded(data):
    return (json.dumps(data, sort_keys=True, indent=2) + '\n').encode()


def purl(name, version):
    return 'pkg:npm/' + quote(name, safe='/') + '@' + quote(version, safe='')


def classify(source):
    if not isinstance(source, str) or not source or '\\' in source or '\x00' in source or '://' in source:
        raise ValueError('unsupported source-map path')
    match = re.fullmatch(r'(?:\.\./)+node_modules/\.pnpm/([^/]+)/node_modules/(.+)', source)
    if match:
        package_id, suffix = match.groups()
        identity = re.fullmatch(r'(@[^@/]+\+[^@/]+|[^@/]+)@(\d+\.\d+\.\d+(?:-[A-Za-z0-9.-]+)?(?:\+[A-Za-z0-9.-]+)?)(?:_.*)?', package_id)
        if not identity: raise ValueError('unversioned or unsupported source package: ' + source)
        name, version = identity.groups(); name = name.replace('+', '/')
        if not re.fullmatch(r'(?:@[A-Za-z0-9_.-]+/)?[A-Za-z0-9_.-]+', name) or not suffix.startswith(name + '/') or any(x in ('.', '..') for x in suffix.split('/')):
            raise ValueError('source package identity differs from pnpm root: ' + source)
        return name, version
    if 'node_modules' in source: raise ValueError('unexplained third-party source: ' + source)
    if source.startswith('../../src/') and not any(x in ('.','..') for x in source[len('../../src/'):].split('/')):
        return None
    match = re.fullmatch(r'\.\./\.\./\.\./([^/]+)/(src|dist)/(.+)', source)
    if match and match.group(1) in SCALAR_ROOTS and not any(x in ('.','..') for x in match.group(3).split('/')):
        return None
    raise ValueError('unexplained source-map source: ' + source)


def observed(source_map):
    if source_map.get('version') != 3 or source_map.get('file') != 'standalone.js' or source_map.get('sourceRoot') not in (None, ''):
        raise ValueError('unexpected standalone source-map identity')
    sources, content = source_map.get('sources'), source_map.get('sourcesContent')
    if not isinstance(sources, list) or not sources or len(sources) > 100000 or not isinstance(content, list) or len(content) != len(sources) or not all(isinstance(c, str) for c in content):
        raise ValueError('incomplete source-map source bytes')
    packages, first_party = {}, []
    for index, source in enumerate(sources):
        identity = classify(source)
        if identity is None:
            first_party.append(index); continue
        packages.setdefault(identity, []).append(index)
    if not packages or not first_party: raise ValueError('empty observed runtime component inventory')
    return packages, first_party


def produce(metadata_bytes, tar_bytes):
    meta = load(metadata_bytes)
    if meta.get('name') != PACKAGE or meta.get('version') != VERSION or meta.get('dist', {}).get('tarball') != TAR_URL or meta.get('dist', {}).get('integrity') != TAR_SRI:
        raise ValueError('unexpected exact-version publishing metadata')
    if len(tar_bytes) > 64*1024*1024 or 'sha512-' + base64.b64encode(hashlib.sha512(tar_bytes).digest()).decode() != TAR_SRI:
        raise ValueError('npm tarball does not match pinned publisher SRI')
    required = {'package/package.json', 'package/dist/browser/standalone.js', 'package/dist/browser/standalone.js.map'}
    files = {}
    with tarfile.open(fileobj=io.BytesIO(tar_bytes), mode='r:gz') as archive:
        members=archive.getmembers(); names=[m.name for m in members]
        if len(members)>100000 or len(set(names))!=len(names):raise ValueError('ambiguous publisher tar inventory')
        for member in members:
            if member.name in required:
                if not member.isfile() or member.size>64*1024*1024:raise ValueError('invalid publisher input file')
                files[member.name]=archive.extractfile(member).read()
    if set(files)!=required:raise ValueError('missing publisher JS/map/manifest')
    manifest=load(files['package/package.json']);js=files['package/dist/browser/standalone.js'];source_map_bytes=files['package/dist/browser/standalone.js.map']
    if manifest.get('name')!=PACKAGE or manifest.get('version')!=VERSION or manifest.get('dependencies')!=meta.get('dependencies') or manifest.get('repository',{}).get('url')!='git+https://github.com/scalar/scalar.git':raise ValueError('publisher root manifest identity mismatch')
    if sha(js)!=JS_SHA or sha(source_map_bytes)!=MAP_SHA or not js.rstrip().endswith(b'//# sourceMappingURL=standalone.js.map'):raise ValueError('published JS/source-map binding mismatch')
    source_map=load(source_map_bytes);packages,first_party=observed(source_map)
    root_ref=purl(PACKAGE,VERSION)
    # Supplier is the actual outer publisher/distributor's own manifest field,
    # not a claim about any upstream constituent's original author/manufacturer.
    supplier={'name':manifest['author'],'url':['https://github.com/scalar/scalar']}
    components=[];inventory=[]
    for (name,version),indices in sorted(packages.items()):
        ref=purl(name,version)
        components.append({'type':'library','name':name,'version':version,'bom-ref':ref,'purl':ref,'supplier':supplier,'properties':[{'name':'xoscal:frontend:role','value':'constituent source observed in publisher browser assembly; upstream authors unspecified'},{'name':'xoscal:frontend:source-indices','value':json.dumps(indices,separators=(',',':'))}]})
        inventory.append({'name':name,'version':version,'purl':ref,'source_indices':indices})
    sbom={'bomFormat':'CycloneDX','specVersion':'1.6','version':1,'metadata':{'tools':[{'vendor':'ckodex-labs','name':'frontend_inventory.py','version':'1'}],'component':{'type':'application','name':PACKAGE,'version':VERSION,'bom-ref':root_ref,'purl':root_ref,'supplier':supplier,'hashes':[{'alg':'SHA-256','content':JS_SHA}]}},'components':components,'dependencies':[{'ref':root_ref,'dependsOn':[c['bom-ref'] for c in components]}]}
    sbom_bytes=encoded(sbom)
    receipt={'schema_version':1,'role':'observed published browser assembly; no complete upstream transitive graph assertion','package':{'name':PACKAGE,'version':VERSION},'tar_url':TAR_URL,'tar_sri':TAR_SRI,'tar_sha256':sha(tar_bytes),'metadata_sha256':sha(metadata_bytes),'manifest_sha256':sha(files['package/package.json']),'js_sha256':JS_SHA,'source_map_sha256':MAP_SHA,'sbom_sha256':sha(sbom_bytes),'source_count':len(source_map['sources']),'first_party_source_indices':first_party,'components':inventory}
    return {'scalar.js':js,'standalone.js.map':source_map_bytes,'package.json':files['package/package.json'],'sbom.cyclonedx.json':sbom_bytes,'frontend-inventory.json':encoded(receipt),'frontend.sha256':(JS_SHA+'  /input/scalar.js\n').encode()}


# Immutable inputs of the repository-owned published-module rebuild. These pins
# are changed only together with an explicitly reviewed dependency lock/builder.
NODE_MAJOR = 22
BUILD_PINS = {'landmark-transform.test.mjs': 'ea1a9698a31ee9b606949ef0dbfc3267b09dbbe8e6065d9e58114da0f2d29c8f', 'landmark-transforms.json': 'c957af1bdf01e8599c8900f02095254e157a21db935b20cd5263081a3f7e24d2', 'landmark-transform.mjs': 'f751e9ce1726d2f6a9bfcebb374a57f8ba8ed5146b41a2a4625e36db76b0acfe', 'build-package.json': '59e31b76b652922756a0a31b6c50ace786dd6e56d6b2a81fe852451e6cea9377', 'package-lock.json': '96f9a32b4e85fe8732da6e75550151e30b68c5b8896d08a746e8b7f3dd29d947', 'entry.js': 'c7199b886c8c6ec93fadbdeb27d819fce80b221bbb87ee0ac6b23b44b17d18b5', 'build.mjs': 'e7007461283f31d27fa52f2d932dc0dab07bcb429e992e5c87f04f26ddb0f898', 'scalar-LICENSE': '380cd0a6ad700e1f821f2a509f0dd9ff835041cee2d43daf5dedc1adb2bcc620', 'scalar-license-source.json': '47fbb8fa80a7ca2e9c6d6597896f91213b32872b79d7cf5849ce3f821d0a4a12'}

def rebuilt(root):
    root=Path(root)
    original=produce((root/'npm-metadata.json').read_bytes(),(root/'npm-package.tgz').read_bytes())
    if (root/'package.json').read_bytes()!=original['package.json']:raise ValueError('rebuilt publisher manifest mismatch')
    for name,pin in BUILD_PINS.items():
        if sha((root/name).read_bytes())!=pin:raise ValueError('unreviewed frontend build input: '+name)
    lock=load((root/'package-lock.json').read_bytes())
    if lock.get('lockfileVersion')!=3:raise ValueError('unsupported frontend lock')
    lock_packages=lock.get('packages',{})
    outer=lock_packages.get('node_modules/'+PACKAGE,{})
    if outer.get('version')!=VERSION or outer.get('resolved')!=TAR_URL or outer.get('integrity')!=TAR_SRI:raise ValueError('lock does not bind exact Scalar publisher tar')
    for name,record in lock_packages.items():
        if name and (not record.get('resolved','').startswith('https://registry.npmjs.org/') or not record.get('integrity','').startswith('sha512-')):raise ValueError('unverified lock registry/integrity')
    toolchain=load((root/'build-toolchain.json').read_bytes())
    if not toolchain.get('node','').startswith('v'+str(NODE_MAJOR)+'.') or toolchain.get('esbuild')!='0.25.12':raise ValueError('unexpected frontend toolchain')
    metafile=load((root/'esbuild-metafile.json').read_bytes());records=load((root/'installed-package-manifests.json').read_bytes())
    inputs=metafile.get('inputs');outputs=metafile.get('outputs')
    if not isinstance(inputs,dict) or not inputs or not isinstance(outputs,dict):raise ValueError('missing observed bundle metafile')
    notices=load((root/'bundle-notices.json').read_bytes())
    if not isinstance(notices,list) or not notices:raise ValueError('missing actual distributed license notices')
    expected_files=set(inputs)|{p+'/package.json' for p in records}|{r['path'] for r in notices}
    raw_inputs=(root/'bundle-inputs.tar.gz').read_bytes();files={}
    if len(raw_inputs)>64*1024*1024:raise ValueError('oversized bundle inputs')
    with tarfile.open(fileobj=io.BytesIO(raw_inputs),mode='r|gz') as archive:
        expanded=0
        for index,member in enumerate(archive):
            expanded+=member.size
            if index>100000 or expanded>256*1024*1024:raise ValueError('oversized expanded bundle inputs')
            if member.name not in expected_files or member.name in files or not member.isfile() or member.size>32*1024*1024:raise ValueError('ambiguous or unsafe bundle input')
            files[member.name]=archive.extractfile(member).read()
    if set(files)!=expected_files:raise ValueError('missing bundle input bytes')
    if files.get('entry.js')!=(root/'entry.js').read_bytes():raise ValueError('bundle entry differs from reviewed entry')
    # Raw npm bytes remain in the input archive. Replay the reviewed source
    # adapter independently before comparing the generated source-map content.
    # This never edits publisher bytes, final minified assets or the served DOM.
    recipe=load((root/'landmark-transforms.json').read_bytes())
    if recipe.get('schema_version')!=1 or not isinstance(recipe.get('transforms'),list):raise ValueError('invalid landmark source recipe')
    adapted={};transform_records=[]
    for change in recipe['transforms']:
        path=change.get('path');package=change.get('package')
        prefix='node_modules/'+str(package)
        if path in adapted or path not in files or not path.startswith(prefix+'/') or change.get('count')!=1:raise ValueError('missing or ambiguous landmark source')
        if lock_packages.get(prefix,{}).get('version')!=change.get('version') or sha(files[path])!=change.get('input_sha256'):raise ValueError('landmark source identity drift')
        text=files[path].decode();before=change.get('find');after=change.get('replace')
        if not isinstance(before,str) or not before or not isinstance(after,str) or text.count(before)!=1:raise ValueError('landmark source replacement count drift')
        output=text.replace(before,after).encode()
        if sha(output)!=change.get('output_sha256'):raise ValueError('landmark transformed source identity drift')
        adapted[path]=output
        transform_records.append({'path':path,'package':package,'version':change['version'],'input_sha256':sha(files[path]),'output_sha256':sha(output),'count':1})
    expected_receipt={'schema_version':1,'recipe_sha256':sha((root/'landmark-transforms.json').read_bytes()),'transforms':sorted(transform_records,key=lambda r:r['path'])}
    if load((root/'landmark-transform-receipt.json').read_bytes())!=expected_receipt:raise ValueError('landmark source adapter receipt differs from replay')
    publisher_prefix='node_modules/'+PACKAGE+'/'
    publisher_inputs={'package/'+path[len(publisher_prefix):]:data for path,data in files.items() if path.startswith(publisher_prefix)}
    with tarfile.open(fileobj=io.BytesIO((root/'npm-package.tgz').read_bytes()),mode='r:gz') as archive:
        matched=set()
        for member in archive.getmembers():
            if member.name in publisher_inputs:
                if not member.isfile() or archive.extractfile(member).read()!=publisher_inputs[member.name]:raise ValueError('Scalar bundled input differs from pinned publisher tar bytes')
                matched.add(member.name)
        if matched!=set(publisher_inputs):raise ValueError('Scalar bundled input absent from pinned publisher tar')
    if set(outputs)!= {'out/scalar.js','out/scalar.js.map','out/scalar.css','out/scalar.css.map'}:raise ValueError('unexpected browser build outputs')
    js=(root/'scalar.js').read_bytes();map_bytes=(root/'scalar.js.map').read_bytes();source_map=load(map_bytes)
    if len(js)!=outputs['out/scalar.js'].get('bytes') or len(map_bytes)!=outputs['out/scalar.js.map'].get('bytes') or not js.rstrip().endswith(b'//# sourceMappingURL=scalar.js.map'):raise ValueError('rebuilt browser output identity mismatch')
    if source_map.get('version')!=3 or not source_map.get('sources') or len(source_map.get('sources',[]))!=len(source_map.get('sourcesContent',[])):raise ValueError('incomplete rebuilt source map')
    for source,content in zip(source_map['sources'],source_map['sourcesContent']):
        if not isinstance(source,str) or not source.startswith('../') or source[3:] not in inputs or not isinstance(content,str):raise ValueError('unexplained final browser source-map source')
        raw=adapted.get(source[3:],files[source[3:]])
        expected=re.sub(rb'^//[#@] sourceMappingURL=.*$',b'',raw,flags=re.MULTILINE) if re.search(r'\.[cm]?js$',source) else raw
        if expected!=content.encode():raise ValueError('source map differs from actual compiled bundle inputs')
    css=(root/'scalar.css').read_bytes();css_map=(root/'scalar.css.map').read_bytes()
    if len(css)!=outputs['out/scalar.css'].get('bytes') or len(css_map)!=outputs['out/scalar.css.map'].get('bytes'):raise ValueError('browser stylesheet output identity mismatch')
    css_string=json.dumps(css.decode(),ensure_ascii=False,separators=(',',':'))
    banner="(function(){const s=document.createElement('style');s.id='scalar-style';const n=document.querySelector('meta[property=\"csp-nonce\"]')?.getAttribute('nonce');if(n)s.setAttribute('nonce',n);s.textContent="+css_string+";document.head.appendChild(s);})();"
    if not js.startswith(banner.encode()):raise ValueError('injected browser stylesheet differs from analyzed stylesheet')
    css_sources=load(css_map)
    if not css_sources.get('sources') or len(css_sources.get('sources',[]))!=len(css_sources.get('sourcesContent',[])):raise ValueError('incomplete stylesheet source map')
    for source,content in zip(css_sources['sources'],css_sources['sourcesContent']):
        if not isinstance(source,str) or not source.startswith('../') or source[3:] not in inputs or not isinstance(content,str) or files[source[3:]]!=content.encode():raise ValueError('stylesheet source map differs from actual inputs')
    observed_components=[];used_inputs=set()
    emitted={path for output in outputs.values() for path,record in output.get('inputs',{}).items() if record.get('bytesInOutput',0)>0}
    if not emitted or not emitted.issubset(inputs):raise ValueError('invalid emitted browser input inventory')
    if not set(adapted).issubset(emitted):raise ValueError('landmark adapter source absent from emitted browser assembly')
    for prefix,record in sorted(records.items()):
        if not re.fullmatch(r'node_modules/(?:@[A-Za-z0-9_.-]+/)?[A-Za-z0-9_.-]+(?:/node_modules/(?:@[A-Za-z0-9_.-]+/)?[A-Za-z0-9_.-]+)*',prefix):raise ValueError('unexplained bundle package root')
        manifest_bytes=files[prefix+'/package.json'];manifest=load(manifest_bytes);locked=lock_packages.get(prefix,{})
        name,version=manifest.get('name'),manifest.get('version')
        if prefix.rsplit('node_modules/',1)[-1]!=name or not version or version!=locked.get('version') or record.get('name')!=name or record.get('version')!=version or sha(manifest_bytes)!=record.get('manifest_sha256') or record.get('manifest')!=manifest:raise ValueError('bundle manifest/lock identity mismatch')
        seen=[]
        for entry in record.get('inputs',[]):
            path=entry.get('path')
            if path not in emitted or not path.startswith(prefix+'/') or path in used_inputs or sha(files[path])!=entry.get('sha256'):raise ValueError('bundle input/manifest mismatch')
            used_inputs.add(path);seen.append(path)
        if not seen:raise ValueError('unobserved package in frontend inventory')
        observed_components.append({'name':name,'version':version,'purl':purl(name,version),'input_paths':sorted(seen),'manifest_sha256':sha(manifest_bytes),'lock_integrity':locked['integrity']})
    if used_inputs|{'entry.js'}!=emitted:raise ValueError('unexplained third-party bundle input')
    notice_text=''
    for notice in notices:
        path=notice.get('path')
        if path not in files or sha(files[path])!=notice.get('sha256'):raise ValueError('unbound runtime license notice')
        if path=='scalar-LICENSE':
            if files[path]!=(root/'scalar-LICENSE').read_bytes() or notice.get('source')!=load((root/'scalar-license-source.json').read_bytes()):raise ValueError('Scalar primary license notice mismatch')
        else:
            prefix=path.rsplit('/',1)[0];manifest_record=records.get(prefix,{})
            if notice.get('name')!=manifest_record.get('name') or notice.get('version')!=manifest_record.get('version') or not re.fullmatch(r'(licen[sc]e|notice|copying)([._-].*)?',path.rsplit('/',1)[-1],flags=re.IGNORECASE):raise ValueError('notice differs from actual observed package')
        notice_text+='\n--- '+notice['name']+'@'+notice['version']+' ('+path+') ---\n'+files[path].decode()
    if (root/'THIRD-PARTY-NOTICES.txt').read_bytes()!=notice_text.encode():raise ValueError('distributed notices differ from actual input notice bytes')
    # This supplier is the repository distributor of the assembled browser bytes,
    # not a fabricated upstream package author/manufacturer.
    supplier={'name':'ckodex-labs','url':['https://github.com/ckodex-labs/ckodex-xoscal']}
    if not re.fullmatch(r'\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z',toolchain.get('built_at','')):raise ValueError('missing actual frontend build timestamp')
    root_ref='pkg:generic/ckodex-labs/xoscal-scalar-browser-assembly@'+VERSION+'?checksum='+quote('sha256:'+sha(js),safe='')
    component_inventory={r['purl']:r for r in observed_components}
    components=[{'type':'library','name':r['name'],'version':r['version'],'purl':r['purl'],'bom-ref':r['purl'],'supplier':supplier,'properties':[{'name':'xoscal:frontend:role','value':'observed input to repository-distributed browser assembly; upstream authors not inferred'}]} for r in component_inventory.values()]
    sbom={'bomFormat':'CycloneDX','specVersion':'1.6','version':1,'metadata':{'timestamp':toolchain['built_at'],'authors':[{'name':'ckodex-labs'}],'tools':[{'vendor':'ckodex-labs','name':'frontend_inventory.py','version':'2'}],'component':{'type':'application','name':'xoscal-scalar-browser-assembly','version':VERSION,'bom-ref':root_ref,'purl':root_ref,'supplier':supplier,'hashes':[{'alg':'SHA-256','content':sha(js)}]}},'components':components,'dependencies':[{'ref':root_ref,'dependsOn':[c['bom-ref'] for c in components]}]}
    sbom_bytes=encoded(sbom)
    receipt={'schema_version':2,'role':'rebuilt published-module browser assembly; observed build inputs, not complete upstream graph','package':{'name':PACKAGE,'version':VERSION},'tar_url':TAR_URL,'tar_sri':TAR_SRI,'tar_sha256':sha((root/'npm-package.tgz').read_bytes()),'metadata_sha256':sha((root/'npm-metadata.json').read_bytes()),'manifest_sha256':sha(original['package.json']),'js_sha256':sha(js),'source_map_sha256':sha(map_bytes),'sbom_sha256':sha(sbom_bytes),'source_count':len(source_map['sources']),'components':observed_components,'build_inputs':{name:sha((root/name).read_bytes()) for name in (*BUILD_PINS,'build-toolchain.json','esbuild-metafile.json','installed-package-manifests.json','bundle-inputs.tar.gz','scalar.css','scalar.css.map','bundle-notices.json','THIRD-PARTY-NOTICES.txt','landmark-transform-receipt.json')}}
    return {'sbom.cyclonedx.json':sbom_bytes,'frontend-inventory.json':encoded(receipt),'frontend.sha256':(sha(js)+'  /input/scalar.js\n').encode()}


def replay(root):
    root=Path(root);outputs=rebuilt(root) if (root/'build-package.json').is_file() else produce((root/'npm-metadata.json').read_bytes(),(root/'npm-package.tgz').read_bytes())
    for name,expected in outputs.items():
        if (root/name).read_bytes()!=expected:raise ValueError('frontend production does not replay: '+name)
    return load(outputs['frontend-inventory.json']),load(outputs['sbom.cyclonedx.json'])


def main():
    parser=argparse.ArgumentParser(description=__doc__);parser.add_argument('--root',required=True,type=Path);parser.add_argument('--replay',action='store_true');parser.add_argument('--build',action='store_true');args=parser.parse_args()
    if args.replay:replay(args.root)
    else:
        for name,data in (rebuilt(args.root) if args.build else produce((args.root/'npm-metadata.json').read_bytes(),(args.root/'npm-package.tgz').read_bytes())).items():(args.root/name).write_bytes(data)

if __name__=='__main__':main()
