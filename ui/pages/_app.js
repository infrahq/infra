import axios from 'axios'
import Head from 'next/head'
import useSWR, { SWRConfig } from 'swr'
import { useRouter } from 'next/router'

import '../lib/dayjs'
import '../styles/globals.css'

// Add the Infra-Version header to requests
axios.interceptors.request.use(
  config => {
    if (config.url.startsWith('/')) {
      config.headers['Infra-Version'] = '0.13.0'
    }

    return config
  },
  error => {
    return Promise.reject(error)
  }
)

const swrConfig = {
  fetcher: url => axios.get(url).then(res => res.data),
  revalidateOnFocus: false,
  revalidateOnReconnect: false,
}

export default function App({ Component, pageProps }) {
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
        <title>Infra</title>
      </Head>
      {layout(<Component {...pageProps} />)}
    </SWRConfig>
  )
}
