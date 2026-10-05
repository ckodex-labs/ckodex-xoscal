#!/usr/bin/env python3
import importlib.util
import unittest
from pathlib import Path
spec=importlib.util.spec_from_file_location('frontend',Path(__file__).resolve().parents[1]/'dagger/frontend_inventory.py')
f=importlib.util.module_from_spec(spec);spec.loader.exec_module(f)

class SourceInventoryTests(unittest.TestCase):
    def test_peer_context_is_not_a_delivered_component(self):
        self.assertEqual(f.classify('../../../node_modules/.pnpm/@vueuse+integrations@13.9.0_axios@1.13.2/node_modules/@vueuse/integrations/index.js'),('@vueuse/integrations','13.9.0'))
    def test_scoped_identity(self):
        self.assertEqual(f.classify('../../../node_modules/.pnpm/@ai-sdk+provider-utils@4.0.5_zod@4.4.3/node_modules/@ai-sdk/provider-utils/dist/index.mjs'),('@ai-sdk/provider-utils','4.0.5'))
    def test_inner_module_must_match_primary_root(self):
        with self.assertRaises(ValueError):f.classify('../../../node_modules/.pnpm/vue@3.5.40/node_modules/axios/index.js')
    def test_unknown_third_party_or_first_party_sources_block(self):
        for path in ('../../node_modules/vue/index.js','../../../unknown/src/example.js','../../src/../escape.js','https://example.com/lib.js'):
            with self.subTest(path=path),self.assertRaises(ValueError):f.classify(path)
    def test_complete_source_content_is_required(self):
        with self.assertRaises(ValueError):f.observed({'version':3,'file':'standalone.js','sources':['../../src/main.ts'],'sourcesContent':[None]})
    def test_nonempty_inventory_and_outer_sources_required(self):
        for sources in (['../../src/main.ts'],['../../node_modules/.pnpm/vue@3.5.40/node_modules/vue/index.js']):
            with self.subTest(sources=sources),self.assertRaises(ValueError):f.observed({'version':3,'file':'standalone.js','sources':sources,'sourcesContent':['x']})
    def test_unknown_map_identity_blocks(self):
        with self.assertRaises(ValueError):f.observed({'version':3,'file':'different.js','sources':[],'sourcesContent':[]})
    def test_duplicate_metadata_blocks(self):
        with self.assertRaises(ValueError):f.load('{"version":"safe","version":"unsafe"}')
    def test_unpinned_publisher_metadata_blocks(self):
        with self.assertRaises(ValueError):f.produce(b'{"name":"@scalar/api-reference","version":"1.25.0"}',b'anything')
    def test_tar_integrity_blocks_before_member_read(self):
        meta=f.encoded({'name':f.PACKAGE,'version':f.VERSION,'dist':{'tarball':f.TAR_URL,'integrity':f.TAR_SRI}})
        with self.assertRaisesRegex(ValueError,'SRI'):f.produce(meta,b'tampered tar')


# Small, byte-real fixture. Pins are patched only in the test module namespace;
# the production CLI exposes no custom pins or test mode.
def make_fixture(root):
    import base64, hashlib, io, json, tarfile
    root=Path(root);root.mkdir(parents=True,exist_ok=True)
    def tar(files):
        output=io.BytesIO()
        with tarfile.open(fileobj=output,mode='w:gz') as archive:
            for name,data in sorted(files.items()):
                member=tarfile.TarInfo(name);member.size=len(data);archive.addfile(member,io.BytesIO(data))
        return output.getvalue()
    author='Scalar (https://github.com/scalar)'
    outer=f.encoded({'name':f.PACKAGE,'version':f.VERSION,'dependencies':{},'repository':{'url':'git+https://github.com/scalar/scalar.git'},'author':author})
    published_js=b'console.log("publisher");\n//# sourceMappingURL=standalone.js.map\n'
    published_map=f.encoded({'version':3,'file':'standalone.js','sources':['../../src/main.ts','../../../node_modules/.pnpm/vue@3.5.43/node_modules/vue/index.js'],'sourcesContent':['outer','library']})
    published_tar=tar({'package/package.json':outer,'package/dist/browser/standalone.js':published_js,'package/dist/browser/standalone.js.map':published_map})
    sri='sha512-'+base64.b64encode(hashlib.sha512(published_tar).digest()).decode()
    root.joinpath('npm-metadata.json').write_bytes(f.encoded({'name':f.PACKAGE,'version':f.VERSION,'dependencies':{},'dist':{'tarball':f.TAR_URL,'integrity':sri}}))
    root.joinpath('npm-package.tgz').write_bytes(published_tar);root.joinpath('package.json').write_bytes(outer)
    entry=b'window.fixture=true;\n';library=b'export const version="3.5.43";\n//# sourceMappingURL=index.js.map\n';css=b'.fixture{color:blue}\n';manifest=f.encoded({'name':'vue','version':'3.5.43'})
    inputs={'entry.js':entry,'node_modules/vue/index.js':library,'node_modules/vue/style.css':css}
    notices=[{'name':f.PACKAGE,'version':f.VERSION,'path':'scalar-LICENSE','sha256':f.sha(b'actual fixture notice'),'source':{'role':'fixture'}}]
    input_tar=tar({**inputs,'node_modules/vue/package.json':manifest,'scalar-LICENSE':b'actual fixture notice'})
    banner="(function(){const s=document.createElement('style');s.id='scalar-style';const n=document.querySelector('meta[property=\"csp-nonce\"]')?.getAttribute('nonce');if(n)s.setAttribute('nonce',n);s.textContent="+json.dumps(css.decode(),ensure_ascii=False,separators=(',',':'))+";document.head.appendChild(s);})();"
    js=(banner+'\nconsole.log("built");\n//# sourceMappingURL=scalar.js.map\n').encode()
    jsmap=f.encoded({'version':3,'sources':['../entry.js','../node_modules/vue/index.js'],'sourcesContent':[entry.decode(),library.decode().replace('//# sourceMappingURL=index.js.map','')]})
    cssmap=f.encoded({'version':3,'sources':['../node_modules/vue/style.css'],'sourcesContent':[css.decode()]})
    lock=f.encoded({'lockfileVersion':3,'packages':{'':{},'node_modules/'+f.PACKAGE:{'version':f.VERSION,'resolved':f.TAR_URL,'integrity':sri},'node_modules/vue':{'version':'3.5.43','resolved':'https://registry.npmjs.org/vue/-/vue-3.5.43.tgz','integrity':'sha512-fixture'}}})
    outputs={'out/scalar.js':{'bytes':len(js),'inputs':{'entry.js':{'bytesInOutput':1},'node_modules/vue/index.js':{'bytesInOutput':1}}},'out/scalar.js.map':{'bytes':len(jsmap)},'out/scalar.css':{'bytes':len(css),'inputs':{'node_modules/vue/style.css':{'bytesInOutput':1}}},'out/scalar.css.map':{'bytes':len(cssmap)}}
    files={'scalar.js':js,'scalar.js.map':jsmap,'scalar.css':css,'scalar.css.map':cssmap,'bundle-inputs.tar.gz':input_tar,'build-package.json':b'{}\n','package-lock.json':lock,'entry.js':entry,'build.mjs':b'fixture builder\n','scalar-LICENSE':b'actual fixture notice','scalar-license-source.json':f.encoded({'role':'fixture'}),'build-toolchain.json':f.encoded({'node':'v22.23.3','npm':'10.9.9','esbuild':'0.25.12','built_at':'2026-10-05T14:38:14.362Z'}),'esbuild-metafile.json':f.encoded({'inputs':{p:{'bytes':len(d)} for p,d in inputs.items()},'outputs':outputs}),'installed-package-manifests.json':f.encoded({'node_modules/vue':{'name':'vue','version':'3.5.43','manifest_sha256':f.sha(manifest),'manifest':f.load(manifest),'inputs':[{'path':p,'sha256':f.sha(d)} for p,d in inputs.items() if p!='entry.js']}}),'bundle-notices.json':f.encoded(notices),'THIRD-PARTY-NOTICES.txt':('\n--- '+f.PACKAGE+'@'+f.VERSION+' (scalar-LICENSE) ---\nactual fixture notice').encode()}
    for name,data in files.items():root.joinpath(name).write_bytes(data)
    for tool in ('npm-ci','build','trivy'):root.joinpath(tool+'.status').write_text('0\n');root.joinpath(tool+'.log').write_text('fixture tool log\n')
    overrides={'TAR_SRI':sri,'JS_SHA':f.sha(published_js),'MAP_SHA':f.sha(published_map),'BUILD_PINS':{name:f.sha(files[name]) for name in f.BUILD_PINS}}
    return overrides

class RebuildTests(unittest.TestCase):
    def setUp(self):
        import tempfile
        from unittest.mock import patch
        self.tmp=tempfile.TemporaryDirectory();self.addCleanup(self.tmp.cleanup);self.root=Path(self.tmp.name)
        self.pins=make_fixture(self.root);self.patch=patch.dict(vars(f),self.pins);self.patch.start();self.addCleanup(self.patch.stop)
    def test_rebuild_replays_real_input_bytes(self):
        outputs=f.rebuilt(self.root)
        for name,data in outputs.items():(self.root/name).write_bytes(data)
        receipt,bom=f.replay(self.root)
        self.assertEqual(receipt['schema_version'],2);self.assertEqual(len(bom['components']),1)
    def test_reviewed_lock_change_blocks(self):
        (self.root/'package-lock.json').write_text('{}')
        with self.assertRaisesRegex(ValueError,'unreviewed'):f.rebuilt(self.root)
    def test_unknown_source_map_input_blocks(self):
        p=self.root/'scalar.js.map';m=f.load(p.read_bytes());m['sources'][0]='../node_modules/unknown/index.js';p.write_bytes(f.encoded(m))
        with self.assertRaises(ValueError):f.rebuilt(self.root)
    def test_empty_emitted_inventory_blocks(self):
        p=self.root/'esbuild-metafile.json';m=f.load(p.read_bytes())
        for output in m['outputs'].values():output['inputs']={}
        p.write_bytes(f.encoded(m))
        with self.assertRaisesRegex(ValueError,'emitted'):f.rebuilt(self.root)
    def test_runtime_version_differs_from_lock_blocks(self):
        p=self.root/'installed-package-manifests.json';m=f.load(p.read_bytes());m['node_modules/vue']['version']='9.9.9';p.write_bytes(f.encoded(m))
        with self.assertRaisesRegex(ValueError,'identity'):f.rebuilt(self.root)
    def test_notice_mutation_blocks(self):
        (self.root/'THIRD-PARTY-NOTICES.txt').write_text('invented attribution')
        with self.assertRaisesRegex(ValueError,'notices'):f.rebuilt(self.root)
    def test_css_mutation_blocks(self):
        (self.root/'scalar.css').write_text('.fixture{color:red}')
        with self.assertRaisesRegex(ValueError,'stylesheet'):f.rebuilt(self.root)
    def test_wrong_toolchain_blocks(self):
        p=self.root/'build-toolchain.json';m=f.load(p.read_bytes());m['node']='v26.10.0';p.write_bytes(f.encoded(m))
        with self.assertRaisesRegex(ValueError,'toolchain'):f.rebuilt(self.root)


class FrontendGateTests(unittest.TestCase):
    def setUp(self):
        RebuildTests.setUp(self)
        from unittest.mock import patch
        import types
        gspec=importlib.util.spec_from_file_location('analysis_gate',Path(__file__).with_name('analysis-gate.py'))
        self.g=importlib.util.module_from_spec(gspec);gspec.loader.exec_module(self.g)
        original_spec=importlib.util.spec_from_file_location
        pins=self.pins
        def patched_spec(name,path):
            result=original_spec(name,path);loader=result.loader
            class FixturePinsLoader:
                def create_module(self,spec):return loader.create_module(spec)
                def exec_module(self,module):
                    loader.exec_module(module)
                    if name=='frontend_inventory':module.__dict__.update(pins)
            result.loader=FixturePinsLoader();return result
        self.loader_patch=patch.object(self.g.importlib.util,'spec_from_file_location',side_effect=patched_spec)
        self.loader_patch.start();self.addCleanup(self.loader_patch.stop)
        for name,data in f.rebuilt(self.root).items():(self.root/name).write_bytes(data)
        self.report={'SchemaVersion':2,'ArtifactType':'cyclonedx','Trivy':{'Version':'0.72.0'},'Results':[{'Target':'fixture browser','Packages':[{'Name':'vue','Version':'3.5.43'}]}]}
        self.write_report()
    def write_report(self):(self.root/'trivy-results.json').write_bytes(f.encoded(self.report))
    def test_frontend_report_passes_with_complete_inventory(self):
        receipt=self.g.frontend_gate(self.root);self.assertEqual(receipt['observed_packages'],1);self.assertEqual(receipt['blocked'],[])
    def test_fixed_critical_blocks_and_all_severities_remain(self):
        self.report['Results'][0]['Vulnerabilities']=[{'VulnerabilityID':'CVE-fixture','Severity':'CRITICAL','FixedVersion':'3.5.44'},{'VulnerabilityID':'GHSA-fixture','Severity':'HIGH','FixedVersion':'3.5.44'}];self.write_report()
        receipt=self.g.frontend_gate(self.root);self.assertEqual(receipt['findings'],2);self.assertEqual(receipt['blocked'],['CVE-fixture'])
    def test_missing_observed_package_blocks(self):
        self.report['Results'][0]['Packages']=[];self.write_report()
        with self.assertRaises(self.g.GateError):self.g.frontend_gate(self.root)
    def test_scanner_failure_never_passes_existing_report(self):
        (self.root/'trivy.status').write_text('1\n')
        with self.assertRaises(self.g.GateError):self.g.frontend_gate(self.root)
    def test_failed_build_never_passes_existing_report(self):
        (self.root/'build.status').write_text('1\n')
        with self.assertRaises(self.g.GateError):self.g.frontend_gate(self.root)
    def test_tampered_browser_bytes_block(self):
        p=self.root/'scalar.js';p.write_bytes(p.read_bytes()+b'evil')
        with self.assertRaises(ValueError):self.g.frontend_gate(self.root)

if __name__=='__main__':unittest.main()
