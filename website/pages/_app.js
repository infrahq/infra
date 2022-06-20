import Head from 'next/head'
import { useRouter } from 'next/router'
import Script from 'next/script'
import * as snippet from '@segment/snippet'

import '../styles/globals.css'

function renderSnippet () {
  const opts = {
    apiKey: '2BEJxBl14gBmVcV4HVZCNYeWOdcpAG9Q',
    page: true
  }

  return snippet.min(opts)
}

export default function App ({ Component, pageProps }) {
  const router = useRouter()

  if (typeof window !== 'undefined') {
    router.events.on('routeChangeStart', url => window?.analytics?.page(url))
  }

  const layout = Component.layout || (page => page)

  return (
    <>
      <Head>
        <link rel='icon' type='image/png' sizes='32x32' href='/favicon-32x32.png' />
        <link rel='icon' type='image/png' sizes='16x16' href='/favicon-16x16.png' />
        <meta property='og:url' content='https://infrahq.com' />
        <meta property='og:type' content='website' />
        <meta property='og:title' content='Infra' />
        <meta property='og:description' content='Single sign-on for your infrastructure' />
        <meta property='og:image' content='/images/og.png' />
      </Head>
      {process.env.NODE_ENV !== 'development' && (
        <Script id='segment-script' dangerouslySetInnerHTML={{ __html: renderSnippet() }} />
      )}
      {layout(<Component {...pageProps} />)}
    </>
  )
}
