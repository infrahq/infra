import { useCallback, useState, useContext } from 'react'
import Router from 'next/router'
import axios from 'axios'
import styled from 'styled-components'

import ExitButton from '../../components/ExitButtn'
import ActionButton from '../../components/ActionButton'
import Setup from '../../components/providers/okta/setup'
import AddAdmin from '../../components/providers/okta/AddAdmin'

import AuthContext from '../../store/AuthContext'

const SetupContainer = styled.section`
  position: relative;
`

const SetupContainerContent = styled.section`
  margin-left: auto;
  margin-right: auto;
  max-width: 24rem;
  padding-top: 1.5rem;

  & > *:not(:first-child) {
    padding-top: 1.75rem;
  }
`

const Nav = styled.section`
  position: absolute;
  right: .5rem;
  top: .5rem;
`

const Footer = styled.section`
  position: fixed;
  bottom: 1.5rem;
  right: .5rem;
`

const SetupOkta = () => {
  const { cookie, setNewProvider } = useContext(AuthContext)

  const page = Object.freeze({ Setup: 1, AddAdmin: 2 })

  const [currentPage, setCurrentPage] = useState(page.Setup)
  const [adminEmail, setAdminEmail] = useState('')
  const [providerId, setProviderId] = useState(null)

  const [value, setValue] = useState({
    name: 'okta',
    domain: '',
    clientId: '',
    clientSecret: ''
  })

  const moveToNext = async () => {
    if (currentPage === page.Setup) {
      axios.post('/v1/providers',
        { name: value.name, url: value.domain, clientID: value.clientId, clientSecret: value.clientSecret },
        { headers: { Authorization: `Bearer ${cookie.accessKey}` } })
        .then((response) => {
          setNewProvider(response.data)
          setProviderId(response.data.id)
          setCurrentPage(page.AddAdmin)
        }).catch((error) => {
          console.log('error:', error)
        })
    }
    if (currentPage === page.AddAdmin) {
      // check if user exists
      const params = {
        email: adminEmail,
        provider_id: providerId
      }
      // TODO: returning 401 at the moment
      const currentUser = await axios.get('/v1/users', { params }, { headers: { Authorization: `Bearer ${cookie.accessKey}` } });

      // if (currentUser == null) {
      //   axios.post
      // }

      // if it doesn't exist, create one

      // otherwise grant access with the user id

      // set the admin email to the infra admin
      // if success then redirect back to dashboard
      await Router.push({
        pathname: '/'
      }, undefined, { shallow: true })
    }
  }

  const updateValue = useCallback((callbackvalue, type) => {
    setValue(previousState => ({
      ...previousState,
      [type]: callbackvalue
    }))
  }, [])

  const updateEmail = useCallback((email) => {
    setAdminEmail(email)
  })

  const content = (pageType) => {
    switch (pageType) {
      case page.Setup:
        return <Setup value={value} parentCallback={updateValue} />
      case page.AddAdmin:
        return <AddAdmin email={adminEmail} parentCallback={updateEmail} />
      default:
        return <Setup value={value} parentCallback={updateValue} />
    }
  }

  return (
    <>
      <SetupContainer>
        <SetupContainerContent>
          {content(currentPage)}
        </SetupContainerContent>
        <Nav>
          <ExitButton />
        </Nav>
      </SetupContainer>
      <Footer>
        <ActionButton onClick={() => moveToNext()} value={currentPage === page.Setup ? 'Connect' : 'Proceed'} size='small' />
      </Footer>
    </>
  )
}

export default SetupOkta
