#!/usr/bin/env python3
"""Retain exact-version upstream publisher fields; never guess from namespaces."""
import argparse
import base64
from concurrent.futures import ThreadPoolExecutor
import hashlib
import io
import json
from pathlib import Path
import re
import urllib.parse
import urllib.request
import xml.etree.ElementTree as ET
import zipfile


def fetch(url, limit=4*1024*1024):
    request=urllib.request.Request(url,headers={'User-Agent':'xoscal-sbom-producer/1'})
    with urllib.request.urlopen(request,timeout=30) as response:
        # Canonical registries are independently selected below, never from input URLs.
        if urllib.parse.urlsplit(response.url).hostname != urllib.parse.urlsplit(url).hostname:
            raise ValueError('cross-host registry redirect')
        data=response.read(limit+1)
    if len(data)>limit:raise ValueError('registry metadata exceeds producer limit')
    return data


def xml(data):
    text=data.decode('utf-8-sig')
    if '<!DOCTYPE' in text.upper() or '<!ENTITY' in text.upper():raise ValueError('unsupported XML entity')
    return ET.fromstring(text)


def field(root, path):
    values=[]
    for item in root.findall(path):
        if item.text and item.text.strip():values.append(item.text.strip())
    return '; '.join(values)


def maven_url(group,artifact,version):
    if not all(re.fullmatch(r'[A-Za-z0-9_.-]+',x) for x in (group,artifact,version)):raise ValueError('invalid Maven identifier')
    return f'https://repo.maven.apache.org/maven2/{group.replace(".","/")}/{artifact}/{version}/{artifact}-{version}.pom'


def maven_fields(chain,fetch_parent=False):
    expected=chain[0]['url']
    for index in range(5):
        if index>=len(chain):
            if not fetch_parent:raise ValueError('missing inherited Maven publisher evidence')
            data=fetch(expected);chain.append({'url':expected,'body':data.decode('utf-8'),'sha256':hashlib.sha256(data).hexdigest()})
        part=chain[index];data=part['body'].encode('utf-8')
        if part['url']!=expected or part['sha256']!=hashlib.sha256(data).hexdigest():raise ValueError('Maven metadata chain mismatch')
        root=xml(data);ns='{http://maven.apache.org/POM/4.0.0}' if root.tag.startswith('{') else ''
        author=field(root,f'{ns}developers/{ns}developer/{ns}name');organization=field(root,f'{ns}organization/{ns}name')
        if author or organization:
            if index!=len(chain)-1:raise ValueError('unused publisher chain metadata')
            return author,({'name':organization} if not author and organization else None)
        group=field(root,f'{ns}parent/{ns}groupId');artifact=field(root,f'{ns}parent/{ns}artifactId');version=field(root,f'{ns}parent/{ns}version')
        if not (group and artifact and version):break
        expected=maven_url(group,artifact,version)
    return '',None


def named_authors(text):
    header='\n'.join(text.splitlines()[:16])
    return '; '.join(sorted(set(re.findall(r'(?im)^\s*(?://|\*|\#)?\s*Copyright(?:\s+\(c\))?(?:\s+\d{4}(?:[-, ]+\d{4})*)?\s+([^\n]*\bauthors\b)',header))))


def go_h1(archive, name, version):
    """Go x/mod dirhash.HashZip/Hash1, with bounded canonical module ZIP input."""
    prefix=name+'@'+version+'/'
    with zipfile.ZipFile(io.BytesIO(archive)) as z:
        entries=z.infolist();names=[i.filename for i in entries]
        if not entries or len(entries)>100000 or len(set(names))!=len(names) or sum(i.file_size for i in entries)>64*1024*1024:raise ValueError('invalid module ZIP inventory')
        outer=hashlib.sha256()
        for filename in sorted(names):
            relative=filename[len(prefix):]
            if not filename.startswith(prefix) or '\n' in filename or '\r' in filename or '\\' in filename or any(x in ('..','.') for x in relative.split('/')):raise ValueError('invalid module ZIP path')
            outer.update((hashlib.sha256(z.read(filename)).hexdigest()+'  '+filename+'\n').encode())
    return 'h1:'+base64.b64encode(outer.digest()).decode()


def verify_go_sum(archive,name,version,go_sum):
    if go_sum is None:raise ValueError('missing byte-backed SDK go.sum')
    actual=go_h1(archive,name,version)
    expected=[line.split()[2] for line in go_sum.decode().splitlines() if len(line.split())==3 and line.split()[:2]==[name,version]]
    if expected!=[actual]:raise ValueError('module ZIP does not match delivered go.sum h1')
    return actual


def swift_pin(resolved,name,version):
    if resolved is None:raise ValueError('missing delivered Package.resolved')
    doc=json.loads(resolved)
    if doc.get('version') not in (2,3):raise ValueError('unsupported Swift lock format')
    pins=[pin for pin in doc['pins'] if pin.get('location','').removesuffix('.git')=='https://'+name and pin.get('state',{}).get('version')==version]
    if len(pins)!=1 or not re.fullmatch(r'[0-9a-f]{40}',pins[0]['state']['revision']):raise ValueError('Swift dependency identity mismatch')
    return pins[0]['state']['revision']


def component_coordinate(component):
    # Syft's POM cataloger can omit `group`; recover it from the actual PURL.
    purl=component.get('purl','')
    m=re.fullmatch(r'pkg:(maven|npm)/(.+)@([^?#]+)(?:\?[^#]*)?(?:#.*)?',purl)
    if m:
        kind,name,version=m.groups();name=urllib.parse.unquote(name)
        if kind=='maven':
            group,artifact=name.rsplit('/',1)
            if component['name']!=artifact or (component.get('group') and component['group']!=group):raise ValueError('component coordinate contradicts PURL')
            return group+':'+artifact
        return name
    return (component.get('group','')+':'+component['name']) if component.get('group') else component['name']

def lookup(component, go_sum=None, resolved=None):
    ref=component['bom-ref'];purl=component.get('purl','')
    match=re.fullmatch(r'pkg:(pypi|maven|npm|nuget|golang|swift)/(.+)@([^?#]+)(?:\?[^#]*)?(?:#.*)?',purl)
    if not match:raise ValueError('no supported exact-version publisher metadata')
    kind,name,version=match.groups();name=urllib.parse.unquote(name);version=urllib.parse.unquote(version)
    quote=lambda x:urllib.parse.quote(x,safe='')
    author='';supplier=None;field_name='';data=b'';url=''
    if kind=='pypi':
        url=f'https://pypi.org/pypi/{quote(name)}/{quote(version)}/json';data=fetch(url);info=json.loads(data)['info']
        if info['version'].lower()!=version.lower():raise ValueError('publisher version mismatch')
        author=info.get('author') or info.get('author_email') or '';field_name='info.author / info.author_email'
    elif kind=='maven':
        group,artifact=name.rsplit('/',1);url=maven_url(group,artifact,version)
        original=fetch(url);chain=[{'url':url,'body':original.decode('utf-8'),'sha256':hashlib.sha256(original).hexdigest()}]
        author,supplier=maven_fields(chain,True);field_name='project or inherited parent publisher fields'
        data=(json.dumps({'maven_pom_chain':chain},sort_keys=True)+'\n').encode()
    elif kind=='npm':
        url=f'https://registry.npmjs.org/{quote(name)}/{quote(version)}';data=fetch(url);info=json.loads(data)
        if info.get('name')!=name or info.get('version')!=version:raise ValueError('publisher package/version mismatch')
        value=info.get('author') or ''
        author=value.get('name','') if isinstance(value,dict) else value
        field_name='author.name / author'
        if not author:
            publisher=info.get('_npmUser',{}).get('name')
            if isinstance(publisher,str) and publisher.strip():
                supplier={'name':'npm publishing account: '+publisher.strip(),'url':[url]}
                field_name='_npmUser.name (exact-version registry publisher; not original author)'
    elif kind=='nuget':
        if not re.fullmatch(r'[A-Za-z0-9_.+-]+',name+version):raise ValueError('invalid NuGet identifier')
        name=name.lower();version=version.lower()
        url=f'https://api.nuget.org/v3-flatcontainer/{name}/{version}/{name}.{version}.nupkg';archive=fetch(url,64*1024*1024)
        with zipfile.ZipFile(io.BytesIO(archive)) as z:
            specs=[x for x in z.namelist() if x.endswith('.nuspec') and '/' not in x]
            if len(specs)!=1 or z.getinfo(specs[0]).file_size>4*1024*1024:raise ValueError('ambiguous NuGet metadata')
            data=z.read(specs[0])
        root=xml(data);ns=root.tag.split('}')[0]+'}' if root.tag.startswith('{') else ''
        author=field(root,f'{ns}metadata/{ns}authors');field_name='package.metadata.authors'
        # Retain the actual nuspec together with archive digest, not reconstructed metadata.
        url += '#'+specs[0]
    elif kind=='swift':
        if not re.fullmatch(r'github.com/[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+',name):raise ValueError('unsupported Swift registry')
        revision=swift_pin(resolved,name,version)
        url='https://raw.githubusercontent.com/'+name[len('github.com/'): ]+'/'+revision+'/Package.swift';data=fetch(url)
        author=named_authors(data.decode());field_name='explicit source Authors at locked Swift commit'
    else:
        if not re.fullmatch(r'[A-Za-z0-9_.~/+!-]+',name+version):raise ValueError('invalid Go module identifier')
        escaped=lambda x:''.join('!'+ch.lower() if ch.isupper() else ch for ch in x)
        url=f'https://proxy.golang.org/{escaped(name)}/@v/{escaped(version)}.zip';archive=fetch(url,64*1024*1024)
        with zipfile.ZipFile(io.BytesIO(archive)) as z:
            prefix=name+'@'+version+'/'
            candidates=[x for x in z.namelist() if x in (prefix+'AUTHORS',prefix+'LICENSE',prefix+'LICENSE.txt',prefix+'COPYRIGHT') or (x.startswith(prefix) and '/' not in x[len(prefix):] and x.endswith('.go') and not x.endswith('_test.go'))]
            for filename in sorted(candidates,key=lambda x:not x.endswith('/AUTHORS')):
                if z.getinfo(filename).file_size>1024*1024:continue
                candidate=z.read(filename);text=candidate.decode('utf-8')
                if filename.endswith('/AUTHORS'):
                    names=[x.strip() for x in text.splitlines() if x.strip() and not x.lstrip().startswith('#')]
                    if names:author='; '.join(names);field_name='original source AUTHORS';data=candidate;url+='#'+filename;break
                found=named_authors(text)
                if found:author=found;field_name='explicit original Authors attribution';data=candidate;url+='#'+filename;break
    if kind=='golang' and not author.strip():
        module_h1=verify_go_sum(archive,name,version,go_sum)
        data=archive;field_name='exact source ZIP distribution; original author remains unknown'
        supplier={'name':'Go module mirror (proxy.golang.org)','url':['https://proxy.golang.org/']}
    if not isinstance(author,str) or (not author.strip() and not supplier):raise ValueError('upstream producer does not supply a known author/supplier field')
    record={'bom_ref':ref,'purl':purl,'source':url,'field':field_name,'raw_sha256':hashlib.sha256(data).hexdigest()}
    if author.strip():record['author']=author.strip()
    if supplier:record['supplier']=supplier
    if kind=='golang' and not author.strip():record.update({'module_h1':module_h1,'go_sum_sha256':hashlib.sha256(go_sum).hexdigest(),'role':'source-mirror; does not identify original author/manufacturer'})
    if kind=='swift':record['package_resolved_sha256']=hashlib.sha256(resolved).hexdigest()
    if kind=='npm' and supplier:record.update({'publisher_account':info['_npmUser'],'role':'registry publication automation/account; does not identify original author/manufacturer or responsible organization'})
    return record,data


def verify_record(component,record,data,go_sum=None,resolved=None):
    if record.get('bom_ref')!=component['bom-ref'] or record.get('purl')!=component.get('purl') or record.get('raw_sha256')!=hashlib.sha256(data).hexdigest():
        raise ValueError('publisher evidence identity/digest mismatch')
    if record.get('source')=='artifact:SDK-PRODUCER.json':
        doc=json.loads(data);coordinate=component_coordinate(component)
        if doc.get('schema_version')!=1 or doc.get('role')!='artifact-producer' or doc.get('entity',{}).get('name')!='ckodex-labs' or doc.get('entity',{}).get('url')!=['https://github.com/ckodex-labs/ckodex-xoscal'] or coordinate.replace('_','-').lower()!=doc.get('package',{}).get('name','').replace('_','-').lower() or (component.get('version') not in (None,'','UNKNOWN') and component.get('version')!=doc['package']['version']) or record.get('version')!=doc['package']['version'] or record.get('supplier')!=doc['entity'] or record.get('author') is not None or record.get('field')!='entity (exact SDK package producer)':raise ValueError('SDK producer identity mismatch')
        return '',doc['entity']
    match=re.fullmatch(r'pkg:(pypi|maven|npm|nuget|golang|swift)/(.+)@([^?#]+)(?:\?[^#]*)?(?:#.*)?',record['purl'])
    if not match:raise ValueError('unsupported publisher identity')
    kind,name,version=match.groups();name=urllib.parse.unquote(name);version=urllib.parse.unquote(version)
    quote=lambda x:urllib.parse.quote(x,safe='');author='';supplier=None;source=record['source'];field_name=record['field']
    if kind=='pypi':
        if source!=f'https://pypi.org/pypi/{quote(name)}/{quote(version)}/json':raise ValueError('unexpected PyPI source')
        info=json.loads(data)['info']
        if info['version'].lower()!=version.lower():raise ValueError('publisher version mismatch')
        author=info.get('author') or info.get('author_email') or ''
        if field_name!='info.author / info.author_email':raise ValueError('unexpected author field')
    elif kind=='npm':
        if source!=f'https://registry.npmjs.org/{quote(name)}/{quote(version)}':raise ValueError('unexpected npm source')
        info=json.loads(data)
        if info.get('name')!=name or info.get('version')!=version:raise ValueError('publisher identity mismatch')
        value=info.get('author') or '';author=value.get('name','') if isinstance(value,dict) else value
        if field_name=='_npmUser.name (exact-version registry publisher; not original author)':
            publisher=info.get('_npmUser',{}).get('name')
            if author or not isinstance(publisher,str) or not publisher.strip():raise ValueError('missing exact-version registry publisher')
            supplier={'name':'npm publishing account: '+publisher.strip(),'url':[source]}
            if record.get('publisher_account')!=info['_npmUser'] or record.get('role')!='registry publication automation/account; does not identify original author/manufacturer or responsible organization':raise ValueError('registry publisher role/details mismatch')
        elif field_name!='author.name / author':raise ValueError('unexpected author field')
    elif kind=='maven':
        group,artifact=name.rsplit('/',1)
        if source!=maven_url(group,artifact,version):raise ValueError('unexpected Maven source')
        chain=json.loads(data)['maven_pom_chain']
        if not chain or chain[0]['url']!=source:raise ValueError('Maven source mismatch')
        author,supplier=maven_fields(chain)
        if field_name!='project or inherited parent publisher fields':raise ValueError('unexpected publisher field')
    elif kind=='nuget':
        package_url=f'https://api.nuget.org/v3-flatcontainer/{name.lower()}/{version.lower()}/{name.lower()}.{version.lower()}.nupkg'
        if not source.startswith(package_url+'#') or '/' in source[len(package_url)+1:] or not source.endswith('.nuspec'):raise ValueError('unexpected NuGet source')
        root=xml(data);ns=root.tag.split('}')[0]+'}' if root.tag.startswith('{') else ''
        if field(root,f'{ns}metadata/{ns}id').lower()!=name.lower() or field(root,f'{ns}metadata/{ns}version').lower()!=version.lower():raise ValueError('NuGet publisher identity mismatch')
        author=field(root,f'{ns}metadata/{ns}authors')
        if field_name!='package.metadata.authors':raise ValueError('unexpected author field')
    elif kind=='swift':
        revision=swift_pin(resolved,name,version)
        expected='https://raw.githubusercontent.com/'+name[len('github.com/'): ]+'/'+revision+'/Package.swift'
        if not re.fullmatch(r'github.com/[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+',name) or source!=expected or field_name!='explicit source Authors at locked Swift commit' or record.get('package_resolved_sha256')!=hashlib.sha256(resolved).hexdigest():raise ValueError('Swift source evidence mismatch')
        author=named_authors(data.decode())
    else:
        escaped=lambda x:''.join('!'+ch.lower() if ch.isupper() else ch for ch in x)
        zip_url=f'https://proxy.golang.org/{escaped(name)}/@v/{escaped(version)}.zip'
        if field_name=='exact source ZIP distribution; original author remains unknown':
            if source!=zip_url or record.get('module_h1')!=verify_go_sum(data,name,version,go_sum) or record.get('go_sum_sha256')!=hashlib.sha256(go_sum).hexdigest() or record.get('role')!='source-mirror; does not identify original author/manufacturer' or record.get('author') is not None:raise ValueError('Go source mirror evidence mismatch')
            supplier={'name':'Go module mirror (proxy.golang.org)','url':['https://proxy.golang.org/']}
            if record.get('supplier')!=supplier:raise ValueError('Go source mirror identity mismatch')
            return '',supplier
        prefix=zip_url+'#'+name+'@'+version+'/'
        if not source.startswith(prefix):raise ValueError('unexpected Go source')
        filename=source[len(prefix):];text=data.decode('utf-8')
        if filename=='AUTHORS' and field_name=='original source AUTHORS':
            author='; '.join(x.strip() for x in text.splitlines() if x.strip() and not x.lstrip().startswith('#'))
        elif (filename in ('LICENSE','LICENSE.txt','COPYRIGHT') or (filename.endswith('.go') and '/' not in filename and not filename.endswith('_test.go'))) and field_name=='explicit original Authors attribution':
            author=named_authors(text)
        else:raise ValueError('unsupported Go publisher field')
    if not isinstance(author,str) or (not author.strip() and not supplier):raise ValueError('no genuine author/supplier')
    if record.get('author')!= (author.strip() or None) or record.get('supplier')!=supplier:raise ValueError('publisher field mismatch')
    return author.strip(),supplier


def collect(sbom, output, producer=None, go_sum_path=None, resolved_path=None, requirements_path=None):
    output.mkdir(parents=True,exist_ok=True)
    from enrich_sbom import BUNDLED
    components=[c for c in sbom['components'] if c.get('type')!='file' and not c.get('supplier') and not c.get('author') and not any(p.get('name')=='syft:package:foundBy' and p.get('value') in BUNDLED for p in c.get('properties',[]))]
    records=[];unknown=[]
    go_sum=go_sum_path.read_bytes() if go_sum_path is not None and go_sum_path.is_file() else None
    resolved=resolved_path.read_bytes() if resolved_path is not None and resolved_path.is_file() else None
    if go_sum is not None:(output/'go.sum').write_bytes(go_sum)
    if resolved is not None:(output/'Package.resolved').write_bytes(resolved)
    if requirements_path is not None and requirements_path.is_file():(output/'requirements.txt').write_bytes(requirements_path.read_bytes())
    producer_bytes=producer.read_bytes() if producer is not None and producer.is_file() else None
    def one(c):
        try:
            if producer_bytes is not None:
                doc=json.loads(producer_bytes)
                if doc.get('schema_version')!=1 or doc.get('role')!='artifact-producer' or doc.get('entity',{}).get('name')!='ckodex-labs' or doc.get('entity',{}).get('url')!=['https://github.com/ckodex-labs/ckodex-xoscal']:raise ValueError('invalid SDK producer identity')
                coordinate=component_coordinate(c)
                if coordinate.replace('_','-').lower()==doc.get('package',{}).get('name','').replace('_','-').lower() and (c.get('version') in (None,'','UNKNOWN') or c.get('version')==doc['package']['version']):
                    record={'bom_ref':c['bom-ref'],'purl':c.get('purl'), 'source':'artifact:SDK-PRODUCER.json','field':'entity (exact SDK package producer)', 'supplier':doc['entity'], 'version':doc['package']['version'], 'raw_sha256':hashlib.sha256(producer_bytes).hexdigest()}
                    return c,(record,producer_bytes),None
            return c,lookup(c,go_sum,resolved),None
        except Exception as error:return c,None,str(error)
    with ThreadPoolExecutor(max_workers=2) as pool:
        for c,result,error in pool.map(one,components):
            if error:unknown.append({'bom_ref':c['bom-ref'],'purl':c.get('purl'),'reason':error});continue
            record,data=result;filename=record['raw_sha256']+'.metadata';(output/filename).write_bytes(data);record['file']=filename;records.append(record)
    ledger={'version':1,'records':records,'unresolved':unknown}
    (output/'publishers.json').write_text(json.dumps(ledger,indent=2,sort_keys=True)+'\n')
    return ledger

if __name__=='__main__':
    p=argparse.ArgumentParser(description=__doc__);p.add_argument('--sbom',required=True,type=Path);p.add_argument('--output',required=True,type=Path);p.add_argument('--producer',type=Path);p.add_argument('--go-sum',type=Path);p.add_argument('--resolved',type=Path);p.add_argument('--requirements',type=Path);a=p.parse_args()
    collect(json.loads(a.sbom.read_text()),a.output,a.producer,a.go_sum,a.resolved,a.requirements)
