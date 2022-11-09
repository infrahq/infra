import { useEffect, useState } from 'react'
import { useRouter } from 'next/router'

import LoginLayout from '../../components/layouts/login'

export default function Callback() {
  const router = useRouter()
  const { isReady } = router
  const { code, state } = router.query
  const [error, setError] = useState('')

  useEffect(() => {
    async function finish({ code, redirectURL, kind }) {
      try {
        let res = await fetch('/api/signup', {
          method: 'POST',
          body: JSON.stringify({
            social: {
              kind,
              code,
              redirectURL,
            },
          }),
        })

        // redirect to the new org subdomain
        let created = await jsonBody(res)

        window.location = `${window.location.protocol}//${created?.organization?.domain}`
      } catch (e) {
        setError(e.message)
      }
    }

    const kind = window.localStorage.getItem('providerKind')
    const redirectURL = window.localStorage.getItem('redirectURL')

    if (state === window.localStorage.getItem('state') && code && kind) {
      finish({ kind, code, redirectURL })
    }
    window.localStorage.removeItem('providerKind')
  }, [code, state, router])

  if (!isReady) {
    return null
  }

  return (
    <>
      {error === '' ? (
        // TODO: make this a loader
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
      ) : (
        <p className='my-1 text-xs text-red-500'>{error}</p>
      )}
    </>
  )
}

Callback.layout = page => <LoginLayout>{page}</LoginLayout>
