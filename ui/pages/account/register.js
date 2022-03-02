import { useContext, useEffect, useState } from 'react'
import styled from 'styled-components'

import Input from '../../components/Input'
import ActionButton from '../../components/ActionButton'
import AccountHeader from '../../components/AccountHeader'

import AuthContext from '../../store/AuthContext'
import { readyToRedirect } from './login'

const RegisterContainer = styled.section`
  margin-left: auto;
  margin-right: auto;
  max-width: 24rem;
  padding-top: 2rem;

  display: grid;
  grid-template-rows: 1fr auto;
  min-height: 100%;
`

const Content = styled.div`
  & > *:not(:first-child) {
    padding-top: 1.5rem;
  }
`

const AccessKeyInputContainer = styled.div`
  margin-top: 1.5rem;
`

const Register = () => {
  const { authReady, register } = useContext(AuthContext)
  const [value, setValue] = useState('')

  useEffect(() => {
    if (authReady) {
      readyToRedirect()
    }
  }, [])

  const handleLogin = async () => {
    await register(value)
  }

  return (
    <RegisterContainer>
      <Content>
        <AccountHeader
          header='Infra Admin API Access Key'
          subheader='Securely manage access to your infrastructure. Take a moment to create your account and start managing access today.'
        />
        <AccessKeyInputContainer>
          <Input
            label='Admin API Access Key'
            value={value}
            onChange={e => setValue(e.target.value)}
            showImage
          />
        </AccessKeyInputContainer>
        <section>
          <ActionButton onClick={handleLogin} value='Login' />
        </section>
      </Content>
    </RegisterContainer>
  )
}

export default Register
