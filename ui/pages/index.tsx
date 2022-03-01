import { useContext, useEffect, useState } from 'react'
import AuthContext from '../store/AuthContext'

export default function Index (): JSX.Element {
  const { logout, user } = useContext(AuthContext)

  // TODO: default value of currentUser
  const [currentUser, setCurrentUser] = useState<string | null>(null)

  useEffect(() => {
    if (user != null) {
      setCurrentUser(user.name)
    }
  }, [])

  const handleLogout = async (): Promise<void> => {
    await logout()
  }

  return (
    <div>
      <p>{currentUser}</p>
      <button onClick={handleLogout}>Logout</button>
    </div>
  )
}
