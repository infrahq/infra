import { useEffect, useState } from 'react'
import { useRouter } from 'next/router'
import Cookies from 'universal-cookie'

import { currentBaseDomain } from './../../lib/login'
import LoginLayout from '../../components/layouts/login'
import Loader from '../../components/loader'

export default function Redirect() {
  const router = useRouter()
  const { isReady } = router
  const { code, state } = router.query
  const [error, setError] = useState('')

  useEffect(() => {
    // if it takes over 3 seconds to get the redirect values, something went wrong
    setTimeout(() => {
      setError(
        'login failed: unable to redirect to finish login, check that you allow cookies'
      )
    }, 3000)

    async function finish({ code, state }) {
      // build the callback URL to finish the login at the org
      const callbackURL =
        window.location.protocol +
        '//' +
        redirect +
        '/login/callback' +
        '?code=' +
        code +
        '&state=' +
        state
      // login redirect is complete so we no longer need this cookie
      cookies.remove('finishLogin', {
        path: '/',
        domain: `.${currentBaseDomain()}`,
      })
      // send the browser to the org specific callback URL to finish login
      router.replace(callbackURL)
    }

    const cookies = new Cookies()
    const redirect = cookies.get('finishLogin') // the org to redirect to is stored in this cookie
    if (code && state && redirect) {
      finish({ code, state, redirect })
    }
  }, [code, state, router])

  if (!isReady) {
    return null
  }

  return (
    <>
      {error ? (
        <p className='my-1 text-xs text-red-500'>{error}</p>
      ) : (
        <Loader className='h-20 w-20' />
      )}
    </>
  )
}

Redirect.layout = page => <LoginLayout>{page}</LoginLayout>
