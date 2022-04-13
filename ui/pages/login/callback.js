import { useEffect } from 'react'
import { useRouter } from 'next/router'
import { useSWRConfig } from 'swr'

export default function () {
  const { mutate } = useSWRConfig()
  const router = useRouter()

  async function login ({ providerID, code, redirectURL }) {
    await fetch('/v1/login', {
      method: 'POST',
      body: JSON.stringify({
        oidc: {
          providerID,
          code,
          redirectURL
        }
      })
    })
    await mutate('/v1/introspect')
    router.replace('/')
  }

  useEffect(() => {
    const urlSearchParams = new URLSearchParams(window.location.search)
    const params = Object.fromEntries(urlSearchParams.entries())
    console.log(params)
    if (params.state === window.localStorage.getItem('state')) {
      login({
        providerID: window.localStorage.getItem('providerId'),
        code: params.code,
        redirectURL: window.localStorage.getItem('redirectURL')
      })
      window.localStorage.removeItem('providerId')
      window.localStorage.removeItem('state')
      window.localStorage.removeItem('redirectURL')
    }
  }, [])

  return null
}
