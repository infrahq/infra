import { useCallback, useState } from 'react'
import Router from 'next/router'
import styled from 'styled-components'

import ExitButton from '../../components/ExitButtn'
import ActionButton from '../../components/ActionButton'
import Setup from '../../components/providers/okta/setup'
import Connected from '../../components/providers/okta/connected'
import AddAdmin from '../../components/providers/okta/AddAdmin'

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
  const page = Object.freeze({ Setup: 1, connected: 2, AddAdmin: 3 })
  const [currentPage, setCurrentPage] = useState(page.Setup)
  const [provider, setProvider] = useState({})
  const [adminEmail, setAdminEmail] = useState('');


  const [value, setValue] = useState({
    name: 'Okta',
    domain: '',
    clientId: '',
    clientSecret: ''
  })

  const moveToNext = async () => {
    // update the state
    if (currentPage === page.Setup) {
      console.log(value)
      // call the endpoint to connect with okta provider
      // when it is successed then update the current page
      setCurrentPage(page.connected)
      // set the return value as provider
      const returnValue = {
        type: 'okta',
        name: 'okta-test',
        id: '3GuiBghzw1',
        created: 1645809213,
        updated: 1646159548,
        url: 'dev-02708987.okta.com',
        clientID: '0oapn0qwiQPiMIyR35d6',
        view: true,
        disabled: true
      }
      setProvider(returnValue)
    }

    if (currentPage === page.connected) {
      setCurrentPage(page.AddAdmin)
    }

    if (currentPage === page.AddAdmin) {
      console.log(adminEmail);
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
      case page.connected:
        return <Connected provider={provider} />
      case page.AddAdmin:
        return <AddAdmin email={adminEmail} parentCallback={updateEmail} />
      default:
        return <Setup parentCallback={callback} />
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
