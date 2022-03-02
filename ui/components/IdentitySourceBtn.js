import styled from 'styled-components'
import PropTypes from 'prop-types'

const IdentitySourceBtnContainer = styled.div`
  & > *:not(:first-child) {
    margin-top: .3rem;
  }
`

const IdentitySourceContainer = styled.button`
  width: 24rem;
  height: 3rem;
  background: rgba(255,255,255,0.02);
  opacity: ${props => props.disabled != null && props.disabled ? '.56' : '1'};
  border-radius: .25rem;
  border: none;
  cursor: ${props => props.disabled != null && props.disabled ? 'default' : 'pointer'};
  color: #FFFFFF;

  ${props => props.disabled != null && props.disabled
    ? ''
    : '&:hover { opacity: .95 }'
  }
`

const IdentitySourceContentContainer = styled.div`
  display: flex;
  flex-direction: row;
  padding: .5rem;
`

const IdentitySourceLogo = styled.div`
  padding-top: .4rem;  
`

const IdentitySourceContentDescriptionContainer = styled.div`
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

const IdentitySourceBtn = ({ providers}) => {
  return (
    <IdentitySourceBtnContainer>
      {providers.map((provider, index) => {
        return (
          <IdentitySourceContainer
            key={index}
            onClick={() => provider.onClick()}
            disabled={provider.disabled || false}
          >
            <IdentitySourceContentContainer>
              <IdentitySourceLogo>
                <img src={`/${provider.type}.svg`} />
              </IdentitySourceLogo>
              <IdentitySourceContentDescriptionContainer>
                <DescriptionHeader>{provider.type}</DescriptionHeader>
                <DescriptionSubheader>{provider.name}</DescriptionSubheader>
              </IdentitySourceContentDescriptionContainer>
            </IdentitySourceContentContainer>
          </IdentitySourceContainer>
        )
      })}
    </IdentitySourceBtnContainer>
  )
}

IdentitySourceBtn.prototype = {
  providers: PropTypes.arrayOf(PropTypes.shape({
    type: PropTypes.string,
    name: PropTypes.string,
    url: PropTypes.string,
    clientID: PropTypes.string,
    id: PropTypes.string,
    created: PropTypes.number,
    updated: PropTypes.number,
    onClick: PropTypes.func,
    disabled: PropTypes.bool
  })).isRequired,
}

export default IdentitySourceBtn
