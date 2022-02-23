import styled from "styled-components";
import Link from 'next/link';
import Router from "next/router";
import { useContext, useEffect } from "react";

import AccountFooter from "../../components/AccountFooter";
import AccountHeader from "../../components/AccountHeader";
import IdentitySourceBtn, { IdentitySourceProvider }  from "../../components/IdentitySourceBtn";

import AuthContext from "../../store/AuthContext";

const LoginContainer = styled.section`
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
`;

const LoginIdentitySourceList = styled.div`
  margin-top: 2rem;
`;

const LoginIdentitySourceComingSoonListContainer = styled.div`
  & > *:not(:first-child) {
    padding-top: 1.25rem;
  }
`;

const LoginIdentitySourceComingSoonListHeader = styled.div`
  font-weight: 100;
  font-size: 12px;
  line-height: 15px;
  display: flex;
  align-items: center;

  color: #FFFFFF;

  opacity: 0.56;
`;

const LoginIdentitySourceComingSoonList = styled.div`
  & > *:not(:first-child) {
    padding-top: .25rem;
  }
`;

const HelpContainer = styled.div`
  margin-top: 3rem;
  font-weight: 100;
  font-size: 11px;
  line-height: 13px;
  max-width: 24rem;
  text-align: center;

  span {
    opacity: .5;
  }

  a {
    padding-left: .5rem;
    color: #93DEFF;
    text-decoration: none;

    :hover {
      opacity: .95;
    }
  }
`;

const Footer = styled.div`
  grid-row-start: 2;
  grid-row-end: 3;
  padding: 2rem 0;
`;

const Login = () => {
  const { providers, authReady } = useContext(AuthContext);
  const comingSoonList: IdentitySourceProvider[] = [
    {
      type: 'google',
    }, 
    {
      type: 'azure',
    },
    {
      type: 'gitlab'
    }
  ];

  const getProviderType = (url: string):string => {
    let tempURL = url;
    return tempURL.replace(/^https?:\/\//, '').split('/')[0].split('.').reverse()[1]; 
  }

  const providerWithType = providers.map((item) => {
    const type = getProviderType(item.url);
    return {...item, type}
  })

  useEffect(() => {
    if(authReady) {
      Router.push({
        pathname: '/',
      }, undefined, { shallow: true });
    }
  }, [])

  return (
    <LoginContainer>
      <Content>
        <AccountHeader
          header='Login to Infra'
          subheader='Securely manage access to your infrastructure. Take a moment to create your account and start managing access today.'
        />
        <LoginIdentitySourceList>
          <IdentitySourceBtn providers={providerWithType} />
        </LoginIdentitySourceList>
        <LoginIdentitySourceComingSoonListContainer>
          <LoginIdentitySourceComingSoonListHeader>Coming Soon</LoginIdentitySourceComingSoonListHeader>
          <LoginIdentitySourceComingSoonList>        
            {comingSoonList.map((identity) => 
              <div key={identity.type}>
                <IdentitySourceBtn providers={[identity]} />
              </div>
            )}
          </LoginIdentitySourceComingSoonList>
        </LoginIdentitySourceComingSoonListContainer>
        <HelpContainer>
          <span>Having trouble logging in?</span>
          <Link href='/account/register'>
            <a>Use API Access Key</a>
          </Link>
        </HelpContainer>
      </Content>
      <Footer>
        <AccountFooter />
      </Footer>
    </LoginContainer>
  )
};

export default Login;