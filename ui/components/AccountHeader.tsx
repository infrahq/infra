import styled from 'styled-components'

interface Header {
  header: string,
  subheader: string
}

const HeaderContainer = styled.div`
  & > *:not(:first-child) {
    padding-top: 1.5rem;
  }
`

const LogoContainer = styled.div`
  text-align: center;
`

const StyledHeader = styled.div`
  font-size: 1.375rem;
  line-height: 1.7rem;
  letter-spacing: -0.035em;
  font-weight: 200;
  text-align: center;
`

const StyledSubheader = styled.div`
  font-weight: 100;
  font-size: .6875rem;
  line-height: 156.52%;
  opacity: .5;
  text-align: center;
  padding: 0 1rem;
`

const AccountHeader = ({ header, subheader }:Header) => {
  return (
    <HeaderContainer>
      <LogoContainer>
        <img src='/infraIcon.svg' />
      </LogoContainer>
      <StyledHeader>{header}</StyledHeader>
      <StyledSubheader>{subheader}</StyledSubheader>
    </HeaderContainer>
  )
}

export default AccountHeader
