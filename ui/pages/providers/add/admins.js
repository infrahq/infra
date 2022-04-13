import { useCallback, useState } from 'react'
import Router from 'next/router'
import Head from 'next/head'

import { AddContainer, AddContainerContent, Nav, Footer } from './[type]'
import ExitButton from '../../../components/ExitButton'
import ActionButton from '../../../components/ActionButton'
import AddAdmin from '../../../components/providers/okta/AddAdmin'

const grantAdminAccess = (userId) => {
  fetch('/v1/grants', {
    method: 'POST',
    body: JSON.stringify({ subject: userId, resource: 'infra', privilege: 'admin' })
  })
  .then(() => {
    Router.push({ pathname: '/providers' }, undefined, { shallow: true })
  }).catch((error) => {
    console.log(error)
  })
}

const Admins = () => {
  const [adminEmail, setAdminEmail] = useState('')

  const updateEmail = useCallback((email) => {
    setAdminEmail(email)
  })

  const moveToNext = async () => {
    fetch(`/v1/identities?name=${adminEmail}`)
    .then((response) => {
      return response.json();
    })
    .then((data) => {
      if(data.length === 0) {
        fetch('/v1/identities', {
          method: 'POST',
          body: JSON.stringify({ name: adminEmail, kind: 'user' })
        })
        .then((response) => {
         return response.json()
        })
        .then((user) => {
          grantAdminAccess(user.id)
        })
        .catch((error) => {
          console.log(error)
        })
      } else {
        grantAdminAccess(data[0].id)
      }
    })
  }

  return (
    <>
      <Head>
        <title>Infra - Providers</title>
      </Head>
      <AddContainer>
        <AddContainerContent>
          <AddAdmin email={adminEmail} parentCallback={updateEmail} />
        </AddContainerContent>
        <Nav>
          <ExitButton previousPage='/providers' />
        </Nav>
      </AddContainer>
      <Footer>
        <ActionButton onClick={() => moveToNext()} value='Proceed' size='small' />
      </Footer>
    </>
  )
}

export default Admins
