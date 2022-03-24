import Router from 'next/router'
import { useContext, useEffect, useState } from 'react'
import styled from 'styled-components'
import axios from 'axios'

import Navigation from '../components/nav/Navigation'
import PageHeader from '../components/PageHeader'

import AuthContext from '../store/AuthContext'
import FormattedTime from '../components/FormattedTime'
import IdentityProvider from '../components/IdentityProvider'
import EmptyPageHeader from '../components/EmptyPageHeader'

const ProvidersHeaderContainer = styled.div`
  padding-top: 3rem;
  padding-bottom: 3rem;
`

const TableHeader = styled.div`
  display: grid;
  opacity: 0.5;
  border-bottom: 1px solid rgba(255, 255, 255, 0.2);
  grid-template-columns: 25% auto 10%;
  align-items: center;
`

const TableHeaderTitle = styled.p`
  font-style: normal;
  font-weight: 400;
  font-size: 11px;
  line-height: 0%;
  text-transform: uppercase;

  img {
    width: 15px;
    height: 15px;
  }
`

const TableContentContainer = styled.div`
  padding-top: 1rem;
`

const TableContent = styled.div`
  display: grid;
  grid-template-columns: 25% auto 10%;
  align-items: center;
`

const TableContentText = styled.div`
  font-weight: 300;
  font-size: 12px;
  line-height: 0px;
`

const Providers = () => {
  const { providers, updateProviders } = useContext(AuthContext)

  const [currentProviders, setCurrentProviders] = useState([])

  useEffect(() => {
    if (providers.length === 0) {
      axios.get('/v1/providers')
        .then((response) => {
          const idpList = response.data.filter((item) => item.name !== 'infra')
          setCurrentProviders(idpList)
          updateProviders(idpList)
        })
        .catch((error) => {
          console.log(error)
        })
    } else {
      setCurrentProviders(providers)
    }
  }, [])

  const handleConnectProviders = async () => {
    await Router.push({
      pathname: '/providers/add/select'
    }, undefined, { shallow: true })
  }

  return (
    <div>
      <Navigation />
      <div>
        <ProvidersHeaderContainer>
          <PageHeader iconPath='/identity-providers.svg' title='Identity Providers' />
        </ProvidersHeaderContainer>
        <TableHeader>
          <TableHeaderTitle>Identity Provider</TableHeaderTitle>
          <TableHeaderTitle>Domain</TableHeaderTitle>
          <TableHeaderTitle>Added</TableHeaderTitle>
        </TableHeader>
        <div>
          {currentProviders.length > 0
            ? (
              <TableContentContainer>
                {currentProviders.map((item) => {
                  return (
                    <TableContent key={item.id}>
                      <IdentityProvider type='okta' name={item.name} />
                      <TableContentText>{item.url}</TableContentText>
                      <TableContentText>
                        <FormattedTime time={item.created} />
                      </TableContentText>
                    </TableContent>
                  )
                })}
              </TableContentContainer>
              )
            : (
              <EmptyPageHeader
                header='Identity Providers'
                subheader='No identity providers connected.'
                actionButtonHeader='Connect Identity Providers'
                onClickActionButton={() => handleConnectProviders()}
              />
              )}
        </div>
      </div>
    </div>
  )
}

export default Providers
