import { createContext, useState, useEffect } from 'react'
import Router from 'next/router'; 
import axios from 'axios';
import { useCookies } from 'react-cookie';


const AuthContext = createContext({
  user: null,
  login: () => {},
  logout: () => {},
  register: (key: string) => {},
  cookie: {},
  authReady: false
})

export const AuthContextProvider = ({ children }:any) => {
  const [user, setUser] = useState(null);
  const [authReady, setAuthReady] = useState(false);
  const [cookie, setCookie] = useCookies(['accessKey']);


  useEffect(() => {
    // on initial load - run auth check 
    authCheck();

  }, []);

  const authCheck = () => {
    // check the /v1/providers
    const hasProvider = [{"id":"2H21T3DkBw","name":"okta","created":-62135596800,"updated":1644606820,"url":"dev-02708987.okta.com","clientID":"0oapn0qwiQPiMIyR35d6"}];
    // const providers:any[] = hasProvider;
    const providers:any[] = [];

    // save the provider to localstorage / redux?
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

    await axios.get('/v1/users', { headers: { Authorization: `Bearer ${key}` } })
    .then((response) => {
      if (response.status === 200) {
        Router.push({
          pathname: '/',
        }, undefined, { shallow: true });
        setAuthReady(true);

        // call machine to set user
        // setUser()
      }
    })
    .catch((error) => {
      console.log(error);
    });

  }

  const context = { user, login, logout, register, cookie, authReady }

  return (
    <AuthContext.Provider value={context}>
      { children }
    </AuthContext.Provider>
  )
}

export default AuthContext