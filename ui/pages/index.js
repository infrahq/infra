import Router from 'next/router'
import { useContext, useEffect, useState } from 'react'
import ActionButton from '../components/ActionButton'

import Nav from '../components/Nav'
import AuthContext from '../store/AuthContext'

export default function Index () {
  const { user, providers } = useContext(AuthContext)
  const [currentUser, setCurrentUser] = useState(null)

  useEffect(() => {
    if (user != null) {
      setCurrentUser(user)
    }
  }, [])

  const handleConnectProviders = async () => {
    await Router.push({
      pathname: '/providers/add/select'
    }, undefined, { shallow: true })
  }

  return (
    <div>
      <Nav />
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
    </div>
  )
}
