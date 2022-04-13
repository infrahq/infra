import { createContext, useState, useEffect } from 'react'
import Router from 'next/router'
import axios from 'axios'

const AuthContext = createContext({
  authReady: false,
  hasRedirected: false,
  loginError: false,
  providers: [],
  user: null,
  accessKey: null,
  loginCallback: async (code, providerID, redirectURL) => {},
  login: (selectedIdp) => {},
  logout: async () => {},
  setup: async () => {},
  register: async (key) => {},
})

// TODO: need to revisit this - when refresh the page, this get call
const redirectAccountPage = async (currentProviders) => {
  if (currentProviders.length > 0) {
    await Router.push({
      pathname: '/account/login'
    }, undefined, { shallow: true })
  } else {
    await Router.push({
      pathname: '/account/register'
    }, undefined, { shallow: true })
  }
}

export const AuthContextProvider = ({ children }) => {
  const [user, setUser] = useState(null)
  const [hasRedirected, setHasRedirected] = useState(false)
  const [loginError, setLoginError] = useState(false)
  const [authReady, setAuthReady] = useState(false)

  const [providers, setProviders] = useState([])

  const [accessKey, setAccessKey] = useState(null)

  useEffect(() => {
    const source = axios.CancelToken.source()
    axios.get('/v1/setup').then(async (response) => {
      if (response.data.required === true) {
        await Router.push({
          pathname: '/account/welcome'
        }, undefined, { shallow: true })
      } else {
        getProviders()
      }
    })
    return function () {
      source.cancel('Cancelling in cleanup')
    }
  }, [])

  const getProviders = () => {
    axios.get('/v1/providers')
      .then(async (response) => {
        const idpList = response.data.filter((item) => item.name !== 'infra')
        setProviders(idpList)
        await redirectAccountPage(idpList)
      })
      .catch(() => {
        setLoginError(true)
      })
  }

  const redirectToDashboard = async (loginInfor) => {
    setUser({
      id: loginInfor.polymorphicID,
      name: loginInfor.name
    })
    setAuthReady(true)

    await Router.push({
      pathname: '/'
    }, undefined, { shallow: true })
  }

  const loginCallback = async (code, providerID, redirectURL) => {
    setHasRedirected(true)
    axios.post('/v1/login', { oidc: { providerID, code, redirectURL } })
      .then((response) => {
        redirectToDashboard(response.data)
      })
      .catch(async () => {
        setAuthReady(false)
        setLoginError(true)
        await Router.push({
          pathname: '/account/login'
        }, undefined, { shallow: true })
      })
  }

  const login = (selectedIdp) => {
    window.localStorage.setItem('providerId', selectedIdp.id)

    const state = [...Array(10)].map(() => (~~(Math.random() * 36)).toString(36)).join('')
    window.localStorage.setItem('state', state)

    const infraRedirect = window.location.origin + '/account/callback'
    window.localStorage.setItem('redirectURL', infraRedirect)

    document.location.href = `https://${selectedIdp.url}/oauth2/v1/authorize?redirect_uri=${infraRedirect}&client_id=${selectedIdp.clientID}&response_type=code&scope=openid+email+groups+offline_access&state=${state}`
  }

  const logout = async () => {
    await axios.post('/v1/logout')
      .then(async () => {
        setAuthReady(false)
        setHasRedirected(false)
        await redirectAccountPage(providers)
      })
  }

  const setup = async () => {
    await axios.post('/v1/setup')
      .then(async (response) => {
        setAccessKey(response.data.accessKey)
        await Router.push({
          pathname: '/account/setup'
        }, undefined, { shallow: true })
      })
  }

  const register = async (key) => {
    await axios.post('/v1/login', { accessKey: key })
      .then((response) => {
        redirectToDashboard(response.data)
      })
      .catch((error) => {
        setLoginError(error)
        setAuthReady(false)
      })
  }

  const context = {
    authReady,
    hasRedirected,
    loginError,
    providers,
    user,
    accessKey,
    loginCallback,
    login,
    logout,
    setup,
    register,
  }

  return (
    <AuthContext.Provider value={context}>
      {children}
    </AuthContext.Provider>
  )
}

export default AuthContext
