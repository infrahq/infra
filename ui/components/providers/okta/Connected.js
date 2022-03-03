import styled from 'styled-components'
import PropTypes from 'prop-types'

import Header from '../Header'
import Logo from './Logo'
import IdentitySourceBtn from '../../IdentitySourceBtn'

const ConnectedStatus = styled.div`
  font-weight: normal;
  font-size: 10px;
  line-height: 12px;
  text-transform: uppercase;
  padding-bottom: .5rem;

  opacity: 0.4;
`

const ImageContainer = styled.div`
  display: flex;
  flex-direction: row;
  justify-content: space-between;
  width: 75%;
`

const ConnectedIcon = styled.img`
  width: 89.16px;
  height: 18.1px;
`

const Connected = ({ provider }) => {
  return (
    <>
      <Header
        header='Connection Successful'
        subheader='Apply your Okta credentials in order to sync your users to Infra.'
      />
      <div>
        <ConnectedStatus>connection status</ConnectedStatus>
        <ImageContainer>
          <ConnectedIcon src='/connectedIcon.svg' />
          <Logo />
        </ImageContainer>
      </div>
      <IdentitySourceBtn providers={[provider]} />
    </>
  )
}

Connected.prototype = {
  provider: PropTypes.shape({
    type: PropTypes.string,
    name: PropTypes.string,
    url: PropTypes.string,
    clientID: PropTypes.string,
    id: PropTypes.string,
    created: PropTypes.number,
    updated: PropTypes.number,
    onClick: PropTypes.func,
    disabled: PropTypes.bool
  }).isRequired
}

export default Connected
