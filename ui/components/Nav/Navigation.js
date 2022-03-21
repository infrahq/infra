import Link from 'next/link'
import { useRouter } from 'next/router'
import styled, { css } from 'styled-components'

import UserDropdown from './UserDropdown'

const NavContainer = styled.div`
  display: flex;
  flex-direction: row;
  justify-content: space-between;
  margin-top: 1rem;
`
const NavOptionsContainer = styled.div`
  & > *:not(:first-child):not(:last-child) {
    margin-left: 2.125rem;
  }

  & > *:last-child {
    margin-left: 1.125rem;
  }
`

const NavItem = styled.a`
  font-style: normal;
  font-weight: 400;
  font-size: 11px;
  line-height: 13px;
  text-transform: uppercase;
  text-decoration: none;
  color: #bdc4d1;
  opacity: .6;
  transition: all .2s ease-in;
  cursor: pointer;

  &:hover {
    opacity: 1
  }

  ${props =>
    props.selected && css`
      opacity: 1;
      color: #FFFFFF;
  `}
`

const Navigation = () => {
  const page = Object.freeze({ access: '/', infrastructure: '/infrastructure', provider: '/providers' })

  const router = useRouter()
  const pathname = router.pathname

  return (
    <NavContainer>
      <>
        <div><img src='/brand.svg' /></div>
        <NavOptionsContainer>
          <Link href='/'>
            <NavItem selected={pathname === page.access}>Access</NavItem>
          </Link>
          <Link href='/infrastructure'>
            <NavItem selected={pathname === page.infrastructure}>Infrastructure</NavItem>
          </Link>
          <Link href='/providers'>
            <NavItem selected={pathname === page.provider}>Identity providers</NavItem>
          </Link>
          <UserDropdown />
        </NavOptionsContainer>
      </>
    </NavContainer>
  )
}

export default Navigation
