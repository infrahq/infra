import { createContext, useState, useEffect } from 'react'
import Router from 'next/router'; 
import axios from 'axios';
import { useCookies } from 'react-cookie';

interface AppContextInterface {
  user: any,
  login: () => void,
  logout: () => void,
  register: (key:string) => void,
  cookie: {},
  authReady: boolean,
  loginError: boolean,
  providers: ProviderField[] | []
}

interface ProviderField {
  id: string,
  name: string,
  created: number,
  updated: number,
  url: string,
  clientID: string
};

const AuthContext = createContext<AppContextInterface>({
  user: null,
  login: () => {},
  logout: () => {},
  register: () => {},
  cookie: {},
  authReady: false,
  loginError: false,
  providers: [],
})

export const AuthContextProvider = ({ children }:any) => {
  const hasProvider: ProviderField[] = 
  [{"id":"2H21T3DkBw","name":"okta","created":-62135596800,"updated":1644606820,"url":"dev-02708987.okta.com","clientID":"0oapn0qwiQPiMIyR35d6"},
  {"id":"2H21T3DkBw","name":"okta2 - invalid","created":-62135596800,"updated":1644606820,"url":"dev-02708988.okta.com","clientID":"0oapn0qwiQPiMIyR35d6"}];
  
  const [user, setUser] = useState(null);
  const [authReady, setAuthReady] = useState(false);
  const [loginError, setLoginError] = useState(false);
  const [providers, setProviders] = useState<ProviderField[] | []>(hasProvider);
  const [cookie, setCookie] = useCookies(['accessKey']);


  useEffect(() => {

    // check the /v1/providers
    console.log(providers);
    console.log(providers.length);

    authCheck();
  }, []);

  const authCheck = () => {
    if (providers.length > 0) {
      Router.push({
        pathname: '/account/login',
      }, undefined, { shallow: true });
    } else {
      Router.push({
        pathname: '/account/register',
      }, undefined, { shallow: true });
    }
  }

  const login = () => {

  }

  const logout = () => {

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

  const context:AppContextInterface = { user, login, logout, register, cookie, authReady, loginError, providers }

  return (
    <AuthContext.Provider value={context}>
      { children }
    </AuthContext.Provider>
  )
}

export default AuthContext