import Router from 'next/router'
import styled from 'styled-components'

import ExitButton from '../../../components/ExitButton'
import IdentityProviderButton from '../../../components/IdentityProviderButton'
import Header from '../../../components/Header'

const ConnectProviderContainer = styled.section`
  position: relative;
`

const ConnectProviderContent = styled.section`
  margin-left: auto;
  margin-right: auto;
  max-width: 24rem;
  padding-top: 1.5rem;

  & > *:not(:first-child) {
    padding-top: 1.75rem;
  }  
`

const NavButton = styled.div`
  position: absolute;
  top: .5rem;
  right: .5rem;
`

const IdentityProviderList = styled.section`
  & > *:not(:first-child) {
    padding-top: 42px;
  }
`

const setupOkta = async () => {
  await Router.push({
    pathname: '/providers/add/okta'
  }, undefined, { shallow: true })
}

const avaliableProviderList = [{
  type: 'okta',
  name: 'Identity Provider',
  onClick: () => setupOkta()
}]

const Select = () => {
  return (
    <ConnectProviderContainer>
      <ConnectProviderContent>
        <Header
          header='Connect Identity Providers'
          subheader='People, Groups and Machines'
        />
        <Header
          header='Choose an Identity Provider'
          subheader={
            <>Currently there are no identity providers connected to Infra. <br />
              Choose your IdP source below and get connected.
            </>
          }
        />
        <IdentityProviderList>
          <div>
            <IdentityProviderButton providers={avaliableProviderList} />
          </div>
        </IdentityProviderList>
      </ConnectProviderContent>
      <NavButton>
        <ExitButton previousPage='/providers' />
      </NavButton>
    </ConnectProviderContainer>
  )
}

export default Select
