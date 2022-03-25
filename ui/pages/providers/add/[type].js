import { useCallback, useState, useContext } from 'react'
import styled from 'styled-components'
import Router, { useRouter } from 'next/router'
import axios from 'axios'

import ExitButton from '../../../components/ExitButton'
import ActionButton from '../../../components/ActionButton'
import Setup from '../../../components/providers/okta/setup'

import AuthContext from '../../../store/AuthContext'

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
  const { setNewProvider } = useContext(AuthContext)

  const router = useRouter()
  const { type } = router.query

  const [value, setValue] = useState({
    name: 'okta',
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

  const addAdmins = async () => {
    await Router.push({
      pathname: '/providers/add/admins'
    }, undefined, { shallow: true })
  }

  const moveToNext = async () => {
    await axios.post('/v1/providers',
      { name: value.name, url: value.domain, clientID: value.clientId, clientSecret: value.clientSecret })
      .then((response) => {
        setNewProvider([response.data])
        addAdmins()
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
