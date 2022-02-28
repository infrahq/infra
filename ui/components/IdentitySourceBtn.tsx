import { useContext } from 'react'
import styled from 'styled-components'
import AuthContext, { ProviderField } from '../store/AuthContext'

export interface IdentitySourceProvider {
  type: string
  name?: string
  url?: string
  clientID?: string
  id?: string
  created?: number
  updated?: number
}

interface IdentitySourceBtnField {
  providers: IdentitySourceProvider[]
  onClick?: () => void
}

const IdentitySourceBtnContainer = styled.div`
  & > *:not(:first-child) {
    margin-top: .3rem;
  }
`

const IdentitySourceContainer = styled.button`
  width: 24rem;
  height: 3rem;
  background: rgba(255,255,255,0.02);
  opacity: ${props => props.disabled ? '.56' : '1'};
  border-radius: .25rem;
  border: none;
  cursor: ${props => (props?.disabled) ? 'default' : 'pointer'};
  color: #FFFFFF;

  ${props => props.disabled
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

const IdentitySourceBtn = ({ providers }: IdentitySourceBtnField): JSX.Element => {
  const { login } = useContext(AuthContext)

  return (
    <IdentitySourceBtnContainer>
      {providers.map((provider, index) => {
        return (
          <IdentitySourceContainer
            key={index}
            onClick={!!provider && !provider.url ? undefined : () => login(provider as ProviderField)}
            disabled={!!provider && !provider.url}
          >
            <IdentitySourceContentContainer>
              <IdentitySourceLogo>
                <img src={`/${provider.type}.svg`} />
              </IdentitySourceLogo>
              <IdentitySourceContentDescriptionContainer>
                <DescriptionHeader>{provider.type}</DescriptionHeader>
                <DescriptionSubheader>{provider.name ? provider.name : 'Identity Source'}</DescriptionSubheader>
              </IdentitySourceContentDescriptionContainer>
            </IdentitySourceContentContainer>
          </IdentitySourceContainer>
        )
      })}
    </IdentitySourceBtnContainer>
  )
}

export default IdentitySourceBtn
