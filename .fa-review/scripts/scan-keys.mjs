// Lists en.json keys used in web/src that are missing from fa.json, grouped by
// folder. Parses every source file with @babel/parser and collects string
// literals, template literals without substitutions and JSX text (so keys
// passed through variables, option lists and static-keys.ts are included).
// Test files are skipped.
//
// Usage (from web/): node ../.fa-review/scripts/scan-keys.mjs [--json out.json]
//   [--folder features/usage-logs] [--all]
// --all lists every used key, not only the ones missing from fa.json.
import fs from 'node:fs/promises'
import { createRequire } from 'node:module'
import path from 'node:path'

const require = createRequire(path.resolve('package.json'))
const { parse } = require('@babel/parser')

const args = process.argv.slice(2)
const option = (name) => {
  const i = args.indexOf(name)
  return i === -1 ? undefined : args[i + 1]
}
const jsonOut = option('--json')
const folderFilter = option('--folder')
const listAll = args.includes('--all')

const en = JSON.parse(
  await fs.readFile('src/i18n/locales/en.json', 'utf8')
).translation
const fa = JSON.parse(
  await fs.readFile('src/i18n/locales/fa.json', 'utf8')
).translation

function folderOf(rel) {
  const parts = rel.split(path.sep)
  if (parts[0] === 'features' && parts.length > 2) return `features/${parts[1]}`
  return parts.length > 1 ? parts[0] : '(root)'
}

function walk(node, visit) {
  if (!node || typeof node.type !== 'string') return
  visit(node)
  for (const [field, child] of Object.entries(node)) {
    if (field === 'loc' || field === 'start' || field === 'end') continue
    if (field === 'leadingComments' || field === 'trailingComments') continue
    const children = Array.isArray(child) ? child : [child]
    for (const item of children) {
      if (item && typeof item.type === 'string') walk(item, visit)
    }
  }
}

const entries = await fs.readdir('src', { recursive: true, withFileTypes: true })
const files = entries
  .filter(
    (e) =>
      e.isFile() &&
      /\.(ts|tsx|js|jsx|mjs)$/.test(e.name) &&
      !/\.(test|spec)\./.test(e.name) &&
      !e.parentPath.split(path.sep).includes('__tests__') &&
      !e.parentPath.includes(path.join('i18n', 'locales'))
  )
  .map((e) => path.join(e.parentPath, e.name))

// folder -> key -> Set of "file:line"
const byFolder = new Map()
for (const file of files) {
  const rel = path.relative('src', file)
  const folder = folderOf(rel)
  if (folderFilter && folder !== folderFilter) continue
  const source = await fs.readFile(file, 'utf8')
  let ast
  try {
    ast = parse(source, {
      sourceType: 'module',
      plugins: ['jsx', 'typescript'],
      errorRecovery: true,
    })
  } catch (err) {
    console.error(`parse error ${rel}: ${err.message}`)
    continue
  }
  walk(ast.program, (node) => {
    let value
    if (node.type === 'StringLiteral') value = node.value
    else if (node.type === 'TemplateLiteral' && node.expressions.length === 0)
      value = node.quasis[0].value.cooked
    else if (node.type === 'JSXText') value = node.value.trim()
    if (!value || !Object.hasOwn(en, value)) return
    if (!listAll && Object.hasOwn(fa, value)) return
    if (!byFolder.has(folder)) byFolder.set(folder, new Map())
    const keys = byFolder.get(folder)
    if (!keys.has(value)) keys.set(value, new Set())
    keys.get(value).add(`${rel}:${node.loc.start.line}`)
  })
}

const result = {}
for (const folder of [...byFolder.keys()].sort()) {
  const keys = byFolder.get(folder)
  result[folder] = Object.fromEntries(
    [...keys.entries()]
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([key, where]) => [key, [...where]])
  )
}
if (jsonOut) await fs.writeFile(jsonOut, JSON.stringify(result, null, 2))
const counts = Object.entries(result)
  .map(([folder, keys]) => [folder, Object.keys(keys).length])
  .sort((a, b) => b[1] - a[1])
const unique = new Set(Object.values(result).flatMap((k) => Object.keys(k)))
for (const [folder, count] of counts) console.log(`${count}\t${folder}`)
console.log(`${unique.size}\tunique keys (${listAll ? 'used' : 'missing from fa.json'})`)
