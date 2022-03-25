import Link from 'next/link'
import { useRouter } from 'next/router'
import styled, { css } from 'styled-components'

import UserDropdown from './UserDropdown'

const NavContainer = styled.div`
  display: grid;
  grid-template-rows: auto 10%;
  border-right: .1rem solid rgba(178,178,178, .1);
  min-height: 100vh;
`

const NavLogo = styled.div`
  padding-top: 3rem;
  padding-left: 21px;
  padding-bottom: 2.5rem;
`

const NavContent = styled.div`
  & > *:not(:first-child) {
    padding-top: 2rem;
  }
`

const NavSubTitle = styled.span`
  padding-left: 21px;
  font-weight: 400;
  font-size: 10px;
  line-height: 0%;
  color: #838383;
  text-transform: uppercase;
`

const NavTitlesGroup = styled.div`
  & > * {
    margin-top: .5rem;
    padding-top: 1rem;
  }
`

const NavItem = styled.div`
  cursor: pointer;

  &:hover {
    display: flex;
    background: #20262C;
    border-radius: 4px;
    width: 95%;
    height: 30px;
  }

  & > *:not(:first-child) {
    padding-left: .75rem;
  }
  
  ${props =>
    props.selected && css`
      a {
        color: #FFFFFF;
      }

      opacity: 1;
      display: flex;
      background: #2F363D;
      border-radius: 4px;
      width: 95%;
      height: 30px;
  `}
`

const NavTitle = styled.a`
  font-weight: 400;
  font-size: 12px;
  line-height: 15px;
  color: #B2B2B2;
`

const NavImg = styled.img`
  width: 1rem;
  height: 1rem;
  vertical-align: middle;
  padding-left: 21px;
`

const NavFooter = styled.div`
  display: flex;
  flex-direction: column-reverse;
  padding-bottom: 20px;
`

const Nav = () => {
  const page = Object.freeze({ access: '/', infrastructure: '/infrastructure', providers: '/providers', users: '/local-user' })

  const router = useRouter()
  const pathname = router.pathname

  return (
    <NavContainer>
      <div>
        <NavLogo><img src='/brand.svg' /></NavLogo>
        <NavContent>
          <div>
            <NavSubTitle>Administration</NavSubTitle>
            <NavTitlesGroup>
              <Link href='/'>
                <NavItem selected={pathname === page.access}>
                  <NavImg src='/access.svg' />
                  <NavTitle>Access</NavTitle>
                </NavItem>
              </Link>
            </NavTitlesGroup>
          </div>
          <div>
            <NavSubTitle>Identities</NavSubTitle>
            <NavTitlesGroup>
              <Link href='/providers'>
                <NavItem selected={pathname === page.providers}>
                  <NavImg src='/identity-providers.svg' />
                  <NavTitle>Identity Providers</NavTitle>
                </NavItem>
              </Link>
              <Link href='/local-user'>
                <NavItem selected={pathname === page.users}>
                  <NavImg src='/local-users.svg' />
                  <NavTitle>Identities</NavTitle>
                </NavItem>
              </Link>
            </NavTitlesGroup>
          </div>
          <div>
            <NavSubTitle>Resources</NavSubTitle>
            <NavTitlesGroup>
              <Link href='/infrastructure'>
                <NavItem selected={pathname === page.infrastructure}>
                  <NavImg src='/infrastructure.svg' />
                  <NavTitle>Infrastructure</NavTitle>
                </NavItem>
              </Link>
            </NavTitlesGroup>
          </div>
        </NavContent>
      </div>
      <NavFooter>
        <UserDropdown />
      </NavFooter>
    </NavContainer>
  )
}

export default Nav
