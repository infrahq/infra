import Router from 'next/router'
import styled from 'styled-components'

import ExitButton from '../../components/ExitButtn'
import IdentitySourceBtn from '../../components/IdentitySourceBtn'
import Header from '../../components/Header'

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

const IdentitySourceList = styled.section`
  & > *:not(:first-child) {
    padding-top: 42px;
  }
`

const setupOkta = async () => {
  await Router.push({
    pathname: '/providers/setupOkta'
  }, undefined, { shallow: true })
}

const avaliableProviderList = [{
  type: 'okta',
  name: 'Identity Provider',
  onClick: () => setupOkta()
}]

const Connect = () => {
  return (
    <ConnectProviderContainer>
      <ConnectProviderContent>
        <Header
          header='connect identity Providers'
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
        <IdentitySourceList>
          <div>
            <IdentitySourceBtn providers={avaliableProviderList} />
          </div>
        </IdentitySourceList>
      </ConnectProviderContent>
      <NavButton>
        <ExitButton />
      </NavButton>
    </ConnectProviderContainer>
  )
}

export default Connect
