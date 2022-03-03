import Link from 'next/link'
import { useContext, useEffect, useState } from 'react'
import AuthContext from '../store/AuthContext'

export default function Index () {
  const { logout, user } = useContext(AuthContext)

  // TODO: default value of currentUser
  const [currentUser, setCurrentUser] = useState(null)

  useEffect(() => {
    if (user != null) {
      setCurrentUser(user.name)
    }
  }, [])

  const handleLogout = async () => {
    await logout()
  }

  return (
    <div>
      <p>{currentUser}</p>
      <Link href='/providers/connect'>
        <a>Connect Identity Providers</a>
      </Link>
      <button onClick={handleLogout}>Logout</button>
    </div>
  )
}
