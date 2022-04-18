import useSWR, { useSWRConfig } from 'swr'
import Head from 'next/head'
import Router from 'next/router'
import styled from 'styled-components'
import Link from 'next/link'

import Dashboard from '../../components/dashboard'
import PageHeader from '../../components/PageHeader'
import FormattedTime from '../../components/FormattedTime'
import IdentityProvider from '../../components/IdentityProvider'
import EmptyPageHeader from '../../components/EmptyPageHeader'

const ProvidersHeaderContainer = styled.div`
  padding-top: 3rem;
  padding-bottom: 3rem;
  display: flex;
  flex-direction: row;
  justify-content: space-between;
`

const AddProviderLink = styled.a`
  font-style: normal;
  font-weight: 400;
  font-size: 11px;
  line-height: 0%;
  text-transform: uppercase;
  cursor: pointer;
  transition: all .2s ease-in;
  opacity: 1;

  span {
    margin-right: .25rem;
  }

  :hover {
    opacity: .6;
  }
`

const TableHeader = styled.div`
  display: grid;
  opacity: 0.5;
  border-bottom: 1px solid rgba(255, 255, 255, 0.2);
  grid-template-columns: 25% auto 10% 5%;
  align-items: center;
`

const TableHeaderTitle = styled.p`
  font-style: normal;
  font-weight: 400;
  font-size: 11px;
  line-height: 0%;
  text-transform: uppercase;
`

const TableContentContainer = styled.div`
  padding-top: 1rem;
`

const TableContent = styled.div`
  display: grid;
  grid-template-columns: 25% auto 10% 5%;
  align-items: center;
`

const TableContentText = styled.div`
  font-weight: 300;
  font-size: 12px;
  line-height: 0px;
`

const ProviderRemoveButton = styled.a`

`

export default function () {
  const { data } = useSWR(() => '/v1/providers')
  const { mutate } = useSWRConfig()

  const handleConnectProviders = async () => {
    await Router.push({
      pathname: '/providers/add/select'
    }, undefined, { shallow: true })
  }

  const handleRemoveProvider = (providerId) => {
    fetch(`/v1/providers/${providerId}`, {
      method: 'DELETE'
    })
      .then(() => {
        mutate('/v1/providers')
      })
      .catch((error) => {
        console.log(error)
      })
  }

  return (
    <Dashboard>
      <Head>
        <title>Providers - Infra</title>
      </Head>
      <div>
        <ProvidersHeaderContainer>
          <PageHeader iconPath='/identity-providers.svg' title='Identity Providers' />
          <Link href='/providers/add/select'>
            <AddProviderLink><span>&#43;</span>Add Provider</AddProviderLink>
          </Link>
        </ProvidersHeaderContainer>
        <TableHeader>
          <TableHeaderTitle>Identity Provider</TableHeaderTitle>
          <TableHeaderTitle>Domain</TableHeaderTitle>
          <TableHeaderTitle>Added</TableHeaderTitle>
          <TableHeaderTitle />
        </TableHeader>
        <div>
          {data && data.length > 0
            ? (
              <TableContentContainer>
                {data.map((item) => {
                  return (
                    <TableContent key={item.id}>
                      <IdentityProvider type='okta' name={item.name} />
                      <TableContentText>{item.url}</TableContentText>
                      <TableContentText>
                        <FormattedTime time={item.created} />
                      </TableContentText>
                      <TableContentText>
                        <ProviderRemoveButton onClick={() => handleRemoveProvider(item.id)}>&#10005;</ProviderRemoveButton>
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
    </Dashboard>
  )
}
