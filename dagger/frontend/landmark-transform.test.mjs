import test from 'node:test'
import assert from 'node:assert/strict'
import { createHash } from 'node:crypto'
import { readFileSync } from 'node:fs'
import { applyLandmarkTransform } from './landmark-transform.mjs'
const sha = bytes => createHash('sha256').update(bytes).digest('hex')
const input = Buffer.from('createElementVNode("main", { class: "reference" })\n')
const output = Buffer.from('createElementVNode("section", { class: "reference" })\n')
const recipe = {path: 'fixture.vue.script.js',version: '1.0.0',count: 1,
  find: 'createElementVNode("main", {',replace: 'createElementVNode("section", {',
  input_sha256: sha(input),output_sha256: sha(output)}
test('source adapter produces exact reviewed source bytes', () => {
  assert.deepEqual(applyLandmarkTransform(input, recipe, '1.0.0'), output)
})
test('input, version and expected-output drift fail closed', () => {
  assert.throws(() => applyLandmarkTransform(Buffer.concat([input,Buffer.from('drift')]),recipe,'1.0.0'), /identity drift/)
  assert.throws(() => applyLandmarkTransform(input,recipe,'1.0.1'), /identity drift/)
  assert.throws(() => applyLandmarkTransform(input,{...recipe, output_sha256: '0'.repeat(64)},'1.0.0'), /output identity drift/)
})
test('zero, multiple or unexpected replacement counts fail closed', () => {
  for (const bytes of [Buffer.from('no matching element'),Buffer.concat([input,input])])
    assert.throws(() => applyLandmarkTransform(bytes,{...recipe,input_sha256:sha(bytes)},'1.0.0'), /count drift/)
  assert.throws(() => applyLandmarkTransform(input,{...recipe,count:2},'1.0.0'), /count drift/)
})
test('reviewed recipe covers all six exact reference and API-client source modules', () => {
  const production = JSON.parse(readFileSync(new URL('./landmark-transforms.json', import.meta.url)))
  assert.equal(production.schema_version, 1)
  assert.equal(production.transforms.length, 6)
  assert.deepEqual(production.transforms.map(r => [r.package,r.version,r.count]), [
    ['@scalar/api-reference','1.72.4',1], ['@scalar/api-client','3.21.4',1], ['@scalar/api-reference','1.72.4',1],
    ['@scalar/api-client','3.21.4',1], ['@scalar/api-client','3.21.4',1], ['@scalar/api-client','3.21.4',1]])
  assert.equal(production.transforms[0].replace,'createElementVNode("section", {')
  assert.equal(production.transforms[1].replace,'createElementBlock("div", _hoisted_1, [')
  assert.equal(production.transforms[2].replace,'renderList(__props.entries.filter((entry) => entry.type !== "text"), (entry) => {')
  assert.match(production.transforms[3].replace, /inheritAttrs: false/)
  assert.match(production.transforms[3].replace, /normalizeClass\(\[_ctx\.\$attrs\.class,/)
  assert.equal(production.transforms[4].replace, 'id: void 0,')
  assert.equal(production.transforms[5].replace, 'allowOutsideClick: true,\n\t\t\tescapeDeactivates: false,')
  for (const recipe of production.transforms) {
    assert.match(recipe.path,/^node_modules\/@scalar\/.+\.vue\.script\.js$/)
    assert.match(recipe.input_sha256,/^[a-f0-9]{64}$/)
    assert.match(recipe.output_sha256,/^[a-f0-9]{64}$/)
  }
})
