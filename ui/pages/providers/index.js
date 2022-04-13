import useSWR from "swr";

import Head from 'next/head'
import Router from 'next/router'
import { useContext, useEffect, useState } from 'react'
import styled from 'styled-components'
import axios from 'axios'
import Link from 'next/link'

import Navigation from '../../components/nav/Navigation'
import PageHeader from '../../components/PageHeader'

import AuthContext from '../../store/AuthContext'
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

const Providers = () => {  
  // const getProvidersList = '/v1/providers'
  // const getProviders = async (url) => await axios.get(url).then((response) => { console.log(response); response.dara })
  const { providers, updateProviders } = useContext(AuthContext)
  const [currentProviders, setCurrentProviders] = useState([])
  // const { data, error } = useSWR(getProvidersList, getProviders)
  

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

  // if (error) return <div>Failed to load</div>
  // if (!providers) return <div>Loading...</div>

  // return (
  //   <>
  //     <Navigation />
  //     {data ? <>
  //       {data.map((item) => {
  //         return (
  //           <TableContent key={item.id}>
  //             <IdentityProvider type='okta' name={item.name} />
  //             <TableContentText>{item.url}</TableContentText>
  //             <TableContentText>
  //               <FormattedTime time={item.created} />
  //             </TableContentText>
  //           </TableContent>
  //         )
  //       })}
  //     </>
  //     : <div>Loading...</div>}
  //   </>
  // )

  const handleConnectProviders = async () => {
    await Router.push({
      pathname: '/providers/add/select'
    }, undefined, { shallow: true })
  }

  // TODO: /v1/providers is 404?!!@@
  const handleRemoveProvider = (providerId) => {
    axios.delete('/v1/providers', { data: { id: providerId } })
      .then((response) => {
        console.log(response)
      })
      .catch((error) => {
        console.log(error)
      })
  }

  return (
    <div>
      <Head>
        <title>Infra - Providers</title>
      </Head>
      <Navigation />
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
          <TableHeaderTitle></TableHeaderTitle>
        </TableHeader>
        <div>
          {currentProviders && currentProviders.length > 0
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
    </div>
  )
}

export default Providers
