import useSWR from 'swr'
import Head from 'next/head'
import Link from 'next/link'
import styled from 'styled-components'
import Router from 'next/router'

import Navigation from '../../components/nav/Navigation'
import PageHeader from '../../components/PageHeader'
import FormattedTime from '../../components/FormattedTime'
import EmptyPageHeader from '../../components/EmptyPageHeader'

const DestinationsHeaderContainer = styled.div`
  padding-top: 3rem;
  padding-bottom: 3rem;
  display: flex;
  flex-direction: row;
  justify-content: space-between;
`

const AddDestinationLink = styled.a`
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
  grid-template-columns: 80% 20%;
  align-items: center;
`

const TableHeaderTitle = styled.p`
  font-style: normal;
  font-weight: 400;
  font-size: 11px;
  line-height: 0%;
  text-transform: uppercase;
`

const TableContent = styled.div`
  display: grid;
  grid-template-columns: 80% 20%;
  align-items: center;
  height: 2rem;
  cursor: pointer;
`

const TableContentLink = styled.button`
  border: none;
  cursor: pointer;
`

const TableContentText = styled.div`
  font-weight: 300;
  font-size: 12px;
  line-height: 0px;

  a {
    cursor: pointer;

    :hover {
      opacity: .6;
    }
  }
`

const TableContentContainer = styled.div`
  padding-top: 1rem;
`

export const getDestinationsList = () => {
  const getDestinationsList = '/v1/destinations'
  const getDestinations = url => fetch(url).then(response => response.json())
  const { data, error } = useSWR(getDestinationsList, getDestinations)
  
  return {
    destinations: data,
    isLoading: !error && !data,
    isError: error
  }
}

const Destinations = () => {
  const { destinations, isLoading, isError } = getDestinationsList()

  const handleAddDestination = () => {
    Router.push({
      pathname: '/destinations/add/connect'
    }, undefined, { shallow: true })
  }

  const handleDestinationDetail = (id) => {
    console.log('id:', id)
    Router.push({
      pathname: `/destinations/details/${id}`
    }, undefined, {shallow: true})
  }

  return (
    <div>
      <Head>
        <title>Infra - Destinations</title>
      </Head>
      <Navigation />
      <div>
        <DestinationsHeaderContainer>
          <PageHeader iconPath='/destinations.svg' title='Destinations' />
          <Link href='/destinations/add/connect'>
            <AddDestinationLink><span>&#43;</span>Add Destination</AddDestinationLink>
          </Link>
        </DestinationsHeaderContainer>
        <TableHeader>
          <TableHeaderTitle>Name</TableHeaderTitle>
          <TableHeaderTitle>Added</TableHeaderTitle>
        </TableHeader>
        <div>
          {destinations && destinations.length > 0
          ? (
            <TableContentContainer>
              {destinations.map((item) => {
              return (
                <TableContent key={item.id}  onClick={() => handleDestinationDetail(item.id)}>
                    <TableContentText>{item.name}</TableContentText>
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
              header='Destinations'
              subheader='No destinations connected.'
              actionButtonHeader='Add Destinations'
              onClickActionButton={() => handleAddDestination()}
            />
            )}
          </div>
      </div>
    </div>
  )
}

export default Destinations
