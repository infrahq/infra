import styled from 'styled-components'
import PropTypes from 'prop-types'

const IdentityProviderButtonContainer = styled.div`
  & > *:not(:first-child) {
    margin-top: .3rem;
  }
`

const IdentityProviderContainer = styled.button`
  width: 24rem;
  height: 3rem;
  background: rgba(255,255,255,0.02);
  opacity: 1;
  border-radius: .25rem;
  border: none;
  cursor: pointer;
  color: #FFFFFF;

  &:hover { opacity: .95 }
`

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
  font-weight: 100;
  font-size: .75rem;
  line-height: 1rem;
  text-transform: capitalize;
`

const DescriptionSubheader = styled.div`
  font-weight: 100;
  font-size: .5rem;
  line-height: .75rem;
  text-transform: uppercase;
  color: #FFFFFF;
  opacity: 0.3;
`

const IdentityProviderButton = ({ providers }) => {
  return (
    <IdentityProviderButtonContainer>
      {providers.map((provider, index) => {
        return (
          <IdentityProviderContainer
            key={index}
            onClick={() => provider.onClick()}
          >
            <IdentityProviderContentContainer>
              <IdentityProviderLogo>
                <img src={`/${provider.type}.svg`} />
              </IdentityProviderLogo>
              <IdentityProviderContentDescriptionContainer>
                <DescriptionHeader>{provider.type}</DescriptionHeader>
                <DescriptionSubheader>{provider.name}</DescriptionSubheader>
              </IdentityProviderContentDescriptionContainer>
            </IdentityProviderContentContainer>
          </IdentityProviderContainer>
        )
      })}
    </IdentityProviderButtonContainer>
  )
}

IdentityProviderButton.prototype = {
  providers: PropTypes.arrayOf(PropTypes.shape({
    type: PropTypes.string,
    name: PropTypes.string,
    url: PropTypes.string,
    clientID: PropTypes.string,
    id: PropTypes.string,
    created: PropTypes.number,
    updated: PropTypes.number,
    onClick: PropTypes.func
  })).isRequired
}

export default IdentityProviderButton
