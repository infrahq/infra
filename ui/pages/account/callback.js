import { useContext, useEffect } from 'react'
import AuthContext from '../../store/AuthContext'

const Callback = () => {
  const { loginCallback } = useContext(AuthContext)

  useEffect(() => {
    const urlSearchParams = new URLSearchParams(window.location.search)
    const params = Object.fromEntries(urlSearchParams.entries())

    if (params.state === window.localStorage.getItem('state')) {
      loginCallback(params.code,
        window.localStorage.getItem('providerId'),
        window.localStorage.getItem('redirectURL')
      )

      window.localStorage.removeItem('providerId')
      window.localStorage.removeItem('state')
      window.localStorage.removeItem('redirectURL')
    }
  }, [])

  return (null)
}

export default Callback
