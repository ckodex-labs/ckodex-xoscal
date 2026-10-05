#!/usr/bin/env python3
"""Real producer-policy boundaries: declarations cannot acquire bundled identity."""
import copy
import json
import hashlib
import tempfile
import sys
import importlib.util
from pathlib import Path
import unittest

spec=importlib.util.spec_from_file_location('enrich',Path(__file__).resolve().parents[1]/'dagger/enrich_sbom.py')
m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
sys.path.insert(0,str(Path(__file__).resolve().parents[1]/"dagger"))
POLICY={'version':1,'role':'release-distributor','entity':{'name':'ckodex-labs','url':['https://github.com/ckodex-labs/ckodex-xoscal']}}

def component(ref,cataloger):
 return {'bom-ref':ref,'type':'library','name':ref,'version':'1.0.0','properties':[{'name':'syft:package:foundBy','value':cataloger},{'name':'syft:location:0:path','value':'observed'}]}

def document(cs):
 return {'bomFormat':'CycloneDX','specVersion':'1.7','version':1,'metadata':{'component':{'bom-ref':'root','type':'file','name':'artifact'}},'components':cs,'dependencies':[]}

class Enrichment(unittest.TestCase):
 def test_binary_supplied_upstream_identity_preserved(self):
  c=component('module','go-module-binary-cataloger');c['manufacturer']={'name':'actual upstream'}
  d,r=m.enrich(document([c]),POLICY)
  self.assertEqual(d['components'][0]['supplier'],POLICY['entity'])
  self.assertEqual(d['components'][0]['manufacturer'],{'name':'actual upstream'})
  self.assertEqual(d['dependencies'],[{'ref':'root','dependsOn':['module']}])
  self.assertEqual(r['unresolved_components'],[])
 def test_dependency_declarations_remain_unknown(self):
  for cataloger in ('java-pom-cataloger','go-module-file-cataloger','python-package-cataloger','javascript-package-cataloger'):
   d,r=m.enrich(document([component('external',cataloger)]),POLICY)
   self.assertNotIn('supplier',d['components'][0]);self.assertEqual(r['unresolved_components'],['external']);self.assertEqual(d['dependencies'],[])
 def test_location_required_for_byte_evidence(self):
  c=component('module','go-module-binary-cataloger');c['properties'].pop()
  d,r=m.enrich(document([c]),POLICY)
  self.assertNotIn('supplier',d['components'][0])
 def test_existing_supplier_preserved(self):
  c=component('module','go-module-binary-cataloger');c['supplier']={'name':'actual existing supplier'}
  d,r=m.enrich(document([c]),POLICY)
  self.assertEqual(d['components'][0]['supplier'],{'name':'actual existing supplier'});self.assertEqual(r['supplier_components'],[])
 def test_dangling_graph_rejected(self):
  d=document([component('module','go-module-binary-cataloger')]);d['dependencies']=[{'ref':'module','dependsOn':['absent']}]
  with self.assertRaises(ValueError):m.enrich(d,POLICY)
 def test_supplier_role_cannot_claim_manufacturer(self):
  p=copy.deepcopy(POLICY);p['role']='manufacturer'
  with self.assertRaises(ValueError):m.enrich(document([component('module','go-module-binary-cataloger')]),p)

 def test_exact_python_requirements_graph(self):
  with tempfile.TemporaryDirectory() as temp:
   p=Path(temp);payload=b'example==1.0.0\n';(p/'requirements.txt').write_bytes(payload);(p/'publishers.json').write_text(json.dumps({'version':1,'records':[],'unresolved':[]}))
   c=component('example','python-package-cataloger');c['purl']='pkg:pypi/example@1.0.0'
   f={'bom-ref':'manifest','name':'/input/unpacked/requirements.txt','type':'file','hashes':[{'alg':'SHA-256','content':hashlib.sha256(payload).hexdigest()}]}
   result,receipt=m.enrich(document([c,f]),POLICY,p)
   self.assertEqual(receipt['root_declares'],['example']);self.assertEqual(result['dependencies'],[{'ref':'root','dependsOn':['example','manifest']}])
   (p/'requirements.txt').write_bytes(b'different==1.0.0\n')
   with self.assertRaises(ValueError):m.enrich(document([c,f]),POLICY,p)
 def test_content_bound_root_and_observed_os_preserve_swid(self):
  c={'bom-ref':'os','type':'operating-system','name':'debian','version':'13','swid':{'tagId':'original-swid'},'properties':[{'name':'syft:distro:id','value':'debian'},{'name':'syft:distro:versionID','value':'13'}]}
  result,receipt=m.enrich(document([c]),POLICY,subject='a'*64)
  self.assertEqual(result['metadata']['component']['version'],'sha256:'+'a'*64)
  self.assertEqual(result['components'][0]['purl'],'pkg:generic/debian@13')
  self.assertEqual(result['components'][0]['swid'],{'tagId':'original-swid'})

if __name__=='__main__':unittest.main()
