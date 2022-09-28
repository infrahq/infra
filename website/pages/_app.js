import Head from 'next/head'
import { useRouter } from 'next/router'
import { useEffect } from 'react'
import analytics from '../lib/analytics'

import '../styles/globals.css'

export default function App({ Component, pageProps }) {
  const router = useRouter()

  useEffect(() => {
    analytics?.page(router.asPath)
    router.events.on('routeChangeStart', url => analytics?.page(url))
  }, [])

  const layout = Component.layout || (page => page)

  return (
    <>
      <Head>
        <link
          rel='icon'
          type='image/png'
          sizes='32x32'
          href='/favicon-32x32.png'
        />
        <link
          rel='icon'
          type='image/png'
          sizes='16x16'
          href='/favicon-16x16.png'
        />
        <meta property='og:url' content='https://infrahq.com' />
        <meta property='og:type' content='website' />
        <meta property='og:title' content='Infra' />
        <meta
          property='og:description'
          content='Connect your team to your infrastructure'
        />
        <meta property='og:image' content='/images/og.png' />
      </Head>
      {layout(<Component {...pageProps} />)}
    </>
  )
}
