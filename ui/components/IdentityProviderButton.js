import styled from 'styled-components'
import PropTypes from 'prop-types'

import IdentityProvider from './IdentityProvider'

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

const IdentityProviderButton = ({ providers }) => {
  return (
    <IdentityProviderButtonContainer>
      {providers.map((provider, index) => {
        return (
          <IdentityProviderContainer
            key={index}
            onClick={() => provider.onClick()}
          >
            <IdentityProvider type={provider.type} name={provider.name} />
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
    onClick: PropTypes.func
  })).isRequired
}

export default IdentityProviderButton
