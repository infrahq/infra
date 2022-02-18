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
  getAccessKey: (code: string, providerID: string) => void,
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

export const AuthContextProvider = ({ children }:any) => { 
  const [user, setUser] = useState(null);
  const [authReady, setAuthReady] = useState(false);
  const [loginError, setLoginError] = useState(false);

  const [providers, setProviders] = useState<ProviderField[]>([]);
  const [cookie, setCookie, removeCookies] = useCookies(['accessKey']);

  useEffect(() => {
    axios.get('/v1/providers')
    .then((response) => {
      setProviders(response.data);

      redirectAccountPage(response.data);
      // if (response.data.length > 0) {
      //   Router.push({
      //     pathname: '/account/login',
      //   }, undefined, { shallow: true });
      // } else {
      //   Router.push({
      //     pathname: '/account/register',
      //   }, undefined, { shallow: true });
      // }
    })
    .catch((error) => {
      console.log(error);
      setLoginError(true);
    })
  }, []);

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

  const getCurrentUser = async() => {
    return await axios.get('/v1/users', { headers: { Authorization: `Bearer ${cookie.accessKey}` } })
    .then((response) => {
      return response.data[0];
    })
  }

  const getAccessKey = async (code: string, providerID: string) => {
    axios.post('/v1/login', {providerID, code})
    .then(async(response) => {
      setCookie('accessKey', response.data.token, { path: '/' });
      setAuthReady(true);

      const userData = await getCurrentUser();
      setUser(userData);
      
      Router.push({
        pathname: '/',
      }, undefined, { shallow: true });
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
    const authorizeURL= `https://${selectedIdp.url}/oauth2/v1/authorize?redirect_uri=${infraRedirect}&client_id=${selectedIdp.clientID}&response_type=code&scope=openid+email+groups+offline_access&state=${state}`;

    document.location.href = authorizeURL;
  }

  const logout = () => {

    console.log('logout:', cookie.accessKey)
    axios.post('/v1/logout', { headers: { Authorization: `Bearer ${cookie.accessKey}` }})
    .then((response) => {
      removeCookies('accessKey', { path: '/' });


      // redirect based on provider[]
      redirectAccountPage(providers);
      console.log(response);
    })
  }

  const register = async (key: string) => {
    setCookie('accessKey', key, { path: '/' });

    // TODO: need to handle multiple axios called

    // const usersList =  axios.get('/v1/users', { headers: { Authorization: `Bearer ${key}` } });
    // const machinesList =  axios.get('/v1/machines', { headers: { Authorization: `Bearer ${key}` } });

    // await axios.all([usersList, machinesList]).then(axios.spread((...responses) => {
    //   setUser(responses[0].data);
    //   setMachine(responses[1].data);

    //   // redirect to '/'

    // })).catch((errors) => {

    // })

    await axios.get('/v1/users', { headers: { Authorization: `Bearer ${key}` } })
    .then((response) => {
      Router.push({
        pathname: '/',
      }, undefined, { shallow: true });
      setAuthReady(true);
    })
    .catch((error) => {
      console.log(error);
      setLoginError(true);
    });
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