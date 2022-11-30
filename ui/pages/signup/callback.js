import { useEffect, useState } from 'react'
import { useRouter } from 'next/router'

import Loader from '../../components/loader'
import LoginLayout from '../../components/layouts/login'

export default function Callback() {
  const router = useRouter()
  const { isReady } = router
  const { code, state } = router.query
  const [error, setError] = useState('')

  useEffect(() => {
    async function finish({ code, redirectURL }) {
      try {
        let res = await fetch('/api/signup', {
          method: 'POST',
          body: JSON.stringify({
            social: {
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

    const redirectURL = window.localStorage.getItem('redirectURL')

    if (state === window.localStorage.getItem('state') && code && redirectURL) {
      finish({ code, redirectURL })
    }
  }, [code, state, router])

  if (!isReady) {
    return null
  }

  return (
    <>
      {error ? (
        <p className='my-1 text-xs text-red-500'>
          An error occurred while signing up: {error}
        </p>
      ) : (
        <Loader className='h-20 w-20' />
      )}
    </>
  )
}

Callback.layout = page => <LoginLayout>{page}</LoginLayout>
