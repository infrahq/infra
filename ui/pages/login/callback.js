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
    await mutate('/v1/users/self')
    router.replace('/')
  }

  useEffect(() => {
    const urlSearchParams = new URLSearchParams(window.location.search)
    const params = Object.fromEntries(urlSearchParams.entries())

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

  return (
    <div className='flex items-center justify-center w-full h-full'>
      <img className='w-40 h-40 animate-spin-fast' src='/spinner.svg' />
    </div>
  )
}
