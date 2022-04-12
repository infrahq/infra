import Head from "next/head"
import Router from 'next/router'
import { useState } from "react"
import styled from 'styled-components'

import ActionButton from "../../../components/ActionButton"
import NameInput from "../../../components/destinations/NameInput"
import CommandInput from "../../../components/destinations/CommandInput"
import ExitButton from "../../../components/ExitButton"
import Header from "../../../components/Header"
import ConnectStatus from "../../../components/destinations/ConnectStatus"

import { DestinationsContextProvider } from "../../../store/DestinationsContext"

const ConnectContainer = styled.section`
  position: relative;
`

const NavButton = styled.div`
  position: absolute;
  top: .5rem;
  right: .5rem;
`

const SetupDestinationContent = styled.div`
  margin-left: auto;
  margin-right: auto;
  max-width: 24rem;
  padding-top: 1.5rem;

  & > *:not(:first-child) {
    padding-top: 1.75rem;
  }
`

const Connect = () => {
  return (
    <DestinationsContextProvider>
      <Head>
        <title>Infra - Destinations</title>
      </Head>
      <ConnectContainer>
        <SetupDestinationContent>
          <Header 
            header='Connect Your Kubernetes Cluster'
            subheader='Run the following command to connect your cluster'
          />
          <NameInput />
          <CommandInput />
          <ConnectStatus />
        </SetupDestinationContent>      
        <NavButton>
          <ExitButton previousPage='/destinations' />
        </NavButton>
      </ConnectContainer>
    </DestinationsContextProvider>
  )
}

export default Connect