import dynamic from 'next/dynamic'
import Head from 'next/head'
import { SWRConfig } from 'swr'

import '../lib/fetch'
import '../lib/dayjs'
import '../styles/globals.css'

async function fetcher (resource, init) {
  const res = await fetch(resource, init)
  const data = await res.json()

  if (!res.ok) {
    throw data
  }

  return data
}

function App ({ Component, pageProps }) {
  const layout = Component.layout || (page => page)

  return (
    <SWRConfig value={{
      fetcher: (resource, init) => fetcher(resource, init),
      revalidateOnFocus: false,
      revalidateOnReconnect: false
    }}
    >
      <Head>
        <link rel='icon' type='image/png' sizes='32x32' href='/favicon-32x32.png' />
        <link rel='icon' type='image/png' sizes='16x16' href='/favicon-16x16.png' />
        <title>Infra</title>
      </Head>
      {layout(<Component {...pageProps} />)}
    </SWRConfig>
  )
}

// disable server-side rendering for pages
export default dynamic(() => Promise.resolve(App), {
  ssr: false
})
