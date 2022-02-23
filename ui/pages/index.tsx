import { useContext, useEffect, useState } from "react"
import AuthContext from "../store/AuthContext"

export default function Index () {
  const { logout, user } = useContext(AuthContext);

  // TODO: default value of currentUser
  const [currentUser, setCurrentUser] = useState(null);

  useEffect(() => {
    if(typeof user !== 'undefined') {
      setCurrentUser(user.name);
    }
  }, [])

  const handleLogout = () => {
    logout();
  }

  return (
    <div>
      <p>{currentUser}</p>
      <button onClick={handleLogout}>Logout</button>
    </div>
  )
}
