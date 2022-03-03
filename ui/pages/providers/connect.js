import Router from 'next/router'
import styled from 'styled-components'

import ExitButton from '../../components/ExitButtn'
import IdentitySourceBtn from '../../components/IdentitySourceBtn'
import Header from '../../components/providers/Header'

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

const ComingSoonList = styled.div`
  & > *:not(:first-child) {
    padding-top: 20px;
  }

  opacity: 0.3;
`

const ComingSoonHeader = styled.div`
  font-weight: normal;
  font-size: 12px;
  line-height: 15px;

  display: flex;
  align-items: center;
`

const setupOkta = async () => {
  await Router.push({
    pathname: '/providers/setupOkta'
  }, undefined, { shallow: true })
}

const avaliableProviderList = [{
  type: 'okta',
  name: 'Identity Source',
  onClick: () => setupOkta()
}]

const commingSoonProviderList = [{
  type: 'google',
  name: 'Identity Source',
  onClick: undefined,
  disabled: true
},
{
  type: 'azure',
  name: 'Identity Source',
  onClick: undefined,
  disabled: true
},
{
  type: 'gitlab',
  name: 'Identity Source',
  onClick: undefined,
  disabled: true
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
          <ComingSoonList>
            <ComingSoonHeader>Comming Soon</ComingSoonHeader>
            <IdentitySourceBtn providers={commingSoonProviderList} />
          </ComingSoonList>
        </IdentitySourceList>
      </ConnectProviderContent>
      <NavButton>
        <ExitButton />
      </NavButton>
    </ConnectProviderContainer>
  )
}

export default Connect
