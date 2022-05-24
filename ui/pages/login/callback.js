import { useEffect } from 'react'
import { useRouter } from 'next/router'
import { useSWRConfig } from 'swr'

export default function () {
  const { mutate } = useSWRConfig()
  const router = useRouter()

  async function login ({ providerID, code, redirectURL }) {
    await fetch('/api/login', {
      method: 'POST',
      body: JSON.stringify({
        oidc: {
          providerID,
          code,
          redirectURL
        }
      })
    })
    await mutate('/api/users/self')
    router.replace('/')
  }

  useEffect(() => {
    const urlSearchParams = new URLSearchParams(window.location.search)
    const params = Object.fromEntries(urlSearchParams.entries())

    const providerID = window.localStorage.getItem('providerID')
    const redirectURL = window.localStorage.getItem('redirectURL')

    if (!params.code || !providerID || !redirectURL) {
      router.replace('/login')
      return
    }

    if (params.state === window.localStorage.getItem('state')) {
      login({
        providerID,
        code: params.code,
        redirectURL: redirectURL
      })
      window.localStorage.removeItem('providerID')
      window.localStorage.removeItem('state')
      window.localStorage.removeItem('redirectURL')
    }
  }, [])

  return (
    <div className='flex items-center justify-center w-full h-full'>
      <img className='w-20 h-20 animate-spin-fast' src='/spinner.svg' />
    </div>
  )
}
