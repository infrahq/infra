import path from 'path'
import fs from 'fs/promises'
import glob from 'glob'
import Head from 'next/head'
import { API } from '@stoplight/elements'
import '@stoplight/elements/styles.min.css'

import DocsLayout from '../../../../components/docs-layout'

export default function OpenAPIDocs({ version, document }) {
  document.info.version = version

  return (
    <>
      <Head>
        <title>{version} - Infra API Docs</title>
      </Head>
      <div data-theme='dark'>
        <API
          apiDescriptionDocument={document}
          layout='stacked'
          router='static'
        />
      </div>
    </>
  )
}

OpenAPIDocs.layout = page => {
  return <DocsLayout>{page}</DocsLayout>
}

const basepath = path.format({
  root: path.dirname(process.cwd()),
  base: '/docs/reference/api-reference/',
})

export async function getStaticProps({ params }) {
  const filepath = path.format({
    root: basepath,
    name: params.version,
    ext: '.json',
  })

  const contents = await fs.readFile(filepath, 'utf-8')
  const document = JSON.parse(contents)
  return {
    props: {
      version: params.version,
      document: document,
    },
  }
}

function apiVersions() {
  return glob.sync('*.json', { cwd: basepath }).map(f => {
    return {
      version: path.basename(f, '.json'),
      filepath: path.join(basepath, f),
    }
  })
}

export async function getStaticPaths() {
  return {
    paths: apiVersions().map(x => {
      return { params: x }
    }),
    fallback: false,
  }
}
