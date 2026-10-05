#!/usr/bin/env python3
"""Reject publisher substitutions, wrong versions, tampering, and XML entities."""
import hashlib
import base64
import io
import zipfile
import importlib.util
import json
from pathlib import Path
import unittest

spec=importlib.util.spec_from_file_location('publisher',Path(__file__).resolve().parents[1]/'dagger/publisher_metadata.py')
m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)

class PublisherProofs(unittest.TestCase):
 def proof(self, info):
  data=json.dumps({'info':info}).encode();c={'bom-ref':'component','purl':'pkg:pypi/example@1.2.3'}
  r={'bom_ref':'component','purl':c['purl'],'source':'https://pypi.org/pypi/example/1.2.3/json','field':'info.author / info.author_email','author':'Actual Author','file':hashlib.sha256(data).hexdigest()+'.metadata','raw_sha256':hashlib.sha256(data).hexdigest()}
  return c,r,data
 def test_genuine_author_field(self):
  c,r,data=self.proof({'version':'1.2.3','author':'Actual Author'});self.assertEqual(m.verify_record(c,r,data),('Actual Author',None))
 def test_author_substitution_rejected(self):
  c,r,data=self.proof({'version':'1.2.3','author':'Actual Author'});r['author']='invented supplier'
  with self.assertRaises(ValueError):m.verify_record(c,r,data)
 def test_metadata_tamper_rejected(self):
  c,r,data=self.proof({'version':'1.2.3','author':'Actual Author'})
  with self.assertRaises(ValueError):m.verify_record(c,r,data+b' ')
 def test_other_package_url_rejected(self):
  c,r,data=self.proof({'version':'1.2.3','author':'Actual Author'});r['source']='https://pypi.org/pypi/other/1.2.3/json'
  with self.assertRaises(ValueError):m.verify_record(c,r,data)
 def test_actual_response_version_must_match(self):
  c,r,data=self.proof({'version':'2.0.0','author':'Actual Author'})
  with self.assertRaises(ValueError):m.verify_record(c,r,data)
 def test_unknown_author_not_guessed(self):
  c,r,data=self.proof({'version':'1.2.3','author':''});r.pop('author')
  with self.assertRaises(ValueError):m.verify_record(c,r,data)
 def test_xml_entities_rejected(self):
  with self.assertRaises(ValueError):m.xml(b'<!DOCTYPE x [<!ENTITY a "invented">]><project><name>&a;</name></project>')
 def test_maven_parent_must_match_actual_child(self):
  url=m.maven_url('example','library','1.0');parent=m.maven_url('example','parent','1.0')
  bodies=['<project><parent><groupId>example</groupId><artifactId>parent</artifactId><version>1.0</version></parent></project>','<project><developers><developer><name>Actual Author</name></developer></developers></project>']
  chain=[{'url':u,'body':b,'sha256':hashlib.sha256(b.encode()).hexdigest()} for u,b in zip([url,parent],bodies)]
  self.assertEqual(m.maven_fields(chain),('Actual Author',None))
  chain[1]['url']=m.maven_url('other','parent','1.0')
  with self.assertRaises(ValueError):m.maven_fields(chain)

 def test_named_authors_only_original_header(self):
  self.assertEqual(m.named_authors('// Copyright The OpenTelemetry Authors\n'), 'The OpenTelemetry Authors')
  self.assertEqual(m.named_authors('// Copyright (c) 2014 - 2018 Apple Inc. and the project authors'), 'Apple Inc. and the project authors')
  self.assertEqual(m.named_authors('\n'*20+'Copyright 2020 The Go Authors'), '')
 def gozip(self):
  b=io.BytesIO()
  with zipfile.ZipFile(b,'w') as z:z.writestr('example.com/library@v1.0.0/a.go',b'package example\n')
  body=b.getvalue();inner=hashlib.sha256(b'package example\n').hexdigest()
  h1='h1:'+base64.b64encode(hashlib.sha256((inner+'  example.com/library@v1.0.0/a.go\n').encode()).digest()).decode()
  return body,h1
 def test_go_source_mirror_requires_exact_delivered_h1(self):
  body,h1=self.gozip();go_sum=('example.com/library v1.0.0 '+h1+'\n').encode()
  self.assertEqual(m.verify_go_sum(body,'example.com/library','v1.0.0',go_sum),h1)
  with self.assertRaises(ValueError):m.verify_go_sum(body,'example.com/library','v1.0.0',b'example.com/library v1.0.0 h1:wrong\n')
  with self.assertRaises(ValueError):m.verify_go_sum(body,'example.com/library','v1.0.0',None)
 def test_swift_version_and_commit_lock_required(self):
  doc=json.dumps({'version':3,'pins':[{'location':'https://github.com/apple/swift-protobuf.git','state':{'version':'1.38.1','revision':'a'*40}}]}).encode()
  self.assertEqual(m.swift_pin(doc,'github.com/apple/swift-protobuf','1.38.1'),'a'*40)
  with self.assertRaises(ValueError):m.swift_pin(doc,'github.com/apple/swift-protobuf','1.38.0')
  with self.assertRaises(ValueError):m.swift_pin(doc,'github.com/other/swift-protobuf','1.38.1')

 def test_registry_publisher_is_supplier_not_original_author(self):
  data=json.dumps({'name':'@bufbuild/protobuf','version':'2.14.1','_npmUser':{'name':'GitHub Actions'}}).encode()
  c={'bom-ref':'dependency','purl':'pkg:npm/%40bufbuild/protobuf@2.14.1'}
  r={'bom_ref':'dependency','purl':c['purl'],'raw_sha256':hashlib.sha256(data).hexdigest(),'source':'https://registry.npmjs.org/%40bufbuild%2Fprotobuf/2.14.1','field':'_npmUser.name (exact-version registry publisher; not original author)','publisher_account':{'name':'GitHub Actions'},'role':'registry publication automation/account; does not identify original author/manufacturer or responsible organization','supplier':{'name':'npm publishing account: GitHub Actions','url':['https://registry.npmjs.org/%40bufbuild%2Fprotobuf/2.14.1']}}
  author,supplier=m.verify_record(c,r,data);self.assertEqual(author,'');self.assertEqual(supplier,r['supplier'])
  r['supplier']['name']='invented original manufacturer'
  with self.assertRaises(ValueError):m.verify_record(c,r,data)
 def test_maven_missing_group_recovered_only_from_matching_purl(self):
  self.assertEqual(m.component_coordinate({'name':'xoscal-sdk','purl':'pkg:maven/io.ckodex/xoscal-sdk@0.0.0-local'}),'io.ckodex:xoscal-sdk')
  with self.assertRaises(ValueError):m.component_coordinate({'name':'different','purl':'pkg:maven/io.ckodex/xoscal-sdk@0.0.0-local'})

if __name__=='__main__':unittest.main()
