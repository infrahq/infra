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

  & > *:not(:first-child) {
    padding-top: 1.5rem;
  }
`;

const AccessKeyInputContainer = styled.div`
  margin-top: 1.5rem;
`;

const Footer = styled.div`
  position: absolute;
  bottom: 0;
  padding-bottom: 1rem;
`;

const Register = () => {
  const [value, setValue] = useState('');

  const handleLogin = () => {
    console.log('handle login with access key');
  };

  return (
    <RegisterContainer>
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
      <Footer>
        <AccountFooter />
      </Footer>
    </RegisterContainer>
  )
};

export default Register;