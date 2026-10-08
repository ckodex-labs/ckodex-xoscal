import { createHash } from 'node:crypto'
const sha = bytes => createHash('sha256').update(bytes).digest('hex')

// A reviewed source adapter. Its input/output hashes and exact count make
// upstream version or source drift a build failure, never a silent DOM repair.
export function applyLandmarkTransform(bytes, recipe, version) {
  if (version !== recipe.version || sha(bytes) !== recipe.input_sha256)
    throw Error('Scalar landmark source identity drift: ' + recipe.path)
  const text = bytes.toString('utf8')
  if (text.split(recipe.find).length - 1 !== recipe.count || recipe.count !== 1)
    throw Error('Scalar landmark replacement count drift: ' + recipe.path)
  const output = Buffer.from(text.replace(recipe.find, recipe.replace))
  if (sha(output) !== recipe.output_sha256)
    throw Error('Scalar landmark output identity drift: ' + recipe.path)
  return output
}
