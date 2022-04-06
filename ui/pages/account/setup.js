import styled from 'styled-components'
import { useContext, useEffect, useState } from 'react'
import Head from 'next/head'

import AuthContext from '../../store/AuthContext'
import AccountHeader from '../../components/AccountHeader'
import ActionButton from '../../components/ActionButton'
import AccessKeyCard from '../../components/AccessKeyCard'
import WarningContainer from '../../components/WarningContainer'

const SetupContainer = styled.section`
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
    margin-top: 1.5rem;
  }
`

const Setup = () => {
  const { accessKey, register } = useContext(AuthContext)
  const [currentAccessKey, setCurrentAccessKey] = useState(null)

  useEffect(() => {
    if (accessKey != null) {
      setCurrentAccessKey(accessKey)
    }
  }, [])

  const handleLogin = async () => {
    register(accessKey)
  }

  return (
    <>
      <Head>
        <title>Infra - Welcome</title>
      </Head>
      <SetupContainer>
        <Content>
          <AccountHeader
            header='Welcome to Infra'
            title='Please backup your Infra Access Key in a safe place.'
            subheader='This access key will allow you to use Infra in the event that you cannot sign in with your configured identity provider.'
          />
          <AccessKeyCard accessKey={currentAccessKey} />
          <WarningContainer text='You will not be able to retrieve this access key again.' />
          <ActionButton
            onClick={handleLogin}
            value='Continue'
          />
        </Content>
      </SetupContainer>
    </>
  )
}

export default Setup
