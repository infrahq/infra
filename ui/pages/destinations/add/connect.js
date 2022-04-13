import Head from 'next/head'
import Router from 'next/router'
import { useCallback, useState } from 'react'
import styled from 'styled-components'
import useSWR from 'swr'

import ActionButton from '../../../components/ActionButton'
import NameInput from '../../../components/destinations/NameInput'
import CommandInput from '../../../components/destinations/CommandInput'
import ExitButton from '../../../components/ExitButton'
import Header from '../../../components/Header'
import ConnectStatus from '../../../components/destinations/ConnectStatus'

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
  const [connected, setConnected] = useState(false)
  const [enabledCommandInput, setEnabledCommandInput] = useState(false)
  const [accessKey, setAccessKey] = useState('')
  const [currentDestinationName, setCurrentDestinationName] = useState('')

  const getDestinationsList = '/v1/destinations'
  const getDestinations = url => fetch(url).then(response => response.json())
  const { data: destinations } = useSWR(getDestinationsList, getDestinations)

  const updateConnectedStatus = useCallback((value) => {
    setConnected(value)
  })

  const updateAccessKey = useCallback((key) => {
    setAccessKey(key)
  })

  const updateCurrentDestinationName = useCallback((name) => {
    setCurrentDestinationName(name)
  })

  const updateEnabledCommandInputStatus = useCallback((status) => {
    setEnabledCommandInput(status)
  })

  const handleFinish = () => {
    Router.push({
      pathname: '/destinations/'
    }, undefined, { shallow: true })
  }

  return (
    <div>
      <Head>
        <title>Infra - Destinations</title>
      </Head>
      <ConnectContainer>
        <SetupDestinationContent>
          <Header
            header='Connect Your Kubernetes Cluster'
            subheader='Run the following command to connect your cluster'
          />
          <NameInput
            accessKey={accessKey}
            connected={connected}
            destinations={destinations}
            updateAccessKey={updateAccessKey}
            updateCurrentDestinationName={updateCurrentDestinationName}
            updateEnabledCommandInputStatus={updateEnabledCommandInputStatus}
            updateConnectedStatus={updateConnectedStatus}
          />
          <CommandInput
            enabledCommandInput={enabledCommandInput}
            accessKey={accessKey}
            currentDestinationName={currentDestinationName}
          />
          <ConnectStatus
            enabledCommandInput={enabledCommandInput}
            connected={connected}
          />
          {enabledCommandInput && <ActionButton disabled={!enabledCommandInput && !connected} onClick={() => handleFinish()} value='Finish' />}
        </SetupDestinationContent>
        <NavButton>
          <ExitButton previousPage='/destinations' />
        </NavButton>
      </ConnectContainer>
    </div>
  )
}

export default Connect
