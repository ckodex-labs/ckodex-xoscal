import { build } from 'esbuild'
import { readFileSync, writeFileSync, mkdirSync, readdirSync, lstatSync } from 'node:fs'
import { createHash } from 'node:crypto'
import { gzipSync } from 'node:zlib'
import { dirname } from 'node:path'
import { applyLandmarkTransform } from './landmark-transform.mjs'
const digest = data => createHash('sha256').update(data).digest('hex')
mkdirSync('out', { recursive: true })
const landmarkRecipe = JSON.parse(readFileSync('landmark-transforms.json'))
if (landmarkRecipe.schema_version !== 1 || landmarkRecipe.transforms.length !== 5)
  throw Error('unexpected reviewed Scalar landmark recipe')
const appliedLandmarks = new Map()
const discardUpstreamMaps = { name: 'reviewed-source-adapters', setup(builder) {
  builder.onLoad({filter: /\.[cm]?js$/}, args => {
    let bytes = readFileSync(args.path)
    for (const recipe of landmarkRecipe.transforms) {
      if (!args.path.endsWith('/' + recipe.path)) continue
      const version = JSON.parse(readFileSync('node_modules/' + recipe.package + '/package.json')).version
      bytes = applyLandmarkTransform(bytes, recipe, version)
      appliedLandmarks.set(recipe.path, { path: recipe.path, package: recipe.package, version,
        input_sha256: recipe.input_sha256, output_sha256: digest(bytes), count: recipe.count })
    }
    return { contents: bytes.toString('utf8').replace(/^\/\/[#@] sourceMappingURL=.*$/gm,''),
      loader: 'js', resolveDir: dirname(args.path) }
  })
} }
const options = {
  entryPoints: ['entry.js'], bundle: true, format: 'iife', platform: 'browser',
  outfile: 'out/scalar.js', sourcemap: 'linked', sourcesContent: true,
  metafile: true, write: false, minify: true, plugins: [discardUpstreamMaps],
  define: { 'process.env.NODE_ENV': '"production"', __VUE_OPTIONS_API__: 'true', __VUE_PROD_DEVTOOLS__: 'false', __VUE_PROD_HYDRATION_MISMATCH_DETAILS__: 'false' },
  loader: { '.svg': 'dataurl', '.woff': 'dataurl', '.woff2': 'dataurl', '.ttf': 'dataurl', '.png': 'dataurl' },
}
const first = await build(options)
const css = first.outputFiles.filter(f => f.path.endsWith('.css')).map(f => f.text).join('\n')
if (!css) throw Error('missing browser stylesheet')
writeFileSync('out/scalar.css',css)
const cssMap = first.outputFiles.find(f=>f.path.endsWith('.css.map'))
if (!cssMap) throw Error('missing stylesheet source map')
writeFileSync('out/scalar.css.map',cssMap.contents)
const banner = `(function(){const s=document.createElement('style');s.id='scalar-style';const n=document.querySelector('meta[property="csp-nonce"]')?.getAttribute('nonce');if(n)s.setAttribute('nonce',n);s.textContent=${JSON.stringify(css)};document.head.appendChild(s);})();`
const result = await build({ ...options, banner: { js: banner } })
if (appliedLandmarks.size !== landmarkRecipe.transforms.length)
  throw Error('Scalar landmark source adapter was not applied to every required module')
writeFileSync('out/landmark-transform-receipt.json', JSON.stringify({schema_version: 1,
  recipe_sha256: digest(readFileSync('landmark-transforms.json')),
  transforms: [...appliedLandmarks.values()].sort((a,b) => a.path.localeCompare(b.path))}, null, 2)+'\n')
for (const file of result.outputFiles) {
  if (file.path.endsWith('scalar.js')) writeFileSync('out/scalar.js', file.contents)
  else if (file.path.endsWith('scalar.js.map')) writeFileSync('out/scalar.js.map', file.contents)
}
writeFileSync('out/esbuild-metafile.json', JSON.stringify(result.metafile, null, 2)+'\n')
const packages = {}
const observedInputs = [...new Set(Object.values(result.metafile.outputs).flatMap(o => Object.entries(o.inputs ?? {}).filter(([p,r]) => r.bytesInOutput > 0).map(([p]) => p)))]
for (const path of observedInputs) {
  if (path === 'entry.js') continue
  const m = path.match(/^((?:.*\/)?node_modules\/(?:@[^/]+\/)?[^/]+)\//)
  if (!m) throw Error('unexplained bundled input: '+path)
  const root = m[1], data = readFileSync(root+'/package.json')
  const pkg = JSON.parse(data)
  const record = packages[root] ??= { name: pkg.name, version: pkg.version, manifest_sha256: digest(data), manifest: pkg, inputs: [] }
  record.inputs.push({ path, sha256: digest(readFileSync(path)) })
}
writeFileSync('out/installed-package-manifests.json', JSON.stringify(packages,null,2)+'\n')
writeFileSync('out/build-toolchain.json', JSON.stringify({ built_at: new Date().toISOString(), node: process.version, npm: JSON.parse(readFileSync('/usr/local/lib/node_modules/npm/package.json')).version, esbuild: JSON.parse(readFileSync('node_modules/esbuild/package.json')).version },null,2)+'\n')

const notices = []
for (const [root,record] of Object.entries(packages).sort(([a],[b])=>a.localeCompare(b))) {
  for (const name of readdirSync(root).sort()) {
    const path=root+'/'+name
    if (/^(licen[sc]e|notice|copying)(?:[._-].*)?$/i.test(name) && lstatSync(path).isFile()) notices.push({name:record.name,version:record.version,path,sha256:digest(readFileSync(path))})
  }
}
notices.push({name:'@scalar/api-reference',version:'1.72.4',path:'scalar-LICENSE',sha256:digest(readFileSync('scalar-LICENSE')),source:JSON.parse(readFileSync('scalar-license-source.json'))})
const noticeText=notices.map(r=>'\n--- '+r.name+'@'+r.version+' ('+r.path+') ---\n'+readFileSync(r.path,'utf8')).join('')
writeFileSync('out/THIRD-PARTY-NOTICES.txt',noticeText)
writeFileSync('out/bundle-notices.json',JSON.stringify(notices,null,2)+'\n')
const inputs = [...new Set([...notices.map(r=>r.path), ...Object.keys(result.metafile.inputs), ...Object.keys(packages).map(p => p+'/package.json')])].sort()
const chunks = []
for (const path of inputs) {
  const bytes = readFileSync(path), header = Buffer.alloc(512)
  let name = path, prefix = ''
  if (Buffer.byteLength(name) > 100) {
    const slash = path.lastIndexOf('/')
    name = path.slice(slash+1); prefix = path.slice(0,slash)
  }
  if (Buffer.byteLength(name)>100 || Buffer.byteLength(prefix)>155) throw Error('tar input path too long: '+path)
  const field = (value, offset, length) => header.write(value,offset,length,'utf8')
  const octal = (value, offset, length) => field(value.toString(8).padStart(length-1,'0')+'\0',offset,length)
  field(name,0,100); octal(0o644,100,8); octal(0,108,8); octal(0,116,8); octal(bytes.length,124,12); octal(0,136,12)
  field('        ',148,8); field('0',156,1); field('ustar\0',257,6); field('00',263,2); field(prefix,345,155)
  const checksum=header.reduce((sum,b)=>sum+b,0)
  field(checksum.toString(8).padStart(6,'0')+'\0 ',148,8)
  chunks.push(header,bytes,Buffer.alloc((512-bytes.length%512)%512))
}
chunks.push(Buffer.alloc(1024))
writeFileSync('out/bundle-inputs.tar.gz',gzipSync(Buffer.concat(chunks),{level:9}))
