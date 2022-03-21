import styled from 'styled-components'
import PropTypes from 'prop-types'

const IdentityProviderContentContainer = styled.div`
  display: flex;
  flex-direction: row;
  padding: .5rem;
`

const IdentityProviderLogo = styled.div`
  padding-top: .4rem;  
`

const IdentityProviderContentDescriptionContainer = styled.div`
  padding-left: 1rem;
  text-align: left;

  & > *:not(:first-child) {
    padding-top: .15rem;
  }
`

const DescriptionHeader = styled.div`
  font-weight: 300;
  font-size: .75rem;
  line-height: 1rem;
  text-transform: capitalize;
`

const DescriptionSubheader = styled.div`
  font-weight: 300;
  font-size: .5rem;
  line-height: .75rem;
  text-transform: uppercase;
  color: #FFFFFF;
  opacity: 0.3;
`

const IdentityProvider = ({ type, name }) => {
  return (
    <IdentityProviderContentContainer>
      <IdentityProviderLogo>
        <img src={`/${type}.svg`} />
      </IdentityProviderLogo>
      <IdentityProviderContentDescriptionContainer>
        <DescriptionHeader>{type}</DescriptionHeader>
        <DescriptionSubheader>{name}</DescriptionSubheader>
      </IdentityProviderContentDescriptionContainer>
    </IdentityProviderContentContainer>
  )
}

IdentityProvider.prototype = {
  type: PropTypes.string.isRequired,
  name: PropTypes.string.isRequired
}

export default IdentityProvider