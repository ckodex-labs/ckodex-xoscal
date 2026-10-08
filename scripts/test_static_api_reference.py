#!/usr/bin/env python3
"""Regression proof for complete prebuilt API HTML, not browser fixtures."""
import importlib.util
import json
from html.parser import HTMLParser
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

ROOT=Path(__file__).resolve().parents[1]
spec=importlib.util.spec_from_file_location('static_api',ROOT/'dagger/render_api_reference.py')
renderer=importlib.util.module_from_spec(spec);spec.loader.exec_module(renderer)


class Inventory(HTMLParser):
    def __init__(self):
        super().__init__();self.ids=[];self.operations={};self.links=[];self.mains=0;self.details={};self.disclosure_depth=0;self.fragment_depths={}
    def handle_starttag(self,tag,attrs):
        attrs=dict(attrs)
        if tag=='details':self.disclosure_depth+=1
        if 'id' in attrs:self.ids.append(attrs['id'])
        if attrs.get('id','').startswith(('api-operation-','api-schema-')):self.fragment_depths[attrs['id']]=self.disclosure_depth
        if tag=='main':self.mains+=1
        if tag=='details' and attrs.get('id'):self.details[attrs['id']]=attrs
        if tag=='article' and 'data-api-operation' in attrs:
            self.operations[attrs['data-api-operation']]=(attrs['data-api-method'],attrs['data-api-path'])
        if tag=='a' and attrs.get('href','').startswith('#'):self.links.append(attrs['href'][1:])
    def handle_endtag(self,tag):
        if tag=='details':self.disclosure_depth-=1


class StaticAPIReferenceTests(unittest.TestCase):
    def setUp(self):
        self.spec_bytes=(ROOT/'site/openapi.json').read_bytes()
        self.spec=json.loads(self.spec_bytes)
        self.template=(ROOT/'site/docs.html').read_text()
    def test_actual_checked_in_html_is_exact_producer_output(self):
        self.assertEqual(renderer.render(self.template,self.spec_bytes),self.template)
    def test_all_actual_operation_content_preexists_without_scripts(self):
        inventory=Inventory();inventory.feed(self.template)
        expected={op['operationId']:(method.upper(),path) for path,item in self.spec['paths'].items()
                  for method,op in item.items() if method in renderer.METHODS}
        self.assertEqual(len(expected),96)
        self.assertEqual(inventory.operations,expected)
        self.assertEqual(inventory.mains,1)
        self.assertEqual(len(inventory.ids),len(set(inventory.ids)))
        self.assertFalse(set(inventory.links)-set(inventory.ids))
        for name,schema in self.spec['components']['schemas'].items():
            self.assertIn(renderer.schema_anchor(name),inventory.ids)
            self.assertIn(renderer.escape(json.dumps(schema,sort_keys=True,indent=2)),self.template)
        for path,item in self.spec['paths'].items():
            for method,op in item.items():
                if method in renderer.METHODS:
                    self.assertIn(renderer.escape(json.dumps(op,sort_keys=True,indent=2)),self.template)
    def test_generated_html_rejects_missing_service_or_duplicate_identity(self):
        for defect in ('missing','duplicate'):
            mutated=json.loads(self.spec_bytes)
            if defect=='missing':
                mutated['paths']={path:{method:op for method,op in item.items()
                  if not op.get('operationId','').startswith('oscal.services.v1.TransparencyGraphService.')}
                  for path,item in mutated['paths'].items()}
            else:
                operations=[op for item in mutated['paths'].values() for method,op in item.items() if method in renderer.METHODS]
                operations[1]['operationId']=operations[0]['operationId']
            with self.subTest(defect=defect),self.assertRaises(ValueError):renderer.reference(json.dumps(mutated).encode())
    def test_template_marker_drift_blocks(self):
        for text in (self.template.replace(renderer.START,''),self.template+renderer.START):
            with self.assertRaisesRegex(ValueError,'marker'):renderer.render(text,self.spec_bytes)
    def test_case_variant_operation_anchor_collision_blocks(self):
        mutated=json.loads(self.spec_bytes)
        operations=[op for item in mutated['paths'].values() for method,op in item.items() if method in renderer.METHODS]
        service,name=operations[0]['operationId'].rsplit('.',1)
        operations[1]['operationId']=service+'.'+name.lower()
        with self.assertRaisesRegex(ValueError,'anchor collision'):renderer.reference(json.dumps(mutated).encode())
    def test_plain_is_canonical_default_and_interactive_is_optional(self):
        inventory=Inventory();inventory.feed(self.template)
        self.assertIn('open',inventory.details['plain-api-view'])
        self.assertNotIn('open',inventory.details['interactive-api-view'])
        self.assertNotIn('open',inventory.details['api-model-definitions'])
        self.assertIn('requires JavaScript',self.template)
    def test_reference_does_not_repeat_operation_and_model_indexes(self):
        content=renderer.reference(self.spec_bytes)
        for path,item in self.spec['paths'].items():
            for method,op in item.items():
                if method in renderer.METHODS:
                    self.assertEqual(content.count('<code>'+method.upper()+' '+renderer.escape(path)+'</code>'),1)
        inventory=Inventory();inventory.feed(content)
        self.assertEqual(len(inventory.links),5) # Four services and the models; no second 96/395-name index.
    def test_static_fragment_targets_are_inside_native_disclosure_bodies(self):
        inventory=Inventory();inventory.feed(self.template)
        self.assertEqual(len(inventory.fragment_depths),96+395)
        self.assertTrue(all(depth>=2 for depth in inventory.fragment_depths.values()))
    def test_untrusted_descriptions_and_names_are_escaped(self):
        mutated=json.loads(self.spec_bytes)
        first=next(iter(next(iter(mutated['paths'].values())).values()))
        first['description']='<script>alert("unsafe")</script>'
        output=renderer.reference(json.dumps(mutated).encode())
        self.assertNotIn('<script>',output);self.assertIn('&lt;script&gt;',output)
    def test_cli_check_detects_mutated_missing_prebuilt_operation(self):
        with tempfile.TemporaryDirectory() as directory:
            root=Path(directory);(root/'openapi.json').write_bytes(self.spec_bytes)
            (root/'docs.html').write_text(self.template.replace('data-api-operation=', 'data-missing-operation=',1))
            result=subprocess.run([sys.executable,str(ROOT/'dagger/render_api_reference.py'),'--root',directory,'--check'],capture_output=True,text=True)
            self.assertNotEqual(result.returncode,0)
            self.assertIn('differs from generated',result.stderr)


if __name__=='__main__':unittest.main()
