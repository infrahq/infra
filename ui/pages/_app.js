import dynamic from 'next/dynamic'
import { useRouter } from 'next/router'
import Head from 'next/head'
import { SWRConfig } from 'swr'
import useSWRImmutable from 'swr/immutable'

import '../lib/dayjs'
import '../styles/globals.css'

const fetcher = async (resource, init) => {
  const res = await fetch(resource, {
    ...init,
    headers: {
      "Infra-Version": "0.12.2",
    },
  })
  const data = await res.json()

  if (!res.ok) {
    throw data
  }

  return data
}

function App ({ Component, pageProps }) {
  const { data: auth, error: authError } = useSWRImmutable('/v1/identities/self', fetcher)
  const { data: signup, error: signupError } = useSWRImmutable('/v1/signup', fetcher)
  const router = useRouter()

  const authLoading = !auth && !authError
  const signupLoading = !signup && !signupError

  if (authLoading || signupLoading) {
    return null
  }

  // redirect to signup if required
  if (signup?.enabled && !router.asPath.startsWith('/signup')) {
    router.replace('/signup')
    return null
  }

  // redirect to login if required
  if (!signup?.enabled && !auth && !router.asPath.startsWith('/login')) {
    router.replace('/login')
    return null
  }

  // redirect to dashboard if logged in
  if (auth?.id && (router.asPath.startsWith('/login') || router.asPath.startsWith('/signup'))) {
    router.replace('/')
    return null
  }

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
