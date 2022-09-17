import path from 'path'
import fs from 'fs'
import glob from 'glob'
import React from 'react'
import Head from 'next/head'
import Markdoc from '@markdoc/markdoc'
import yaml from 'js-yaml'

import Layout from '../../components/layout'
import DocsLayout from '../../components/docs-layout'
import config from '../../lib/markdoc/config'
import components from '../../lib/markdoc/components'

export default function Docs({ markdoc, title }) {
  return (
    <>
      <Head>
        <title>{`${title} - Infra Documentation`}</title>
        <meta property='og:title' content={title} key='title' />
        <meta property='og:url' content='https://infrahq.com' />
        <meta property='og:description' content='Infra Documentation' />
      </Head>
      {Markdoc.renderers.react(JSON.parse(markdoc), React, { components })}
    </>
  )
}

Docs.layout = page => {
  return (
    <Layout>
      <DocsLayout
        headings={page.props.headings}
        items={page.props.items}
        icon={page.props.icon}
      >
        {page}
      </DocsLayout>
    </Layout>
  )
}

function title(filename) {
  const uppercase = ['ssh', 'sdk', 'api', 'cli', 'faq']

  const capitalize = ['infra', 'kubernetes']

  filename = filename.replace('.md', '')

  return filename.replace(/-/g, ' ').replace(/\w\S*/g, (w, offset) => {
    if (uppercase.indexOf(w) !== -1) {
      return w.toUpperCase()
    }

    if (offset === 0 || capitalize.indexOf(w) !== -1) {
      return w.charAt(0).toUpperCase() + w.slice(1)
    }

    return w
  })
}

const rootDir = path.join(process.cwd(), '..')

function items() {
  // discover categories
  const categories = glob.sync('docs/**/.category', { cwd: rootDir }).map(f => {
    const contents = fs.readFileSync(path.join(rootDir, f), 'utf-8')
    const fm = yaml.load(contents)
    fm.title = fm.title || title(path.basename(path.dirname(f)))

    return {
      ...fm,
      href: `/${path.dirname(f)}`,
      empty: !fs.existsSync(path.join(rootDir, path.dirname(f), 'README.md')),
    }
  })

  // discover individual documents
  const docs = glob
    .sync('docs/**/*.md', { cwd: rootDir })
    .filter(f => !f.endsWith('README.md'))
    .map(f => {
      const contents = fs.readFileSync(path.join(rootDir, f), 'utf-8')
      const ast = Markdoc.parse(contents)
      const frontmatter = ast?.attributes?.frontmatter
        ? yaml.load(ast.attributes.frontmatter)
        : {}
      const filename = path.basename(f)
      if (!frontmatter.title) {
        frontmatter.title = title(filename)
      }

      return { ...frontmatter, href: `/${f.replace('.md', '')}` }
    })

  // build the tree of categories and documents
  function build(href, links = []) {
    const cs = categories.filter(c => path.relative(c.href, href) === '..')
    const ds = docs.filter(d => path.relative(d.href, href) === '..')

    for (const c of cs) {
      c.items = build(c.href, c.links)
    }

    return [...cs, ...ds, ...links.map(e => ({ ...e, link: true }))]
      .sort((a, b) => a?.title?.localeCompare(b?.title))
      .sort((a, b) => {
        if (a.position === undefined) {
          return 1
        }

        if (b.position === undefined) {
          return -1
        }

        return a.position - b.position
      })
  }

  return build('/docs')
}

function allitems() {
  // if `all` was specified, traverse the tree to put every item in a single list
  function traverse(items) {
    let ret = items

    for (const i of items) {
      if (i.items) {
        ret = [...ret, ...traverse(i.items)]
      }
    }

    return ret
  }

  return traverse(items())
}

export function texts(node) {
  if (typeof node === 'string') {
    return [node]
  }

  if (node.attributes.content) {
    return [node.attributes.content]
  }

  if (!node?.children?.length) {
    return []
  }

  return node.children
    .map(c => texts(c))
    .flat()
    .map(t => t.trim())
}

export function nodeid(node) {
  return texts(node)
    .join(' ')
    .replace(/[\s+@]/g, '-')
    .replace(/[^A-Za-z0-9-]/g, '')
    .toLowerCase()
}

function headers(node) {
  const hs = []

  if (node.name === 'Tabs') {
    return hs
  }

  if (node.name === 'Heading' && node.attributes.level > 1) {
    hs.push(node)
  }

  return [...hs, ...(node.children?.map(c => headers(c)) || []).flat()]
}

export function addids(node) {
  const count = {}
  const hs = headers(node)

  for (const h of hs) {
    const id = nodeid(h)
    if (count[id]) {
      h.attributes.id = `${id}-${count[id]}`
    } else {
      h.attributes.id = id
    }

    count[id] = (count[id] || 0) + 1
  }
}

function headings(node) {
  if (!node) {
    return []
  }

  let hs = []

  if (node.name === 'Heading' && node.attributes.level > 1) {
    const title = texts(node).join(' ')

    hs.push({
      ...node.attributes,
      title,
    })
  }

  if (node.children) {
    for (const child of node.children) {
      hs = [...hs, ...headings(child)]
    }
  }

  return hs
}

function isRelative(href) {
  return href?.startsWith('./') || href?.startsWith('../')
}

function checklinks(node) {
  if (node.type === 'link' && isRelative(node.attributes.href)) {
    let filename = node.attributes?.href?.split('#')[0]
    filename = filename.endsWith('.md') ? filename : filename + '.md'
    if (!fs.existsSync(path.join(path.dirname(node.location.file), filename))) {
      throw Error(
        `Broken link in ${node.location.file}: ${node.attributes.href}`
      )
    }
  }

  for (const child of node.children || []) {
    checklinks(child)
  }
}

export async function getStaticProps({ params }) {
  const slug = '/' + ['docs', ...params.slug].join('/')
  const item = allitems().find(i => i.href === slug)
  const filepath = item?.items
    ? path.join(rootDir, `${slug}/README.md`)
    : path.join(rootDir, `${slug}.md`)

  const content = fs.readFileSync(filepath, 'utf-8')
  const ast = Markdoc.parse(content, filepath)

  checklinks(ast)

  const transformed = Markdoc.transform(ast, config)

  addids(transformed)

  const markdoc = JSON.stringify(transformed)

  // add image
  // todo: store me with docs frontmatter
  const base = path.basename(slug)
  let icon = ''
  if (
    fs.existsSync(path.join(process.cwd(), 'public', 'icons', base + '.svg'))
  ) {
    icon = path.join('/', 'icons', `${base}.svg`)
  }

  return {
    props: {
      markdoc,
      items: items(),
      icon,
      title: item.title,
      headings: headings(transformed),
    },
  }
}

export async function getStaticPaths() {
  return {
    paths: allitems()
      .filter(i => !i?.items || !i?.empty)
      .filter(i => !i.link)
      .map(i => i.href),
    fallback: false,
  }
}
