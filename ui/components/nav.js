import Link from 'next/link'
import styled from 'styled-components'

import UserDropdown from './UserDropdown'

const NavContainer = styled.div`
  display: flex;
  flex-direction: row;
  justify-content: space-between;
  margin-top: 1rem;
`
const NavOptionsContainer = styled.div`
  a {
    font-style: normal;
    font-weight: 400;
    font-size: 11px;
    line-height: 13px;
    text-transform: uppercase;
    text-decoration: none;
    color: #FFFFFF;
  }

  & > *:not(:first-child):not(:last-child) {
    margin-left: 2.125rem;
  }

  & > *:last-child {
    margin-left: 1.125rem;
  }
`


const Nav = () => {
  return (
    <NavContainer>
      <>
        <div><img src='/brand.svg' /></div>
        <NavOptionsContainer>
          <Link href='/'>
            <a>Access</a>
          </Link>
          <Link href='/infrastructure'>
            <a>Infrastructure</a>
          </Link>
          <Link href='/providers'>
            <a>Identity providers</a>
          </Link>
          <UserDropdown />
        </NavOptionsContainer>
      </>
    </NavContainer>
  )
}

export default Nav
