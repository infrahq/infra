import styled from 'styled-components'
import PropTypes from 'prop-types'

const LogoContainer = styled.div`
  text-align: center;
`

const StyledHeader = styled.div`
  font-weight: 400;
  font-size: 22px;
  line-height: 27px;
  text-align: center;
  letter-spacing: -0.035em;
`

const StyledSubheader = styled.div`
  font-weight: 400;
  font-size: 11px;
  line-height: 156.52%;
  opacity: .5;
  text-align: center;
  padding: .5rem .5rem 1rem .5rem;
`

const StyledTitle = styled.div`
  font-weight: 700;
  font-size: 11px;
  line-height: 156.52%;
  text-align: center;
  padding: 1.5rem .5rem 1rem;
`

const AccountHeader = ({ header, subheader, title }) => {
  return (
    <>
      <LogoContainer>
        <img src='/infra-icon.svg' />
      </LogoContainer>
      <StyledHeader>{header}</StyledHeader>
      <StyledTitle>{title}</StyledTitle>
      <StyledSubheader>{subheader}</StyledSubheader>
    </>
  )
}

AccountHeader.prototype = {
  header: PropTypes.string,
  title: PropTypes.string,
  subheader: PropTypes.string
}

export default AccountHeader
