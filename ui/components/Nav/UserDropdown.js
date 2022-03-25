import styled from 'styled-components'
import { useContext, useEffect, useState, useRef } from 'react'

import AuthContext from '../../store/AuthContext'

const UserDropdownContainer = styled.span`
  position: relative;
`

const UserDropdownButton = styled.button`
  display: flex;
  flex-direction: row;
  border: 0;
  background-color: transparent;
  cursor: pointer;
  opacity: .8;
  padding: 0;
  padding-left: .5rem;

  &:hover {
    display: flex;
    opacity: 1;
    background: #20262C;
    border-radius: 4px;
    width: 95%;
    height: 3rem;
    padding-top: 0.75rem;

  }
`

const UserDropdownHeader = styled.div`
  width: 26px;
  height: 26px;
  background-color: #373C41;
  border-radius: 4px;
  color: #FFFFFF;

  span {
    display: inline-block;
    margin-top: .35rem;
  }
`

const UserNameText = styled.span`
  padding-left: .5rem;
  font-weight: 400;
  font-size: 11px;
  line-height: 13px;
  color: #B2B2B2;
  margin-top: .35rem;
`

const UserDropdownContent = styled.div`
  background: #373C41;
  position: absolute;
  bottom: -20px;
  left: -2rem;
  width: 100%;
  min-height: auto;
  z-index: 991;
  max-height: 300px;
  max-width: calc(100vw);
  padding: 1rem 1.05rem 0px;
`

const UserDropdownContentHeader = styled.div`
  display: flex;
  flex-direction: column;
  padding-bottom: 1rem;
  padding-left: 19px;

  & > *:not(:first-child) {
    padding-top: .5rem
  }
`

const Avatar = styled.div`
  width: 40px;
  height: 40px;
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

const LogoutContainer = styled.div`
  margin-left: -.45rem;
  margin-right: -.45rem;
  padding-left: 19px;
`

const LogoutButton = styled.a`
  display: flex;
  flex-direction: row;
  font-weight: 400;
  font-size: 11px;
  line-height: 13px;
  opacity: .5;
  box-sizing: border-box;
  padding: 6px 6px 9px;
  cursor: pointer;

  span {
    padding-left: 11px;
  }

  &:hover {
    opacity: 1;
  }
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
      <UserDropdownButton onClick={() => setDropdownOpen(!dropdownOpen)}>
        <UserDropdownHeader>
          <span>{iconText}</span>
        </UserDropdownHeader>
        {user && <UserNameText>{user.name}</UserNameText>}
      </UserDropdownButton>
      {dropdownOpen &&
        <UserDropdownContent>
          <UserDropdownContentHeader>
            <Avatar>
              <div>{iconText}</div>
            </Avatar>
            <Content>
              <div>{user.name}</div>
            </Content>
          </UserDropdownContentHeader>
          <LogoutContainer>
            <LogoutButton onClick={() => handleLogout()}>
              <img src='/sign-out.svg' />
              <span>Sign Out</span>
            </LogoutButton>
          </LogoutContainer>
        </UserDropdownContent>}
    </UserDropdownContainer>
  )
}

export default UserDropdown
