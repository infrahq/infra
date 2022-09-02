import { useEffect } from 'react'
import { useRouter } from 'next/router'
import { useSWRConfig } from 'swr'
import Cookies from 'universal-cookie'
import { useServerConfig } from '../../lib/serverconfig'

export default function Callback() {
  const { mutate } = useSWRConfig()
  const { baseDomain } = useServerConfig()

  const router = useRouter()
  const { isReady } = router
  const { code, state } = router.query

  useEffect(() => {
    const cookies = new Cookies()

    async function login({ providerID, code, redirectURL, next }) {
      await fetch('/api/login', {
        method: 'POST',
        body: JSON.stringify({
          oidc: {
            providerID,
            code,
            redirectURL,
          },
        }),
      })

      await mutate('/api/users/self')

      if (next) {
        router.replace(`/${next}`)
      } else {
        router.replace('/')
      }
      window.localStorage.removeItem('next')

      let visitedOrgs = cookies.get('orgs') || []
      if (visitedOrgs.findIndex(x => x.url === window.location.origin) === -1) {
        visitedOrgs.push({
          name: window.location.host.split('.')[0],
          url: window.location.origin,
        })

        cookies.set('orgs', visitedOrgs, {
          path: '/',
          domain: `.${baseDomain}`,
        })
      }
    }

    const providerID = window.localStorage.getItem('providerID')
    const redirectURL = window.localStorage.getItem('redirectURL')
    const next = window.localStorage.getItem('next')

    if (
      state === window.localStorage.getItem('state') &&
      code &&
      providerID &&
      redirectURL
    ) {
      login({
        providerID,
        code,
        redirectURL,
        next,
      })
      window.localStorage.removeItem('providerID')
      window.localStorage.removeItem('state')
      window.localStorage.removeItem('redirectURL')
    }
  }, [code, state, mutate, router, baseDomain])

  if (!isReady) {
    return null
  }

  if (!state || !code) {
    const next = window.localStorage.getItem('next')
    next ? router.replace(`/login?next=${next}`) : router.replace('/login')
    return null
  }

  return (
    <div className='flex h-full w-full items-center justify-center'>
      <img
        alt='loading'
        className='h-20 w-20 animate-spin-fast'
        src='/spinner.svg'
      />
    </div>
  )
}
