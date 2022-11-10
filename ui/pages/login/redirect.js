import { useEffect } from 'react'
import { useRouter } from 'next/router'
import Cookies from 'universal-cookie'

import { currentBaseDomain } from '.'
import LoginLayout from '../../components/layouts/login'
import Loader from '../../components/loader'

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
      const org = cookies.get('finishLogin') // the org to redirect to is stored in this cookie
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
    <Loader fullscreen={true}/>
  )
}

Redirect.layout = page => <LoginLayout>{page}</LoginLayout>
