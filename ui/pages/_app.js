import dynamic from 'next/dynamic'
import Head from 'next/head'
import useSWR, { SWRConfig } from 'swr'
import { useRouter } from 'next/router'

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

const swrConfig = {
  fetcher,
  revalidateOnFocus: false,
  revalidateOnReconnect: false
}

function App ({ Component, pageProps }) {
  const { data: signup } = useSWR('/api/signup', swrConfig)
  const router = useRouter()

  if (!signup) {
    return null
  }

  if (signup.enabled && router.pathname !== '/signup') {
    router.replace('/signup')
    return null
  }

  const layout = Component.layout || (page => page)

  return (
    <SWRConfig value={swrConfig}>
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
