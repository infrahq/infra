import styled from "styled-components";
import Link from 'next/link';

const AccountFooterContainer = styled.section`
  display: flex;
  flex-direction: row;
  justify-content: space-between;
  width: 24rem;
  height: .6875rem;
  font-weight: 100;
  font-size: 9px;
  line-height: 11px;
  align-items: center;
  text-align: center;
  color: #FFFFFF;
  opacity: .2;

  a {
    color: #FFFFFF;
    text-decoration: none;

    :hover {
      opacity: .75;
    }
  }
`;

const AccountFooter = () => {
  return (
    <AccountFooterContainer>
      <div>© {new Date().getUTCFullYear()} Infra Technologies, Inc. All rights reserved. </div>
      <Link href='/'>
        <a>Privacy Policy</a>
      </Link>
      <Link href='/'>
        <a>Terms of Use</a>
      </Link>
    </AccountFooterContainer>
  )
};

export default AccountFooter;