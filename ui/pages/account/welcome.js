import styled from 'styled-components'
import { useContext } from 'react'

import AccountHeader from '../../components/AccountHeader'
import ActionButton from '../../components/ActionButton'

import AuthContext from '../../store/AuthContext'

const WelcomeContainer = styled.section`
  margin-left: auto;
  margin-right: auto;
  max-width: 24rem;
  padding-top: 2rem;

  display: grid;
  grid-template-rows: 1fr auto;
  min-height: 100%;
`

const WelcomeImg = styled.img`
  margin-left: -5rem;
`

const Welcome = () => {
  const { setup } = useContext(AuthContext)

  const handleSetup = async () => {
    await setup()
  }

  return (
    <WelcomeContainer>
      <div>
        <AccountHeader
          header='Welcome to Infra'
          subheader='Infra has been successfully installed. Please click Get Started below to obtain your Infra Access Key.'
        />
        <WelcomeImg src='/welcome.svg' />
        <ActionButton
          onClick={handleSetup}
          value='Get Started'
        />
      </div>
    </WelcomeContainer>
  )
}

export default Welcome
