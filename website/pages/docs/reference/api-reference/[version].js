import Head from 'next/head'
import { API } from '@stoplight/elements'
import '@stoplight/elements/styles.min.css'

import DocsLayout from '../../../../components/docs-layout'

export default function OpenAPIDocs({ version }) {
  const apiDescriptionUrl = `https://raw.githubusercontent.com/infrahq/infra/${version}/internal/server/testdata/openapi3.json`

  return (
    <>
      <Head>
        <title>{version} - Infra API Docs</title>
      </Head>
      <div data-theme='dark'>
        <API
          apiDescriptionUrl={apiDescriptionUrl}
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

export async function getStaticProps({ params }) {
  return {
    props: {
      version: params.version,
    },
  }
}

export async function getStaticPaths() {
  return {
    paths: [
      { params: { version: 'v0.14.1' } },
      { params: { version: 'v0.14.0' } },
    ],
    fallback: false,
  }
}
