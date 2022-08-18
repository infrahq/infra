import { useEffect } from 'react'
import { useRouter } from 'next/router'
import { useSWRConfig } from 'swr'

export default function Callback() {
  const { mutate } = useSWRConfig()
  const router = useRouter()
  const { isReady } = router
  const { code, state } = router.query

  useEffect(() => {
    async function login({ providerID, code, redirectURL }) {
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
      router.replace('/')
    }

    const providerID = window.localStorage.getItem('providerID')
    const redirectURL = window.localStorage.getItem('redirectURL')

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
      })
      window.localStorage.removeItem('providerID')
      window.localStorage.removeItem('state')
      window.localStorage.removeItem('redirectURL')
    }
  }, [code, state, mutate, router])

  if (!isReady) {
    return null
  }

  if (!state || !code) {
    router.replace('/login')
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
