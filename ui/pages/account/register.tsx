import styled from 'styled-components';
import { faUnlockKeyhole } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome'

const RegisterContainer = styled.section`
  margin-left: auto;
  margin-right: auto;
  max-width: 24rem;
  padding-top: 1.5rem;

`;

const RegisterHeader = styled.h1`
  font-size: 1.375rem;
  line-height: 1.7rem;
  letter-spacing: -0.035em;
  font-weight: normal;
`;

const RegisterDescription = styled.p`
  font-weight: normal;
  font-size: .6875rem;
  line-height: 156.52%;
  opacity: .5;
`;

const Register = () => {
  return (
    <RegisterContainer>
      <RegisterHeader>
        Infra Admin API Access Key
      </RegisterHeader>
      <RegisterDescription>
        Securely manage access to your infrastructure. Take a moment to create your account and start managing access today.
      </RegisterDescription>
      <FontAwesomeIcon icon={faUnlockKeyhole} />
    </RegisterContainer>
  )
};

export default Register;