import styled from 'styled-components'

import ExitButton from '../../components/ExitButtn'
import ActionButton from '../../components/ActionButton'
import { useCallback, useState } from 'react'
import Setup from '../../components/providers/okta/setup'

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
  const page = Object.freeze({"Setup":1, "connected":2, "AddAdmin":3})
  const [currentPage, setCurrentPage] = useState(page.Setup);

  const [name, setName] = useState('Okta')
  const [domain, setDomain] = useState('')
  const [clientId, setClientId] = useState('')
  const [clientSecret, setClientSecret] = useState('')

  const moveToNext = () => {
    // update the state
    console.log('moving to next')
  }

  const callback = useCallback((value, type) => {
    console.log(value)
    console.log(type)
  }, [])

  return (
    <>
      <SetupContainer>
        <SetupContainerContent>
          <Setup parentCallback={callback} />
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