import Router from 'next/router'
import { useContext, useEffect, useState } from 'react'
import ActionButton from '../components/ActionButton'

import AuthContext from '../store/AuthContext'

export default function Index () {
  const { logout, user, providers } = useContext(AuthContext)
  const [currentUser, setCurrentUser] = useState(null)

  useEffect(() => {
    if (user != null) {
      setCurrentUser(user)
    }
  }, [])

  const handleLogout = async () => {
    await logout()
  }

  const handleConnectProviders = async () => {
    await Router.push({
      pathname: '/providers/connect'
    }, undefined, { shallow: true })
  }

  return (
    <div>
      {currentUser ? <p>{currentUser.name}</p> : <></>}
      {providers.length > 0
        ? (
          <div>
            {providers.map((item) => {
              return (
                <div key={item.id}>
                  <span>{item.name} / </span>
                  <span>{item.url}</span>
                </div>
              )
            })}
          </div>
        )
        : (
          <ActionButton
            onClick={() => handleConnectProviders()}
            value='Connect Identity Providers'
            size='small'
          />
          )}
      <button onClick={handleLogout}>Logout</button>
    </div>
  )
}
