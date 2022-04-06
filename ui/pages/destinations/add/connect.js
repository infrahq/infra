import Head from "next/head"
import Router from 'next/router'
import { useState } from "react"
import styled from 'styled-components'

import ActionButton from "../../../components/ActionButton"
import NameInput from "../../../components/destinations/NameInput"
import ExitButton from "../../../components/ExitButton"
import Header from "../../../components/Header"
import Input from "../../../components/Input"
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
	const [enabled, setEnabled] = useState(false)
	const [connected, setConnected] = useState(false)

	const handleFinish = async () => {
		await Router.push({
      pathname: '/destinations/'
    }, undefined, { shallow: true })
	}

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
					{/* <ActionButton disabled={!enabled && !connected} onClick={() => handleFinish()} value='Finish' /> */}
        </SetupDestinationContent>      
        <NavButton>
          <ExitButton previousPage='/destinations' />
        </NavButton>
      </ConnectContainer>
    </DestinationsContextProvider>
  )
}

export default Connect