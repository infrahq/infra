import styled from 'styled-components'

import Header from '../Header'
import Logo from './Logo'
import Input from '../../Input'

const HelpContainer = styled.div`
  font-weight: 100;
  font-size: 11px;
  line-height: 13px;
  max-width: 24rem;

  span {
    opacity: .5;
  }

  a {
    padding-left: .5rem;
    color: #93DEFF;
    text-decoration: none;

    :hover {
      opacity: .95;
      text-decoration: underline;
    }
  }
`

const InputList = styled.div`
  & > *:not(:first-child) {
    padding-top: .75rem;
  }
`

const Setup = ({ value, parentCallback }) => {
  return (
    <>
      <Header
        header='Connect Okta'
        subheader='Apply your Okta credentials in order to sync your users to Infra.'
      />
      <Logo />
      <HelpContainer>
        <span>To find these values</span>
        <a
          target='_blank'
          rel='noreferrer'
          href='https://github.com/infrahq/infra/blob/main/docs/providers/okta.md'
        >
          click here
        </a>
      </HelpContainer>
      <InputList>
        <div>
          <Input
            label='Name of Provider'
            value={value.name}
            onChange={e => parentCallback(e.target.value, 'name')}
          />
        </div>
        <div>
          <Input
            label='Okta Domain'
            value={value.domain}
            onChange={e => parentCallback(e.target.value, 'domain')}
          />
        </div>
        <div>
          <Input
            label='Okta Client ID'
            value={value.clientId}
            onChange={e => parentCallback(e.target.value, 'clientId')}
          />
        </div>
        <div>
          <Input
            label='Okta Client Secret'
            value={value.clientSecret}
            onChange={e => parentCallback(e.target.value, 'clientSecret')}
          />
        </div>

      </InputList>
    </>
  )
}

export default Setup
