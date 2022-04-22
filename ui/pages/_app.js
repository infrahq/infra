import dynamic from 'next/dynamic'
import { useRouter } from 'next/router'
import Head from 'next/head'
import useSWRImmutable from 'swr/immutable'
import { SWRConfig } from 'swr'
import relativeTime from 'dayjs/plugin/relativeTime'
import updateLocale from 'dayjs/plugin/updateLocale'
import dayjs from 'dayjs'

import '../styles/globals.css'
import '../styles/typography.css'
import '../styles/buttons.css'

dayjs.extend(relativeTime)
dayjs.extend(updateLocale)
dayjs.updateLocale('en', {
  relativeTime: {
    future: 'in %s',
    past: '%s',
    s: 'just now',
    m: 'a minute ago',
    mm: '%d minutes ago',
    h: 'an hour ago',
    hh: '%d hours ago',
    d: 'a day ago',
    dd: '%d days ago',
    M: 'a month ago',
    MM: '%d months ago',
    y: 'a year ago',
    yy: '%d years ago'
  }
})

const fetcher = async (resource, init) => {
  const res = await fetch(resource, init)
  const data = await res.json()

  if (!res.ok) {
    throw data
  }

  return data
}

function App ({ Component, pageProps }) {
  const { data: auth, error: authError } = useSWRImmutable('/v1/introspect', fetcher)
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

  console.log(signup?.enabled, auth)

  // redirect to login if required
  if (!signup?.enabled && !auth && !router.asPath.startsWith('/login')) {
    router.replace('/login')
    return null
  }

  // redirect to dashboard
  if (auth?.id && (router.asPath.startsWith('/login') || router.asPath.startsWith('/signup'))) {
    router.replace('/')
  }

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
      <Component {...pageProps} />
    </SWRConfig>
  )
}

// disable server-side rendering for pages
export default dynamic(() => Promise.resolve(App), {
  ssr: false
})
