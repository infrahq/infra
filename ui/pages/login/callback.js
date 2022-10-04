import { useEffect } from 'react'
import { useRouter } from 'next/router'
import { useSWRConfig } from 'swr'

import { useServerConfig } from '../../lib/serverconfig'
import { saveToVisitedOrgs } from '.'

import LoginLayout from '../../components/layouts/login'

export default function Callback() {
  const { mutate } = useSWRConfig()
  const { baseDomain } = useServerConfig()

  const router = useRouter()
  const { isReady } = router
  const { code, state } = router.query

  useEffect(() => {
    async function login({ providerID, code, redirectURL, next }) {
      const res = await fetch('/api/login', {
        method: 'POST',
        body: JSON.stringify({
          oidc: {
            providerID,
            code,
            redirectURL,
          },
        }),
      })

      if (!res.ok) {
        throw await res.json()
      }

      const data = await res.json()

      router.replace(next ? decodeURIComponent(next) : '/')

      window.localStorage.removeItem('next')
      saveToVisitedOrgs(
        window.location.host,
        baseDomain,
        data?.organizationName
      )
    }

    const providerID = window.localStorage.getItem('providerID')
    const redirectURL = window.localStorage.getItem('redirectURL')
    const next = window.localStorage.getItem('next')

    if (
      state === window.localStorage.getItem('state') &&
      code &&
      providerID &&
      redirectURL &&
      baseDomain
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
    <div className='my-32 flex h-full w-full items-center justify-center'>
      <svg
        xmlns='http://www.w3.org/2000/svg'
        width='200px'
        height='200px'
        viewBox='0 0 100 100'
        preserveAspectRatio='xMidYMid'
        className='h-24 w-24 animate-spin-fast stroke-current text-gray-500'
      >
        <circle
          cx='50'
          cy='50'
          fill='none'
          strokeWidth='1'
          r='24'
          strokeDasharray='113.09733552923255 39.69911184307752'
        ></circle>
      </svg>
    </div>
  )
}

Callback.layout = page => <LoginLayout>{page}</LoginLayout>
