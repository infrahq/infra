import { useContext, useEffect, useState } from "react"
import AuthContext from "../store/AuthContext"
import styled from 'styled-components';

export default function Index () {
  const { logout, user } = useContext(AuthContext);

  // TODO: default value of currentUser
  const [currentUser, setCurrentUser] = useState('user');
  console.log(currentUser);

  useEffect(() => {
    if(user !== null) {
      setCurrentUser(user.email);
    }
  }, [])

  const handleLogout = () => {
    console.log('handlelogout')
    logout();
  }

  return (
    <div>
      <p>{currentUser}</p>
      <button onClick={handleLogout}>Logout</button>
    </div>
  )
}
