import { useEffect } from 'react'
import { useRouter } from 'next/router'
import Cookies from 'universal-cookie'

import { currentBaseDomain } from '.'
import LoginLayout from '../../components/layouts/login'

export default function Redirect() {
  const router = useRouter()
  const { isReady } = router
  const { code, state } = router.query

  useEffect(() => {
    async function finish({ code, state }) {
      const cookies = new Cookies()
      // start with the base of the login callback URL we will redirect to
      let callbackURL =
        currentBaseDomain() +
        '/login/callback' +
        '?code=' +
        code +
        '&state=' +
        state
      let org = cookies.get('finishLogin') // the org to redirect to is stored in this cookie
      if (org !== undefined && org !== '') {
        // build the callback URL to finish the login at the org
        callbackURL = org + '.' + callbackURL
        // login redirect is complete so we no longer need this cookie
        cookies.remove('finishLogin', {
          path: '/',
          domain: `.${currentBaseDomain()}`,
        })
      }
      callbackURL = window.location.protocol + '//' + callbackURL
      // send the browser to the org specific callback URL to finish login
      router.replace(callbackURL)
    }

    if (code && state) {
      finish({ code, state })
    }
  }, [code, state, router])

  if (!isReady) {
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

Redirect.layout = page => <LoginLayout>{page}</LoginLayout>
