import { useEffect, useState } from 'react';
import styled from 'styled-components';
import axios from 'axios';
import Router from 'next/router'; 


import AccessKeyInput from '../../components/AccessKeyInput';
import ActionButton from '../../components/ActionButton';
import AccountFooter from '../../components/AccountFooter';
import AccountHeader from '../../components/AccountHeader';
import { useCookies } from 'react-cookie';

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
  const [cookie, setCookie] = useCookies(['accessKey']);
  
  useEffect(() => {
    if(!!cookie.accessKey && cookie.accessKey !== undefined) {
      Router.push({
        pathname: '/',
      }, undefined, { shallow: true });
    }
  }, [])

  const handleLogin = async () => {
    // get cookie to access to the api
    // document.cookie = `access_key=${value}`
    setCookie('accessKey', value, { path: '/' });

    await axios.get('/v1/users', { headers: { Authorization: `Bearer ${value}` } })
    .then((response) => {
      Router.push({
        pathname: '/',
      }, undefined, { shallow: true });
    })
    .catch((error) => {
      console.log(error);
    });
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