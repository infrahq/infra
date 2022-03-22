import { createContext, useState, useEffect } from 'react'
import Router from 'next/router'
import axios from 'axios'
import { useCookies } from 'react-cookie'

const AuthContext = createContext({
  authReady: false,
  cookie: {},
  hasRedirected: false,
  loginError: false,
  providers: [],
  user: null,
  newestProvider: null,
  accessKey: null,
  loginCallback: async (code, providerID, redirectURL) => {},
  login: (selectedIdp) => {},
  logout: async () => {},
  setup: async () => {},
  register: async (key) => {},
  setNewProvider: (provider) => {}
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

  const [newestProvider, setNewestProvider] = useState(null)
  const [accessKey, setAccessKey] = useState(null)
  const [cookie, setCookie, removeCookies] = useCookies(['accessKey'])

  useEffect(() => {
    const source = axios.CancelToken.source()
    axios.get('/v1/setup').then((async (response) => {
      if (response.data.required === true) {
        await Router.push({
          pathname: '/account/welcome'
        }, undefined, { shallow: true })
      } else {
        getProviders();
      }
    }))
    return function () {
      source.cancel('Cancelling in cleanup')
    }
  }, [])

  const getProviders = () => {
    axios.get('/v1/providers')
    .then(async (response) => {
      setProviders(response.data)
      await redirectAccountPage(response.data)
    })
    .catch(() => {
      setLoginError(true)
    })
  }

  const getCurrentUser = async (key) => {
    return await axios.get('/v1/introspect', { headers: { Authorization: `Bearer ${key}` } })
      .then((response) => {
        return response.data
      })
      .catch(() => {
        setAuthReady(false)
        setLoginError(true)
      })
  }

  const setNewProvider = (provider) => {
    setProviders(currentProviders => [...currentProviders, provider])
    setNewestProvider(provider)
  }

  const redirectToDashboard = async (key) => {
    try {
      const currentUser = await getCurrentUser(key)

      if (currentUser) {
        setUser(currentUser)
        setAuthReady(true)

        await Router.push({
          pathname: '/'
        }, undefined, { shallow: true })
      }
    } catch (error) {
      setLoginError(true)
    }
  }

  const loginCallback = async (code, providerID, redirectURL) => {
    setHasRedirected(true)
    axios.post('/v1/login', { oidc: { providerID, code, redirectURL } })
      .then(async (response) => {
        setCookie('accessKey', response.data.accessKey, { path: '/' })
        await redirectToDashboard(response.data.accessKey)
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
    await axios.post('/v1/logout', {}, { headers: { Authorization: `Bearer ${cookie.accessKey}` } })
      .then(async () => {
        setAuthReady(false)
        setHasRedirected(false)
        await redirectAccountPage(providers)
        removeCookies('accessKey', { path: '/' })
      })
  }

  const setup = async () => {
    await axios.post('/v1/setup')
      .then(async (response) => {
        console.log(response.data)
        setAccessKey(response.data.accessKey)
        await Router.push({
          pathname: '/account/setup'
        }, undefined, { shallow: true })
      })
  }

  // TODO: verify access key
  const register = async (key) => {
    setCookie('accessKey', key, { path: '/' })
    await redirectToDashboard(key)
  }

  const context = {
    authReady,
    cookie,
    hasRedirected,
    loginError,
    providers,
    user,
    newestProvider,
    accessKey,
    loginCallback,
    login,
    logout,
    setup,
    register,
    setNewProvider
  }

  return (
    <AuthContext.Provider value={context}>
      {children}
    </AuthContext.Provider>
  )
}

export default AuthContext
