import { useEffect } from 'react'
import { useRouter } from 'next/router'
import { useSWRConfig } from 'swr'

import { useUser } from '../../lib/hooks'
import { saveToVisitedOrgs } from '../../lib/login'

import LoginLayout from '../../components/layouts/login'
import Loader from '../../components/loader'

export default function Callback() {
  const { mutate } = useSWRConfig()
  const { login } = useUser()

  const router = useRouter()
  const { isReady } = router
  const { code, state } = router.query

  useEffect(() => {
    async function finish({ providerID, code, redirectURL, next }) {
      const user = await login({
        oidc: {
          providerID,
          code,
          redirectURL,
        },
      })

      router.replace(next ? decodeURIComponent(next) : '/')

      window.localStorage.removeItem('next')
      saveToVisitedOrgs(window.location.host, user?.organizationName)
    }

    const providerID = window.localStorage.getItem('providerID')
    const redirectURL = window.localStorage.getItem('redirectURL')
    const next = window.localStorage.getItem('next')

    if (state === window.localStorage.getItem('state') && code && redirectURL) {
      finish({
        providerID,
        code,
        redirectURL,
        next,
      })
      window.localStorage.removeItem('providerID')
      window.localStorage.removeItem('state')
      window.localStorage.removeItem('redirectURL')
    }
  }, [code, state, mutate, router, login])

  if (!isReady) {
    return null
  }

  if (!state || !code) {
    const next = window.localStorage.getItem('next')
    next ? router.replace(`/login?next=${next}`) : router.replace('/login')
    return null
  }

  return <Loader className='h-20 w-20' />
}

Callback.layout = page => <LoginLayout>{page}</LoginLayout>
