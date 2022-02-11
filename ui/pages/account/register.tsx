import { useState } from 'react';
import styled from 'styled-components';

import AccessKeyInput from '../../components/AccessKeyInput';
import ActionButton from '../../components/ActionButton';
import AccountFooter from '../../components/AccountFooter';
import AccountHeader from '../../components/AccountHeader';

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
  const [value, setValue] = useState('');

  const handleLogin = () => {
    console.log('handle login with access key');
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