import Head from 'next/head'
import Link from 'next/link'
import styled from 'styled-components'

import Navigation from '../../components/nav/Navigation'
import PageHeader from '../../components/PageHeader'
import { DestinationsContextProvider } from '../../store/DestinationsContext'
import Dashboard from '../../components/destinations/Dashboard'

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
  grid-template-columns: 80% 18% auto;
  align-items: center;
`

const TableHeaderTitle = styled.p`
  font-style: normal;
  font-weight: 400;
  font-size: 11px;
  line-height: 0%;
  text-transform: uppercase;
`

const Destinations = () => {
  return (
    <DestinationsContextProvider>
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
          <Dashboard />
        </div>
    </DestinationsContextProvider>
  )
}

export default Destinations
