import { useEffect } from 'react'
import { useRouter } from 'next/router'
import { useSWRConfig } from 'swr'

export default function Callback() {
  const { mutate } = useSWRConfig()
  const router = useRouter()

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

    return
  }

  useEffect(() => {
    const urlSearchParams = new URLSearchParams(window.location.search)
    const params = Object.fromEntries(urlSearchParams.entries())

    const providerID = window.localStorage.getItem('providerID')
    const redirectURL = window.localStorage.getItem('redirectURL')

    const next = window.localStorage.getItem('next')

    if (!params.code || !providerID || !redirectURL) {
      next ? router.replace(`/login?next=${next}`) : router.replace('/login')
      return
    }

    if (params.state === window.localStorage.getItem('state')) {
      login({
        providerID,
        code: params.code,
        redirectURL,
        next,
      })
      window.localStorage.removeItem('providerID')
      window.localStorage.removeItem('state')
      window.localStorage.removeItem('redirectURL')
    }
  })

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
