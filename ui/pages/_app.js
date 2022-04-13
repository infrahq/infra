import dynamic from 'next/dynamic'
import { useRouter } from 'next/router'
import useSWRImmutable from 'swr/immutable'
import { SWRConfig } from 'swr'
import relativeTime from 'dayjs/plugin/relativeTime'
import updateLocale from 'dayjs/plugin/updateLocale'
import dayjs from 'dayjs'

import '../styles/globals.css'

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

  // If the status code is not in the range 200-299,
  // we still try to parse and throw it.
  if (!res.ok) {
    const error = new Error('An error occurred while fetching the data.')
    // Attach extra info to the error object.
    error.info = await res.json()
    error.status = res.status
    throw error
  }

  return res.json()
}

function App ({ Component, pageProps }) {
  const { data: auth } = useSWRImmutable('/v1/introspect', fetcher)
  const { data: setup } = useSWRImmutable('/v1/setup', fetcher)
  const router = useRouter()

  console.log(auth, setup)

  if (!auth && !setup) {
    return null
  }

  // redirect to signup if required
  if (setup?.required && !router.asPath.startsWith('/signup')) {
    router.replace('/signup')
    return null
  }

  // redirect to login if required
  if (!setup?.required && !auth && !router.asPath.startsWith('/login') && !router.asPath.startsWith('/signup')) {
    router.replace('/login')
    return null
  }

  // redirect to dashboard
  if (auth?.id && (router.asPath.startsWith('/login') || router.asPath.startsWith('/signup'))) {
    router.replace('/')
  }

  return (
    <SWRConfig value={{ fetcher: (resource, init) => fetcher(resource, init) }}>
      <Component {...pageProps} />
    </SWRConfig>
  )
}

// disable server-side rendering for pages
export default dynamic(() => Promise.resolve(App), {
  ssr: false
})
