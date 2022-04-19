import { useCallback, useState } from 'react'
import styled from 'styled-components'
import Router, { useRouter } from 'next/router'

import ExitButton from '../../../components/ExitButton'
import ActionButton from '../../../components/ActionButton'
import Setup from '../../../components/providers/okta/setup'

export const AddContainer = styled.section`
  position: relative;
`

export const AddContainerContent = styled.section`
  margin-left: auto;
  margin-right: auto;
  max-width: 24rem;
  padding-top: 1.5rem;

  & > *:not(:first-child) {
    padding-top: 1.75rem;
  }
`

export const Nav = styled.section`
  position: absolute;
  right: .5rem;
  top: .5rem;
`

export const Footer = styled.section`
  position: fixed;
  bottom: 1.5rem;
  right: .5rem;
`

const Details = () => {
  const router = useRouter()
  const { type } = router.query

  const [value, setValue] = useState({
    domain: '',
    clientId: '',
    clientSecret: ''
  })

  const content = (type) => {
    if (type === 'okta') {
      return <Setup value={value} parentCallback={updateValue} />
    }
  }

  const updateValue = useCallback((callbackvalue, type) => {
    setValue(previousState => ({
      ...previousState,
      [type]: callbackvalue
    }))
  }, [])

  const moveToNext = () => {
    const name = type + '-' + value.domain

    fetch('/v1/providers', {
      method: 'POST',
      body: JSON.stringify({ name, url: value.domain, clientID: value.clientId, clientSecret: value.clientSecret })
    })
      .then(() => {
        Router.push({
          pathname: '/providers'
        }, undefined, { shallow: true })
      }).catch((error) => {
        console.log('error:', error)
      })
  }

  return (
    <>
      <AddContainer>
        <AddContainerContent>
          {content(type)}
        </AddContainerContent>
        <Nav>
          <ExitButton previousPage='/providers' />
        </Nav>
      </AddContainer>
      <Footer>
        <ActionButton onClick={() => moveToNext()} value='Connect' size='small' />
      </Footer>
    </>
  )
}

export default Details
