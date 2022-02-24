import { useContext, useEffect } from "react";
import AuthContext from "../../store/AuthContext";

const Callback = () => {
  const { getAccessKey } = useContext(AuthContext);

  useEffect(() => {
    const urlSearchParams = new URLSearchParams(window.location.search);
    const params = Object.fromEntries(urlSearchParams.entries());

    if(params.state === localStorage.getItem('state')) {
      getAccessKey(params.code,
        localStorage.getItem('providerId') as string,
        localStorage.getItem('redirectURL') as string
      );

      localStorage.removeItem('providerId');
      localStorage.removeItem('state');
      localStorage.removeItem('redirectURL');
    }
  }, []);

  return (null);

};

export default Callback;