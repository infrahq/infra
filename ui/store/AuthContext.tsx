import { createContext, useState, useEffect } from 'react'
import Router from 'next/router'; 
import axios from 'axios';
import { useCookies } from 'react-cookie';

interface AppContextInterface {
  authReady: boolean,
  cookie: {},
  loginError: boolean,
  providers: ProviderField[]
  user: any,
  getAccessKey: (code: string, providerID: string, redirectURL: string) => void,
  login: (selectedIdp: ProviderField) => void,
  logout: () => void,
  register: (key:string) => void,
}

export interface ProviderField {
  id: string,
  clientID: string,
  created: number,
  name: string,
  updated: number,
  url: string,
};

const AuthContext = createContext<AppContextInterface>({
  authReady: false,
  cookie: {},
  loginError: false,
  providers: [],
  user: null,
  getAccessKey: () => {},
  login: () => {},
  logout: () => {},
  register: () => {},
})

const redirectAccountPage = (currentProviders : ProviderField[]) => {
  if (currentProviders.length > 0) {
    Router.push({
      pathname: '/account/login',
    }, undefined, { shallow: true });
  } else {
    Router.push({
      pathname: '/account/register',
    }, undefined, { shallow: true });
  }
}

export const AuthContextProvider = ({ children }:any) => { 
  const [user, setUser] = useState(null);
  const [loginError, setLoginError] = useState(false);
  const [authReady, setAuthReady] = useState(false);
  
  const [providers, setProviders] = useState<ProviderField[]>([]);
  const [cookie, setCookie, removeCookies] = useCookies(['accessKey']);

  // TODO: need to revisit this - potential memory leak somewhere
  useEffect(() => {
    const source = axios.CancelToken.source();
    axios.get('/v1/providers')
      .then((response) => {
        setProviders(response.data);
        redirectAccountPage(response.data);
      })
      .catch((error) => {
        setLoginError(true);
    });
    return function () {
      source.cancel("Cancelling in cleanup");
    };
  }, []);


  const getCurrentUser = async (key: string) => {
    return await axios.get('/v1/introspect', { headers: { Authorization: `Bearer ${key}` } })
    // return await axios.get('/v1/introspect')
    .then((response) => {
      return response.data;
    })
    .catch((error) => {
      setAuthReady(false);
      setLoginError(true);
    })
  }

  const redirectToDashboard = async (key: string) => {
    try {
      const currentUser = await getCurrentUser(key)
      
      setUser(currentUser);
      setAuthReady(true);

      Router.push({
        pathname: '/',
      }, undefined, { shallow: true })
    } catch(error) {
      setLoginError(true);
    }
  }

  const getAccessKey = async (code: string, providerID: string, redirectURL: string) => {
    axios.post('/v1/login', { providerID, code, redirectURL })
    .then(async(response) => {
      setCookie('accessKey', response.data.accessKey, { path: '/' });
      await redirectToDashboard(response.data.accessKey);
    })
    .catch((error) => {
      setAuthReady(false);
      setLoginError(true);
      Router.push({
        pathname: '/account/login',
      }, undefined, { shallow: true });
    })
  }

  const login = (selectedIdp: ProviderField) => {
    localStorage.setItem('providerId', selectedIdp.id);

    const state = [...Array(10)].map(i=>(~~(Math.random()*36)).toString(36)).join('');
    localStorage.setItem('state', state);

    const infraRedirect = window.location.origin + `/account/callback`;
    localStorage.setItem('redirectURL', infraRedirect);

    document.location.href = `https://${selectedIdp.url}/oauth2/v1/authorize?redirect_uri=${infraRedirect}&client_id=${selectedIdp.clientID}&response_type=code&scope=openid+email+groups+offline_access&state=${state}`;
  }

  // TODO: it is not working right now
  const logout = () => {
    axios.post('/v1/logout', {}, { headers: { Authorization: `Bearer ${cookie.accessKey}` }})
    .then((response) => {
      removeCookies('accessKey', { path: '/' });

      // redirect based on provider[]
      redirectAccountPage(providers);
    })
  }

  const register = async (key: string) => {
    setCookie('accessKey', key, { path: '/' });
    await redirectToDashboard(key);
  }

  const context:AppContextInterface = { 
    authReady,
    cookie,
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
      { children }
    </AuthContext.Provider>
  )
}

export default AuthContext