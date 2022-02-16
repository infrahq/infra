import { useContext, useEffect, useState } from 'react';
import styled from 'styled-components';
import Router from 'next/router'; 

import AccessKeyInput from '../../components/AccessKeyInput';
import ActionButton from '../../components/ActionButton';
import AccountFooter from '../../components/AccountFooter';
import AccountHeader from '../../components/AccountHeader';
import AuthContext from '../../store/AuthContext';

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
`;

const AccessKeyInputContainer = styled.div`
  margin-top: 1.5rem;
`;

const Footer = styled.div`
  grid-row-start: 2;
  grid-row-end: 3;
  padding: 2rem 0;
`;

const Register = () => {
  const { authReady, register } = useContext(AuthContext)
  const [value, setValue] = useState('');
  
  useEffect(() => {
    if(authReady) {
      Router.push({
        pathname: '/',
      }, undefined, { shallow: true });
    }
  }, [])

  const handleLogin = async () => {
    register(value);
  };

  return (
    <RegisterContainer>
      <Content>
        <AccountHeader 
          header='Infra Admin API Access Key'
          subheader='Securely manage access to your infrastructure. Take a moment to create your account and start managing access today.'
        />
        <AccessKeyInputContainer>
          <AccessKeyInput 
            value={value}
            onChange={e => setValue(e.target.value)}
          />
        </AccessKeyInputContainer>
        <section>
          <ActionButton onClick={handleLogin} children='Login'/>
        </section>
      </Content>
      <Footer>
        <AccountFooter />
      </Footer>
    </RegisterContainer>
  )
};

export default Register;