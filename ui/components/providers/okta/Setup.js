import styled from 'styled-components'
import Link from 'next/link'

import Header from "../Header"
import Logo from "./Logo"
import { useState } from 'react'
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
    }
  }
`

const Setup = ({ parentCallback }) => {
  const [name, setName] = useState('Okta')
  const [domain, setDomain] = useState('')
  const [clientId, setClientId] = useState('')
  const [clientSecret, setClientSecret] = useState('')

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
      <div>
        <Input
          label='Name of Provider'
          value={name}
          onChange={(e) => {
            setName(e.target.value)
            parentCallback(e.target.value, 'name')
          }}
        />
        <Input
          label='Okta Domain'
          value={domain}
          onChange={(e) => {
            setDomain(e.target.value)
            parentCallback(e.target.value, 'domain')
          }}
        />
        <Input
          label='Okta Client ID'
          value={clientId}
          onChange={(e) => {
            setClientId(e.target.value)
            parentCallback(e.target.value, 'id')
          }}
        />
        <Input
          label='Okta Client Secret'
          value={clientSecret}
          onChange={(e) => {
            setClientSecret(e.target.value)
            parentCallback(e.target.value, 'secret')
          }}
        />

      </div>
    </>
  )
}

export default Setup