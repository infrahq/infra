import { createContext, useState, useEffect } from 'react'
import Router from 'next/router'
import axios from 'axios'
import { useCookies } from 'react-cookie'

export interface User {
  id: string
  name: string
  identityType: string
}

export interface ProviderField {
  id: string
  clientID: string
  created: number
  name: string
  updated: number
  url: string
}

interface AppContextInterface {
  authReady: boolean
  cookie: {}
  hasRedirected: boolean
  loginError: boolean
  providers: ProviderField[]
  user: User | null
  getAccessKey: (code: string, providerID: string, redirectURL: string) => Promise<void>
  login: (selectedIdp: ProviderField) => void
  logout: () => Promise<void>
  register: (key: string) => Promise<void>
}

const AuthContext = createContext<AppContextInterface>({
  authReady: false,
  cookie: {},
  hasRedirected: false,
  loginError: false,
  providers: [],
  user: null,
  getAccessKey: async () => {},
  login: () => {},
  logout: async () => {},
  register: async () => {}
})

// TODO: need to revisit this - when refresh the page, this get call
const redirectAccountPage = async (currentProviders: ProviderField[]): Promise<void> => {
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

export const AuthContextProvider = ({ children }: any): any => {
  const [user, setUser] = useState<User | null>(null)
  const [hasRedirected, setHasRedirected] = useState<boolean>(false)
  const [loginError, setLoginError] = useState<boolean>(false)
  const [authReady, setAuthReady] = useState<boolean>(false)

  const [providers, setProviders] = useState<ProviderField[]>([])
  const [cookie, setCookie, removeCookies] = useCookies(['accessKey'])

  useEffect(() => {
    const source = axios.CancelToken.source()
    axios.get('/v1/providers')
      .then(async (response) => {
        setProviders(response.data)
        await redirectAccountPage(response.data)
      })
      .catch(() => {
        setLoginError(true)
      })
    return function () {
      source.cancel('Cancelling in cleanup')
    }
  }, [])

  const getCurrentUser = async (key: string): Promise<User> => {
    return await axios.get('/v1/introspect', { headers: { Authorization: `Bearer ${key}` } })
      .then((response) => {
        return response.data
      })
      .catch(() => {
        setAuthReady(false)
        setLoginError(true)
      })
  }

  const redirectToDashboard = async (key: string): Promise<void> => {
    try {
      const currentUser = await getCurrentUser(key)

      setUser(currentUser)
      setAuthReady(true)

      await Router.push({
        pathname: '/'
      }, undefined, { shallow: true })
    } catch (error) {
      setLoginError(true)
    }
  }

  const getAccessKey = async (code: string, providerID: string, redirectURL: string): Promise<void> => {
    setHasRedirected(true)
    axios.post('/v1/login', { providerID, code, redirectURL })
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

  const login = (selectedIdp: ProviderField): void => {
    localStorage.setItem('providerId', selectedIdp.id)

    const state = [...Array(10)].map(() => (~~(Math.random() * 36)).toString(36)).join('')
    localStorage.setItem('state', state)

    const infraRedirect = window.location.origin + '/account/callback'
    localStorage.setItem('redirectURL', infraRedirect)

    document.location.href = `https://${selectedIdp.url}/oauth2/v1/authorize?redirect_uri=${infraRedirect}&client_id=${selectedIdp.clientID}&response_type=code&scope=openid+email+groups+offline_access&state=${state}`
  }

  const logout = async (): Promise<void> => {
    await axios.post('/v1/logout', {}, { headers: { Authorization: `Bearer ${cookie.accessKey}` } })
      .then(async () => {
        setAuthReady(false)
        setHasRedirected(false)
        await redirectAccountPage(providers)
        removeCookies('accessKey', { path: '/' })
      })
  }

  const register = async (key: string): Promise<void> => {
    setCookie('accessKey', key, { path: '/' })
    await redirectToDashboard(key)
  }

  const context: AppContextInterface = {
    authReady,
    cookie,
    hasRedirected,
    loginError,
    providers,
    user,
    getAccessKey,
    login,
    logout,
    register
  }

  return (
    <AuthContext.Provider value={context}>
      {children}
    </AuthContext.Provider>
  )
}

export default AuthContext
