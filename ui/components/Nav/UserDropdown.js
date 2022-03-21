import styled from 'styled-components'
import { useContext, useEffect, useState, useRef } from 'react'

import AuthContext from '../../store/AuthContext'

const UserDropdownContainer = styled.span`
  position: relative;
`

const UserDropdownHeader = styled.button`
  width: 26px;
  height: 26px;
  background-color: #373C41;
  border: 0;
  border-radius: 4px;
  color: #FFFFFF;
  cursor: pointer;
  opacity: .8;

  &:hover {
    opacity: 1;
  }
`

const UserDropdownContent = styled.div`
  background: #373C41;
  border-radius: 4px;
  position: absolute;
  top: 0;
  width: 183px;
  min-height: auto;
  z-index: 991;
  margin-top: 2rem;
  right: 0;
  max-height: 117px;
  max-width: calc(-24px + 100vw);
  padding: 11px 7px 0px;
`

const UserDropdownContentHeader = styled.div`
  display: flex;
  flex-direction: row;
  padding-bottom: 1rem;

  & > *:not(:first-child) {
    padding-left: .5rem
  }
`

const Avatar = styled.div`
  width: 36px;
  height: 36px;
  background-color: #505559;
  border: 0;
  border-radius: 4px;
  color: #FFFFFF;
  display: flex;
  justify-content: center;
  align-items: center;
`

const Content = styled.div`
  font-weight: 400;
  font-size: 11px;
  line-height: 13px;
  padding-top: .25rem;
`

const Role = styled.div`
  color: rgba(255, 255, 255, 0.48);
`

const LogoutContainer = styled.div`
  border-top: 1px solid rgba(255, 255, 255, .1);
  margin-left: -.45rem;
  margin-right: -.45rem
`

const LogoutButton = styled.a`
  font-weight: 400;
  font-size: 11px;
  line-height: 13px;
  text-transform: uppercase;
  opacity: .5;
  box-sizing: border-box;
  display: block;
  height: 28px;
  padding: 6px 6px 9px;
  white-space: nowrap;
  width: 100%;
  cursor: pointer;
`

const UserDropdown = () => {
  const { user, logout } = useContext(AuthContext)
  const [iconText, setIconText] = useState(null)
  const [dropdownOpen, setDropdownOpen] = useState(false)

  const wrapperRef = useRef(null)

  const useOutsideAlerter = (ref) => {
    useEffect(() => {
      const handleClickOutside = (event) => {
        if (ref.current && !ref.current.contains(event.target)) {
          setDropdownOpen(false)
        }
      }
      document.addEventListener('mousedown', handleClickOutside)
      return () => {
        document.removeEventListener('mousedown', handleClickOutside)
      }
    }, [ref])
  }

  useOutsideAlerter(wrapperRef)

  useEffect(() => {
    if (user != null) {
      getIconText(user.name)
    }
  }, [])

  const getIconText = (name) => {
    setIconText(name[0].toUpperCase())
  }

  const handleLogout = async () => {
    setDropdownOpen(false)
    await logout()
  }

  return (
    <UserDropdownContainer ref={wrapperRef}>
      <UserDropdownHeader onClick={() => setDropdownOpen(!dropdownOpen)}>
        {iconText}
      </UserDropdownHeader>
      {dropdownOpen &&
        <UserDropdownContent>
          <UserDropdownContentHeader>
            <Avatar>
              <div>{iconText}</div>
            </Avatar>
            <Content>
              <div>{user.name}</div>
              <Role>{user.identityType}</Role>
            </Content>
          </UserDropdownContentHeader>
          <LogoutContainer>
            <LogoutButton onClick={() => handleLogout()}>LOGOUT</LogoutButton>
          </LogoutContainer>
        </UserDropdownContent>}
    </UserDropdownContainer>
  )
}

export default UserDropdown
