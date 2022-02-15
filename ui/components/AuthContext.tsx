import { createContext, useState, useEffect } from 'react'
import Router from 'next/router'; 


const AuthContext = createContext({
  user: null,
  authReady: false
})

export const AuthContextProvider = ({ children }:any) => {
  const [user, setUser] = useState(null)
  const [authReady, setAuthReady] = useState(false)

  useEffect(() => {
    // on initial load - run auth check 
    authCheck();

  }, []);

  const authCheck = () => {
    // check if there is user
    // setUser(...)

    // check the /v1/providers
    const hasProvider = [{"id":"2H21T3DkBw","name":"okta","created":-62135596800,"updated":1644606820,"url":"dev-02708987.okta.com","clientID":"0oapn0qwiQPiMIyR35d6"}];
    // const providers:any[] = hasProvider;
    const providers:any[] = [];

    // save the provider to localstorage / redux?
    // check if user is exist
    // if(...) {
    setAuthReady(false);
    console.log(providers);
    if (providers.length > 0) {
      Router.push({
        pathname: '/account/login',
        // query: { returnUrl: router.asPath }
      }, undefined, { shallow: true });
    } else {
      Router.push({
        pathname: '/account/register',
        // query: { returnUrl: router.asPath }
      }, undefined, { shallow: true });
    }

    // } else { setAuthReady(true) }
  }

  const context = { user, authReady }

  return (
    <AuthContext.Provider value={context}>
      { children }
    </AuthContext.Provider>
  )
}

export default AuthContext