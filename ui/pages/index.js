import { useContext, useEffect, useState } from 'react'

import Navigation from '../components/nav/Navigation'

import AuthContext from '../store/AuthContext'

export default function Index () {
  const { user } = useContext(AuthContext)
  const [currentUser, setCurrentUser] = useState(null)

  useEffect(() => {
    if (user != null) {
      setCurrentUser(user)
    }
  }, [])

  return (
    <div>
      <Navigation />
      {currentUser ? <p>{currentUser.name}</p> : <></>}
    </div>
  )
}
