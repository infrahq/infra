import styled from 'styled-components'
import { useContext, useEffect, useState } from 'react'
import Head from 'next/head'

import AuthContext from '../../store/AuthContext'
import AccountHeader from '../../components/AccountHeader'
import ActionButton from '../../components/ActionButton'
import AccessKeyCard from '../../components/AccessKeyCard'

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

const WarningSection = styled.div`
  display: flex;
  flex-direction: row;
  height: 58px;
  width: auto;
  background: rgba(255, 255, 255, 0.02);
  border-left: 2px solid #808EF9;
  box-sizing: border-box;
  box-shadow: 0px 4px 4px rgba(0, 0, 0, 0.25);
  border-radius: 2px;

  & > *:not(:first-child) {
    padding-left: 1rem;
  }
`

const WarningImg = styled.img`
  width: 15.17px;
  height: 13.92px;
  padding-top: 21px;
  padding-left: 20px;
`

const WarningContentText = styled.div`
  font-style: normal;
  font-weight: 400;
  font-size: 11px;
  line-height: 156.52%;
  color: #FFFFFF;
  opacity: 0.5;
  padding-top: 1rem;
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
          <WarningSection>
            <WarningImg src='/warning-icon.svg' />
            <WarningContentText>You will not be able to retrieve this access key again.</WarningContentText>
          </WarningSection>
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
