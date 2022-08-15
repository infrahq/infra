import path from 'path'
import glob from 'glob'
import fs from 'fs'
import yaml from 'js-yaml'
import Markdoc from '@markdoc/markdoc'

import config from './markdoc/config'

const rootDir = path.join(process.cwd(), '..')

// todo: pagination
export function posts() {
  return glob
    .sync('blog/**/*.md', { cwd: rootDir })
    .map(f => {
      const contents = fs.readFileSync(path.join(rootDir, f), 'utf-8')
      const ast = Markdoc.parse(contents, path.join(rootDir, f))
      const frontmatter = ast?.attributes?.frontmatter
        ? yaml.load(ast.attributes.frontmatter)
        : {}

      // convert date to string since next.js doesn't serialize dates
      frontmatter.date = new Date(frontmatter.date).toISOString()

      const markdoc = JSON.stringify(Markdoc.transform(ast, config))

      return { ...frontmatter, href: `/${f.replace('.md', '')}`, markdoc }
    })
    .sort((a, b) => {
      return b.date?.localeCompare(a.date)
    })
}
