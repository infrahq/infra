import styled from "styled-components";
import Link from 'next/link';

import AccountFooter from "../../components/AccountFooter";
import AccountHeader from "../../components/AccountHeader";
import IdentitySourceBtn from "../../components/IdentitySourceBtn";

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

const LoginIdentiySourceList = styled.div`
  margin-top: 2rem;
`;

const LoginIdentiySourceComingSoonListContainer = styled.div`
  & > *:not(:first-child) {
    padding-top: 1.25rem;
  }
`;

const LoginIdentiySourceComingSoonListHeader = styled.div`
  font-weight: 100;
  font-size: 12px;
  line-height: 15px;
  display: flex;
  align-items: center;

  color: #FFFFFF;

  opacity: 0.56;
`;

const LoginIdentiySourceComingSoonList = styled.div`
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
  const handleOktaLogin = () => {
    console.log('you click to log in with Okta')
  }

  const comingSoonList = ['google', 'azure', 'gitlab'];

  return (
    <LoginContainer>
      <Content>
        <AccountHeader
          header='Login to Infra'
          subheader='Securely manage access to your infrastructure. Take a moment to create your account and start managing access today.'
        />
        <LoginIdentiySourceList>
          <IdentitySourceBtn type='okta' disabled={false} onClick={handleOktaLogin} />
        </LoginIdentiySourceList>
        <LoginIdentiySourceComingSoonListContainer>
          <LoginIdentiySourceComingSoonListHeader>Coming Soon</LoginIdentiySourceComingSoonListHeader>
          <LoginIdentiySourceComingSoonList>        
            {comingSoonList.map((identity) => 
              <div key={identity}>
                <IdentitySourceBtn type={identity} disabled={true} />
              </div>
            )}
          </LoginIdentiySourceComingSoonList>
        </LoginIdentiySourceComingSoonListContainer>
        <HelpContainer>
          <span>Having trouble loggin in?</span>
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